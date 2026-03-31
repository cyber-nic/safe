package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type statusResponse struct {
	Service string `json:"service"`
	Env     string `json:"env"`
	Status  string `json:"status"`
}

type devSessionResponse struct {
	AccountID string `json:"accountId"`
	DeviceID  string `json:"deviceId"`
	Env       string `json:"env"`
}

type storageConfigResponse struct {
	Bucket    string `json:"bucket"`
	Endpoint  string `json:"endpoint"`
	Region    string `json:"region"`
	AccountID string `json:"accountId"`
	DeviceID  string `json:"deviceId"`
}

func main() {
	env := os.Getenv("SAFE_ENV")
	if env == "" {
		env = "development"
	}

	accountID := getenvDefault("SAFE_DEV_ACCOUNT_ID", "acct-dev-001")
	deviceID := getenvDefault("SAFE_DEV_DEVICE_ID", "dev-web-001")
	bucket := getenvDefault("SAFE_S3_BUCKET", "safe-dev")
	endpoint := getenvDefault("SAFE_S3_ENDPOINT", "http://localstack:4566")
	region := getenvDefault("AWS_REGION", "us-east-1")

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, statusResponse{
			Service: "safe-control-plane",
			Env:     env,
			Status:  "ok",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, statusResponse{
			Service: "safe-control-plane",
			Env:     env,
			Status:  "healthy",
		})
	})
	mux.HandleFunc("/v1/dev/session", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, http.StatusOK, devSessionResponse{
			AccountID: accountID,
			DeviceID:  deviceID,
			Env:       env,
		})
	})
	mux.HandleFunc("/v1/dev/storage-config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, http.StatusOK, storageConfigResponse{
			Bucket:    bucket,
			Endpoint:  endpoint,
			Region:    region,
			AccountID: accountID,
			DeviceID:  deviceID,
		})
	})

	addr := ":8080"
	log.Printf("safe control plane listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
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
