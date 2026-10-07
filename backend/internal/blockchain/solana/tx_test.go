package solana

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func newTestSigner(t *testing.T) Ed25519Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewEd25519Signer(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCompactU16RoundTrip(t *testing.T) {
	for _, n := range []int{0, 1, 127, 128, 255, 256, 16383, 16384} {
		b := appendCompactU16(nil, n)
		got, used, err := decodeCompactU16(b)
		if err != nil || got != n || used != len(b) {
			t.Fatalf("n=%d: got %d used %d err %v", n, got, used, err)
		}
	}
}

func TestCompileMessage_OrdersAccountsAndSigns(t *testing.T) {
	feePayer := newTestSigner(t)
	owner := newTestSigner(t)
	mint := MustPublicKey("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
	source, _ := AssociatedTokenAddress(owner.PublicKey(), mint, TokenProgram)
	dest, _ := AssociatedTokenAddress(feePayer.PublicKey(), mint, TokenProgram)
	blockhash := MustPublicKey("GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC")

	ixs := []Instruction{
		ComputeBudgetSetUnitLimit(50_000),
		TokenTransferChecked(TokenProgram, source, mint, dest, owner.PublicKey(), 1_000_000, 6),
		TokenCloseAccount(TokenProgram, source, feePayer.PublicKey(), owner.PublicKey()),
	}
	msg, err := CompileMessage(feePayer.PublicKey(), blockhash, ixs)
	if err != nil {
		t.Fatal(err)
	}
	if msg.AccountKeys[0] != feePayer.PublicKey() {
		t.Fatal("fee payer must be the first account")
	}
	if msg.NumRequiredSignatures != 2 {
		t.Fatalf("required signatures = %d, want 2 (fee payer + owner)", msg.NumRequiredSignatures)
	}
	if msg.NumReadonlySignedAccounts != 1 {
		t.Fatalf("readonly signed = %d, want 1 (owner only signs)", msg.NumReadonlySignedAccounts)
	}
	// mint, token program and compute budget program are readonly unsigned.
	if msg.NumReadonlyUnsignedAccounts != 3 {
		t.Fatalf("readonly unsigned = %d, want 3", msg.NumReadonlyUnsignedAccounts)
	}
	// Writable non-signers (source, dest) come before readonly ones.
	for i, k := range msg.AccountKeys {
		if k == source || k == dest {
			if i < 2 || i > 3 {
				t.Fatalf("token account at index %d, want 2 or 3", i)
			}
		}
	}
	tx, err := Sign(msg, []Signer{owner, feePayer})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Verify(); err != nil {
		t.Fatal(err)
	}
	if len(tx.Serialize()) > MaxTransactionSize {
		t.Fatal("tx exceeds packet size")
	}
	if tx.Signature() == "" {
		t.Fatal("empty signature")
	}
	if _, err := Sign(msg, []Signer{feePayer}); err == nil {
		t.Fatal("signing without the owner must fail")
	}
}

func TestInstructionLayouts(t *testing.T) {
	a, b := newTestSigner(t).PublicKey(), newTestSigner(t).PublicKey()
	ix := TokenTransferChecked(TokenProgram, a, b, a, b, 0x0102030405060708, 6)
	want := []byte{12, 8, 7, 6, 5, 4, 3, 2, 1, 6}
	if string(ix.Data) != string(want) {
		t.Fatalf("transferChecked data = %x, want %x", ix.Data, want)
	}
	if d := ComputeBudgetSetUnitPrice(1000).Data; string(d) != string([]byte{3, 0xe8, 3, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("unit price data = %x", d)
	}
	if d := SystemTransfer(a, b, 1).Data; string(d) != string([]byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatalf("system transfer data = %x", d)
	}
	cre, ata, err := CreateAssociatedTokenAccountIdempotent(a, b, a, TokenProgram)
	if err != nil {
		t.Fatal(err)
	}
	if cre.Accounts[1].Pubkey != ata || len(cre.Accounts) != 6 || cre.Data[0] != 1 {
		t.Fatal("create ATA idempotent layout wrong")
	}
}
