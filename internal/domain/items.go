package domain

import "encoding/json"

type VaultItemKind string

const (
	VaultItemKindLogin  VaultItemKind = "login"
	VaultItemKindNote   VaultItemKind = "note"
	VaultItemKindAPIKey VaultItemKind = "apiKey"
	VaultItemKindSSHKey VaultItemKind = "sshKey"
	VaultItemKindTOTP   VaultItemKind = "totp"
)

type VaultItemSummary struct {
	ID          string
	Kind        VaultItemKind
	Title       string
	Description string
}

type VaultItem struct {
	ID            string        `json:"id"`
	Kind          VaultItemKind `json:"kind"`
	Title         string        `json:"title"`
	Tags          []string      `json:"tags"`
	Username      string        `json:"username,omitempty"`
	URLs          []string      `json:"urls,omitempty"`
	BodyPreview   string        `json:"bodyPreview,omitempty"`
	Service       string        `json:"service,omitempty"`
	Host          string        `json:"host,omitempty"`
	Issuer        string        `json:"issuer,omitempty"`
	AccountName   string        `json:"accountName,omitempty"`
	Digits        int           `json:"digits,omitempty"`
	PeriodSeconds int           `json:"periodSeconds,omitempty"`
	Algorithm     string        `json:"algorithm,omitempty"`
	SecretRef     string        `json:"secretRef,omitempty"`
}

type VaultItemRecord struct {
	SchemaVersion int       `json:"schemaVersion"`
	Item          VaultItem `json:"item"`
}

func (item VaultItem) Summary() VaultItemSummary {
	switch item.Kind {
	case VaultItemKindLogin:
		return VaultItemSummary{
			ID:          item.ID,
			Kind:        item.Kind,
			Title:       item.Title,
			Description: "Login for " + item.Username,
		}
	case VaultItemKindTOTP:
		return VaultItemSummary{
			ID:          item.ID,
			Kind:        item.Kind,
			Title:       item.Title,
			Description: "Built-in authenticator for " + item.Issuer + " (" + item.AccountName + ")",
		}
	default:
		return VaultItemSummary{
			ID:          item.ID,
			Kind:        item.Kind,
			Title:       item.Title,
			Description: string(item.Kind) + " item",
		}
	}
}

func (record VaultItemRecord) Validate() error {
	if record.SchemaVersion != 1 {
		return ErrInvalidVaultItemRecord("schemaVersion")
	}

	if record.Item.ID == "" {
		return ErrInvalidVaultItemRecord("item.id")
	}

	if record.Item.Title == "" {
		return ErrInvalidVaultItemRecord("item.title")
	}

	if record.Item.Kind == "" {
		return ErrInvalidVaultItemRecord("item.kind")
	}

	if record.Item.Tags == nil {
		return ErrInvalidVaultItemRecord("item.tags")
	}

	switch record.Item.Kind {
	case VaultItemKindLogin:
		if record.Item.Username == "" {
			return ErrInvalidVaultItemRecord("item.username")
		}
		if len(record.Item.URLs) == 0 {
			return ErrInvalidVaultItemRecord("item.urls")
		}
	case VaultItemKindTOTP:
		if record.Item.Issuer == "" {
			return ErrInvalidVaultItemRecord("item.issuer")
		}
		if record.Item.AccountName == "" {
			return ErrInvalidVaultItemRecord("item.accountName")
		}
		if record.Item.Digits != 6 {
			return ErrInvalidVaultItemRecord("item.digits")
		}
		if record.Item.PeriodSeconds != 30 {
			return ErrInvalidVaultItemRecord("item.periodSeconds")
		}
		if record.Item.Algorithm != "SHA1" {
			return ErrInvalidVaultItemRecord("item.algorithm")
		}
		if record.Item.SecretRef == "" {
			return ErrInvalidVaultItemRecord("item.secretRef")
		}
	default:
		return ErrInvalidVaultItemRecord("item.kind")
	}

	return nil
}

func (record VaultItemRecord) CanonicalJSON() ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}

	type canonicalLoginItem struct {
		ID       string        `json:"id"`
		Kind     VaultItemKind `json:"kind"`
		Title    string        `json:"title"`
		Tags     []string      `json:"tags"`
		Username string        `json:"username"`
		URLs     []string      `json:"urls"`
	}

	type canonicalTotpItem struct {
		ID            string        `json:"id"`
		Kind          VaultItemKind `json:"kind"`
		Title         string        `json:"title"`
		Tags          []string      `json:"tags"`
		Issuer        string        `json:"issuer"`
		AccountName   string        `json:"accountName"`
		Digits        int           `json:"digits"`
		PeriodSeconds int           `json:"periodSeconds"`
		Algorithm     string        `json:"algorithm"`
		SecretRef     string        `json:"secretRef"`
	}

	switch record.Item.Kind {
	case VaultItemKindLogin:
		return json.Marshal(struct {
			SchemaVersion int                `json:"schemaVersion"`
			Item          canonicalLoginItem `json:"item"`
		}{
			SchemaVersion: record.SchemaVersion,
			Item: canonicalLoginItem{
				ID:       record.Item.ID,
				Kind:     record.Item.Kind,
				Title:    record.Item.Title,
				Tags:     record.Item.Tags,
				Username: record.Item.Username,
				URLs:     record.Item.URLs,
			},
		})
	case VaultItemKindTOTP:
		return json.Marshal(struct {
			SchemaVersion int               `json:"schemaVersion"`
			Item          canonicalTotpItem `json:"item"`
		}{
			SchemaVersion: record.SchemaVersion,
			Item: canonicalTotpItem{
				ID:            record.Item.ID,
				Kind:          record.Item.Kind,
				Title:         record.Item.Title,
				Tags:          record.Item.Tags,
				Issuer:        record.Item.Issuer,
				AccountName:   record.Item.AccountName,
				Digits:        record.Item.Digits,
				PeriodSeconds: record.Item.PeriodSeconds,
				Algorithm:     record.Item.Algorithm,
				SecretRef:     record.Item.SecretRef,
			},
		})
	default:
		return nil, ErrInvalidVaultItemRecord("item.kind")
	}
}

func ParseVaultItemRecordJSON(data []byte) (VaultItemRecord, error) {
	var record VaultItemRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return VaultItemRecord{}, err
	}

	if err := record.Validate(); err != nil {
		return VaultItemRecord{}, err
	}

	return record, nil
}

type invalidVaultItemRecordError string

func (field invalidVaultItemRecordError) Error() string {
	return "invalid vault item record field: " + string(field)
}

func ErrInvalidVaultItemRecord(field string) error {
	return invalidVaultItemRecordError(field)
}

func StarterVaultItems() []VaultItemSummary {
	records := StarterVaultItemRecords()
	summaries := make([]VaultItemSummary, 0, len(records))

	for _, record := range records {
		summaries = append(summaries, record.Item.Summary())
	}

	return summaries
}

func StarterVaultItemRecords() []VaultItemRecord {
	return []VaultItemRecord{
		{
			SchemaVersion: 1,
			Item: VaultItem{
				ID:       "login-gmail-primary",
				Kind:     VaultItemKindLogin,
				Title:    "Gmail",
				Tags:     []string{"email", "personal"},
				Username: "alice@example.com",
				URLs:     []string{"https://accounts.google.com"},
			},
		},
		{
			SchemaVersion: 1,
			Item: VaultItem{
				ID:            "totp-gmail-primary",
				Kind:          VaultItemKindTOTP,
				Title:         "Gmail 2FA",
				Tags:          []string{"2fa", "authenticator"},
				Issuer:        "Google",
				AccountName:   "alice@example.com",
				Digits:        6,
				PeriodSeconds: 30,
				Algorithm:     "SHA1",
				SecretRef:     "vault-secret://totp/gmail-primary",
			},
		},
	}
}
