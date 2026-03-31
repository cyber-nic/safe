package main

import (
	"io"
	"net/http"
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
