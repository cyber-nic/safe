package storage

import (
	"encoding/base64"
	"fmt"
)

func EventObjectKey(accountID, collectionID, eventID string) string {
	return fmt.Sprintf("accounts/%s/collections/%s/events/%s.json", accountID, collectionID, eventID)
}

func ItemObjectKey(accountID, collectionID, itemID string) string {
	return fmt.Sprintf("accounts/%s/collections/%s/items/%s.json", accountID, collectionID, itemID)
}

func EventPrefix(accountID, collectionID string) string {
	return fmt.Sprintf("accounts/%s/collections/%s/events/", accountID, collectionID)
}

func ItemPrefix(accountID, collectionID string) string {
	return fmt.Sprintf("accounts/%s/collections/%s/items/", accountID, collectionID)
}

func CollectionHeadKey(accountID, collectionID string) string {
	return fmt.Sprintf("accounts/%s/collections/%s/head.json", accountID, collectionID)
}

func AccountConfigKey(accountID string) string {
	return fmt.Sprintf("accounts/%s/account.json", accountID)
}

func LocalUnlockKey(accountID string) string {
	return fmt.Sprintf("accounts/%s/unlock.json", accountID)
}

func SecretMaterialKey(accountID, collectionID, secretRef string) string {
	encodedRef := base64.RawURLEncoding.EncodeToString([]byte(secretRef))
	return fmt.Sprintf("accounts/%s/collections/%s/secrets/%s.txt", accountID, collectionID, encodedRef)
}

func DeviceRecordKey(accountID, deviceID string) string {
	return fmt.Sprintf("accounts/%s/devices/%s.json", accountID, deviceID)
}
