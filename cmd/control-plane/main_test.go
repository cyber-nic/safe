package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDevSessionEndpoint(t *testing.T) {
	server := newServer(serverConfig{
		env:       "test",
		accountID: "acct-test-001",
		deviceID:  "dev-test-001",
		bucket:    "safe-test",
		endpoint:  "http://localstack:4566",
		region:    "us-east-1",
	})

	request := httptest.NewRequest(http.MethodPost, "/v1/dev/session", strings.NewReader(`{}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var payload devSessionResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.AccountID != "acct-test-001" || payload.DeviceID != "dev-test-001" || payload.Env != "test" {
		t.Fatalf("unexpected session payload: %+v", payload)
	}
}

func TestStorageConfigEndpoint(t *testing.T) {
	server := newServer(serverConfig{
		env:       "test",
		accountID: "acct-test-001",
		deviceID:  "dev-test-001",
		bucket:    "safe-test",
		endpoint:  "http://localstack:4566",
		region:    "us-east-1",
	})

	request := httptest.NewRequest(http.MethodGet, "/v1/dev/storage-config", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var payload storageConfigResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Bucket != "safe-test" || payload.Endpoint != "http://localstack:4566" || payload.Region != "us-east-1" {
		t.Fatalf("unexpected storage payload: %+v", payload)
	}
}

func TestDevSessionMethodGuard(t *testing.T) {
	server := newServer(serverConfig{})
	request := httptest.NewRequest(http.MethodGet, "/v1/dev/session", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, response.Code)
	}
}
