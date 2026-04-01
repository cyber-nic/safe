package crypto

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateTOTPSHA1RFCVector(t *testing.T) {
	code, err := GenerateTOTP("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Unix(59, 0).UTC(), 8, 30, "SHA1")
	if err != nil {
		t.Fatalf("generate totp: %v", err)
	}

	if code != "94287082" {
		t.Fatalf("unexpected totp code: %s", code)
	}
}

func TestGenerateTOTPRejectsUnsupportedAlgorithm(t *testing.T) {
	_, err := GenerateTOTP("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", time.Unix(59, 0).UTC(), 6, 30, "SHA256")
	if err == nil {
		t.Fatal("expected unsupported algorithm error")
	}

	if !strings.Contains(err.Error(), "unsupported totp algorithm") {
		t.Fatalf("unexpected error: %v", err)
	}
}
