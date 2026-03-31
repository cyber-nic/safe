package storage

import "fmt"

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
