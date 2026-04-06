package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// S3ObjectStoreWithCAS is an S3-backed ObjectStoreWithCAS. All objects live
// under a single bucket. CAS is implemented using S3 conditional writes:
//   - PutIfMatch("", ...) uses If-None-Match: * (object must not yet exist)
//   - PutIfMatch(etag, ...) uses If-Match: "<etag>" (must match current ETag)
//
// ETags are returned without surrounding quotes.
type S3ObjectStoreWithCAS struct {
	client *s3.Client
	bucket string
}

// S3Config holds the parameters needed to connect to an S3-compatible endpoint.
type S3Config struct {
	Bucket          string
	Region          string
	Endpoint        string // optional; set for LocalStack or other S3-compatible services
	AccessKeyID     string // optional; defaults to credential chain
	SecretAccessKey string // optional; defaults to credential chain
}

// NewS3ObjectStoreWithCAS creates an S3ObjectStoreWithCAS. All fields in cfg
// are optional except Bucket; missing credentials fall through to the standard
// AWS credential chain (env vars, ~/.aws/credentials, EC2 instance profile).
func NewS3ObjectStoreWithCAS(ctx context.Context, cfg S3Config) (*S3ObjectStoreWithCAS, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	if cfg.AccessKeyID != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("s3 store: load aws config: %w", err)
	}

	s3Opts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // required for LocalStack and most S3-compatible services
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)
	return &S3ObjectStoreWithCAS{client: client, bucket: cfg.Bucket}, nil
}

// Put unconditionally writes value at key.
func (s *S3ObjectStoreWithCAS) Put(key string, value []byte) error {
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(value),
	})
	if err != nil {
		return fmt.Errorf("s3 store: put %q: %w", key, err)
	}
	return nil
}

// Get retrieves the value at key, returning ErrObjectNotFound if absent.
func (s *S3ObjectStoreWithCAS) Get(key string) ([]byte, error) {
	out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, ErrObjectNotFound(key)
		}
		return nil, fmt.Errorf("s3 store: get %q: %w", key, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 store: read body %q: %w", key, err)
	}
	return data, nil
}

// List returns all keys with the given prefix, sorted lexicographically.
func (s *S3ObjectStoreWithCAS) List(prefix string) ([]string, error) {
	var keys []string
	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			return nil, fmt.Errorf("s3 store: list %q: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
	}
	return keys, nil
}

// GetWithETag returns the value and its ETag (without surrounding quotes).
// Returns ErrObjectNotFound if the key does not exist.
func (s *S3ObjectStoreWithCAS) GetWithETag(key string) ([]byte, string, error) {
	out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, "", ErrObjectNotFound(key)
		}
		return nil, "", fmt.Errorf("s3 store: get-with-etag %q: %w", key, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, "", fmt.Errorf("s3 store: read body %q: %w", key, err)
	}
	etag := unquoteETag(aws.ToString(out.ETag))
	return data, etag, nil
}

// PutIfMatch writes value at key only when the current ETag matches expectedETag.
// Pass expectedETag="" to create an object that must not already exist.
// Returns the new ETag on success, or ErrCASConflict on mismatch.
func (s *S3ObjectStoreWithCAS) PutIfMatch(key string, value []byte, expectedETag string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(value),
	}

	if expectedETag == "" {
		input.IfNoneMatch = aws.String("*")
	} else {
		input.IfMatch = aws.String(quoteETag(expectedETag))
	}

	out, err := s.client.PutObject(context.Background(), input)
	if err != nil {
		if isS3PreconditionFailed(err) {
			return "", fmt.Errorf("%w: key %s", ErrCASConflict, key)
		}
		return "", fmt.Errorf("s3 store: put-if-match %q: %w", key, err)
	}
	return unquoteETag(aws.ToString(out.ETag)), nil
}

// unquoteETag strips the surrounding double-quotes that S3 includes in ETags.
// If the ETag is not quoted it is returned as-is.
func unquoteETag(etag string) string {
	return strings.Trim(etag, `"`)
}

// quoteETag wraps an ETag in double-quotes as S3 expects in If-Match headers.
// If the ETag is already quoted it is returned as-is.
func quoteETag(etag string) string {
	if strings.HasPrefix(etag, `"`) {
		return etag
	}
	return `"` + etag + `"`
}

// isS3NotFound returns true when err represents an S3 NoSuchKey or 404.
func isS3NotFound(err error) bool {
	var noSuchKey *types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return true
	}
	// S3-compatible services (including LocalStack) sometimes return a plain 404
	// without a structured NoSuchKey body.
	var httpErr *smithyhttp.ResponseError
	if errors.As(err, &httpErr) && httpErr.HTTPStatusCode() == 404 {
		return true
	}
	return false
}

// isS3PreconditionFailed returns true when err is a 412 Precondition Failed,
// which S3 returns when an If-Match or If-None-Match condition is not met.
func isS3PreconditionFailed(err error) bool {
	var httpErr *smithyhttp.ResponseError
	if errors.As(err, &httpErr) && httpErr.HTTPStatusCode() == 412 {
		return true
	}
	return false
}
