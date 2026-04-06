package sync

import "github.com/ndelorme/safe/internal/domain"

// VerifyHeadFunc authenticates and freshness-checks a candidate signed head
// against the most recently trusted head plus the immutable event and active
// device record it references. It must return nil only when the candidate is
// signed correctly and is not stale or divergent relative to trusted.
type VerifyHeadFunc func(trusted domain.CollectionHeadRecord, candidate SignedCollectionHead, latestEvent domain.VaultEventRecord, authoringDevice domain.LocalDeviceRecord) error

// HeadSignerFunc signs a canonical head record before it is persisted.
type HeadSignerFunc func(candidate domain.CollectionHeadRecord) (SignedCollectionHead, error)
