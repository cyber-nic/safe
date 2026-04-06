package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ndelorme/safe/internal/auth"
)

func TestDevSessionEndpoint(t *testing.T) {
	server := newServer(newTestServerConfig(t))

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
	server := newServer(newTestServerConfig(t))

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
	server := newServer(newTestServerConfig(t))
	request := httptest.NewRequest(http.MethodGet, "/v1/dev/session", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, response.Code)
	}
}

func TestAccountAccessEndpoint(t *testing.T) {
	server := newServer(newTestServerConfig(t))
	request := httptest.NewRequest(http.MethodPost, "/v1/access/account", strings.NewReader(`{"accountId":"acct-test-001","deviceId":"dev-test-001"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var payload accountAccessResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Capability.Prefix != "accounts/acct-test-001/" {
		t.Fatalf("unexpected prefix: %s", payload.Capability.Prefix)
	}
	if len(payload.Capability.AllowedActions) != 2 || payload.Capability.AllowedActions[0] != auth.ActionGet || payload.Capability.AllowedActions[1] != auth.ActionPut {
		t.Fatalf("unexpected actions: %#v", payload.Capability.AllowedActions)
	}
}

func TestAccountAccessEndpointRejectsMismatchedIdentity(t *testing.T) {
	server := newServer(newTestServerConfig(t))
	request := httptest.NewRequest(http.MethodPost, "/v1/access/account", strings.NewReader(`{"accountId":"acct-other-001","deviceId":"dev-test-001"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, response.Code)
	}
}

func TestAccountAccessEndpointAllowsExplicitList(t *testing.T) {
	server := newServer(newTestServerConfig(t))
	request := httptest.NewRequest(http.MethodPost, "/v1/access/account", strings.NewReader(`{"accountId":"acct-test-001","deviceId":"dev-test-001","allowedActions":["list","get"]}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}

	var payload accountAccessResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Capability.AllowedActions) != 2 || payload.Capability.AllowedActions[0] != auth.ActionGet || payload.Capability.AllowedActions[1] != auth.ActionList {
		t.Fatalf("unexpected actions: %#v", payload.Capability.AllowedActions)
	}
}

func newTestServerConfig(t *testing.T) serverConfig {
	t.Helper()

	capability, err := auth.NewCapabilitySigner("test-key", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	capability.SetNowForTest(func() time.Time {
		return time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC)
	})

	return serverConfig{
		env:        "test",
		accountID:  "acct-test-001",
		deviceID:   "dev-test-001",
		bucket:     "safe-test",
		endpoint:   "http://localstack:4566",
		region:     "us-east-1",
		accessTTL:  5 * time.Minute,
		capability: capability,
	}
}
