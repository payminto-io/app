package models

import "testing"

func TestSecretsVault_TableName(t *testing.T) {
	if (SecretsVault{}).TableName() != "secrets_vaults" {
		t.Error("wrong table name")
	}
}

func TestSecretsVaultActivity_TableName(t *testing.T) {
	if (SecretsVaultActivity{}).TableName() != "secrets_vault_activities" {
		t.Error("wrong table name")
	}
}

func TestSecretTypeConstants(t *testing.T) {
	if SecretTypeMnemonic != "mnemonic" {
		t.Error("wrong constant")
	}
	if SecretTypePrivateKey != "private_key" {
		t.Error("wrong constant")
	}
	if SecretTypeSMTPPassword != "smtp_password" {
		t.Error("wrong constant")
	}
}

func TestVaultActionConstants(t *testing.T) {
	if VaultActionUnlock != "unlock" {
		t.Error("wrong constant")
	}
	if VaultActionStore != "store" {
		t.Error("wrong constant")
	}
	if VaultActionRotateKey != "rotate_master_key" {
		t.Error("wrong constant")
	}
}
