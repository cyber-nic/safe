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

	safecrypto "github.com/ndelorme/safe/internal/crypto"
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

type cliOptions struct {
	json bool
}

var nowFunc = time.Now

func main() {
	if err := runWithIO(os.Args[1:], os.Stdin, os.Stdout); err != nil {
		panic(err)
	}
}

func run(args []string, out io.Writer) error {
	return runWithIO(args, os.Stdin, out)
}

func runWithIO(args []string, in io.Reader, out io.Writer) error {
	options, args, err := parseCLIOptions(args)
	if err != nil {
		return err
	}

	state, err := bootstrapCLIState()
	if err != nil {
		return err
	}

	if !options.json {
		fmt.Fprintln(out, "safe CLI bootstrap")
		fmt.Fprintf(out, "control plane bootstrap:\n- env=%s account=%s device=%s\n", state.session.Env, state.session.AccountID, state.session.DeviceID)
		fmt.Fprintf(out, "- storage bucket=%s region=%s endpoint=%s\n", state.storageConfig.Bucket, state.storageConfig.Region, state.storageConfig.Endpoint)
	}

	if len(args) == 0 {
		return printOverview(out, state, options)
	}

	switch args[0] {
	case "secret":
		return runSecretCommand(in, out, state, options, args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func parseCLIOptions(args []string) (cliOptions, []string, error) {
	options := cliOptions{}
	filtered := make([]string, 0, len(args))

	for _, arg := range args {
		switch arg {
		case "--json":
			options.json = true
		default:
			filtered = append(filtered, arg)
		}
	}

	return options, filtered, nil
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
	if _, err := storage.StoreSecretMaterial(store, session.AccountID, "vault-personal", "vault-secret://totp/gmail-primary", "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"); err != nil {
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

func printOverview(out io.Writer, state cliState, options cliOptions) error {
	head, projection, err := loadVerifiedState(state)
	if err != nil {
		return err
	}

	if options.json {
		return writeJSON(out, struct {
			AccountID         string `json:"accountId"`
			DefaultCollection string `json:"defaultCollection"`
			LatestSeq         int    `json:"latestSeq"`
			ItemCount         int    `json:"itemCount"`
			HeadEventID       string `json:"headEventId"`
		}{
			AccountID:         state.accountConfig.AccountID,
			DefaultCollection: state.accountConfig.DefaultCollectionID,
			LatestSeq:         projection.LatestSeq,
			ItemCount:         len(projection.Items),
			HeadEventID:       head.LatestEventID,
		})
	}

	fmt.Fprintln(out, "sync replay:")
	fmt.Fprintf(out, "- account=%s defaultCollection=%s latestSeq=%d items=%d headEvent=%s\n", state.accountConfig.AccountID, state.accountConfig.DefaultCollectionID, projection.LatestSeq, len(projection.Items), head.LatestEventID)
	return nil
}

func runSecretCommand(in io.Reader, out io.Writer, state cliState, options cliOptions, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("secret command requires a subcommand")
	}

	switch args[0] {
	case "list":
		return secretList(out, state, options)
	case "import":
		if len(args) != 1 {
			return fmt.Errorf("usage: safe secret import")
		}
		return secretImport(in, out, state, options)
	case "export":
		if len(args) > 2 {
			return fmt.Errorf("usage: safe secret export [item-id]")
		}
		itemID := ""
		if len(args) == 2 {
			itemID = args[1]
		}
		return secretExport(out, state, itemID)
	case "history":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret history <item-id>")
		}
		return secretHistory(out, state, options, args[1])
	case "restore":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret restore <item-id>")
		}
		return secretRestore(out, state, options, args[1])
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret delete <item-id>")
		}
		return secretDelete(out, state, options, args[1])
	case "update":
		if len(args) < 4 {
			return fmt.Errorf("usage: safe secret update <item-id> <title> <username>")
		}
		return secretUpdate(out, state, options, args[1], args[2], args[3])
	case "search":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret search <query>")
		}
		return secretSearch(out, state, options, strings.Join(args[1:], " "))
	case "show":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret show <item-id>")
		}
		return secretShow(out, state, options, args[1])
	case "code":
		if len(args) < 2 {
			return fmt.Errorf("usage: safe secret code <item-id>")
		}
		return secretCode(out, state, options, args[1], nowFunc().UTC())
	case "add":
		if len(args) < 3 {
			return fmt.Errorf("usage: safe secret add <title> <username>")
		}
		return secretAdd(out, state, options, args[1], args[2])
	default:
		return fmt.Errorf("unknown secret subcommand: %s", args[0])
	}
}

func secretList(out io.Writer, state cliState, options cliOptions) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(projection.Items))
	for id := range projection.Items {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	if options.json {
		type listEntry struct {
			ID       string               `json:"id"`
			Kind     domain.VaultItemKind `json:"kind"`
			Title    string               `json:"title"`
			Username string               `json:"username,omitempty"`
		}

		items := make([]listEntry, 0, len(ids))
		for _, id := range ids {
			item := projection.Items[id].Item
			items = append(items, listEntry{
				ID:       item.ID,
				Kind:     item.Kind,
				Title:    item.Title,
				Username: item.Username,
			})
		}

		return writeJSON(out, struct {
			Items []listEntry `json:"items"`
		}{Items: items})
	}

	fmt.Fprintln(out, "secret list:")
	for _, id := range ids {
		item := projection.Items[id].Item
		fmt.Fprintf(out, "- %s (%s)\n", item.Title, item.Username)
	}

	return nil
}

func secretSearch(out io.Writer, state cliState, options cliOptions, query string) error {
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

	if options.json {
		items := make([]domain.VaultItemSummary, 0, len(ids))
		for _, id := range ids {
			items = append(items, projection.Items[id].Item.Summary())
		}

		return writeJSON(out, struct {
			Query string                    `json:"query"`
			Items []domain.VaultItemSummary `json:"items"`
		}{
			Query: query,
			Items: items,
		})
	}

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

func secretShow(out io.Writer, state cliState, options cliOptions, itemID string) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	record, ok := projection.Items[itemID]
	if !ok {
		return fmt.Errorf("secret not found: %s", itemID)
	}

	if options.json {
		return writeJSON(out, record)
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
	case domain.VaultItemKindNote:
		fmt.Fprintf(out, "- bodyPreview=%s\n", item.BodyPreview)
	case domain.VaultItemKindAPIKey:
		fmt.Fprintf(out, "- service=%s\n", item.Service)
	case domain.VaultItemKindSSHKey:
		fmt.Fprintf(out, "- username=%s\n", item.Username)
		fmt.Fprintf(out, "- host=%s\n", item.Host)
	case domain.VaultItemKindTOTP:
		fmt.Fprintf(out, "- issuer=%s account=%s digits=%d period=%d algorithm=%s secretRef=%s\n", item.Issuer, item.AccountName, item.Digits, item.PeriodSeconds, item.Algorithm, item.SecretRef)
	}

	return nil
}

func secretCode(out io.Writer, state cliState, options cliOptions, itemID string, at time.Time) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	record, ok := projection.Items[itemID]
	if !ok {
		return fmt.Errorf("secret not found: %s", itemID)
	}
	if record.Item.Kind != domain.VaultItemKindTOTP {
		return fmt.Errorf("secret code only supports totp items: %s", itemID)
	}

	secret, err := storage.LoadSecretMaterial(state.store, state.session.AccountID, state.accountConfig.DefaultCollectionID, record.Item.SecretRef)
	if err != nil {
		return fmt.Errorf("secret code secret material not found: %s", record.Item.SecretRef)
	}

	code, err := safecrypto.GenerateTOTP(secret, at, record.Item.Digits, record.Item.PeriodSeconds, record.Item.Algorithm)
	if err != nil {
		return err
	}

	generatedAt := at.UTC()
	expiresAt := generatedAt.Truncate(time.Duration(record.Item.PeriodSeconds) * time.Second).Add(time.Duration(record.Item.PeriodSeconds) * time.Second)

	if options.json {
		return writeJSON(out, struct {
			ItemID        string `json:"itemId"`
			Title         string `json:"title"`
			Code          string `json:"code"`
			GeneratedAt   string `json:"generatedAt"`
			ExpiresAt     string `json:"expiresAt"`
			PeriodSeconds int    `json:"periodSeconds"`
		}{
			ItemID:        itemID,
			Title:         record.Item.Title,
			Code:          code,
			GeneratedAt:   generatedAt.Format(time.RFC3339),
			ExpiresAt:     expiresAt.Format(time.RFC3339),
			PeriodSeconds: record.Item.PeriodSeconds,
		})
	}

	fmt.Fprintln(out, "secret code:")
	fmt.Fprintf(out, "- id=%s title=%s code=%s generatedAt=%s expiresAt=%s\n", itemID, record.Item.Title, code, generatedAt.Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	return nil
}

func secretImport(in io.Reader, out io.Writer, state cliState, options cliOptions) error {
	payload, err := io.ReadAll(in)
	if err != nil {
		return err
	}

	records, err := parseSecretImportRecords(payload)
	if err != nil {
		return err
	}

	projection := safesync.Projection{}
	for index, record := range records {
		occurredAt := fmt.Sprintf("2026-03-31T10:06:%02dZ", index)
		projection, _, err = persistItemMutation(state, record, occurredAt)
		if err != nil {
			return err
		}
	}

	if options.json {
		return writeJSON(out, struct {
			Imported  int `json:"imported"`
			LatestSeq int `json:"latestSeq"`
			ItemCount int `json:"itemCount"`
		}{
			Imported:  len(records),
			LatestSeq: projection.LatestSeq,
			ItemCount: len(projection.Items),
		})
	}

	fmt.Fprintln(out, "secret import:")
	fmt.Fprintf(out, "- imported=%d latestSeq=%d items=%d\n", len(records), projection.LatestSeq, len(projection.Items))
	return nil
}

func secretExport(out io.Writer, state cliState, itemID string) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")

	if itemID != "" {
		record, ok := projection.Items[itemID]
		if !ok {
			return fmt.Errorf("secret not found: %s", itemID)
		}

		payload := struct {
			AccountID    string                 `json:"accountId"`
			CollectionID string                 `json:"collectionId"`
			LatestSeq    int                    `json:"latestSeq"`
			Item         domain.VaultItemRecord `json:"item"`
		}{
			AccountID:    projection.AccountID,
			CollectionID: projection.CollectionID,
			LatestSeq:    projection.LatestSeq,
			Item:         record,
		}

		return encoder.Encode(payload)
	}

	ids := make([]string, 0, len(projection.Items))
	for id := range projection.Items {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]domain.VaultItemRecord, 0, len(ids))
	for _, id := range ids {
		items = append(items, projection.Items[id])
	}

	payload := struct {
		AccountID    string                   `json:"accountId"`
		CollectionID string                   `json:"collectionId"`
		LatestSeq    int                      `json:"latestSeq"`
		Items        []domain.VaultItemRecord `json:"items"`
	}{
		AccountID:    projection.AccountID,
		CollectionID: projection.CollectionID,
		LatestSeq:    projection.LatestSeq,
		Items:        items,
	}

	return encoder.Encode(payload)
}

func parseSecretImportRecords(payload []byte) ([]domain.VaultItemRecord, error) {
	payload = bytes.TrimSpace(payload)
	if len(payload) == 0 {
		return nil, fmt.Errorf("secret import payload is empty")
	}

	type listImportPayload struct {
		Items []json.RawMessage `json:"items"`
	}
	type singleImportPayload struct {
		Item json.RawMessage `json:"item"`
	}

	var listPayload listImportPayload
	if err := json.Unmarshal(payload, &listPayload); err == nil && len(listPayload.Items) > 0 {
		records := make([]domain.VaultItemRecord, 0, len(listPayload.Items))
		for _, rawRecord := range listPayload.Items {
			record, err := domain.ParseVaultItemRecordJSON(rawRecord)
			if err != nil {
				return nil, fmt.Errorf("secret import invalid item: %w", err)
			}
			records = append(records, record)
		}
		return records, nil
	}

	record, err := domain.ParseVaultItemRecordJSON(payload)
	if err == nil {
		return []domain.VaultItemRecord{record}, nil
	}

	var singlePayload singleImportPayload
	if err := json.Unmarshal(payload, &singlePayload); err == nil && len(singlePayload.Item) > 0 {
		record, err := domain.ParseVaultItemRecordJSON(singlePayload.Item)
		if err != nil {
			return nil, fmt.Errorf("secret import invalid item: %w", err)
		}
		return []domain.VaultItemRecord{record}, nil
	}

	return nil, fmt.Errorf("secret import payload must be a vault item record or secret export JSON")
}

func secretHistory(out io.Writer, state cliState, options cliOptions, itemID string) error {
	events, err := loadEvents(state)
	if err != nil {
		return err
	}

	matches := make([]domain.VaultEventRecord, 0)
	for _, event := range events {
		if eventTargetsItem(event, itemID) {
			matches = append(matches, event)
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("secret history not found: %s", itemID)
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Sequence < matches[j].Sequence
	})

	if options.json {
		return writeJSON(out, struct {
			ItemID string                    `json:"itemId"`
			Events []domain.VaultEventRecord `json:"events"`
		}{
			ItemID: itemID,
			Events: matches,
		})
	}

	fmt.Fprintln(out, "secret history:")
	for _, event := range matches {
		fmt.Fprintf(out, "- seq=%d action=%s event=%s at=%s\n", event.Sequence, event.Action, event.EventID, event.OccurredAt)
	}

	return nil
}

func secretAdd(out io.Writer, state cliState, options cliOptions, title, username string) error {
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

	if options.json {
		return writeJSON(out, struct {
			Item      domain.VaultItem `json:"item"`
			EventID   string           `json:"eventId"`
			LatestSeq int              `json:"latestSeq"`
			ItemCount int              `json:"itemCount"`
		}{
			Item:      itemRecord.Item,
			EventID:   newEvent.EventID,
			LatestSeq: projection.LatestSeq,
			ItemCount: len(projection.Items),
		})
	}

	fmt.Fprintln(out, "secret add:")
	fmt.Fprintf(out, "- added=%s username=%s event=%s latestSeq=%d items=%d\n", title, username, newEvent.EventID, projection.LatestSeq, len(projection.Items))
	return nil
}

func secretUpdate(out io.Writer, state cliState, options cliOptions, itemID, title, username string) error {
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

	if options.json {
		return writeJSON(out, struct {
			Item      domain.VaultItem `json:"item"`
			EventID   string           `json:"eventId"`
			LatestSeq int              `json:"latestSeq"`
		}{
			Item:      updated.Item,
			EventID:   newEvent.EventID,
			LatestSeq: projection.LatestSeq,
		})
	}

	fmt.Fprintln(out, "secret update:")
	fmt.Fprintf(out, "- id=%s title=%s username=%s event=%s latestSeq=%d\n", itemID, title, username, newEvent.EventID, projection.LatestSeq)
	return nil
}

func secretDelete(out io.Writer, state cliState, options cliOptions, itemID string) error {
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

	if options.json {
		return writeJSON(out, struct {
			ItemID    string `json:"itemId"`
			EventID   string `json:"eventId"`
			LatestSeq int    `json:"latestSeq"`
			ItemCount int    `json:"itemCount"`
		}{
			ItemID:    itemID,
			EventID:   newEvent.EventID,
			LatestSeq: projection.LatestSeq,
			ItemCount: len(projection.Items),
		})
	}

	fmt.Fprintln(out, "secret delete:")
	fmt.Fprintf(out, "- id=%s event=%s latestSeq=%d items=%d\n", itemID, newEvent.EventID, projection.LatestSeq, len(projection.Items))
	return nil
}

func secretRestore(out io.Writer, state cliState, options cliOptions, itemID string) error {
	projection, err := loadProjection(state)
	if err != nil {
		return err
	}

	if _, ok := projection.Items[itemID]; ok {
		return fmt.Errorf("secret already active: %s", itemID)
	}

	record, err := storage.LoadItemRecord(state.store, state.session.AccountID, state.accountConfig.DefaultCollectionID, itemID)
	if err != nil {
		return fmt.Errorf("secret version not found: %s", itemID)
	}

	projection, newEvent, err := persistItemMutation(state, record, "2026-03-31T10:05:00Z")
	if err != nil {
		return err
	}

	if options.json {
		return writeJSON(out, struct {
			ItemID    string `json:"itemId"`
			EventID   string `json:"eventId"`
			LatestSeq int    `json:"latestSeq"`
			ItemCount int    `json:"itemCount"`
		}{
			ItemID:    itemID,
			EventID:   newEvent.EventID,
			LatestSeq: projection.LatestSeq,
			ItemCount: len(projection.Items),
		})
	}

	fmt.Fprintln(out, "secret restore:")
	fmt.Fprintf(out, "- id=%s event=%s latestSeq=%d items=%d\n", itemID, newEvent.EventID, projection.LatestSeq, len(projection.Items))
	return nil
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func loadProjection(state cliState) (safesync.Projection, error) {
	_, projection, err := loadVerifiedState(state)
	if err != nil {
		return safesync.Projection{}, err
	}

	return projection, nil
}

func loadVerifiedState(state cliState) (domain.CollectionHeadRecord, safesync.Projection, error) {
	head, err := loadHead(state)
	if err != nil {
		return domain.CollectionHeadRecord{}, safesync.Projection{}, err
	}

	storedEvents, err := loadEvents(state)
	if err != nil {
		return domain.CollectionHeadRecord{}, safesync.Projection{}, err
	}

	projection, err := safesync.ReplayCollectionAgainstHead(storedEvents, head)
	if err != nil {
		return domain.CollectionHeadRecord{}, safesync.Projection{}, err
	}

	return head, projection, nil
}

func loadEvents(state cliState) ([]domain.VaultEventRecord, error) {
	storedEvents, err := storage.LoadCollectionEventRecords(state.store, state.session.AccountID, state.accountConfig.DefaultCollectionID)
	if err != nil {
		return nil, err
	}

	return storedEvents, nil
}

func persistItemMutation(state cliState, itemRecord domain.VaultItemRecord, occurredAt string) (safesync.Projection, domain.VaultEventRecord, error) {
	head, err := loadHead(state)
	if err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}
	if err := ensureHeadMatchesEvents(state, head); err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}

	newEvent, newHead, err := safesync.BuildPutItemMutation(head, state.session.DeviceID, itemRecord, occurredAt)
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
	head, err := loadHead(state)
	if err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}
	if err := ensureHeadMatchesEvents(state, head); err != nil {
		return safesync.Projection{}, domain.VaultEventRecord{}, err
	}

	newEvent, newHead, err := safesync.BuildDeleteItemMutation(head, state.session.DeviceID, itemID, occurredAt)
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

func loadHead(state cliState) (domain.CollectionHeadRecord, error) {
	return storage.LoadCollectionHeadRecord(state.store, state.session.AccountID, state.accountConfig.DefaultCollectionID)
}

func ensureHeadMatchesEvents(state cliState, head domain.CollectionHeadRecord) error {
	events, err := loadEvents(state)
	if err != nil {
		return err
	}

	_, err = safesync.ReplayCollectionAgainstHead(events, head)
	return err
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

func eventTargetsItem(event domain.VaultEventRecord, itemID string) bool {
	switch event.Action {
	case domain.VaultEventActionPutItem:
		return event.ItemRecord.Item.ID == itemID
	case domain.VaultEventActionDeleteItem:
		return event.ItemID == itemID
	default:
		return false
	}
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
