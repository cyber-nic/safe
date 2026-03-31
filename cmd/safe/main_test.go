package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestFetchDevSession(t *testing.T) {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v1/dev/session" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", r.Method)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"accountId":"acct-test-001","deviceId":"dev-test-001","env":"test"}`)),
			}, nil
		}),
	}
	session, err := fetchDevSession(client, "http://control-plane.test")
	if err != nil {
		t.Fatalf("fetch dev session: %v", err)
	}

	if session.AccountID != "acct-test-001" || session.DeviceID != "dev-test-001" || session.Env != "test" {
		t.Fatalf("unexpected session payload: %+v", session)
	}
}

func TestFetchDevSessionRejectsIncompletePayload(t *testing.T) {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"service":"safe-control-plane","env":"development","status":"ok"}`)),
			}, nil
		}),
	}
	_, err := fetchDevSession(client, "http://control-plane.test")
	if err == nil {
		t.Fatal("expected incomplete payload error")
	}

	if !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("expected incomplete payload error, got %v", err)
	}
}

func TestFetchStorageConfig(t *testing.T) {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v1/dev/storage-config" {
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected method: %s", r.Method)
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"bucket":"safe-test","endpoint":"http://localstack:4566","region":"us-east-1","accountId":"acct-test-001","deviceId":"dev-test-001"}`)),
			}, nil
		}),
	}
	config, err := fetchStorageConfig(client, "http://control-plane.test")
	if err != nil {
		t.Fatalf("fetch storage config: %v", err)
	}

	if config.Bucket != "safe-test" || config.Endpoint != "http://localstack:4566" || config.Region != "us-east-1" {
		t.Fatalf("unexpected storage payload: %+v", config)
	}
}

func TestRunSecretList(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "list"}, &buffer); err != nil {
			t.Fatalf("run secret list: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, "secret list:") {
			t.Fatalf("expected secret list output, got %s", output)
		}
		if !strings.Contains(output, "Gmail (alice@example.com)") {
			t.Fatalf("expected Gmail entry, got %s", output)
		}
	})
}

func TestRunSecretAdd(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "add", "GitHub", "alice"}, &buffer); err != nil {
			t.Fatalf("run secret add: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, "secret add:") {
			t.Fatalf("expected secret add output, got %s", output)
		}
		if !strings.Contains(output, "added=GitHub") {
			t.Fatalf("expected GitHub add output, got %s", output)
		}
		if !strings.Contains(output, "latestSeq=3") {
			t.Fatalf("expected latestSeq=3 after add, got %s", output)
		}
	})
}

func TestRunSecretShow(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "show", "login-gmail-primary"}, &buffer); err != nil {
			t.Fatalf("run secret show: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, "secret show:") {
			t.Fatalf("expected secret show output, got %s", output)
		}
		if !strings.Contains(output, "id=login-gmail-primary kind=login title=Gmail") {
			t.Fatalf("expected Gmail identity output, got %s", output)
		}
		if !strings.Contains(output, "username=alice@example.com") {
			t.Fatalf("expected Gmail username output, got %s", output)
		}
	})
}

func TestRunSecretShowMissingItem(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		err := run([]string{"secret", "show", "missing-item"}, &buffer)
		if err == nil {
			t.Fatal("expected missing item error")
		}

		if !strings.Contains(err.Error(), "secret not found: missing-item") {
			t.Fatalf("expected missing item error, got %v", err)
		}
	})
}

func withFakeBootstrap(t *testing.T, fn func()) {
	t.Helper()

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1/dev/session":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"accountId":"acct-dev-001","deviceId":"dev-web-001","env":"test"}`)),
			}, nil
		case "/v1/dev/storage-config":
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"bucket":"safe-test","endpoint":"http://localstack:4566","region":"us-east-1","accountId":"acct-dev-001","deviceId":"dev-web-001"}`)),
			}, nil
		default:
			t.Fatalf("unexpected bootstrap path: %s", r.URL.Path)
			return nil, nil
		}
	})
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	previousURL := os.Getenv("SAFE_CONTROL_PLANE_URL")
	if err := os.Setenv("SAFE_CONTROL_PLANE_URL", "http://control-plane.test"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer func() {
		if previousURL == "" {
			_ = os.Unsetenv("SAFE_CONTROL_PLANE_URL")
		} else {
			_ = os.Setenv("SAFE_CONTROL_PLANE_URL", previousURL)
		}
	}()

	fn()
}
