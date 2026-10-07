package solana

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

var (
	fxOwner   = MustPublicKey("HAgk14JpMQLgt6rVgv7cBQFJWFto5Dqxi472uT3DKpqk")
	fxPayer   = MustPublicKey("9h1cLBiraaUqM1CdJTaVaew1oQtgQUW24FZ8YdnLLgJY")
	fxUSDC    = MustPublicKey("EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v")
	fxUSDT    = MustPublicKey("Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYb")
	fxUSDCATA = MustPublicKey("5N3f1tj9v1vc5TUZ8S7mCAnVmjVKrfnzXWhxLaxyZAgt")
	fxUSDTATA = MustPublicKey("4fz24twEFEWmsAKeeD7hgGZtVBHiGjRLEFuciSgRcgdw")
)

func usdcWatch() Watch {
	return Watch{Owner: fxOwner, TokenAccount: fxUSDCATA, Mint: fxUSDC, TokenProgram: TokenProgram, Decimals: 6}
}

func usdtWatch() Watch {
	return Watch{Owner: fxOwner, TokenAccount: fxUSDTATA, Mint: fxUSDT, TokenProgram: TokenProgram, Decimals: 6}
}

func fixtureTx(t *testing.T, name string) *ParsedTransaction {
	t.Helper()
	tx, err := LoadFixtureTransaction(filepath.Join("testdata", "tx", name))
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

func TestExtractDeposits_USDCTransferChecked(t *testing.T) {
	credit, anomalies := ExtractDeposits(fixtureTx(t, "usdc_transfer_checked.json"), usdcWatch())
	if credit == nil {
		t.Fatal("expected a credit")
	}
	if !credit.Amount.Equal(decimal.RequireFromString("25")) {
		t.Fatalf("amount = %s, want 25", credit.Amount)
	}
	if credit.From != fxPayer.String() || credit.To != fxUSDCATA.String() || credit.Mint != fxUSDC.String() {
		t.Fatalf("credit routing wrong: %+v", credit)
	}
	if credit.Slot != 250000123 || credit.Signature == "" {
		t.Fatalf("credit slot/signature wrong: %+v", credit)
	}
	if len(anomalies) != 0 {
		t.Fatalf("unexpected anomalies: %+v", anomalies)
	}
}

func TestExtractDeposits_USDTPlainTransferResolvesMintFromBalances(t *testing.T) {
	credit, anomalies := ExtractDeposits(fixtureTx(t, "usdt_transfer.json"), usdtWatch())
	if credit == nil || !credit.Amount.Equal(decimal.RequireFromString("10")) {
		t.Fatalf("credit = %+v, want 10 USDT", credit)
	}
	if credit.Mint != fxUSDT.String() {
		t.Fatalf("mint = %s, want USDT", credit.Mint)
	}
	if len(anomalies) != 0 {
		t.Fatalf("unexpected anomalies: %+v", anomalies)
	}
	// The same transaction is not a USDC deposit.
	if c, _ := ExtractDeposits(fixtureTx(t, "usdt_transfer.json"), usdcWatch()); c != nil {
		t.Fatalf("USDT transfer credited to USDC watch: %+v", c)
	}
}

func TestExtractDeposits_USDTToUSDCPaymentIsWrongMintAnomaly(t *testing.T) {
	credit, anomalies := ExtractDeposits(fixtureTx(t, "wrong_mint_usdt_to_usdc_payment.json"), usdcWatch())
	if credit != nil {
		t.Fatalf("wrong-mint transfer must never be credited: %+v", credit)
	}
	if len(anomalies) != 1 || anomalies[0].Kind != AnomalyWrongMint {
		t.Fatalf("anomalies = %+v, want one wrong_mint", anomalies)
	}
	a := anomalies[0]
	if a.Mint != fxUSDT.String() || a.To != fxUSDTATA.String() || !a.Amount.Equal(decimal.RequireFromString("25")) {
		t.Fatalf("anomaly detail wrong: %+v", a)
	}
}

func TestExtractDeposits_PaymentToOwnerCreatesATAAndCredits(t *testing.T) {
	credit, anomalies := ExtractDeposits(fixtureTx(t, "owner_payment_creates_ata.json"), usdcWatch())
	if credit == nil || !credit.Amount.Equal(decimal.RequireFromString("40")) {
		t.Fatalf("credit = %+v, want 40 USDC", credit)
	}
	if len(anomalies) != 0 {
		t.Fatalf("unexpected anomalies: %+v", anomalies)
	}
}

func TestExtractDeposits_NativeSOLToOwnerIsAnomaly(t *testing.T) {
	credit, anomalies := ExtractDeposits(fixtureTx(t, "native_sol_to_owner.json"), usdcWatch())
	if credit != nil {
		t.Fatal("SOL must not be credited as USDC")
	}
	if len(anomalies) != 1 || anomalies[0].Kind != AnomalyNativeToOwner || !anomalies[0].Amount.Equal(decimal.RequireFromString("0.5")) {
		t.Fatalf("anomalies = %+v", anomalies)
	}
}

func TestExtractDeposits_FailedTransactionYieldsNothing(t *testing.T) {
	credit, anomalies := ExtractDeposits(fixtureTx(t, "failed_transfer.json"), usdcWatch())
	if credit != nil || len(anomalies) != 0 {
		t.Fatalf("failed tx produced credit=%+v anomalies=%+v", credit, anomalies)
	}
}

func TestExtractDeposits_InnerCPITransferIsCredited(t *testing.T) {
	credit, anomalies := ExtractDeposits(fixtureTx(t, "cpi_inner_transfer.json"), usdcWatch())
	if credit == nil || !credit.Amount.Equal(decimal.RequireFromString("12.5")) {
		t.Fatalf("credit = %+v, want 12.5", credit)
	}
	if len(anomalies) != 0 {
		t.Fatalf("unexpected anomalies: %+v", anomalies)
	}
}

func TestExtractDeposits_BalanceRiseWithoutParsedTransferIsFlagged(t *testing.T) {
	tx := fixtureTx(t, "usdc_transfer_checked.json")
	tx.Transaction.Message.Instructions = nil
	credit, anomalies := ExtractDeposits(tx, usdcWatch())
	if credit != nil {
		t.Fatal("no parsed transfer means no credit")
	}
	if len(anomalies) != 1 || anomalies[0].Kind != AnomalyUnparsedCredit || !anomalies[0].Amount.Equal(decimal.RequireFromString("25")) {
		t.Fatalf("anomalies = %+v", anomalies)
	}
}

func TestExtractDeposits_Token2022ProgramRejectedForSPLAsset(t *testing.T) {
	tx := fixtureTx(t, "usdc_transfer_checked.json")
	tx.Transaction.Message.Instructions[0].ProgramID = Token2022Program.String()
	credit, anomalies := ExtractDeposits(tx, usdcWatch())
	if credit != nil {
		t.Fatal("token-2022 transfer credited to an SPL asset")
	}
	if len(anomalies) == 0 || anomalies[0].Kind != AnomalyWrongTokenProgram {
		t.Fatalf("anomalies = %+v", anomalies)
	}
}

func TestRentMovements(t *testing.T) {
	tx := fixtureTx(t, "usdc_transfer_checked.json")
	if r, f := RentMovements(tx); r != 0 || f != 0 {
		t.Fatalf("plain transfer: reclaimed %d funded %d", r, f)
	}
	// A sweep closes the deposit ATA (index 1) and creates the hot ATA (index 2).
	tx.Meta.PreBalances = []uint64{1_000_000_000, 2039280, 0, 374004672, 934087680}
	tx.Meta.PostBalances = []uint64{1_000_000_000 - 5000, 0, 2039280, 374004672, 934087680}
	if r, f := RentMovements(tx); r != 2039280 || f != 2039280 {
		t.Fatalf("sweep: reclaimed %d funded %d", r, f)
	}
}

// I4: a token program that withholds part of a transfer (Token-2022 transfer fee) must credit the
// balance delta, never the instruction amount, and flag the difference.
func TestExtractDeposits_WithheldAmountCreditsDeltaAndFlags(t *testing.T) {
	tx := fixtureTx(t, "usdc_transfer_checked.json")
	tx.Meta.PostTokenBalances[1].UITokenAmount.Amount = "24000000"
	credit, anomalies := ExtractDeposits(tx, usdcWatch())
	if credit == nil || !credit.Amount.Equal(decimal.RequireFromString("24")) {
		t.Fatalf("credit = %+v, want the delivered 24", credit)
	}
	if len(anomalies) != 1 || anomalies[0].Kind != AnomalyWithheldAmount || !anomalies[0].Amount.Equal(decimal.RequireFromString("1")) {
		t.Fatalf("anomalies = %+v", anomalies)
	}
}

// M2: a wallet that treats the ATA as an owner sends to ATA(ATA, mint); nobody can sign for it.
func TestExtractDeposits_TokenStrandedInATAOfATAIsFlagged(t *testing.T) {
	tx := fixtureTx(t, "usdc_transfer_checked.json")
	stranded, _ := AssociatedTokenAddress(fxUSDCATA, fxUSDC, TokenProgram)
	tx.Transaction.Message.AccountKeys[2].Pubkey = stranded.String()
	ix := &tx.Transaction.Message.Instructions[0]
	ix.Parsed = []byte(strings.Replace(string(ix.Parsed), fxUSDCATA.String(), stranded.String(), 1))
	for i := range tx.Meta.PostTokenBalances {
		if tx.Meta.PostTokenBalances[i].AccountIndex == 2 {
			tx.Meta.PostTokenBalances[i].Owner = fxUSDCATA.String()
			tx.Meta.PreTokenBalances[i].Owner = fxUSDCATA.String()
		}
	}
	credit, anomalies := ExtractDeposits(tx, usdcWatch())
	if credit != nil {
		t.Fatal("stranded token credited")
	}
	if len(anomalies) != 1 || anomalies[0].Kind != AnomalyStrandedInPDA || anomalies[0].To != stranded.String() {
		t.Fatalf("anomalies = %+v", anomalies)
	}
}
