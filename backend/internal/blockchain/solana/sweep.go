package solana

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DefaultSweepBatchSize is the most deposit accounts one sweep packet holds with closes enabled:
// each adds a signature, two keys and two instructions (~152 bytes) on a ~390-byte base, limit 1232.
const DefaultSweepBatchSize = 5

// SweepItem is one deposit token account to drain; Amount is the full balance in base units.
type SweepItem struct {
	Owner        PublicKey
	TokenAccount PublicKey
	Amount       uint64
	// Close reclaims the account's rent; only when Amount is the whole balance, or closeAccount fails.
	Close bool
}

// SweepParams describes a batched, fee-sponsored sweep of one mint into the hot wallet's ATA.
type SweepParams struct {
	FeePayer                 PublicKey
	HotWalletOwner           PublicKey
	Mint                     PublicKey
	TokenProgram             PublicKey
	Decimals                 uint8
	Items                    []SweepItem
	ComputeUnitLimit         uint32
	PriorityFeeMicroLamports uint64
	// CloseAccounts reclaims each emptied deposit ATA's rent to the fee payer.
	CloseAccounts bool
}

// SweepInstructions builds the instruction list: compute budget, idempotent hot ATA creation,
// then per item a transferChecked and optional closeAccount. Returns the hot wallet ATA.
func SweepInstructions(p SweepParams) ([]Instruction, PublicKey, error) {
	if len(p.Items) == 0 {
		return nil, PublicKey{}, errors.New("solana: sweep has no items")
	}
	var ixs []Instruction
	if p.ComputeUnitLimit > 0 {
		ixs = append(ixs, ComputeBudgetSetUnitLimit(p.ComputeUnitLimit))
	}
	if p.PriorityFeeMicroLamports > 0 {
		ixs = append(ixs, ComputeBudgetSetUnitPrice(p.PriorityFeeMicroLamports))
	}
	create, hotATA, err := CreateAssociatedTokenAccountIdempotent(p.FeePayer, p.HotWalletOwner, p.Mint, p.TokenProgram)
	if err != nil {
		return nil, PublicKey{}, err
	}
	ixs = append(ixs, create)
	for _, it := range p.Items {
		if it.Amount == 0 {
			return nil, PublicKey{}, fmt.Errorf("solana: sweep item %s has zero amount", it.TokenAccount)
		}
		ixs = append(ixs, TokenTransferChecked(p.TokenProgram, it.TokenAccount, p.Mint, hotATA, it.Owner, it.Amount, p.Decimals))
		if p.CloseAccounts && it.Close {
			ixs = append(ixs, TokenCloseAccount(p.TokenProgram, it.TokenAccount, p.FeePayer, it.Owner))
		}
	}
	return ixs, hotATA, nil
}

// BuildSweepMessage compiles the sweep against blockhash and reports the required signers.
func BuildSweepMessage(p SweepParams, blockhash string) (Message, []PublicKey, error) {
	ixs, _, err := SweepInstructions(p)
	if err != nil {
		return Message{}, nil, err
	}
	bh, err := ParsePublicKey(blockhash)
	if err != nil {
		return Message{}, nil, fmt.Errorf("solana: blockhash: %w", err)
	}
	msg, err := CompileMessage(p.FeePayer, bh, ixs)
	if err != nil {
		return Message{}, nil, err
	}
	return msg, msg.Signers(), nil
}

// SendOptions bounds SendAndConfirm.
type SendOptions struct {
	// Commitment to wait for; the sweeper waits for confirmed and lets the tracker see finalized.
	Commitment string
	// MaxAttempts rebuilds with a fresh blockhash after expiry; 0 means 3.
	MaxAttempts int
	// Poll interval for signature statuses; 0 means 2s.
	Poll time.Duration
	// Wait is the total time per attempt before checking blockhash expiry; 0 means 60s.
	Wait time.Duration
}

// SendResult is the outcome of SendAndConfirm.
type SendResult struct {
	Signature            string
	Slot                 uint64
	LastValidBlockHeight uint64
	Attempts             int
	// Landed is false when the final attempt was sent but not seen before Wait elapsed; the
	// caller records the signature and lets the tracker resolve it.
	Landed bool
}

// ErrBlockhashExpired is returned by WaitForSignature when the chain passed lastValidBlockHeight.
var ErrBlockhashExpired = errors.New("solana: blockhash expired before the transaction landed")

// SendAndConfirm fetches a blockhash, builds, signs, sends and waits. On blockhash expiry it rebuilds
// with a fresh blockhash (recent-blockhash retry; durable nonces are not used, see README).
func SendAndConfirm(ctx context.Context, c *Client, build func(blockhash string) (Message, error), signers []Signer, opts SendOptions) (SendResult, error) {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 3
	}
	if opts.Commitment == "" {
		opts.Commitment = CommitmentConfirmed
	}
	var res SendResult
	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		res.Attempts = attempt
		bh, err := c.GetLatestBlockhash(ctx, CommitmentConfirmed)
		if err != nil {
			return res, fmt.Errorf("solana: latest blockhash: %w", err)
		}
		msg, err := build(bh.Blockhash)
		if err != nil {
			return res, err
		}
		tx, err := Sign(msg, signers)
		if err != nil {
			return res, err
		}
		if size := len(tx.Serialize()); size > MaxTransactionSize {
			return res, fmt.Errorf("solana: transaction is %d bytes, limit %d", size, MaxTransactionSize)
		}
		sig, err := c.SendTransaction(ctx, tx, false, 0)
		if err != nil {
			return res, fmt.Errorf("solana: send: %w", err)
		}
		res.Signature = sig
		res.LastValidBlockHeight = bh.LastValidBlockHeight
		slot, err := c.WaitForSignature(ctx, sig, bh.LastValidBlockHeight, opts)
		if err == nil {
			res.Slot = slot
			res.Landed = true
			return res, nil
		}
		if errors.Is(err, ErrBlockhashExpired) {
			continue
		}
		return res, err
	}
	return res, ErrBlockhashExpired
}

// WaitForSignature polls until the signature reaches the commitment, the blockhash expires, or Wait elapses.
// A timeout without expiry returns ErrNotLanded so the caller keeps tracking the signature.
func (c *Client) WaitForSignature(ctx context.Context, sig string, lastValidBlockHeight uint64, opts SendOptions) (uint64, error) {
	poll := opts.Poll
	if poll <= 0 {
		poll = 2 * time.Second
	}
	wait := opts.Wait
	if wait <= 0 {
		wait = 60 * time.Second
	}
	deadline := time.Now().Add(wait)
	for {
		statuses, err := c.GetSignatureStatuses(ctx, []string{sig})
		if err != nil {
			return 0, err
		}
		if len(statuses) == 1 && statuses[0] != nil {
			st := statuses[0]
			if st.Failed() {
				return 0, fmt.Errorf("solana: transaction %s failed: %s", sig, string(st.Err))
			}
			if commitmentReached(st.ConfirmationStatus, opts.Commitment) {
				return st.Slot, nil
			}
		} else {
			height, err := c.GetBlockHeight(ctx, CommitmentConfirmed)
			if err != nil {
				return 0, err
			}
			if lastValidBlockHeight > 0 && height > lastValidBlockHeight {
				return 0, ErrBlockhashExpired
			}
		}
		if time.Now().After(deadline) {
			return 0, ErrNotLanded
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(poll):
		}
	}
}

// ErrNotLanded means the wait elapsed while the blockhash was still valid.
var ErrNotLanded = errors.New("solana: transaction not yet seen")

func commitmentReached(got, want string) bool {
	rank := map[string]int{CommitmentProcessed: 1, CommitmentConfirmed: 2, CommitmentFinalized: 3}
	return rank[got] >= rank[want]
}

// Sent is one broadcast attempt; the tracker owns what happens to it afterwards.
type Sent struct {
	Signature            string
	Blockhash            string
	LastValidBlockHeight uint64
}

// BlockhashValidityBlocks is how many blocks past its own a blockhash stays usable.
const BlockhashValidityBlocks = 150

// Signed is a transaction ready to send: its signature is known before any node sees it, so the
// caller persists it first and a transport failure can never lose it.
type Signed struct {
	Tx                   Transaction
	Signature            string
	Blockhash            string
	LastValidBlockHeight uint64
}

// SignForSend builds against a fresh blockhash and signs; nothing is sent.
func SignForSend(ctx context.Context, c *Client, build func(blockhash string) (Message, error), signers []Signer) (Signed, error) {
	bh, err := c.GetLatestBlockhash(ctx, CommitmentConfirmed)
	if err != nil {
		return Signed{}, fmt.Errorf("solana: latest blockhash: %w", err)
	}
	msg, err := build(bh.Blockhash)
	if err != nil {
		return Signed{}, err
	}
	tx, err := Sign(msg, signers)
	if err != nil {
		return Signed{}, err
	}
	if size := len(tx.Serialize()); size > MaxTransactionSize {
		return Signed{}, fmt.Errorf("solana: transaction is %d bytes, limit %d", size, MaxTransactionSize)
	}
	return Signed{Tx: tx, Signature: tx.Signature(), Blockhash: bh.Blockhash, LastValidBlockHeight: bh.LastValidBlockHeight}, nil
}

// SendSigned broadcasts a signed transaction. A *RPCError means the node answered and refused
// (IsRejection); any other error is transport and the transaction may still have been forwarded.
func SendSigned(ctx context.Context, c *Client, s Signed) error {
	sig, err := c.SendTransaction(ctx, s.Tx, false, 0)
	if err != nil {
		return err
	}
	if sig != "" && sig != s.Signature {
		return fmt.Errorf("solana: node returned signature %s for %s", sig, s.Signature)
	}
	return nil
}

// IsRejection reports a JSON-RPC error: the node refused the transaction, it was never forwarded.
func IsRejection(err error) bool {
	var rpcErr *RPCError
	return errors.As(err, &rpcErr)
}

// SendOnce is SignForSend then SendSigned for callers that do not persist between the two (tests, tools).
func SendOnce(ctx context.Context, c *Client, build func(blockhash string) (Message, error), signers []Signer) (Sent, error) {
	signed, err := SignForSend(ctx, c, build, signers)
	if err != nil {
		return Sent{}, err
	}
	if err := SendSigned(ctx, c, signed); err != nil {
		return Sent{}, fmt.Errorf("solana: send: %w", err)
	}
	return Sent{Signature: signed.Signature, Blockhash: signed.Blockhash, LastValidBlockHeight: signed.LastValidBlockHeight}, nil
}
