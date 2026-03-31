package sync

import (
	"fmt"

	"github.com/ndelorme/safe/internal/domain"
)

func BuildPutItemMutation(head domain.CollectionHeadRecord, deviceID string, itemRecord domain.VaultItemRecord, occurredAt string) (domain.VaultEventRecord, domain.CollectionHeadRecord, error) {
	if err := head.Validate(); err != nil {
		return domain.VaultEventRecord{}, domain.CollectionHeadRecord{}, err
	}
	if deviceID == "" {
		return domain.VaultEventRecord{}, domain.CollectionHeadRecord{}, ErrReplayInvariant("deviceId")
	}
	if err := itemRecord.Validate(); err != nil {
		return domain.VaultEventRecord{}, domain.CollectionHeadRecord{}, err
	}
	if occurredAt == "" {
		return domain.VaultEventRecord{}, domain.CollectionHeadRecord{}, ErrReplayInvariant("occurredAt")
	}

	nextSeq := head.LatestSeq + 1
	eventID := fmt.Sprintf("evt-%s-v%d", itemRecord.Item.ID, nextSeq)

	event := domain.VaultEventRecord{
		SchemaVersion: 1,
		EventID:       eventID,
		AccountID:     head.AccountID,
		DeviceID:      deviceID,
		CollectionID:  head.CollectionID,
		Sequence:      nextSeq,
		OccurredAt:    occurredAt,
		Action:        domain.VaultEventActionPutItem,
		ItemRecord:    itemRecord,
	}

	newHead := domain.CollectionHeadRecord{
		SchemaVersion: 1,
		AccountID:     head.AccountID,
		CollectionID:  head.CollectionID,
		LatestEventID: eventID,
		LatestSeq:     nextSeq,
	}

	return event, newHead, nil
}

func BuildDeleteItemMutation(head domain.CollectionHeadRecord, deviceID, itemID, occurredAt string) (domain.VaultEventRecord, domain.CollectionHeadRecord, error) {
	if err := head.Validate(); err != nil {
		return domain.VaultEventRecord{}, domain.CollectionHeadRecord{}, err
	}
	if deviceID == "" {
		return domain.VaultEventRecord{}, domain.CollectionHeadRecord{}, ErrReplayInvariant("deviceId")
	}
	if itemID == "" {
		return domain.VaultEventRecord{}, domain.CollectionHeadRecord{}, ErrReplayInvariant("itemId")
	}
	if occurredAt == "" {
		return domain.VaultEventRecord{}, domain.CollectionHeadRecord{}, ErrReplayInvariant("occurredAt")
	}

	nextSeq := head.LatestSeq + 1
	eventID := fmt.Sprintf("evt-%s-delete-v%d", itemID, nextSeq)

	event := domain.VaultEventRecord{
		SchemaVersion: 1,
		EventID:       eventID,
		AccountID:     head.AccountID,
		DeviceID:      deviceID,
		CollectionID:  head.CollectionID,
		Sequence:      nextSeq,
		OccurredAt:    occurredAt,
		Action:        domain.VaultEventActionDeleteItem,
		ItemID:        itemID,
	}

	newHead := domain.CollectionHeadRecord{
		SchemaVersion: 1,
		AccountID:     head.AccountID,
		CollectionID:  head.CollectionID,
		LatestEventID: eventID,
		LatestSeq:     nextSeq,
	}

	return event, newHead, nil
}
