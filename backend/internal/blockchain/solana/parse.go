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
	// AnomalyWithheldAmount: the instruction said more than the account received (transfer fees).
	AnomalyWithheldAmount = "withheld_amount"
	// AnomalyStrandedInPDA: a wallet treated the ATA as an owner and sent to ATA(ATA, mint); unsignable.
	AnomalyStrandedInPDA = "stranded_in_pda_ata"
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
			case balances[dest].Owner == ata:
				anomalies = append(anomalies, Anomaly{Kind: AnomalyStrandedInPDA, Signature: sig, Slot: tx.Slot, From: from, To: dest, Mint: ixMint,
					Amount: decimal.NewFromBigInt(raw, -int32(decimals)), Detail: "token account owned by the deposit ATA " + ata})
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

	// The balance delta is what the account received and is what gets credited; any difference
	// from the parsed instructions is flagged, in both directions.
	if delta := tokenDelta(tx, ata, mint); delta != nil {
		parsed := new(big.Int)
		if credit != nil {
			parsed = credit.RawAmount
		}
		switch delta.Cmp(parsed) {
		case 1:
			diff := new(big.Int).Sub(delta, parsed)
			anomalies = append(anomalies, Anomaly{Kind: AnomalyUnparsedCredit, Signature: sig, Slot: tx.Slot, To: ata, Mint: mint,
				Amount: decimal.NewFromBigInt(diff, -int32(w.Decimals)), Detail: "balance rose more than parsed transfers"})
		case -1:
			diff := new(big.Int).Sub(parsed, delta)
			anomalies = append(anomalies, Anomaly{Kind: AnomalyWithheldAmount, Signature: sig, Slot: tx.Slot, To: ata, Mint: mint,
				Amount: decimal.NewFromBigInt(diff, -int32(w.Decimals)), Detail: "instructions exceed the balance delta"})
		}
		if credit != nil && delta.Sign() > 0 {
			credit.RawAmount = delta
			credit.Amount = decimal.NewFromBigInt(delta, -int32(w.Decimals))
		} else if credit != nil {
			credit = nil
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

// RentMovements reads per-account lamport changes: reclaimed is the rent of accounts emptied to zero
// (closed deposit ATAs), funded the rent of accounts created from zero (a new hot wallet ATA).
// The fee payer (index 0) is excluded; its delta is fee plus funded minus reclaimed.
func RentMovements(tx *ParsedTransaction) (reclaimed, funded uint64) {
	if tx == nil || tx.Meta == nil {
		return 0, 0
	}
	for i := 1; i < len(tx.Meta.PreBalances) && i < len(tx.Meta.PostBalances); i++ {
		pre, post := tx.Meta.PreBalances[i], tx.Meta.PostBalances[i]
		switch {
		case pre > 0 && post == 0:
			reclaimed += pre
		case pre == 0 && post > 0:
			funded += post
		}
	}
	return reclaimed, funded
}
