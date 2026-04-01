package crypto

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

func GenerateTOTP(secretBase32 string, at time.Time, digits, periodSeconds int, algorithm string) (string, error) {
	if strings.ToUpper(algorithm) != "SHA1" {
		return "", fmt.Errorf("unsupported totp algorithm: %s", algorithm)
	}
	if digits < 1 || digits > 10 {
		return "", fmt.Errorf("invalid totp digits: %d", digits)
	}
	if periodSeconds < 1 {
		return "", fmt.Errorf("invalid totp period: %d", periodSeconds)
	}

	decoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	secret, err := decoder.DecodeString(strings.ToUpper(secretBase32))
	if err != nil {
		return "", fmt.Errorf("decode totp secret: %w", err)
	}

	counter := uint64(at.Unix()) / uint64(periodSeconds)
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)

	mac := hmac.New(sha1.New, secret)
	if _, err := mac.Write(message[:]); err != nil {
		return "", fmt.Errorf("hash totp counter: %w", err)
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	code := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)

	modulus := int(math.Pow10(digits))
	value := code % modulus
	return fmt.Sprintf("%0*d", digits, value), nil
}
