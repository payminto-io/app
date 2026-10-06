package models

import "testing"

func TestWalletXpub_TableName(t *testing.T) {
	if (WalletXpub{}).TableName() != "wallet_xpubs" {
		t.Error("wrong table name")
	}
}

func TestWalletSCW_TableName(t *testing.T) {
	if (WalletSCW{}).TableName() != "wallet_scws" {
		t.Error("wrong table name")
	}
}

func TestWalletFunction_TableName(t *testing.T) {
	if (WalletFunction{}).TableName() != "wallet_functions" {
		t.Error("wrong table name")
	}
}

func TestAddressContractSignature_TableName(t *testing.T) {
	if (AddressContractSignature{}).TableName() != "address_contract_signatures" {
		t.Error("wrong table name")
	}
}

func TestAddressDeployment_TableName(t *testing.T) {
	if (AddressDeployment{}).TableName() != "address_deployments" {
		t.Error("wrong table name")
	}
}

func TestEntrypointSCAddress_TableName(t *testing.T) {
	if (EntrypointSCAddress{}).TableName() != "entrypoint_sc_addresses" {
		t.Error("wrong table name")
	}
}

func TestInternalBlockchainTransaction_TableName(t *testing.T) {
	if (InternalBlockchainTransaction{}).TableName() != "internal_blockchain_transactions" {
		t.Error("wrong table name")
	}
}

func TestMissedDeposit_TableName(t *testing.T) {
	if (MissedDeposit{}).TableName() != "missed_deposits" {
		t.Error("wrong table name")
	}
}

func TestMissedDepositStatusConstants(t *testing.T) {
	if MissedDepositStatusPending != "pending" {
		t.Error("wrong constant")
	}
}
