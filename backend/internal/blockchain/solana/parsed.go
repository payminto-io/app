package solana

import "encoding/json"

// ParsedTransaction is getTransaction/getBlock with encoding jsonParsed; only the fields deposit
// detection and fee accounting read are typed.
type ParsedTransaction struct {
	Slot        uint64       `json:"slot"`
	BlockTime   *int64       `json:"blockTime"`
	Transaction ParsedTxBody `json:"transaction"`
	Meta        *ParsedMeta  `json:"meta"`
}

// Signature is the transaction id.
func (t *ParsedTransaction) Signature() string {
	if t == nil || len(t.Transaction.Signatures) == 0 {
		return ""
	}
	return t.Transaction.Signatures[0]
}

// Failed reports meta.err.
func (t *ParsedTransaction) Failed() bool {
	return t != nil && t.Meta != nil && len(t.Meta.Err) > 0 && string(t.Meta.Err) != "null"
}

type ParsedTxBody struct {
	Signatures []string      `json:"signatures"`
	Message    ParsedMessage `json:"message"`
}

type ParsedMessage struct {
	AccountKeys     []ParsedAccountKey  `json:"accountKeys"`
	Instructions    []ParsedInstruction `json:"instructions"`
	RecentBlockhash string              `json:"recentBlockhash"`
}

type ParsedAccountKey struct {
	Pubkey   string `json:"pubkey"`
	Signer   bool   `json:"signer"`
	Writable bool   `json:"writable"`
	Source   string `json:"source"`
}

// ParsedInstruction is either parsed (program, parsed) or raw (programId, accounts, data).
type ParsedInstruction struct {
	Program   string          `json:"program"`
	ProgramID string          `json:"programId"`
	Parsed    json.RawMessage `json:"parsed"`
	Accounts  []string        `json:"accounts"`
	Data      string          `json:"data"`
}

type ParsedMeta struct {
	Err               json.RawMessage          `json:"err"`
	Fee               uint64                   `json:"fee"`
	PreBalances       []uint64                 `json:"preBalances"`
	PostBalances      []uint64                 `json:"postBalances"`
	InnerInstructions []ParsedInnerInstruction `json:"innerInstructions"`
	PreTokenBalances  []ParsedTokenBalance     `json:"preTokenBalances"`
	PostTokenBalances []ParsedTokenBalance     `json:"postTokenBalances"`
	LogMessages       []string                 `json:"logMessages"`
}

type ParsedInnerInstruction struct {
	Index        int                 `json:"index"`
	Instructions []ParsedInstruction `json:"instructions"`
}

type ParsedTokenBalance struct {
	AccountIndex  int         `json:"accountIndex"`
	Mint          string      `json:"mint"`
	Owner         string      `json:"owner"`
	ProgramID     string      `json:"programId"`
	UITokenAmount TokenAmount `json:"uiTokenAmount"`
}

// tokenInstruction is the parsed payload of spl-token transfer, transferChecked, mintTo, closeAccount.
type tokenInstruction struct {
	Type string `json:"type"`
	Info struct {
		Source      string       `json:"source"`
		Destination string       `json:"destination"`
		Account     string       `json:"account"`
		Mint        string       `json:"mint"`
		Authority   string       `json:"authority"`
		Amount      string       `json:"amount"`
		TokenAmount *TokenAmount `json:"tokenAmount"`
	} `json:"info"`
}

// systemInstruction is the parsed payload of system transfer.
type systemInstruction struct {
	Type string `json:"type"`
	Info struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		Lamports    uint64 `json:"lamports"`
	} `json:"info"`
}
