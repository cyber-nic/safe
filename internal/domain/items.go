package domain

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

func StarterVaultItems() []VaultItemSummary {
	return []VaultItemSummary{
		{
			ID:          "login-gmail-primary",
			Kind:        VaultItemKindLogin,
			Title:       "Gmail",
			Description: "Login for alice@example.com",
		},
		{
			ID:          "totp-gmail-primary",
			Kind:        VaultItemKindTOTP,
			Title:       "Gmail 2FA",
			Description: "Built-in authenticator for Google (alice@example.com)",
		},
	}
}
