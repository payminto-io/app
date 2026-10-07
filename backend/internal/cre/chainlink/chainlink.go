// Package chainlink triggers deployed CRE workflows over the authenticated HTTP trigger and reads
// attestations back from the consumer contract over the gateway's own RPC (SPEC sections 6 and 7).
// It holds no key: the trigger JWT is signed by the signer service through a key reference.
package chainlink

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/payminto/payminto/backend/internal/cre"
)

// Signer is the signer service's surface: it resolves a key reference and signs an EIP-191 message.
// The API process never holds the key (SPEC section 8).
type Signer interface {
	Address(ctx context.Context, keyRef string) (common.Address, error)
	// SignMessage signs keccak256("\x19Ethereum Signed Message:\n" + len(msg) + msg) and returns 65 bytes, v in {27, 28}.
	SignMessage(ctx context.Context, keyRef string, msg []byte) ([]byte, error)
}

// UnavailableSigner is the default until the signer service is wired; every call explains what is missing.
type UnavailableSigner struct{}

var ErrSignerUnavailable = errors.New("cre: signer service is not configured for the CRE trigger key")

func (UnavailableSigner) Address(context.Context, string) (common.Address, error) {
	return common.Address{}, ErrSignerUnavailable
}

func (UnavailableSigner) SignMessage(context.Context, string, []byte) ([]byte, error) {
	return nil, ErrSignerUnavailable
}

// Log is the subset of an EVM log the reader returns.
type Log struct {
	Address     common.Address
	TxHash      common.Hash
	BlockNumber uint64
	Index       uint
	Topics      []common.Hash
	Data        []byte
}

// LogReader is the gateway's own RPC view of the attestation chain.
type LogReader interface {
	// FinalizedHead is the newest block the reader treats as final.
	FinalizedHead(ctx context.Context) (uint64, error)
	FilterLogs(ctx context.Context, address common.Address, from, to uint64, topics [][]common.Hash) ([]Log, error)
}

// The consumer emits ReportAccepted per report and one event per item (contracts/src/cre/GatewayAttestations.sol);
// the reader rebuilds the report from those and the verifier checks it hashes to the logged reportHash.
const (
	EventReportAccepted = "ReportAccepted"
	EventSolvency       = "SolvencyAttested"
	EventDeposit        = "DepositAttested"
	EventConversion     = "ConversionReferenceAttested"
)

// Config is what the provider needs; every value is a reference or an address, never a secret.
type Config struct {
	GatewayURL  string
	WorkflowIDs map[cre.Kind][32]byte
	KeyRef      string
	Consumer    common.Address
	GatewayID   [32]byte
	// ChunkBlocks bounds one eth_getLogs range; public RPCs often cap it.
	ChunkBlocks uint64
	// TokenTTL is the JWT lifetime; the CRE gateway accepts at most five minutes.
	TokenTTL time.Duration
}

type Provider struct {
	cfg    Config
	signer Signer
	reader LogReader
	http   *http.Client
	now    func() time.Time
}

var _ cre.Attester = (*Provider)(nil)

type Option func(*Provider)

func WithHTTPClient(c *http.Client) Option  { return func(p *Provider) { p.http = c } }
func WithClock(now func() time.Time) Option { return func(p *Provider) { p.now = now } }

func New(cfg Config, signer Signer, reader LogReader, opts ...Option) *Provider {
	if cfg.ChunkBlocks == 0 {
		cfg.ChunkBlocks = 2000
	}
	if cfg.TokenTTL <= 0 || cfg.TokenTTL > 5*time.Minute {
		cfg.TokenTTL = 5 * time.Minute
	}
	if signer == nil {
		signer = UnavailableSigner{}
	}
	p := &Provider{cfg: cfg, signer: signer, reader: reader, http: &http.Client{Timeout: 30 * time.Second}, now: func() time.Time { return time.Now().UTC() }}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *Provider) Name() string { return cre.ProviderChainlink }

// --- HTTP trigger ---

// TriggerRequest is the signed request for workflows.execute; Build returns it without sending.
type TriggerRequest struct {
	Body []byte
	JWT  string
}

// BuildTriggerRequest assembles the JSON-RPC body and the alg=ETH JWT: digest of the key-sorted body, iss, iat, exp, jti.
func BuildTriggerRequest(ctx context.Context, signer Signer, keyRef string, workflowID [32]byte, input json.RawMessage, now time.Time, ttl time.Duration) (TriggerRequest, error) {
	if len(input) == 0 {
		input = json.RawMessage("{}")
	}
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      uuid.NewString(),
		"method":  "workflows.execute",
		"params": map[string]any{
			"input":    input,
			"workflow": map[string]any{"workflowID": hex.EncodeToString(workflowID[:])},
		},
	}
	canonical, err := canonicalJSON(body)
	if err != nil {
		return TriggerRequest{}, err
	}
	issuer, err := signer.Address(ctx, keyRef)
	if err != nil {
		return TriggerRequest{}, err
	}
	digest := sha256.Sum256(canonical)
	header := map[string]any{"alg": "ETH", "typ": "JWT"}
	claims := map[string]any{
		"digest": "0x" + hex.EncodeToString(digest[:]),
		"iss":    issuer.Hex(),
		"iat":    now.Unix(),
		"exp":    now.Add(ttl).Unix(),
		"jti":    uuid.NewString(),
	}
	h, _ := json.Marshal(header)
	c, _ := json.Marshal(claims)
	signingInput := base64.RawURLEncoding.EncodeToString(h) + "." + base64.RawURLEncoding.EncodeToString(c)
	sig, err := signer.SignMessage(ctx, keyRef, []byte(signingInput))
	if err != nil {
		return TriggerRequest{}, err
	}
	if len(sig) != 65 {
		return TriggerRequest{}, fmt.Errorf("cre: signer returned a %d-byte signature, want 65", len(sig))
	}
	return TriggerRequest{Body: canonical, JWT: signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)}, nil
}

// canonicalJSON re-encodes with keys sorted at every level so the digest matches what the gateway computes.
func canonicalJSON(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var generic any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&generic); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeCanonical(&buf, generic); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeCanonical(buf *bytes.Buffer, v any) error {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			if err := writeCanonical(buf, t[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return err
		}
		buf.Write(b)
	}
	return nil
}

// VerifyTriggerJWT checks a JWT the way the CRE gateway does; it exists so tests and the signer service agree.
func VerifyTriggerJWT(token string, body []byte, now time.Time) (common.Address, error) {
	parts := bytes.Split([]byte(token), []byte("."))
	if len(parts) != 3 {
		return common.Address{}, errors.New("jwt: want three parts")
	}
	var header struct{ Alg string }
	h, err := base64.RawURLEncoding.DecodeString(string(parts[0]))
	if err != nil || json.Unmarshal(h, &header) != nil || header.Alg != "ETH" {
		return common.Address{}, errors.New("jwt: header is not alg ETH")
	}
	var claims struct {
		Digest string `json:"digest"`
		Iss    string `json:"iss"`
		Iat    int64  `json:"iat"`
		Exp    int64  `json:"exp"`
		Jti    string `json:"jti"`
	}
	c, err := base64.RawURLEncoding.DecodeString(string(parts[1]))
	if err != nil || json.Unmarshal(c, &claims) != nil {
		return common.Address{}, errors.New("jwt: claims do not decode")
	}
	digest := sha256.Sum256(body)
	if claims.Digest != "0x"+hex.EncodeToString(digest[:]) {
		return common.Address{}, errors.New("jwt: digest does not match the body")
	}
	if claims.Jti == "" || claims.Exp <= claims.Iat || claims.Exp-claims.Iat > int64((5*time.Minute).Seconds()) || now.Unix() > claims.Exp || now.Unix() < claims.Iat-60 {
		return common.Address{}, errors.New("jwt: iat/exp/jti out of policy")
	}
	sig, err := base64.RawURLEncoding.DecodeString(string(parts[2]))
	if err != nil || len(sig) != 65 {
		return common.Address{}, errors.New("jwt: signature is not 65 bytes")
	}
	s := make([]byte, 65)
	copy(s, sig)
	if s[64] >= 27 {
		s[64] -= 27
	}
	msg := []byte(string(parts[0]) + "." + string(parts[1]))
	pub, err := crypto.SigToPub(EIP191Hash(msg), s)
	if err != nil {
		return common.Address{}, fmt.Errorf("jwt: recover: %w", err)
	}
	addr := crypto.PubkeyToAddress(*pub)
	if addr != common.HexToAddress(claims.Iss) {
		return common.Address{}, errors.New("jwt: signature does not recover to iss")
	}
	return addr, nil
}

// EIP191Hash is the Ethereum signed-message hash the trigger signature covers.
func EIP191Hash(msg []byte) []byte {
	return crypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(msg))), msg)
}

type triggerResponse struct {
	Status              string `json:"status"`
	WorkflowExecutionID string `json:"workflow_execution_id"`
	Error               any    `json:"error"`
	Result              *struct {
		Status              string `json:"status"`
		WorkflowExecutionID string `json:"workflow_execution_id"`
	} `json:"result"`
}

// Trigger sends workflows.execute for the kind's workflow; the body is the batch input the workflow would otherwise pull.
func (p *Provider) Trigger(ctx context.Context, kind cre.Kind, input []byte) (string, error) {
	id, ok := p.cfg.WorkflowIDs[kind]
	if !ok || id == ([32]byte{}) {
		return "", fmt.Errorf("%w: no workflow id configured for %s", cre.ErrUnsupported, kind)
	}
	req, err := BuildTriggerRequest(ctx, p.signer, p.cfg.KeyRef, id, input, p.now(), p.cfg.TokenTTL)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.GatewayURL, bytes.NewReader(req.Body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.JWT)
	resp, err := p.http.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("cre: trigger %s: %w", kind, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	var out triggerResponse
	_ = json.Unmarshal(raw, &out)
	status, execID := out.Status, out.WorkflowExecutionID
	if out.Result != nil {
		status, execID = out.Result.Status, out.Result.WorkflowExecutionID
	}
	if resp.StatusCode/100 != 2 || status != "ACCEPTED" {
		return "", fmt.Errorf("cre: trigger %s refused: http %d, status %q", kind, resp.StatusCode, status)
	}
	return execID, nil
}

// --- Consumer-contract reader ---

// Poll scans the consumer contract from the cursor to the finalized head and returns one raw attestation per
// accepted report of the kind, rebuilt from ReportAccepted and its item events, with on-chain evidence.
func (p *Provider) Poll(ctx context.Context, kind cre.Kind, cursor cre.Cursor) ([]cre.RawAttestation, cre.Cursor, error) {
	if p.reader == nil {
		return nil, cursor, fmt.Errorf("%w: no RPC reader", cre.ErrUnsupported)
	}
	contract, err := cre.ContractABI()
	if err != nil {
		return nil, cursor, err
	}
	head, err := p.reader.FinalizedHead(ctx)
	if err != nil {
		return nil, cursor, fmt.Errorf("cre: read head: %w", err)
	}
	from := cursor.Block
	if from == 0 {
		// A fresh cursor starts at the head: history before the module was turned on is not replayed.
		return nil, cre.Cursor{Block: head + 1}, nil
	}
	if from > head {
		return nil, cursor, nil
	}
	topics := [][]common.Hash{
		{contract.Events[EventReportAccepted].ID, contract.Events[EventSolvency].ID, contract.Events[EventDeposit].ID, contract.Events[EventConversion].ID},
		{common.BytesToHash(p.cfg.GatewayID[:])},
	}
	var logs []Log
	for start := from; start <= head; start += p.cfg.ChunkBlocks {
		end := start + p.cfg.ChunkBlocks - 1
		if end > head {
			end = head
		}
		chunk, err := p.reader.FilterLogs(ctx, p.cfg.Consumer, start, end, topics)
		if err != nil {
			return nil, cre.Cursor{Block: start}, fmt.Errorf("cre: get logs %d-%d: %w", start, end, err)
		}
		logs = append(logs, chunk...)
	}
	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		return logs[i].Index < logs[j].Index
	})
	out, err := RebuildReports(contract, logs, kind, head)
	if err != nil {
		return nil, cursor, err
	}
	return out, cre.Cursor{Block: head + 1}, nil
}

// acceptedEvent is the decoded ReportAccepted log.
type acceptedEvent struct {
	WorkflowOwner common.Address `abi:"workflowOwner"`
	WorkflowName  [10]byte       `abi:"workflowName"`
	ReportID      [2]byte        `abi:"reportId"`
	ObservedAt    uint64         `abi:"observedAt"`
	ItemCount     *big.Int       `abi:"itemCount"`
	ReportHash    [32]byte       `abi:"reportHash"`
}

// RebuildReports groups the consumer's logs per transaction and rebuilds each accepted report of kind.
// A report whose items cannot be rebuilt is returned with a zero ReportHash so the verifier refuses it.
func RebuildReports(contract abi.ABI, logs []Log, kind cre.Kind, head uint64) ([]cre.RawAttestation, error) {
	type txGroup struct {
		accepted *Log
		meta     acceptedEvent
		workflow [32]byte
		items    []Log
		log      Log
	}
	groups := map[common.Hash]*txGroup{}
	var order []common.Hash
	var kindTopic common.Hash
	kindTopic[31] = kind.Code()
	for i := range logs {
		l := logs[i]
		if len(l.Topics) < 2 {
			continue
		}
		g := groups[l.TxHash]
		if g == nil {
			g = &txGroup{}
			groups[l.TxHash] = g
			order = append(order, l.TxHash)
		}
		switch l.Topics[0] {
		case contract.Events[EventReportAccepted].ID:
			if len(l.Topics) != 4 || l.Topics[2] != kindTopic {
				continue
			}
			var ev acceptedEvent
			if err := contract.UnpackIntoInterface(&ev, EventReportAccepted, l.Data); err != nil {
				continue
			}
			g.accepted, g.meta, g.workflow, g.log = &logs[i], ev, l.Topics[3], l
		case itemEventID(contract, kind):
			g.items = append(g.items, l)
		}
	}
	var out []cre.RawAttestation
	for _, tx := range order {
		g := groups[tx]
		if g.accepted == nil {
			continue
		}
		meta := cre.Metadata{WorkflowID: g.workflow, WorkflowName: g.meta.WorkflowName, ReportID: g.meta.ReportID}
		copy(meta.Owner[:], g.meta.WorkflowOwner.Bytes())
		var emitter [20]byte
		copy(emitter[:], g.log.Address.Bytes())
		raw := cre.RawAttestation{
			Kind: kind, Metadata: meta.Encode(),
			Evidence: cre.Evidence{Emitter: emitter, TxHash: g.log.TxHash.Bytes(), BlockNumber: g.log.BlockNumber, LogIndex: g.log.Index, HeadBlock: head},
		}
		var gateway [32]byte
		copy(gateway[:], g.log.Topics[1].Bytes())
		report, err := rebuildReport(contract, kind, gateway, g.meta.ObservedAt, g.items)
		if err == nil && g.meta.ItemCount != nil && g.meta.ItemCount.IsInt64() && int(g.meta.ItemCount.Int64()) == len(g.items) {
			raw.Report = report
			raw.Evidence.ReportHash = g.meta.ReportHash
		} else {
			raw.Report = report
		}
		out = append(out, raw)
	}
	return out, nil
}

func itemEventID(contract abi.ABI, kind cre.Kind) common.Hash {
	switch kind {
	case cre.KindSolvency:
		return contract.Events[EventSolvency].ID
	case cre.KindDepositFinality:
		return contract.Events[EventDeposit].ID
	default:
		return contract.Events[EventConversion].ID
	}
}

func rebuildReport(contract abi.ABI, kind cre.Kind, gateway [32]byte, observedAt uint64, items []Log) ([]byte, error) {
	r := cre.Report{Kind: kind, GatewayID: gateway, ObservedAt: time.Unix(int64(observedAt), 0).UTC()}
	switch kind {
	case cre.KindSolvency:
		rows := make([]cre.SolvencyItem, 0, len(items))
		for _, l := range items {
			var ev struct {
				CheckpointHash [32]byte `abi:"checkpointHash"`
				Liabilities    *big.Int `abi:"liabilities"`
				Reserves       *big.Int `abi:"reserves"`
				Decimals       uint8    `abi:"decimals"`
				ObservedAt     uint64   `abi:"observedAt"`
			}
			if len(l.Topics) != 3 {
				return nil, fmt.Errorf("%w: solvency event topics", cre.ErrInvalidReport)
			}
			if err := contract.UnpackIntoInterface(&ev, EventSolvency, l.Data); err != nil {
				return nil, err
			}
			rows = append(rows, cre.SolvencyItem{CheckpointHash: ev.CheckpointHash, Asset: l.Topics[2], Liabilities: ev.Liabilities, Reserves: ev.Reserves, Decimals: ev.Decimals})
		}
		r.Items = rows
	case cre.KindDepositFinality:
		rows := make([]cre.DepositItem, 0, len(items))
		for _, l := range items {
			var ev struct {
				ChainId     [32]byte `abi:"chainId"`
				TxRef       [32]byte `abi:"txRef"`
				Token       [32]byte `abi:"token"`
				Amount      *big.Int `abi:"amount"`
				Destination [32]byte `abi:"destination"`
				SlotOrBlock uint64   `abi:"slotOrBlock"`
				ObservedAt  uint64   `abi:"observedAt"`
			}
			if len(l.Topics) != 4 {
				return nil, fmt.Errorf("%w: deposit event topics", cre.ErrInvalidReport)
			}
			if err := contract.UnpackIntoInterface(&ev, EventDeposit, l.Data); err != nil {
				return nil, err
			}
			rows = append(rows, cre.DepositItem{DepositID: l.Topics[2], ChainID: ev.ChainId, TxRef: ev.TxRef, Token: ev.Token, Amount: ev.Amount, Destination: ev.Destination, SlotOrBlock: ev.SlotOrBlock, Verdict: l.Topics[3][31]})
		}
		r.Items = rows
	default:
		rows := make([]cre.ConversionItem, 0, len(items))
		for _, l := range items {
			var ev struct {
				ReferenceRate     *big.Int       `abi:"referenceRate"`
				ReferenceDecimals uint8          `abi:"referenceDecimals"`
				DeviationBps      *big.Int       `abi:"deviationBps"`
				Feed              common.Address `abi:"feed"`
				RoundId           *big.Int       `abi:"roundId"`
				ObservedAt        uint64         `abi:"observedAt"`
			}
			if len(l.Topics) != 4 {
				return nil, fmt.Errorf("%w: conversion event topics", cre.ErrInvalidReport)
			}
			if err := contract.UnpackIntoInterface(&ev, EventConversion, l.Data); err != nil {
				return nil, err
			}
			rows = append(rows, cre.ConversionItem{ConversionID: l.Topics[2], Pair: l.Topics[3], ReferenceRate: ev.ReferenceRate, ReferenceDecimals: ev.ReferenceDecimals, DeviationBps: ev.DeviationBps, Feed: ev.Feed, RoundID: ev.RoundId})
		}
		r.Items = rows
	}
	return cre.EncodeReport(r)
}

func (p *Provider) Health(ctx context.Context) cre.Health {
	h := cre.Health{Status: cre.HealthOK, Message: "chainlink provider"}
	if addr, err := p.signer.Address(ctx, p.cfg.KeyRef); err != nil {
		h.Status, h.Message = cre.HealthDegraded, "trigger signer unavailable: "+err.Error()
	} else {
		h.SignerAddress = addr.Hex()
	}
	if p.reader == nil {
		h.Status, h.Message = cre.HealthDown, "no RPC reader for the attestation chain"
		return h
	}
	rctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := p.reader.FinalizedHead(rctx); err != nil {
		h.Status, h.Message = cre.HealthDown, "attestation chain RPC unreachable: "+err.Error()
	}
	return h
}
