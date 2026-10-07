package solana

import (
	"testing"
)

func TestSweepInstructions_BatchTransfersCloseAndSponsorFees(t *testing.T) {
	feePayer := newTestSigner(t)
	hot := newTestSigner(t)
	ownerA, ownerB := newTestSigner(t), newTestSigner(t)
	ataA, _ := AssociatedTokenAddress(ownerA.PublicKey(), fxUSDC, TokenProgram)
	ataB, _ := AssociatedTokenAddress(ownerB.PublicKey(), fxUSDC, TokenProgram)
	hotATA, _ := AssociatedTokenAddress(hot.PublicKey(), fxUSDC, TokenProgram)

	p := SweepParams{
		FeePayer: feePayer.PublicKey(), HotWalletOwner: hot.PublicKey(), Mint: fxUSDC, TokenProgram: TokenProgram, Decimals: 6,
		Items:            []SweepItem{{Owner: ownerA.PublicKey(), TokenAccount: ataA, Amount: 25_000_000, Close: true}, {Owner: ownerB.PublicKey(), TokenAccount: ataB, Amount: 10_000_000, Close: true}},
		ComputeUnitLimit: 60_000, PriorityFeeMicroLamports: 1_000, CloseAccounts: true,
	}
	ixs, gotHot, err := SweepInstructions(p)
	if err != nil {
		t.Fatal(err)
	}
	if gotHot != hotATA {
		t.Fatal("hot ATA derivation wrong")
	}
	// budget limit, budget price, create hot ATA, then (transfer, close) per item.
	if len(ixs) != 3+2*len(p.Items) {
		t.Fatalf("instructions = %d", len(ixs))
	}
	if ixs[0].ProgramID != ComputeBudgetProgram || ixs[1].ProgramID != ComputeBudgetProgram || ixs[2].ProgramID != AssociatedTokenProgram {
		t.Fatal("prefix instructions wrong")
	}
	if ixs[2].Accounts[0].Pubkey != feePayer.PublicKey() || !ixs[2].Accounts[0].Signer {
		t.Fatal("hot ATA rent must be paid by the fee payer")
	}
	tr, cl := ixs[3], ixs[4]
	if tr.Data[0] != tokenTransferChecked || tr.Accounts[0].Pubkey != ataA || tr.Accounts[2].Pubkey != hotATA || tr.Accounts[3].Pubkey != ownerA.PublicKey() || !tr.Accounts[3].Signer {
		t.Fatalf("transfer instruction wrong: %+v", tr)
	}
	if cl.Data[0] != tokenCloseAccount || cl.Accounts[0].Pubkey != ataA || cl.Accounts[1].Pubkey != feePayer.PublicKey() || cl.Accounts[2].Pubkey != ownerA.PublicKey() {
		t.Fatalf("close instruction wrong: %+v", cl)
	}

	msg, signers, err := BuildSweepMessage(p, "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC")
	if err != nil {
		t.Fatal(err)
	}
	if signers[0] != feePayer.PublicKey() || len(signers) != 3 {
		t.Fatalf("signers = %v", signers)
	}
	for _, s := range signers {
		if s == hot.PublicKey() {
			t.Fatal("the hot wallet never signs a sweep")
		}
	}
	tx, err := Sign(msg, []Signer{feePayer, ownerA, ownerB})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Verify(); err != nil {
		t.Fatal(err)
	}
	if n := len(tx.Serialize()); n > MaxTransactionSize {
		t.Fatalf("two-item sweep is %d bytes", n)
	}
	// DefaultSweepBatchSize deposits fit one packet; one more does not.
	var items []SweepItem
	var sigs []Signer
	for i := 0; i < DefaultSweepBatchSize+1; i++ {
		o := newTestSigner(t)
		ata, _ := AssociatedTokenAddress(o.PublicKey(), fxUSDC, TokenProgram)
		items = append(items, SweepItem{Owner: o.PublicKey(), TokenAccount: ata, Amount: 1, Close: true})
		sigs = append(sigs, o)
	}
	p.Items = items[:DefaultSweepBatchSize]
	msg, _, err = BuildSweepMessage(p, "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC")
	if err != nil {
		t.Fatal(err)
	}
	tx, _ = Sign(msg, append(append([]Signer{}, sigs[:DefaultSweepBatchSize]...), feePayer))
	if n := len(tx.Serialize()); n > MaxTransactionSize {
		t.Fatalf("%d-item sweep is %d bytes, over the packet limit", DefaultSweepBatchSize, n)
	}
	p.Items = items
	msg, _, _ = BuildSweepMessage(p, "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC")
	tx, _ = Sign(msg, append(sigs, feePayer))
	if n := len(tx.Serialize()); n <= MaxTransactionSize {
		t.Fatalf("%d-item sweep is %d bytes; raise DefaultSweepBatchSize", DefaultSweepBatchSize+1, n)
	}
}

func TestSweepInstructions_RejectsEmptyAndZero(t *testing.T) {
	if _, _, err := SweepInstructions(SweepParams{}); err == nil {
		t.Fatal("empty sweep accepted")
	}
	o := newTestSigner(t)
	ata, _ := AssociatedTokenAddress(o.PublicKey(), fxUSDC, TokenProgram)
	p := SweepParams{FeePayer: o.PublicKey(), HotWalletOwner: o.PublicKey(), Mint: fxUSDC, TokenProgram: TokenProgram, Items: []SweepItem{{Owner: o.PublicKey(), TokenAccount: ata}}}
	if _, _, err := SweepInstructions(p); err == nil {
		t.Fatal("zero amount accepted")
	}
}
