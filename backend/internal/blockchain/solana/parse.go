package solana

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

// Watch describes one payment's deposit accounts: the owner keypair's address and its ATA for the
// expected mint. Payers may send to either; only credits of Mint into TokenAccount count.
type Watch struct {
	Owner        PublicKey
	TokenAccount PublicKey
	Mint         PublicKey
	TokenProgram PublicKey
	Decimals     uint8
}

// Credit is a transfer of the expected mint into the watched token account, summed per transaction.
type Credit struct {
	Signature string
	Slot      uint64
	From      string
	To        string
	Mint      string
	RawAmount *big.Int
	Amount    decimal.Decimal
}

// Anomaly kinds recorded to missed_deposits rather than credited.
const (
	AnomalyWrongMint         = "wrong_mint"
	AnomalyWrongTokenProgram = "wrong_token_program"
	AnomalyNativeToOwner     = "native_sol_to_owner"
	AnomalyUnparsedCredit    = "unparsed_credit"
)

// Anomaly is value that reached the payment's accounts but must not be credited as the payment asset.
type Anomaly struct {
	Kind      string
	Signature string
	Slot      uint64
	From      string
	To        string
	Mint      string
	Amount    decimal.Decimal
	Detail    string
}

// ExtractDeposits walks top-level and inner instructions of a jsonParsed transaction and returns the
// credit to w.TokenAccount (nil when none) plus anomalies. Failed transactions yield nothing.
func ExtractDeposits(tx *ParsedTransaction, w Watch) (*Credit, []Anomaly) {
	if tx == nil || tx.Failed() || tx.Meta == nil {
		return nil, nil
	}
	sig := tx.Signature()
	ata := w.TokenAccount.String()
	owner := w.Owner.String()
	mint := w.Mint.String()
	balances := indexTokenBalances(tx)

	var credit *Credit
	var anomalies []Anomaly
	addCredit := func(from string, raw *big.Int, decimals uint8) {
		if credit == nil {
			credit = &Credit{Signature: sig, Slot: tx.Slot, From: from, To: ata, Mint: mint, RawAmount: new(big.Int)}
		}
		credit.RawAmount.Add(credit.RawAmount, raw)
		credit.Amount = decimal.NewFromBigInt(credit.RawAmount, -int32(decimals))
	}

	for _, ix := range allInstructions(tx) {
		switch ix.Program {
		case "spl-token":
			var ti tokenInstruction
			if err := json.Unmarshal(ix.Parsed, &ti); err != nil {
				continue
			}
			dest := ti.Info.Destination
			if ti.Type == "mintTo" || ti.Type == "mintToChecked" {
				dest = ti.Info.Account
			}
			switch ti.Type {
			case "transfer", "transferChecked", "mintTo", "mintToChecked":
			default:
				continue
			}
			raw, decimals, ok := tokenQuantity(ti, balances[dest], w.Decimals)
			if !ok {
				continue
			}
			ixMint := ti.Info.Mint
			if ixMint == "" {
				ixMint = balances[dest].Mint
			}
			from := ti.Info.Authority
			if src, ok := balances[ti.Info.Source]; ok && src.Owner != "" {
				from = src.Owner
			}
			switch {
			case dest == ata:
				if ix.ProgramID != w.TokenProgram.String() {
					anomalies = append(anomalies, Anomaly{Kind: AnomalyWrongTokenProgram, Signature: sig, Slot: tx.Slot, From: from, To: dest, Mint: ixMint,
						Amount: decimal.NewFromBigInt(raw, -int32(decimals)), Detail: "token program " + ix.ProgramID})
					continue
				}
				if ixMint != "" && ixMint != mint {
					anomalies = append(anomalies, Anomaly{Kind: AnomalyWrongMint, Signature: sig, Slot: tx.Slot, From: from, To: dest, Mint: ixMint,
						Amount: decimal.NewFromBigInt(raw, -int32(decimals)), Detail: "expected mint " + mint})
					continue
				}
				addCredit(from, raw, decimals)
			case balances[dest].Owner == owner && ixMint != mint:
				// The payer sent another token to the owner address; it sits in the owner's other ATA.
				anomalies = append(anomalies, Anomaly{Kind: AnomalyWrongMint, Signature: sig, Slot: tx.Slot, From: from, To: dest, Mint: ixMint,
					Amount: decimal.NewFromBigInt(raw, -int32(decimals)), Detail: "expected mint " + mint})
			}
		case "system":
			var si systemInstruction
			if err := json.Unmarshal(ix.Parsed, &si); err != nil || si.Type != "transfer" || si.Info.Destination != owner {
				continue
			}
			anomalies = append(anomalies, Anomaly{Kind: AnomalyNativeToOwner, Signature: sig, Slot: tx.Slot, From: si.Info.Source, To: owner, Mint: "",
				Amount: decimal.New(int64(si.Info.Lamports), -9), Detail: fmt.Sprintf("%d lamports", si.Info.Lamports)})
		}
	}

	// The balance delta is the ground truth; a larger delta than the parsed credits means a
	// transfer we did not understand, which is flagged rather than silently credited.
	if delta := tokenDelta(tx, ata, mint); delta != nil {
		parsed := new(big.Int)
		if credit != nil {
			parsed = credit.RawAmount
		}
		if delta.Cmp(parsed) > 0 {
			diff := new(big.Int).Sub(delta, parsed)
			anomalies = append(anomalies, Anomaly{Kind: AnomalyUnparsedCredit, Signature: sig, Slot: tx.Slot, To: ata, Mint: mint,
				Amount: decimal.NewFromBigInt(diff, -int32(w.Decimals)), Detail: "balance rose more than parsed transfers"})
		}
	}
	return credit, anomalies
}

func allInstructions(tx *ParsedTransaction) []ParsedInstruction {
	out := append([]ParsedInstruction{}, tx.Transaction.Message.Instructions...)
	for _, inner := range tx.Meta.InnerInstructions {
		out = append(out, inner.Instructions...)
	}
	return out
}

// indexTokenBalances maps token account pubkey to its post balance entry (mint, owner, program).
func indexTokenBalances(tx *ParsedTransaction) map[string]ParsedTokenBalance {
	out := map[string]ParsedTokenBalance{}
	keys := tx.Transaction.Message.AccountKeys
	for _, b := range append(append([]ParsedTokenBalance{}, tx.Meta.PreTokenBalances...), tx.Meta.PostTokenBalances...) {
		if b.AccountIndex >= 0 && b.AccountIndex < len(keys) {
			out[keys[b.AccountIndex].Pubkey] = b
		}
	}
	return out
}

func tokenQuantity(ti tokenInstruction, dest ParsedTokenBalance, fallbackDecimals uint8) (*big.Int, uint8, bool) {
	raw := new(big.Int)
	decimals := fallbackDecimals
	switch {
	case ti.Info.TokenAmount != nil:
		if _, ok := raw.SetString(ti.Info.TokenAmount.Amount, 10); !ok {
			return nil, 0, false
		}
		decimals = ti.Info.TokenAmount.Decimals
	case ti.Info.Amount != "":
		if _, ok := raw.SetString(ti.Info.Amount, 10); !ok {
			return nil, 0, false
		}
		if dest.Mint != "" {
			decimals = dest.UITokenAmount.Decimals
		}
	default:
		return nil, 0, false
	}
	return raw, decimals, true
}

// tokenDelta returns post minus pre balance of account for mint, nil when the account has no entries.
func tokenDelta(tx *ParsedTransaction, account, mint string) *big.Int {
	keys := tx.Transaction.Message.AccountKeys
	find := func(list []ParsedTokenBalance) (*big.Int, bool) {
		for _, b := range list {
			if b.AccountIndex < len(keys) && keys[b.AccountIndex].Pubkey == account && b.Mint == mint {
				v, ok := new(big.Int).SetString(b.UITokenAmount.Amount, 10)
				return v, ok
			}
		}
		return new(big.Int), false
	}
	post, okPost := find(tx.Meta.PostTokenBalances)
	pre, okPre := find(tx.Meta.PreTokenBalances)
	if !okPost && !okPre {
		return nil
	}
	return new(big.Int).Sub(post, pre)
}

// FeePayerRentRefund returns lamports the fee payer gained beyond the fee (rent from closed accounts).
func FeePayerRentRefund(tx *ParsedTransaction) uint64 {
	if tx == nil || tx.Meta == nil || len(tx.Meta.PreBalances) == 0 || len(tx.Meta.PostBalances) == 0 {
		return 0
	}
	pre, post := tx.Meta.PreBalances[0], tx.Meta.PostBalances[0]
	if post+tx.Meta.Fee <= pre {
		return 0
	}
	return post + tx.Meta.Fee - pre
}
