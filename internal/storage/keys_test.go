package storage

import "testing"

func TestEventObjectKey(t *testing.T) {
	got := EventObjectKey("acct-dev-001", "vault-personal", "evt-login-gmail-primary-v1")
	want := "accounts/acct-dev-001/collections/vault-personal/events/evt-login-gmail-primary-v1.json"

	if got != want {
		t.Fatalf("unexpected event object key\nwant: %s\ngot:  %s", want, got)
	}
}

func TestItemObjectKey(t *testing.T) {
	got := ItemObjectKey("acct-dev-001", "vault-personal", "totp-gmail-primary")
	want := "accounts/acct-dev-001/collections/vault-personal/items/totp-gmail-primary.json"

	if got != want {
		t.Fatalf("unexpected item object key\nwant: %s\ngot:  %s", want, got)
	}
}

func TestDeviceListPrefix(t *testing.T) {
	got := DeviceListPrefix("acct-dev-001")
	want := "accounts/acct-dev-001/devices/"
	if got != want {
		t.Fatalf("unexpected device list prefix\nwant: %s\ngot:  %s", want, got)
	}
}

func TestEnrollmentRequestKey(t *testing.T) {
	got := EnrollmentRequestKey("acct-dev-001", "dev-new-001")
	want := "accounts/acct-dev-001/enrollments/dev-new-001/request.json"
	if got != want {
		t.Fatalf("unexpected enrollment request key\nwant: %s\ngot:  %s", want, got)
	}
}

func TestEnrollmentBundleKey(t *testing.T) {
	got := EnrollmentBundleKey("acct-dev-001", "dev-new-001")
	want := "accounts/acct-dev-001/enrollments/dev-new-001/bundle.json"
	if got != want {
		t.Fatalf("unexpected enrollment bundle key\nwant: %s\ngot:  %s", want, got)
	}
}

func TestEnrollmentListPrefix(t *testing.T) {
	got := EnrollmentListPrefix("acct-dev-001")
	want := "accounts/acct-dev-001/enrollments/"
	if got != want {
		t.Fatalf("unexpected enrollment list prefix\nwant: %s\ngot:  %s", want, got)
	}
}
