package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ndelorme/safe/internal/domain"
	"github.com/ndelorme/safe/internal/storage"
)

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
	fmt.Println("safe CLI bootstrap")
	if err := printControlPlaneBootstrap(); err != nil {
		panic(err)
	}

	fmt.Println("supported starter items:")

	for _, item := range domain.StarterVaultItems() {
		fmt.Printf("- [%s] %s: %s\n", item.Kind, item.Title, item.Description)
	}

	fmt.Println("canonical starter records:")

	for _, record := range domain.StarterVaultItemRecords() {
		canonical, err := record.CanonicalJSON()
		if err != nil {
			panic(err)
		}

		fmt.Printf("- %s\n", canonical)
	}

	fmt.Println("canonical starter events:")

	for _, record := range domain.StarterVaultEventRecords() {
		canonical, err := record.CanonicalJSON()
		if err != nil {
			panic(err)
		}

		fmt.Printf("- %s\n", canonical)
	}
}

func printControlPlaneBootstrap() error {
	baseURL := os.Getenv("SAFE_CONTROL_PLANE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	session, err := fetchDevSession(httpClient, baseURL)
	if err != nil {
		return err
	}

	storageConfig, err := fetchStorageConfig(httpClient, baseURL)
	if err != nil {
		return err
	}

	fmt.Println("control plane bootstrap:")
	fmt.Printf("- env=%s account=%s device=%s\n", session.Env, session.AccountID, session.DeviceID)
	fmt.Printf("- storage bucket=%s region=%s endpoint=%s\n", storageConfig.Bucket, storageConfig.Region, storageConfig.Endpoint)
	fmt.Println("storage plan:")

	for _, event := range domain.StarterVaultEventRecords() {
		fmt.Printf("- event %s\n", storage.EventObjectKey(session.AccountID, event.CollectionID, event.EventID))
		fmt.Printf("- item  %s\n", storage.ItemObjectKey(session.AccountID, event.CollectionID, event.ItemRecord.Item.ID))
	}

	return nil
}

func fetchDevSession(httpClient *http.Client, baseURL string) (devSessionResponse, error) {
	request, err := http.NewRequest(http.MethodPost, baseURL+"/v1/dev/session", bytes.NewReader([]byte("{}")))
	if err != nil {
		return devSessionResponse{}, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		return devSessionResponse{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return devSessionResponse{}, fmt.Errorf("control plane session request failed: %s", response.Status)
	}

	var payload devSessionResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return devSessionResponse{}, err
	}

	if payload.Env == "" || payload.AccountID == "" || payload.DeviceID == "" {
		return devSessionResponse{}, fmt.Errorf("control plane session response incomplete; restart the control-plane service and verify /v1/dev/session")
	}

	return payload, nil
}

func fetchStorageConfig(httpClient *http.Client, baseURL string) (storageConfigResponse, error) {
	response, err := httpClient.Get(baseURL + "/v1/dev/storage-config")
	if err != nil {
		return storageConfigResponse{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return storageConfigResponse{}, fmt.Errorf("control plane storage config request failed: %s", response.Status)
	}

	var payload storageConfigResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return storageConfigResponse{}, err
	}

	if payload.Bucket == "" || payload.Region == "" || payload.Endpoint == "" {
		return storageConfigResponse{}, fmt.Errorf("control plane storage config response incomplete; restart the control-plane service and verify /v1/dev/storage-config")
	}

	return payload, nil
}
