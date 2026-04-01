package sync

import (
	"fmt"
	"sort"

	"github.com/ndelorme/safe/internal/domain"
)

type Projection struct {
	AccountID     string
	CollectionID  string
	LatestSeq     int
	LatestEventID string
	Items         map[string]domain.VaultItemRecord
}

func ReplayCollection(events []domain.VaultEventRecord) (Projection, error) {
	if len(events) == 0 {
		return Projection{}, nil
	}

	ordered := append([]domain.VaultEventRecord(nil), events...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Sequence < ordered[j].Sequence
	})

	projection := Projection{
		AccountID:    ordered[0].AccountID,
		CollectionID: ordered[0].CollectionID,
		Items:        make(map[string]domain.VaultItemRecord),
	}

	expectedSequence := 1
	for _, event := range ordered {
		if err := event.Validate(); err != nil {
			return Projection{}, err
		}

		if event.AccountID != projection.AccountID {
			return Projection{}, ErrReplayInvariant("accountId")
		}

		if event.CollectionID != projection.CollectionID {
			return Projection{}, ErrReplayInvariant("collectionId")
		}

		if event.Sequence != expectedSequence {
			return Projection{}, ErrSequenceGap(expectedSequence, event.Sequence)
		}

		switch event.Action {
		case domain.VaultEventActionPutItem:
			projection.Items[event.ItemRecord.Item.ID] = event.ItemRecord
		case domain.VaultEventActionDeleteItem:
			delete(projection.Items, event.ItemID)
		default:
			return Projection{}, ErrReplayInvariant("action")
		}

		projection.LatestSeq = event.Sequence
		projection.LatestEventID = event.EventID
		expectedSequence++
	}

	return projection, nil
}

func ReplayCollectionAgainstHead(events []domain.VaultEventRecord, head domain.CollectionHeadRecord) (Projection, error) {
	if err := head.Validate(); err != nil {
		return Projection{}, err
	}

	projection, err := ReplayCollection(events)
	if err != nil {
		return Projection{}, err
	}

	if projection.AccountID != head.AccountID {
		return Projection{}, ErrReplayInvariant("head.accountId")
	}

	if projection.CollectionID != head.CollectionID {
		return Projection{}, ErrReplayInvariant("head.collectionId")
	}

	if projection.LatestSeq != head.LatestSeq {
		return Projection{}, ErrHeadMismatch("latestSeq", head.LatestSeq, projection.LatestSeq)
	}

	if projection.LatestEventID != head.LatestEventID {
		return Projection{}, ErrHeadEventMismatch(head.LatestEventID, projection.LatestEventID)
	}

	return projection, nil
}

func EnsureMonotonicHead(trusted, candidate domain.CollectionHeadRecord) error {
	if err := trusted.Validate(); err != nil {
		return err
	}
	if err := candidate.Validate(); err != nil {
		return err
	}

	if trusted.AccountID != candidate.AccountID {
		return ErrReplayInvariant("trustedHead.accountId")
	}
	if trusted.CollectionID != candidate.CollectionID {
		return ErrReplayInvariant("trustedHead.collectionId")
	}

	if candidate.LatestSeq < trusted.LatestSeq {
		return ErrStaleHead(trusted, candidate)
	}

	if candidate.LatestSeq == trusted.LatestSeq && candidate.LatestEventID != trusted.LatestEventID {
		return ErrHeadEventMismatch(trusted.LatestEventID, candidate.LatestEventID)
	}

	return nil
}

type replayInvariantError string

func (field replayInvariantError) Error() string {
	return "sync replay invariant violated: " + string(field)
}

func ErrReplayInvariant(field string) error {
	return replayInvariantError(field)
}

type sequenceGapError struct {
	expected int
	got      int
}

func (err sequenceGapError) Error() string {
	return fmt.Sprintf("sync replay sequence gap: expected %d got %d", err.expected, err.got)
}

func ErrSequenceGap(expected, got int) error {
	return sequenceGapError{expected: expected, got: got}
}

type headMismatchError struct {
	field    string
	expected int
	got      int
}

func (err headMismatchError) Error() string {
	return fmt.Sprintf("sync head mismatch: %s expected %d got %d", err.field, err.expected, err.got)
}

func ErrHeadMismatch(field string, expected, got int) error {
	return headMismatchError{field: field, expected: expected, got: got}
}

type headEventMismatchError struct {
	expected string
	got      string
}

func (err headEventMismatchError) Error() string {
	return fmt.Sprintf("sync head mismatch: latestEventId expected %s got %s", err.expected, err.got)
}

func ErrHeadEventMismatch(expected, got string) error {
	return headEventMismatchError{expected: expected, got: got}
}

type staleHeadError struct {
	trustedSeq   int
	candidateSeq int
}

func (err staleHeadError) Error() string {
	return fmt.Sprintf("sync stale head rejected: trusted %d candidate %d", err.trustedSeq, err.candidateSeq)
}

func ErrStaleHead(trusted, candidate domain.CollectionHeadRecord) error {
	return staleHeadError{trustedSeq: trusted.LatestSeq, candidateSeq: candidate.LatestSeq}
}
