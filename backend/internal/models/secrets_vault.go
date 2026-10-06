package models

import "time"

// SecretsVault is an encrypted storage entry. Each row has a unique label
// (e.g., "hd_wallet.eth.mnemonic"), the AES-256-GCM encrypted ciphertext,
// and a flag marking which type of secret it holds. The master passphrase
// used to derive the AES key is NEVER stored in the database.
type SecretsVault struct {
	PaymintoModel
	Label       string     `gorm:"type:varchar(200);not null;uniqueIndex" json:"label"`
	Ciphertext  string     `gorm:"type:text;not null" json:"-"`
	SecretType  string     `gorm:"type:varchar(50);not null" json:"secretType"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	CreatedByID *uint      `json:"createdByID,omitempty"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
}

func (SecretsVault) TableName() string { return "secrets_vaults" }

// Well-known secret types.
const (
	SecretTypeMnemonic     = "mnemonic"
	SecretTypePrivateKey   = "private_key"
	SecretTypeAPIKey       = "api_key"
	SecretTypeSMTPPassword = "smtp_password"
	SecretTypeOther        = "other"
)

// SecretsVaultActivity is the audit trail of every vault operation. Uses the
// BaseModel base (not PaymintoModel) because it's an audit-log style table.
type SecretsVaultActivity struct {
	BaseModel
	VaultID    *uint   `json:"vaultID,omitempty"`
	Label      string  `gorm:"type:varchar(200);not null" json:"label"`
	Action     string  `gorm:"type:varchar(50);not null" json:"action"`
	MemberID   *uint   `json:"memberID,omitempty"`
	Details    *string `gorm:"type:text" json:"details,omitempty"`
	Successful bool    `gorm:"default:true" json:"successful"`
}

func (SecretsVaultActivity) TableName() string { return "secrets_vault_activities" }

// Well-known action types.
const (
	VaultActionInitialize   = "initialize"
	VaultActionUnlock       = "unlock"
	VaultActionLock         = "lock"
	VaultActionStore        = "store"
	VaultActionRetrieve     = "retrieve"
	VaultActionDelete       = "delete"
	VaultActionRotateKey    = "rotate_master_key"
	VaultActionFailedUnlock = "failed_unlock"
)
