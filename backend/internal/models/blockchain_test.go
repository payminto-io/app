package models

import "testing"

func TestBlockchain_TableName(t *testing.T) {
	b := Blockchain{}
	if b.TableName() != "blockchains" {
		t.Errorf("expected 'blockchains', got '%s'", b.TableName())
	}
}

func TestBlockchainFamily_TableName(t *testing.T) {
	bf := BlockchainFamily{}
	if bf.TableName() != "blockchain_families" {
		t.Errorf("expected 'blockchain_families', got '%s'", bf.TableName())
	}
}

func TestCurrency_TableName(t *testing.T) {
	c := Currency{}
	if c.TableName() != "currencies" {
		t.Errorf("expected 'currencies', got '%s'", c.TableName())
	}
}

func TestBlockchainCurrency_TableName(t *testing.T) {
	bc := BlockchainCurrency{}
	if bc.TableName() != "blockchain_currencies" {
		t.Errorf("expected 'blockchain_currencies', got '%s'", bc.TableName())
	}
}
