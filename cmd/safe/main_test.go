package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ndelorme/safe/internal/domain"
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

func TestRunOverviewJSON(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"--json"}, &buffer); err != nil {
			t.Fatalf("run overview json: %v", err)
		}

		if strings.Contains(buffer.String(), "safe CLI bootstrap") {
			t.Fatalf("did not expect bootstrap banner in json output: %s", buffer.String())
		}

		var payload struct {
			AccountID         string `json:"accountId"`
			DefaultCollection string `json:"defaultCollection"`
			LatestSeq         int    `json:"latestSeq"`
			ItemCount         int    `json:"itemCount"`
			HeadEventID       string `json:"headEventId"`
		}
		if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode overview json: %v", err)
		}

		if payload.AccountID != "acct-dev-001" || payload.DefaultCollection != "vault-personal" {
			t.Fatalf("unexpected overview payload: %+v", payload)
		}
		if payload.LatestSeq != 2 || payload.ItemCount != 2 {
			t.Fatalf("unexpected overview counts: %+v", payload)
		}
	})
}

func TestRunSecretListJSON(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"--json", "secret", "list"}, &buffer); err != nil {
			t.Fatalf("run secret list json: %v", err)
		}

		if strings.Contains(buffer.String(), "safe CLI bootstrap") {
			t.Fatalf("did not expect bootstrap banner in json output: %s", buffer.String())
		}

		var payload struct {
			Items []struct {
				ID       string `json:"id"`
				Kind     string `json:"kind"`
				Title    string `json:"title"`
				Username string `json:"username"`
			} `json:"items"`
		}
		if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode list json: %v", err)
		}

		if len(payload.Items) != 2 {
			t.Fatalf("expected 2 items, got %+v", payload)
		}
		if payload.Items[0].ID != "login-gmail-primary" || payload.Items[1].ID != "totp-gmail-primary" {
			t.Fatalf("expected sorted json list, got %+v", payload.Items)
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

func TestRunSecretAddJSON(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"--json", "secret", "add", "GitHub", "alice"}, &buffer); err != nil {
			t.Fatalf("run secret add json: %v", err)
		}

		var payload struct {
			Item struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Username string `json:"username"`
			} `json:"item"`
			EventID   string `json:"eventId"`
			LatestSeq int    `json:"latestSeq"`
			ItemCount int    `json:"itemCount"`
		}
		if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode add json: %v", err)
		}

		if payload.Item.ID != "login-github-primary" || payload.Item.Title != "GitHub" {
			t.Fatalf("unexpected add payload: %+v", payload)
		}
		if payload.LatestSeq != 3 || payload.ItemCount != 3 {
			t.Fatalf("unexpected add counts: %+v", payload)
		}
	})
}

func TestRunSecretUpdate(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "update", "login-gmail-primary", "Gmail Work", "alice@work.example"}, &buffer); err != nil {
			t.Fatalf("run secret update: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, "secret update:") {
			t.Fatalf("expected secret update output, got %s", output)
		}
		if !strings.Contains(output, "id=login-gmail-primary title=Gmail Work username=alice@work.example") {
			t.Fatalf("expected updated login output, got %s", output)
		}
		if !strings.Contains(output, "latestSeq=3") {
			t.Fatalf("expected latestSeq=3 after update, got %s", output)
		}
	})
}

func TestRunSecretUpdateMissingItem(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		err := run([]string{"secret", "update", "missing-item", "Title", "alice"}, &buffer)
		if err == nil {
			t.Fatal("expected missing item error")
		}

		if !strings.Contains(err.Error(), "secret not found: missing-item") {
			t.Fatalf("expected missing item error, got %v", err)
		}
	})
}

func TestRunSecretUpdateRejectsNonLoginItem(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		err := run([]string{"secret", "update", "totp-gmail-primary", "Gmail 2FA", "alice@example.com"}, &buffer)
		if err == nil {
			t.Fatal("expected non-login item error")
		}

		if !strings.Contains(err.Error(), "secret update only supports login items: totp-gmail-primary") {
			t.Fatalf("expected non-login item error, got %v", err)
		}
	})
}

func TestRunSecretDelete(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "delete", "login-gmail-primary"}, &buffer); err != nil {
			t.Fatalf("run secret delete: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, "secret delete:") {
			t.Fatalf("expected secret delete output, got %s", output)
		}
		if !strings.Contains(output, "id=login-gmail-primary") {
			t.Fatalf("expected deleted item ID, got %s", output)
		}
		if !strings.Contains(output, "latestSeq=3") {
			t.Fatalf("expected latestSeq=3 after delete, got %s", output)
		}
		if !strings.Contains(output, "items=1") {
			t.Fatalf("expected one remaining item after delete, got %s", output)
		}
	})
}

func TestRunSecretDeleteMissingItem(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		err := run([]string{"secret", "delete", "missing-item"}, &buffer)
		if err == nil {
			t.Fatal("expected missing item error")
		}

		if !strings.Contains(err.Error(), "secret not found: missing-item") {
			t.Fatalf("expected missing item error, got %v", err)
		}
	})
}

func TestRunSecretHistory(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "history", "login-gmail-primary"}, &buffer); err != nil {
			t.Fatalf("run secret history: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, "secret history:") {
			t.Fatalf("expected secret history output, got %s", output)
		}
		if !strings.Contains(output, "seq=1 action=put_item event=evt-login-gmail-primary-v1") {
			t.Fatalf("expected initial Gmail history entry, got %s", output)
		}
	})
}

func TestRunSecretHistoryJSON(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"--json", "secret", "history", "login-gmail-primary"}, &buffer); err != nil {
			t.Fatalf("run secret history json: %v", err)
		}

		var payload struct {
			ItemID string `json:"itemId"`
			Events []struct {
				EventID  string `json:"eventId"`
				Action   string `json:"action"`
				Sequence int    `json:"sequence"`
			} `json:"events"`
		}
		if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode history json: %v", err)
		}

		if payload.ItemID != "login-gmail-primary" || len(payload.Events) != 1 {
			t.Fatalf("unexpected history payload: %+v", payload)
		}
		if payload.Events[0].EventID != "evt-login-gmail-primary-v1" {
			t.Fatalf("unexpected history event: %+v", payload.Events[0])
		}
	})
}

func TestSecretRestore(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var deleteBuffer bytes.Buffer
		if err := secretDelete(&deleteBuffer, state, cliOptions{}, "login-gmail-primary"); err != nil {
			t.Fatalf("delete secret before restore: %v", err)
		}

		var restoreBuffer bytes.Buffer
		if err := secretRestore(&restoreBuffer, state, cliOptions{}, "login-gmail-primary"); err != nil {
			t.Fatalf("restore secret: %v", err)
		}

		output := restoreBuffer.String()
		if !strings.Contains(output, "secret restore:") {
			t.Fatalf("expected secret restore output, got %s", output)
		}
		if !strings.Contains(output, "id=login-gmail-primary") {
			t.Fatalf("expected restored item ID, got %s", output)
		}
		if !strings.Contains(output, "latestSeq=4") {
			t.Fatalf("expected latestSeq=4 after delete and restore, got %s", output)
		}
		if !strings.Contains(output, "items=2") {
			t.Fatalf("expected two items after restore, got %s", output)
		}
	})
}

func TestSecretHistoryAfterDeleteAndRestore(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var deleteBuffer bytes.Buffer
		if err := secretDelete(&deleteBuffer, state, cliOptions{}, "login-gmail-primary"); err != nil {
			t.Fatalf("delete secret before history check: %v", err)
		}

		var restoreBuffer bytes.Buffer
		if err := secretRestore(&restoreBuffer, state, cliOptions{}, "login-gmail-primary"); err != nil {
			t.Fatalf("restore secret before history check: %v", err)
		}

		var historyBuffer bytes.Buffer
		if err := secretHistory(&historyBuffer, state, cliOptions{}, "login-gmail-primary"); err != nil {
			t.Fatalf("history after restore: %v", err)
		}

		output := historyBuffer.String()
		if !strings.Contains(output, "seq=1 action=put_item event=evt-login-gmail-primary-v1") {
			t.Fatalf("expected initial put event, got %s", output)
		}
		if !strings.Contains(output, "seq=3 action=delete_item event=evt-login-gmail-primary-delete-v3") {
			t.Fatalf("expected delete event, got %s", output)
		}
		if !strings.Contains(output, "seq=4 action=put_item event=evt-login-gmail-primary-v4") {
			t.Fatalf("expected restore put event, got %s", output)
		}
	})
}

func TestSecretHistoryMissingItem(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var buffer bytes.Buffer
		err = secretHistory(&buffer, state, cliOptions{}, "missing-item")
		if err == nil {
			t.Fatal("expected missing history error")
		}

		if !strings.Contains(err.Error(), "secret history not found: missing-item") {
			t.Fatalf("expected missing history error, got %v", err)
		}
	})
}

func TestSecretRestoreRejectsActiveItem(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var buffer bytes.Buffer
		err = secretRestore(&buffer, state, cliOptions{}, "login-gmail-primary")
		if err == nil {
			t.Fatal("expected active item error")
		}

		if !strings.Contains(err.Error(), "secret already active: login-gmail-primary") {
			t.Fatalf("expected active item error, got %v", err)
		}
	})
}

func TestSecretRestoreMissingVersion(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var buffer bytes.Buffer
		err = secretRestore(&buffer, state, cliOptions{}, "missing-item")
		if err == nil {
			t.Fatal("expected missing version error")
		}

		if !strings.Contains(err.Error(), "secret version not found: missing-item") {
			t.Fatalf("expected missing version error, got %v", err)
		}
	})
}

func TestRunSecretSearchByTitle(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "search", "gmail"}, &buffer); err != nil {
			t.Fatalf("run secret search by title: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, `secret search: query="gmail"`) {
			t.Fatalf("expected search header, got %s", output)
		}
		if !strings.Contains(output, "id=login-gmail-primary kind=login title=Gmail") {
			t.Fatalf("expected Gmail login match, got %s", output)
		}
		if !strings.Contains(output, "id=totp-gmail-primary kind=totp title=Gmail 2FA") {
			t.Fatalf("expected Gmail totp match, got %s", output)
		}
	})
}

func TestRunSecretSearchByTag(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "search", "authenticator"}, &buffer); err != nil {
			t.Fatalf("run secret search by tag: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, "id=totp-gmail-primary kind=totp title=Gmail 2FA") {
			t.Fatalf("expected TOTP tag match, got %s", output)
		}
		if strings.Contains(output, "id=login-gmail-primary kind=login title=Gmail") {
			t.Fatalf("did not expect login match for authenticator tag, got %s", output)
		}
	})
}

func TestRunSecretSearchNoMatches(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"secret", "search", "nomatch"}, &buffer); err != nil {
			t.Fatalf("run secret search no matches: %v", err)
		}

		output := buffer.String()
		if !strings.Contains(output, "- no matches") {
			t.Fatalf("expected no matches output, got %s", output)
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

func TestRunSecretShowJSON(t *testing.T) {
	withFakeBootstrap(t, func() {
		var buffer bytes.Buffer
		if err := run([]string{"--json", "secret", "show", "login-gmail-primary"}, &buffer); err != nil {
			t.Fatalf("run secret show json: %v", err)
		}

		var payload domain.VaultItemRecord
		if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode show json: %v", err)
		}

		if payload.Item.ID != "login-gmail-primary" || payload.Item.Username != "alice@example.com" {
			t.Fatalf("unexpected show payload: %+v", payload)
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

func TestSecretExport(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var buffer bytes.Buffer
		if err := secretExport(&buffer, state, ""); err != nil {
			t.Fatalf("export secrets: %v", err)
		}

		var payload struct {
			AccountID    string                   `json:"accountId"`
			CollectionID string                   `json:"collectionId"`
			LatestSeq    int                      `json:"latestSeq"`
			Items        []domain.VaultItemRecord `json:"items"`
		}
		if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode export payload: %v", err)
		}

		if payload.AccountID != "acct-dev-001" {
			t.Fatalf("expected account export, got %+v", payload)
		}
		if payload.CollectionID != "vault-personal" {
			t.Fatalf("expected default collection export, got %+v", payload)
		}
		if payload.LatestSeq != 2 {
			t.Fatalf("expected latestSeq=2, got %+v", payload)
		}
		if len(payload.Items) != 2 {
			t.Fatalf("expected two active items, got %+v", payload)
		}
		if payload.Items[0].Item.ID != "login-gmail-primary" {
			t.Fatalf("expected sorted first item, got %+v", payload.Items)
		}
		if payload.Items[1].Item.ID != "totp-gmail-primary" {
			t.Fatalf("expected sorted second item, got %+v", payload.Items)
		}
	})
}

func TestSecretExportSingleItem(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var buffer bytes.Buffer
		if err := secretExport(&buffer, state, "totp-gmail-primary"); err != nil {
			t.Fatalf("export single secret: %v", err)
		}

		var payload struct {
			AccountID    string                 `json:"accountId"`
			CollectionID string                 `json:"collectionId"`
			LatestSeq    int                    `json:"latestSeq"`
			Item         domain.VaultItemRecord `json:"item"`
		}
		if err := json.Unmarshal(buffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode single export payload: %v", err)
		}

		if payload.Item.Item.ID != "totp-gmail-primary" {
			t.Fatalf("expected targeted item export, got %+v", payload)
		}
		if payload.Item.Item.Kind != domain.VaultItemKindTOTP {
			t.Fatalf("expected totp item export, got %+v", payload)
		}
	})
}

func TestSecretExportMissingItem(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var buffer bytes.Buffer
		err = secretExport(&buffer, state, "missing-item")
		if err == nil {
			t.Fatal("expected missing item error")
		}

		if !strings.Contains(err.Error(), "secret not found: missing-item") {
			t.Fatalf("expected missing item error, got %v", err)
		}
	})
}

func TestSecretImportSingleItemExport(t *testing.T) {
	withFakeBootstrap(t, func() {
		exportState, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap export state: %v", err)
		}

		var exportBuffer bytes.Buffer
		if err := secretExport(&exportBuffer, exportState, "login-gmail-primary"); err != nil {
			t.Fatalf("export single secret: %v", err)
		}

		importState, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap import state: %v", err)
		}

		var deleteBuffer bytes.Buffer
		if err := secretDelete(&deleteBuffer, importState, cliOptions{}, "login-gmail-primary"); err != nil {
			t.Fatalf("delete before import: %v", err)
		}

		var importBuffer bytes.Buffer
		if err := secretImport(bytes.NewReader(exportBuffer.Bytes()), &importBuffer, importState, cliOptions{}); err != nil {
			t.Fatalf("import single secret: %v", err)
		}

		output := importBuffer.String()
		if !strings.Contains(output, "secret import:") {
			t.Fatalf("expected import output, got %s", output)
		}
		if !strings.Contains(output, "imported=1 latestSeq=4 items=2") {
			t.Fatalf("expected import summary, got %s", output)
		}

		projection, err := loadProjection(importState)
		if err != nil {
			t.Fatalf("load projection after import: %v", err)
		}
		if _, ok := projection.Items["login-gmail-primary"]; !ok {
			t.Fatalf("expected imported login item in projection, got %+v", projection.Items)
		}
	})
}

func TestRunSecretImportFullExport(t *testing.T) {
	withFakeBootstrap(t, func() {
		exportState, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap export state: %v", err)
		}

		var exportBuffer bytes.Buffer
		if err := secretExport(&exportBuffer, exportState, ""); err != nil {
			t.Fatalf("export vault: %v", err)
		}

		var importBuffer bytes.Buffer
		if err := runWithIO([]string{"secret", "import"}, bytes.NewReader(exportBuffer.Bytes()), &importBuffer); err != nil {
			t.Fatalf("run secret import: %v", err)
		}

		output := importBuffer.String()
		if !strings.Contains(output, "secret import:") {
			t.Fatalf("expected import output, got %s", output)
		}
		if !strings.Contains(output, "imported=2 latestSeq=4 items=2") {
			t.Fatalf("expected full import summary, got %s", output)
		}
	})
}

func TestRunSecretImportJSON(t *testing.T) {
	withFakeBootstrap(t, func() {
		exportState, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap export state: %v", err)
		}

		var exportBuffer bytes.Buffer
		if err := secretExport(&exportBuffer, exportState, ""); err != nil {
			t.Fatalf("export vault: %v", err)
		}

		var importBuffer bytes.Buffer
		if err := runWithIO([]string{"--json", "secret", "import"}, bytes.NewReader(exportBuffer.Bytes()), &importBuffer); err != nil {
			t.Fatalf("run secret import json: %v", err)
		}

		var payload struct {
			Imported  int `json:"imported"`
			LatestSeq int `json:"latestSeq"`
			ItemCount int `json:"itemCount"`
		}
		if err := json.Unmarshal(importBuffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode import json: %v", err)
		}

		if payload.Imported != 2 || payload.LatestSeq != 4 || payload.ItemCount != 2 {
			t.Fatalf("unexpected import payload: %+v", payload)
		}
	})
}

func TestSecretImportFullExportRestoresDeletedVault(t *testing.T) {
	withFakeBootstrap(t, func() {
		exportState, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap export state: %v", err)
		}

		var exportBuffer bytes.Buffer
		if err := secretExport(&exportBuffer, exportState, ""); err != nil {
			t.Fatalf("export vault: %v", err)
		}

		importState, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap import state: %v", err)
		}

		var deleteLogin bytes.Buffer
		if err := secretDelete(&deleteLogin, importState, cliOptions{}, "login-gmail-primary"); err != nil {
			t.Fatalf("delete login before import: %v", err)
		}

		var deleteTotp bytes.Buffer
		if err := secretDelete(&deleteTotp, importState, cliOptions{}, "totp-gmail-primary"); err != nil {
			t.Fatalf("delete totp before import: %v", err)
		}

		var importBuffer bytes.Buffer
		if err := secretImport(bytes.NewReader(exportBuffer.Bytes()), &importBuffer, importState, cliOptions{}); err != nil {
			t.Fatalf("import full export: %v", err)
		}

		if !strings.Contains(importBuffer.String(), "imported=2 latestSeq=6 items=2") {
			t.Fatalf("expected restore summary, got %s", importBuffer.String())
		}

		projection, err := loadProjection(importState)
		if err != nil {
			t.Fatalf("load projection after full import: %v", err)
		}
		if len(projection.Items) != 2 {
			t.Fatalf("expected restored projection items, got %+v", projection.Items)
		}
	})
}

func TestSecretImportRejectsInvalidPayload(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap import state: %v", err)
		}

		var importBuffer bytes.Buffer
		err = secretImport(strings.NewReader(`{"items":[{"schemaVersion":2}]}`), &importBuffer, state, cliOptions{})
		if err == nil {
			t.Fatal("expected invalid payload error")
		}

		if !strings.Contains(err.Error(), "secret import invalid item") {
			t.Fatalf("expected invalid payload error, got %v", err)
		}
	})
}

func TestSecretImportRejectsUnsupportedPayloadShape(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap import state: %v", err)
		}

		var importBuffer bytes.Buffer
		err = secretImport(strings.NewReader(`{"latestSeq":2}`), &importBuffer, state, cliOptions{})
		if err == nil {
			t.Fatal("expected unsupported payload shape error")
		}

		if !strings.Contains(err.Error(), "secret import payload must be a vault item record or secret export JSON") {
			t.Fatalf("expected unsupported payload shape error, got %v", err)
		}
	})
}

func TestSecretImportNoteItemAndShow(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		importPayload := `{
  "schemaVersion": 1,
  "item": {
    "id": "note-server-bootstrap",
    "kind": "note",
    "title": "Server Bootstrap",
    "tags": ["infra", "runbook"],
    "bodyPreview": "Bootstrap checklist"
  }
}`

		var importBuffer bytes.Buffer
		if err := secretImport(strings.NewReader(importPayload), &importBuffer, state, cliOptions{}); err != nil {
			t.Fatalf("import note item: %v", err)
		}

		var showBuffer bytes.Buffer
		if err := secretShow(&showBuffer, state, cliOptions{}, "note-server-bootstrap"); err != nil {
			t.Fatalf("show note item: %v", err)
		}

		output := showBuffer.String()
		if !strings.Contains(output, "kind=note title=Server Bootstrap") {
			t.Fatalf("expected note identity output, got %s", output)
		}
		if !strings.Contains(output, "bodyPreview=Bootstrap checklist") {
			t.Fatalf("expected note body preview output, got %s", output)
		}
	})
}

func TestSecretImportAPIKeyItemAndShowJSON(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		importPayload := `{
  "schemaVersion": 1,
  "item": {
    "id": "apikey-stripe-primary",
    "kind": "apiKey",
    "title": "Stripe Primary",
    "tags": ["payments"],
    "service": "Stripe"
  }
}`

		var importBuffer bytes.Buffer
		if err := secretImport(strings.NewReader(importPayload), &importBuffer, state, cliOptions{}); err != nil {
			t.Fatalf("import api key item: %v", err)
		}

		var showBuffer bytes.Buffer
		if err := secretShow(&showBuffer, state, cliOptions{json: true}, "apikey-stripe-primary"); err != nil {
			t.Fatalf("show api key item json: %v", err)
		}

		var payload domain.VaultItemRecord
		if err := json.Unmarshal(showBuffer.Bytes(), &payload); err != nil {
			t.Fatalf("decode api key show json: %v", err)
		}

		if payload.Item.Kind != domain.VaultItemKindAPIKey || payload.Item.Service != "Stripe" {
			t.Fatalf("unexpected api key show payload: %+v", payload)
		}
	})
}

func TestSecretImportSSHKeyItemSearchAndShow(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		importPayload := `{
  "schemaVersion": 1,
  "item": {
    "id": "ssh-prod-root",
    "kind": "sshKey",
    "title": "Prod Root",
    "tags": ["infra", "ssh"],
    "username": "root",
    "host": "prod-01.example.com"
  }
}`

		var importBuffer bytes.Buffer
		if err := secretImport(strings.NewReader(importPayload), &importBuffer, state, cliOptions{}); err != nil {
			t.Fatalf("import ssh key item: %v", err)
		}

		var searchBuffer bytes.Buffer
		if err := secretSearch(&searchBuffer, state, cliOptions{}, "prod-01"); err != nil {
			t.Fatalf("search ssh key item: %v", err)
		}
		if !strings.Contains(searchBuffer.String(), "id=ssh-prod-root kind=sshKey title=Prod Root") {
			t.Fatalf("expected ssh key search output, got %s", searchBuffer.String())
		}

		var showBuffer bytes.Buffer
		if err := secretShow(&showBuffer, state, cliOptions{}, "ssh-prod-root"); err != nil {
			t.Fatalf("show ssh key item: %v", err)
		}
		if !strings.Contains(showBuffer.String(), "host=prod-01.example.com") {
			t.Fatalf("expected ssh host output, got %s", showBuffer.String())
		}
	})
}

func TestMatchesSecretQueryEmptyQuery(t *testing.T) {
	if matchesSecretQuery(domain.VaultItem{Title: "Gmail"}, "   ") {
		t.Fatal("expected empty query to return false")
	}
}

func TestEventTargetsItem(t *testing.T) {
	putEvent := domain.VaultEventRecord{
		Action: domain.VaultEventActionPutItem,
		ItemRecord: domain.VaultItemRecord{
			Item: domain.VaultItem{ID: "login-gmail-primary"},
		},
	}
	if !eventTargetsItem(putEvent, "login-gmail-primary") {
		t.Fatal("expected put event to match item")
	}

	deleteEvent := domain.VaultEventRecord{
		Action: domain.VaultEventActionDeleteItem,
		ItemID: "login-gmail-primary",
	}
	if !eventTargetsItem(deleteEvent, "login-gmail-primary") {
		t.Fatal("expected delete event to match item")
	}
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
