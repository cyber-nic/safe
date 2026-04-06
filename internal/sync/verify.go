package sync

import "github.com/ndelorme/safe/internal/domain"

// VerifyHeadFunc authenticates and freshness-checks a candidate head record
// against the most recently trusted head. It must return nil only when the
// candidate is strictly at least as advanced as trusted and its integrity holds.
//
// W12 will replace the default stub (MonotonicVerifyHead) with a function that
// also verifies the Ed25519 signature carried by the event the head references.
// Callers should inject VerifyHeadFunc via NewSyncWriter / NewSyncReader rather
// than calling MonotonicVerifyHead directly, so the W12 upgrade is a one-line
// swap at the call site.
type VerifyHeadFunc func(trusted, candidate domain.CollectionHeadRecord) error

// MonotonicVerifyHead checks that candidate is a valid monotonic advancement of
// trusted: same account and collection, non-decreasing sequence, and no
// divergence at the same sequence number.
//
// This is the W15 stub. W12 will add Ed25519 event-signature verification on
// top of these monotonicity rules without changing the function signature.
func MonotonicVerifyHead(trusted, candidate domain.CollectionHeadRecord) error {
	// Genesis: no trusted state to compare against.
	if trusted.LatestSeq == 0 && trusted.LatestEventID == "" {
		return nil
	}
	return EnsureMonotonicHead(trusted, candidate)
}
