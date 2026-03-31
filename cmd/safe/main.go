package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ndelorme/safe/internal/domain"
	"github.com/ndelorme/safe/internal/storage"
	safesync "github.com/ndelorme/safe/internal/sync"
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

type cliState struct {
	session       devSessionResponse
	storageConfig storageConfigResponse
	accountConfig domain.AccountConfigRecord
	head          domain.CollectionHeadRecord
	store         *storage.MemoryObjectStore
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		panic(err)
	}
}

func run(args []string, out io.Writer) error {
	state, err := bootstrapCLIState()
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "safe CLI bootstrap")
	fmt.Fprintf(out, "control plane bootstrap:\n- env=%s account=%s device=%s\n", state.session.Env, state.session.AccountID, state.session.DeviceID)
	fmt.Fprintf(out, "- storage bucket=%s region=%s endpoint=%s\n", state.storageConfig.Bucket, state.storageConfig.Region, state.storageConfig.Endpoint)

	if len(args) == 0 {
		return printOverview(out, state)
	}

	switch args[0] {
	case "secret":
		return runSecretCommand(out, state, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func bootstrapCLIState() (cliState, error) {
	baseURL := os.Getenv("SAFE_CONTROL_PLANE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	httpClient := &http.Client{Timeout: 5 * time.Second}
	session, err := fetchDevSession(httpClient, baseURL)
	if err != nil {
		return cliState{}, err
	}

	storageConfig, err := fetchStorageConfig(httpClient, baseURL)
	if err != nil {
		return cliState{}, err
	}

	store := storage.NewMemoryObjectStore()
	for _, record := range domain.StarterVaultItemRecords() {
		if _, err := storage.StoreItemRecord(store, session.AccountID, "vault-personal", record); err != nil {
			return cliState{}, err
		}
	}
	for _, record := range domain.StarterVaultEventRecords() {
		if _, err := storage.StoreEventRecord(store, record); err != nil {
			return cliState{}, err
		}
	}
	if _, err := storage.StoreAccountConfigRecord(store, domain.StarterAccountConfigRecord()); err != nil {
		return cliState{}, err
	}
	if _, err := storage.StoreCollectionHeadRecord(store, domain.StarterCollectionHeadRecord()); err != nil {
		return cliState{}, err
	}

	accountConfig, err := storage.LoadAccountConfigRecord(store, session.AccountID)
	if err != nil {
		return cliState{}, err
	}
	head, err := storage.LoadCollectionHeadRecord(store, session.AccountID, accountConfig.DefaultCollectionID)
	if err != nil {
		return cliState{}, err
	}

	return cliState{
		session:       session,
		storageConfig: storageConfig,
		accountConfig: accountConfig,
		head:          head,
		store:         store,
	}, nil
}

func printOverview(out io.Writer, state cliState) error {
	storedEvents, err := storage.LoadCollectionEventRecords(state.store, state.session.AccountID, state.accountConfig.DefaultCollectionID)
	if err != nil {
		return err
	}

	projection, err := safesync.ReplayCollection(storedEvents)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "sync replay:")
	fmt.Fprintf(out, "- account=%s defaultCollection=%s latestSeq=%d items=%d headEvent=%s\n", state.accountConfig.AccountID, state.accountConfig.DefaultCollectionID, projection.LatestSeq, len(projection.Items), state.head.LatestEventID)
	return nil
}

func runSecretCommand(out io.Writer, state cliState, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("secret command requires a subcommand")
	}

	switch args[0] {
	case "list":
		return secretList(out, state)
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret delete <item-id>")
		}
		return secretDelete(out, state, args[1])
	case "update":
		if len(args) < 4 {
			return fmt.Errorf("usage: safe secret update <item-id> <title> <username>")
		}
		return secretUpdate(out, state, args[1], args[2], args[3])
	case "search":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret search <query>")
		}
		return secretSearch(out, state, strings.Join(args[1:], " "))
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret show <item-id>")
		}
		return secretShow(out, state, args[1])
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("usage: safe secret add <title> <username>")
		}
		return secretAdd(out, state, args[1], args[2])
	default:
		return fmt.Errorf("unknown secret subcommand: %s", args[0])
	}
}

func secretList(out io.Writer, state cliState) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(projection.Items))
	for id := range projection.Items {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	fmt.Fprintln(out, "secret list:")
	for _, id := range ids {
		item := projection.Items[id].Item
		fmt.Fprintf(out, "- %s (%s)\n", item.Title, item.Username)
	}

	return nil
}

func secretSearch(out io.Writer, state cliState, query string) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(projection.Items))
	for id, record := range projection.Items {
		if matchesSecretQuery(record.Item, query) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	fmt.Fprintf(out, "secret search: query=%q\n", query)
	if len(ids) == 0 {
		fmt.Fprintln(out, "- no matches")
		return nil
	}

	for _, id := range ids {
		item := projection.Items[id].Item
		fmt.Fprintf(out, "- id=%s kind=%s title=%s\n", item.ID, item.Kind, item.Title)
	}

	return nil
}

func secretShow(out io.Writer, state cliState, itemID string) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	record, ok := projection.Items[itemID]
	if !ok {
		return fmt.Errorf("secret not found: %s", itemID)
	}

	item := record.Item
	fmt.Fprintln(out, "secret show:")
	fmt.Fprintf(out, "- id=%s kind=%s title=%s\n", item.ID, item.Kind, item.Title)
	if len(item.Tags) > 0 {
		fmt.Fprintf(out, "- tags=%s\n", strings.Join(item.Tags, ","))
	}

	switch item.Kind {
	case domain.VaultItemKindLogin:
		fmt.Fprintf(out, "- username=%s\n", item.Username)
		fmt.Fprintf(out, "- urls=%s\n", strings.Join(item.URLs, ","))
	case domain.VaultItemKindTOTP:
		fmt.Fprintf(out, "- issuer=%s account=%s digits=%d period=%d algorithm=%s secretRef=%s\n", item.Issuer, item.AccountName, item.Digits, item.PeriodSeconds, item.Algorithm, item.SecretRef)
	}

	return nil
}

func secretAdd(out io.Writer, state cliState, title, username string) error {
	itemID := fmt.Sprintf("login-%s-primary", slugify(title))
	itemRecord := domain.VaultItemRecord{
		SchemaVersion: 1,
		Item: domain.VaultItem{
			ID:       itemID,
			Kind:     domain.VaultItemKindLogin,
			Title:    title,
			Tags:     []string{"manual", "password"},
			Username: username,
			URLs:     []string{"https://example.invalid/login"},
		},
	}

	projection, newEvent, err := persistItemMutation(state, itemRecord, "2026-03-31T10:02:00Z")
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "secret add:")
	fmt.Fprintf(out, "- added=%s username=%s event=%s latestSeq=%d items=%d\n", title, username, newEvent.EventID, projection.LatestSeq, len(projection.Items))
	return nil
}

func secretUpdate(out io.Writer, state cliState, itemID, title, username string) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	record, ok := projection.Items[itemID]
	if !ok {
		return fmt.Errorf("secret not found: %s", itemID)
	}
	if record.Item.Kind != domain.VaultItemKindLogin {
		return fmt.Errorf("secret update only supports login items: %s", itemID)
	}

	updated := record
	updated.Item.Title = title
	updated.Item.Username = username

	projection, newEvent, err := persistItemMutation(state, updated, "2026-03-31T10:03:00Z")
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "secret update:")
	fmt.Fprintf(out, "- id=%s title=%s username=%s event=%s latestSeq=%d\n", itemID, title, username, newEvent.EventID, projection.LatestSeq)
	return nil
}

func secretDelete(out io.Writer, state cliState, itemID string) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	if _, ok := projection.Items[itemID]; !ok {
		return fmt.Errorf("secret not found: %s", itemID)
	}

	projection, newEvent, err := persistDeleteMutation(state, itemID, "2026-03-31T10:04:00Z")
	if err != nil {
		return err
	}

	fmt.Fprintln(out, "secret delete:")
	fmt.Fprintf(out, "- id=%s event=%s latestSeq=%d items=%d\n", itemID, newEvent.EventID, projection.LatestSeq, len(projection.Items))
	return nil
}

func loadProjection(state cliState) (safesync.Projection, error) {
	storedEvents, err := storage.LoadCollectionEventRecords(state.store, state.session.AccountID, state.accountConfig.DefaultCollectionID)
	if err != nil {
		return safesync.Projection{}, err
	}

	return safesync.ReplayCollection(storedEvents)
}

func persistItemMutation(state cliState, itemRecord domain.VaultItemRecord, occurredAt string) (safesync.Projection, domain.VaultEventRecord, error) {
	newEvent, newHead, err := safesync.BuildPutItemMutation(state.head, state.session.DeviceID, itemRecord, occurredAt)
	if err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}

	if _, err := storage.StoreItemRecord(state.store, state.session.AccountID, state.accountConfig.DefaultCollectionID, itemRecord); err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}
	if _, err := storage.StoreEventRecord(state.store, newEvent); err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}
	if _, err := storage.StoreCollectionHeadRecord(state.store, newHead); err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}

	projection, err := loadProjection(state)
	if err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}

	return projection, newEvent, nil
}

func persistDeleteMutation(state cliState, itemID, occurredAt string) (safesync.Projection, domain.VaultEventRecord, error) {
	newEvent, newHead, err := safesync.BuildDeleteItemMutation(state.head, state.session.DeviceID, itemID, occurredAt)
	if err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}

	if _, err := storage.StoreEventRecord(state.store, newEvent); err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}
	if _, err := storage.StoreCollectionHeadRecord(state.store, newHead); err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}

	projection, err := loadProjection(state)
	if err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}

	return projection, newEvent, nil
}

func matchesSecretQuery(item domain.VaultItem, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return false
	}

	fields := []string{
		item.ID,
		item.Title,
		item.Username,
		item.BodyPreview,
		item.Service,
		item.Host,
		item.Issuer,
		item.AccountName,
	}
	fields = append(fields, item.Tags...)
	fields = append(fields, item.URLs...)

	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}

	return false
}

func slugify(value string) string {
	buffer := make([]rune, 0, len(value))
	for _, char := range value {
		switch {
		case char >= 'A' && char <= 'Z':
			buffer = append(buffer, char+('a'-'A'))
		case (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9'):
			buffer = append(buffer, char)
		case char == ' ' || char == '-' || char == '_':
			buffer = append(buffer, '-')
		}
	}

	if len(buffer) == 0 {
		return "item"
	}

	return string(buffer)
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
