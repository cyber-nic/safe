package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ndelorme/safe/internal/auth"
)

type statusResponse struct {
	Service string `json:"service"`
	Env     string `json:"env"`
	Status  string `json:"status"`
}

type sessionResponse struct {
	AccountID string `json:"accountId"`
	Env       string `json:"env"`
	Bucket    string `json:"bucket"`
	Endpoint  string `json:"endpoint"`
	Region    string `json:"region"`
}

type accountAccessRequest struct {
	AccountID      string   `json:"accountId"`
	DeviceID       string   `json:"deviceId"`
	AllowedActions []string `json:"allowedActions"`
}

type accountAccessResponse struct {
	Bucket     string                `json:"bucket"`
	Endpoint   string                `json:"endpoint"`
	Region     string                `json:"region"`
	KeyID      string                `json:"keyId"`
	Token      string                `json:"token"`
	Capability auth.AccessCapability `json:"capability"`
}

type serverConfig struct {
	env        string
	accountID  string
	deviceID   string
	bucket     string
	endpoint   string
	region     string
	accessTTL  time.Duration
	capability *auth.CapabilitySigner
	oauth      *auth.OAuthVerifier
}

func main() {
	capability, err := auth.NewCapabilitySigner(
		getenvDefault("SAFE_CONTROL_PLANE_KEY_ID", "dev-hmac-v1"),
		[]byte(getenvDefault("SAFE_CONTROL_PLANE_HMAC_SECRET", "0123456789abcdef0123456789abcdef")),
	)
	if err != nil {
		log.Fatal(err)
	}
	oauth, err := auth.NewOAuthVerifier(
		getenvDefault("SAFE_OAUTH_ISSUER", "https://auth.safe.local"),
		getenvDefault("SAFE_OAUTH_AUDIENCE", "safe-control-plane"),
		[]byte(getenvDefault("SAFE_OAUTH_HS256_SECRET", "0123456789abcdef0123456789abcdef")),
	)
	if err != nil {
		log.Fatal(err)
	}

	cfg := serverConfig{
		env:        getenvDefault("SAFE_ENV", "development"),
		accountID:  getenvDefault("SAFE_OAUTH_ACCOUNT_ID", getenvDefault("SAFE_DEV_ACCOUNT_ID", "acct-dev-001")),
		deviceID:   getenvDefault("SAFE_DEVICE_ID", getenvDefault("SAFE_DEV_DEVICE_ID", "dev-web-001")),
		bucket:     getenvDefault("SAFE_S3_BUCKET", "safe-dev"),
		endpoint:   getenvDefault("SAFE_S3_ENDPOINT", "http://localstack:4566"),
		region:     getenvDefault("AWS_REGION", "us-east-1"),
		accessTTL:  5 * time.Minute,
		capability: capability,
		oauth:      oauth,
	}

	addr := ":8080"
	log.Printf("safe control plane listening on %s", addr)
	if err := http.ListenAndServe(addr, newServer(cfg)); err != nil {
		log.Fatal(err)
	}
}

func newServer(cfg serverConfig) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, statusResponse{
			Service: "safe-control-plane",
			Env:     cfg.env,
			Status:  "ok",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, statusResponse{
			Service: "safe-control-plane",
			Env:     cfg.env,
			Status:  "healthy",
		})
	})
	mux.HandleFunc("/v1/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		identity, err := cfg.oauth.VerifyBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			writeOAuthError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, sessionResponse{
			AccountID: identity.AccountID,
			Env:       cfg.env,
			Bucket:    cfg.bucket,
			Endpoint:  cfg.endpoint,
			Region:    cfg.region,
		})
	})
	mux.HandleFunc("/v1/access/account", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var request accountAccessRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		identity, err := cfg.oauth.VerifyBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			writeOAuthError(w, err)
			return
		}
		if request.AccountID != identity.AccountID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := auth.ValidateActiveDevice(request.AccountID, request.DeviceID, cfg.deviceLookup); err != nil {
			writeAccessError(w, err)
			return
		}

		actions := request.AllowedActions
		if len(actions) == 0 {
			actions = []string{auth.ActionGet, auth.ActionPut}
		}

		signed, err := cfg.capability.IssueAccountCapability(auth.AccountAccessRequest{
			AccountID:      request.AccountID,
			DeviceID:       request.DeviceID,
			Bucket:         cfg.bucket,
			AllowedActions: actions,
			TTL:            cfg.accessTTL,
		})
		if err != nil {
			writeAccessError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, accountAccessResponse{
			Bucket:     cfg.bucket,
			Endpoint:   cfg.endpoint,
			Region:     cfg.region,
			KeyID:      signed.KeyID,
			Token:      signed.Token,
			Capability: signed.Capability,
		})
	})

	return mux
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"status":"encode_error"}`, http.StatusInternalServerError)
	}
}

func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func (cfg serverConfig) deviceLookup(accountID, deviceID string) (bool, error) {
	return accountID == cfg.accountID && deviceID == cfg.deviceID, nil
}

func writeAccessError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	case errors.Is(err, auth.ErrForbiddenAccess("device")):
		http.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, auth.ErrInvalidCapability("accountId")),
		errors.Is(err, auth.ErrInvalidCapability("deviceId")),
		errors.Is(err, auth.ErrInvalidCapability("bucket")),
		errors.Is(err, auth.ErrInvalidCapability("allowedActions")),
		errors.Is(err, auth.ErrInvalidCapability("ttl")):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func writeOAuthError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	case errors.Is(err, auth.ErrMissingOAuthToken()),
		errors.Is(err, auth.ErrExpiredOAuthToken()):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, auth.ErrInvalidOAuthToken("authorization")),
		errors.Is(err, auth.ErrInvalidOAuthToken("token")),
		errors.Is(err, auth.ErrInvalidOAuthToken("signature")),
		errors.Is(err, auth.ErrInvalidOAuthToken("issuer")),
		errors.Is(err, auth.ErrInvalidOAuthToken("audience")),
		errors.Is(err, auth.ErrInvalidOAuthToken("subject")),
		errors.Is(err, auth.ErrInvalidOAuthToken("accountId")),
		errors.Is(err, auth.ErrInvalidOAuthToken("env")),
		errors.Is(err, auth.ErrInvalidOAuthToken("issuedAt")),
		errors.Is(err, auth.ErrInvalidOAuthToken("expiresAt")):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
