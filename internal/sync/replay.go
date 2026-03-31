package sync

import (
	"fmt"
	"sort"

	"github.com/ndelorme/safe/internal/domain"
)

type Projection struct {
	AccountID    string
	CollectionID string
	LatestSeq    int
	Items        map[string]domain.VaultItemRecord
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
		default:
			return Projection{}, ErrReplayInvariant("action")
		}

		projection.LatestSeq = event.Sequence
		expectedSequence++
	}

	return projection, nil
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
