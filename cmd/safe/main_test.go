package main

import (
	"bytes"
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

func TestSecretRestore(t *testing.T) {
	withFakeBootstrap(t, func() {
		state, err := bootstrapCLIState()
		if err != nil {
			t.Fatalf("bootstrap cli state: %v", err)
		}

		var deleteBuffer bytes.Buffer
		if err := secretDelete(&deleteBuffer, state, "login-gmail-primary"); err != nil {
			t.Fatalf("delete secret before restore: %v", err)
		}

		var restoreBuffer bytes.Buffer
		if err := secretRestore(&restoreBuffer, state, "login-gmail-primary"); err != nil {
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
		if err := secretDelete(&deleteBuffer, state, "login-gmail-primary"); err != nil {
			t.Fatalf("delete secret before history check: %v", err)
		}

		var restoreBuffer bytes.Buffer
		if err := secretRestore(&restoreBuffer, state, "login-gmail-primary"); err != nil {
			t.Fatalf("restore secret before history check: %v", err)
		}

		var historyBuffer bytes.Buffer
		if err := secretHistory(&historyBuffer, state, "login-gmail-primary"); err != nil {
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
		err = secretHistory(&buffer, state, "missing-item")
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
		err = secretRestore(&buffer, state, "login-gmail-primary")
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
		err = secretRestore(&buffer, state, "missing-item")
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
