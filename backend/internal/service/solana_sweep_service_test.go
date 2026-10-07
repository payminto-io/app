package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

type solanaKeys map[string][]byte

func (k solanaKeys) PrivateKeyForAddress(address string) ([]byte, string, error) {
	priv, ok := k[address]
	if !ok {
		return nil, "", errors.New("no key for " + address)
	}
	return append([]byte(nil), priv...), "SOL_Family", nil
}

type sweepFixture struct {
	*solanaFixture
	keys     solanaKeys
	feePayer solana.Ed25519Signer
	hot      solana.PublicKey
	svc      *SolanaSweepService
	sweepSvc *SweepService
}

func newSweepFixture(t *testing.T) *sweepFixture {
	t.Helper()
	base := newSolanaFixture(t)
	gen := func() solana.Ed25519Signer {
		_, priv, _ := ed25519.GenerateKey(rand.Reader)
		s, _ := solana.NewEd25519Signer(priv)
		return s
	}
	f := &sweepFixture{solanaFixture: base, keys: solanaKeys{}, feePayer: gen(), hot: gen().PublicKey()}
	sweepRepo := repository.NewSweepRepository(f.db)
	sweepTxRepo := repository.NewSweepTransactionRepository(f.db)
	f.sweepSvc = NewSweepService(f.db, sweepRepo, sweepTxRepo, repository.NewBlockchainRepository(f.db), f.ledger)
	sweepTxSvc := NewSweepTransactionService(sweepTxRepo, sweepRepo, f.ledger)
	f.svc = NewSolanaSweepService(f.db, solana.NewClient(f.rpc), f.chain, f.deposits, f.accounts, repository.NewBlockchainCurrencyRepository(f.db),
		sweepRepo, sweepTxRepo, f.sweepSvc, sweepTxSvc, f.keys, f.feePayer, f.hot, f.journal,
		SolanaSweepConfig{CloseAccounts: true, DropGrace: time.Minute, Send: solana.SendOptions{Poll: time.Millisecond, Wait: time.Second}})
	f.svc.now = func() time.Time { return f.now }
	return f
}

// newOwner creates an owner keypair with its USDC ATA, a payment and n confirmed deposits.
func (f *sweepFixture) newOwner(amounts ...string) (solana.Ed25519Signer, string, []models.Deposit) {
	f.t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	owner, _ := solana.NewEd25519Signer(priv)
	ata, _ := solana.AssociatedTokenAddress(owner.PublicKey(), solana.MustPublicKey(fxUSDC), solana.TokenProgram)
	f.keys[owner.PublicKey().String()] = priv
	pr, _ := f.newPayment("0", f.usdc, owner.PublicKey().String(), ata.String())
	var deps []models.Deposit
	for i, a := range amounts {
		d := models.Deposit{TxID: "deposit-" + ata.String()[:6] + string(rune('a'+i)), Amount: decimal.RequireFromString(a), Status: models.DepositStatusConfirmed,
			Confirmations: 32, RequiredConfirmations: 32, FromAddress: fxPayer, ToAddress: ata.String(), BlockchainCurrencyID: f.usdc.ID, PaymentRequestID: &pr.ID, MemberID: f.member.ID}
		must(f.t, f.db.Create(&d).Error)
		deps = append(deps, d)
	}
	return owner, ata.String(), deps
}

func (f *sweepFixture) depositStatus(id uint) string {
	var d models.Deposit
	must(f.t, f.db.First(&d, id).Error)
	return d.Status
}

func TestSolanaSweep_BatchesPerMintClosesAccountsAndBooksGasInSOL(t *testing.T) {
	f := newSweepFixture(t)
	ownerA, ataA, depsA := f.newOwner("25", "5")
	ownerB, ataB, depsB := f.newOwner("10")
	balances := map[string]string{ataA: "30000000", ataB: "10000000"}
	f.rpc.On("getTokenAccountBalance", func(p []any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": balances[solana.FirstParamString(p)], "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	var sent []string
	f.rpc.On("sendTransaction", func(p []any) (any, error) {
		sent = append(sent, solana.FirstParamString(p))
		return "SWEEPSIG1", nil
	})
	f.statuses(map[string]any{"slot": 300, "confirmations": 2, "err": nil, "confirmationStatus": "confirmed"})

	n, err := f.svc.SweepConfirmed(context.Background())
	must(t, err)
	if n != 1 || len(sent) != 1 {
		t.Fatalf("sweeps sent = %d (%d transactions)", n, len(sent))
	}
	tx, err := solana.DecodeTransactionBase64(sent[0])
	must(t, err)
	if err := tx.Verify(); err != nil {
		t.Fatal(err)
	}
	msg := tx.Message
	if msg.AccountKeys[0] != f.feePayer.PublicKey() {
		t.Fatal("fee payer must pay")
	}
	signers := map[solana.PublicKey]bool{}
	for _, s := range msg.Signers() {
		signers[s] = true
	}
	if len(signers) != 3 || !signers[ownerA.PublicKey()] || !signers[ownerB.PublicKey()] || signers[f.hot] {
		t.Fatalf("signers = %v", msg.Signers())
	}
	hotATA, _ := solana.AssociatedTokenAddress(f.hot, solana.MustPublicKey(fxUSDC), solana.TokenProgram)
	var kinds []string
	for _, ci := range msg.Instructions {
		ix, err := msg.Instruction(ci)
		must(t, err)
		switch {
		case ix.ProgramID == solana.ComputeBudgetProgram:
			kinds = append(kinds, "budget")
		case ix.ProgramID == solana.AssociatedTokenProgram:
			kinds = append(kinds, "createHotATA")
			if ix.Accounts[1].Pubkey != hotATA || ix.Accounts[2].Pubkey != f.hot {
				t.Fatalf("hot ATA create wrong: %+v", ix)
			}
		case ix.ProgramID == solana.TokenProgram && ix.Data[0] == 12:
			kinds = append(kinds, "transfer")
			if ix.Accounts[2].Pubkey != hotATA || ix.Accounts[1].Pubkey != solana.MustPublicKey(fxUSDC) {
				t.Fatalf("transfer destination wrong: %+v", ix)
			}
		case ix.ProgramID == solana.TokenProgram && ix.Data[0] == 9:
			kinds = append(kinds, "close")
			if ix.Accounts[1].Pubkey != f.feePayer.PublicKey() {
				t.Fatal("rent must go back to the fee payer")
			}
		default:
			t.Fatalf("unexpected instruction %+v", ix)
		}
	}
	want := []string{"budget", "createHotATA", "transfer", "close", "transfer", "close"}
	if len(kinds) != len(want) {
		t.Fatalf("instructions = %v", kinds)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("instructions = %v, want %v", kinds, want)
		}
	}

	for _, d := range append(depsA, depsB...) {
		if f.depositStatus(d.ID) != models.DepositStatusSwept {
			t.Fatalf("deposit %d = %s, want swept", d.ID, f.depositStatus(d.ID))
		}
	}
	var sweeps []models.Sweep
	must(t, f.db.Find(&sweeps).Error)
	if len(sweeps) != 1 || sweeps[0].Status != SweepStatusPending {
		t.Fatalf("sweeps = %+v", sweeps)
	}
	var sweepTxs []models.SweepTransaction
	must(t, f.db.Order("id").Find(&sweepTxs).Error)
	if len(sweepTxs) != 2 || sweepTxs[0].TxHash != "SWEEPSIG1" || !sweepTxs[0].Amount.Equal(decimal.RequireFromString("30")) || !sweepTxs[1].Amount.Equal(decimal.RequireFromString("10")) || sweepTxs[0].ToAddress != hotATA.String() {
		t.Fatalf("sweep txs = %+v", sweepTxs)
	}
	acct, _ := f.accounts.GetByTokenAccount(ataA)
	if acct.Status != models.SolanaDepositAccountClosed {
		t.Fatal("closed account must stop being watched")
	}
	// A second round finds nothing to sweep.
	if n, _ := f.svc.SweepConfirmed(context.Background()); n != 0 {
		t.Fatal("re-swept")
	}

	// Still only confirmed: tracked, not completed.
	done, err := f.svc.TrackConfirmations(context.Background())
	must(t, err)
	if done != 0 {
		t.Fatal("completed before finalization")
	}
	must(t, f.db.Order("id").Find(&sweepTxs).Error)
	if sweepTxs[0].Status != SweepTxStatusConfirming {
		t.Fatalf("sweep tx status = %s", sweepTxs[0].Status)
	}

	// Finalized: fee 10000 lamports, two closed ATAs refund 2 * 2039280 to the fee payer.
	f.statuses(map[string]any{"slot": 300, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	// Account order: fee payer, ATA A (closed), ATA B (closed), hot ATA (created).
	f.rpc.Result("getTransaction", map[string]any{
		"slot": 300, "transaction": map[string]any{"signatures": []string{"SWEEPSIG1"}, "message": map[string]any{"accountKeys": []any{
			map[string]any{"pubkey": f.feePayer.PublicKey().String(), "signer": true, "writable": true},
			map[string]any{"pubkey": ataA, "signer": false, "writable": true},
			map[string]any{"pubkey": ataB, "signer": false, "writable": true},
			map[string]any{"pubkey": hotATA.String(), "signer": false, "writable": true},
		}, "instructions": []any{}}},
		"meta": map[string]any{"err": nil, "fee": 10000, "preBalances": []uint64{1_000_000_000, 2039280, 2039280, 0}, "postBalances": []uint64{1_000_000_000 - 10000 + 2039280, 0, 0, 2039280}, "innerInstructions": []any{}, "preTokenBalances": []any{}, "postTokenBalances": []any{}},
	})
	done, err = f.svc.TrackConfirmations(context.Background())
	must(t, err)
	if done != 1 {
		t.Fatal("sweep not completed")
	}
	must(t, f.db.Find(&sweeps).Error)
	if sweeps[0].Status != SweepStatusCompleted || !sweeps[0].TotalAmount.Equal(decimal.RequireFromString("40")) || !sweeps[0].TotalGasFee.Equal(decimal.RequireFromString("0.00001")) {
		t.Fatalf("sweep = %+v", sweeps[0])
	}
	must(t, f.db.Order("id").Find(&sweepTxs).Error)
	if sweepTxs[0].Status != SweepTxStatusConfirmed || !sweepTxs[0].GasFee.Add(sweepTxs[1].GasFee).Equal(decimal.RequireFromString("0.00001")) {
		t.Fatalf("sweep txs = %+v", sweepTxs)
	}

	lines := f.journalLines("sweep", "1")
	byKey := map[string]decimal.Decimal{}
	for _, l := range lines {
		byKey[l.OwnerID+"/"+l.Asset] = l.Amount
	}
	expect := map[string]string{
		"cold_wallet_assets/USDC.SOLANA": "40",
		"crypto_assets/USDC.SOLANA":      "-40",
		"sweep_gas/SOL.SOLANA":           "0.00001",
		"crypto_assets/SOL.SOLANA":       "-0.00001",
	}
	if len(lines) != len(expect) {
		t.Fatalf("sweep journal = %+v", lines)
	}
	for k, v := range expect {
		if got, ok := byKey[k]; !ok || !got.Equal(decimal.RequireFromString(v)) {
			t.Fatalf("sweep journal %s = %s, want %s (all: %+v)", k, got, v, lines)
		}
	}
	rent := f.journalLines("solana_rent", "1")
	rentByKey := map[string]decimal.Decimal{}
	for _, l := range rent {
		rentByKey[l.OwnerID+"/"+l.Kind] = rentByKey[l.OwnerID+"/"+l.Kind].Add(l.Amount)
		if l.Asset != "SOL.SOLANA" {
			t.Fatalf("rent line in %s", l.Asset)
		}
	}
	if len(rent) != 4 || !rentByKey["rent_reclaimed/income"].Equal(decimal.RequireFromString("-0.00407856")) ||
		!rentByKey["token_account_rent/asset"].Equal(decimal.RequireFromString("0.00203928")) ||
		!rentByKey["crypto_assets/asset"].Equal(decimal.RequireFromString("0.00203928")) {
		t.Fatalf("rent journal = %+v", rent)
	}
	// Tracking again is a no-op.
	if done, _ := f.svc.TrackConfirmations(context.Background()); done != 0 {
		t.Fatal("completed twice")
	}
}

func TestSolanaSweep_DroppedTransactionReleasesDeposits(t *testing.T) {
	f := newSweepFixture(t)
	_, ata, deps := f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": "25000000", "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	f.rpc.Result("sendTransaction", "SWEEPSIG2")
	f.statuses(map[string]any{"slot": 300, "confirmations": 1, "err": nil, "confirmationStatus": "confirmed"})
	if n, err := f.svc.SweepConfirmed(context.Background()); err != nil || n != 1 {
		t.Fatalf("sweep: %d %v", n, err)
	}
	if f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatal("not claimed")
	}
	// The cluster forgets the signature and the sweep ages past the grace period.
	f.rpc.On("getSignatureStatuses", func([]any) (any, error) { return solana.ContextValue(1, []any{nil}), nil })
	f.rpc.Result("getTransaction", nil)
	f.svc.now = func() time.Time { return f.now.Add(time.Hour) }
	f.db.Model(&models.Sweep{}).Where("1 = 1").Update("created_at", f.now.Add(-time.Hour))
	if done, err := f.svc.TrackConfirmations(context.Background()); err != nil || done != 0 {
		t.Fatalf("track: %d %v", done, err)
	}
	if f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed {
		t.Fatalf("deposit = %s, want confirmed again", f.depositStatus(deps[0].ID))
	}
	var sweeps []models.Sweep
	f.db.Find(&sweeps)
	if sweeps[0].Status != SweepStatusFailed {
		t.Fatalf("sweep = %s", sweeps[0].Status)
	}
	acct, _ := f.accounts.GetByTokenAccount(ata)
	if acct.Status != models.SolanaDepositAccountWatching {
		t.Fatal("account must be watched again")
	}
	if lines := f.journalLines("sweep", "1"); len(lines) != 0 {
		t.Fatalf("journal for a failed sweep: %+v", lines)
	}
}

func TestSolanaSweep_BroadcastFailureReleasesClaim(t *testing.T) {
	f := newSweepFixture(t)
	_, _, deps := f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return solana.ContextValue(1, map[string]any{"amount": "25000000", "decimals": 6}), nil
	})
	f.rpc.Result("getLatestBlockhash", solana.ContextValue(1, map[string]any{"blockhash": "GH7ome3EiwEr7tu9JuTh2dpYWBJK3z69Xm1ZE3MEE6JC", "lastValidBlockHeight": 500}))
	f.rpc.On("sendTransaction", func([]any) (any, error) {
		return nil, &solana.RPCError{Code: -32002, Message: "Transaction simulation failed"}
	})
	if n, _ := f.svc.SweepConfirmed(context.Background()); n != 0 {
		t.Fatal("counted a failed broadcast")
	}
	if f.depositStatus(deps[0].ID) != models.DepositStatusConfirmed {
		t.Fatalf("deposit = %s, want confirmed", f.depositStatus(deps[0].ID))
	}
	var count int64
	f.db.Model(&models.Sweep{}).Count(&count)
	if count != 0 {
		t.Fatal("sweep row for nothing broadcast")
	}
}

func TestSolanaSweep_EmptyAccountIsMarkedSweptWithoutBroadcast(t *testing.T) {
	f := newSweepFixture(t)
	_, _, deps := f.newOwner("25")
	f.rpc.On("getTokenAccountBalance", func([]any) (any, error) {
		return nil, &solana.RPCError{Code: -32602, Message: "Invalid param: could not find account"}
	})
	if n, err := f.svc.SweepConfirmed(context.Background()); err != nil || n != 0 {
		t.Fatalf("sweep: %d %v", n, err)
	}
	if f.rpc.Count("sendTransaction") != 0 {
		t.Fatal("broadcast for an empty account")
	}
	if f.depositStatus(deps[0].ID) != models.DepositStatusSwept {
		t.Fatalf("deposit = %s, want swept", f.depositStatus(deps[0].ID))
	}
}
