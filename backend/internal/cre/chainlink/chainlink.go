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
	// Removed marks a log the node reported as reorged out; it is never used.
	Removed bool
}

// LogReader is the gateway's own RPC view of the attestation chain.
type LogReader interface {
	// FinalizedHead is the newest block the reader treats as final: the finalized tag, or latest minus
	// confirmations when the tag is unavailable and confirmations are configured.
	FinalizedHead(ctx context.Context) (uint64, error)
	FilterLogs(ctx context.Context, address common.Address, from, to uint64, topics [][]common.Hash) ([]Log, error)
	// TransactionInput is the calldata of the transaction that emitted a log; the report bytes come from it.
	TransactionInput(ctx context.Context, txHash common.Hash) ([]byte, error)
}

// The consumer emits ReportAccepted per report and one event per item (contracts/src/cre/GatewayAttestations.sol);
// the reader rebuilds the report from those and the verifier checks it hashes to the logged reportHash.
const (
	EventReportAccepted  = "ReportAccepted"
	EventSolvency        = "SolvencyAttested"
	EventSolvencyIgnored = "SolvencyIgnored"
	EventDeposit         = "DepositAttested"
	EventConversion      = "ConversionReferenceAttested"
)

// Keystone raw report layout (KeystoneForwarder._getMetadata): 45 forwarder bytes, 64 metadata bytes, then the
// receiver's report. The forwarder slices [45:109] and [109:]; the verifier takes the same bytes from calldata.
const (
	forwarderMetadataLength = 45
	rawMetadataEnd          = 109
)

// forwarderReportABI is KeystoneForwarder.report(address receiver, bytes rawReport, bytes reportContext, bytes[] signatures).
var forwarderReportABI = func() abi.Arguments {
	addr, _ := abi.NewType("address", "", nil)
	b, _ := abi.NewType("bytes", "", nil)
	bs, _ := abi.NewType("bytes[]", "", nil)
	return abi.Arguments{{Name: "receiver", Type: addr}, {Name: "rawReport", Type: b}, {Name: "reportContext", Type: b}, {Name: "signatures", Type: bs}}
}()

// ForwarderReportSelector is the 4-byte selector of KeystoneForwarder.report.
var ForwarderReportSelector = crypto.Keccak256([]byte("report(address,bytes,bytes,bytes[])"))[:4]

// SplitRawReport returns (metadata, report) from a forwarder rawReport.
func SplitRawReport(raw []byte) (metadata, report []byte, err error) {
	if len(raw) < rawMetadataEnd {
		return nil, nil, fmt.Errorf("%w: raw report is %d bytes, want at least %d", cre.ErrInvalidReport, len(raw), rawMetadataEnd)
	}
	return raw[forwarderMetadataLength:rawMetadataEnd], raw[rawMetadataEnd:], nil
}

// DecodeForwarderCall decodes a KeystoneForwarder.report transaction input into (receiver, metadata, report).
func DecodeForwarderCall(input []byte) (common.Address, []byte, []byte, error) {
	if len(input) < 4 || string(input[:4]) != string(ForwarderReportSelector) {
		return common.Address{}, nil, nil, fmt.Errorf("%w: transaction is not KeystoneForwarder.report", cre.ErrInvalidReport)
	}
	values, err := forwarderReportABI.Unpack(input[4:])
	if err != nil || len(values) != 4 {
		return common.Address{}, nil, nil, fmt.Errorf("%w: forwarder calldata: %v", cre.ErrInvalidReport, err)
	}
	receiver, _ := values[0].(common.Address)
	raw, _ := values[1].([]byte)
	metadata, report, err := SplitRawReport(raw)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return receiver, metadata, report, nil
}

// EncodeForwarderCall builds the calldata the forwarder would receive; tests and the demo chain use it.
func EncodeForwarderCall(receiver common.Address, metadata, report []byte, reportContext []byte, signatures [][]byte) ([]byte, error) {
	if len(metadata) != cre.MetadataLength {
		return nil, fmt.Errorf("%w: metadata is %d bytes", cre.ErrInvalidReport, len(metadata))
	}
	raw := make([]byte, 0, rawMetadataEnd+len(report))
	raw = append(raw, make([]byte, forwarderMetadataLength)...)
	raw[0] = 1
	raw = append(raw, metadata...)
	raw = append(raw, report...)
	if signatures == nil {
		signatures = [][]byte{}
	}
	packed, err := forwarderReportABI.Pack(receiver, raw, reportContext, signatures)
	if err != nil {
		return nil, err
	}
	return append(append([]byte{}, ForwarderReportSelector...), packed...), nil
}

// Config is what the provider needs; every value is a reference or an address, never a secret.
type Config struct {
	GatewayURL  string
	WorkflowIDs map[cre.Kind][32]byte
	KeyRef      string
	Consumer    common.Address
	GatewayID   [32]byte
	// ChunkBlocks bounds one eth_getLogs range; public RPCs often cap it.
	ChunkBlocks uint64
	// StartBlock is where a fresh cursor begins (the consumer's deployment block), so pre-enable history is read.
	StartBlock uint64
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
		return "", fmt.Errorf("cre: trigger %s: %s", kind, cre.SanitizeError(err))
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

// Poll scans the consumer contract from the cursor to the finality bound and returns one raw attestation per
// accepted report of the kind. The report bytes come from the forwarder calldata (the receiver slice), the
// contract's events say it was accepted and which solvency items were superseded, and every log is marked final.
func (p *Provider) Poll(ctx context.Context, kind cre.Kind, cursor cre.Cursor) ([]cre.RawAttestation, cre.Cursor, error) {
	if p.reader == nil {
		return nil, cursor, fmt.Errorf("%w: no RPC reader", cre.ErrUnsupported)
	}
	contract, err := cre.ContractABI()
	if err != nil {
		return nil, cursor, err
	}
	bound, err := p.reader.FinalizedHead(ctx)
	if err != nil {
		return nil, cursor, fmt.Errorf("cre: read finality bound: %s", cre.SanitizeError(err))
	}
	from := cursor.Block
	if from == 0 {
		// A fresh cursor starts at the configured start block, never at the head: history is read.
		from = p.cfg.StartBlock
		if from == 0 {
			return nil, cursor, fmt.Errorf("%w: CRE_START_BLOCK is not configured", cre.ErrUnsupported)
		}
	}
	if from > bound {
		return nil, cursor, nil
	}
	topics := [][]common.Hash{
		{contract.Events[EventReportAccepted].ID, contract.Events[EventSolvency].ID, contract.Events[EventSolvencyIgnored].ID},
		{common.BytesToHash(p.cfg.GatewayID[:])},
	}
	var logs []Log
	for start := from; start <= bound; start += p.cfg.ChunkBlocks {
		end := start + p.cfg.ChunkBlocks - 1
		if end > bound {
			end = bound
		}
		chunk, err := p.reader.FilterLogs(ctx, p.cfg.Consumer, start, end, topics)
		if err != nil {
			return nil, cre.Cursor{Block: start}, fmt.Errorf("cre: get logs %d-%d: %s", start, end, cre.SanitizeError(err))
		}
		logs = append(logs, chunk...)
	}
	sort.SliceStable(logs, func(i, j int) bool {
		if logs[i].BlockNumber != logs[j].BlockNumber {
			return logs[i].BlockNumber < logs[j].BlockNumber
		}
		return logs[i].Index < logs[j].Index
	})
	out, err := p.assemble(ctx, contract, logs, kind, bound)
	if err != nil {
		return nil, cursor, err
	}
	return out, cre.Cursor{Block: bound + 1}, nil
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

// assemble groups the logs per transaction, reads each transaction's calldata and pairs the receiver slice
// with the ReportAccepted event of the kind. A calldata read failure is returned so the cursor does not pass it.
func (p *Provider) assemble(ctx context.Context, contract abi.ABI, logs []Log, kind cre.Kind, bound uint64) ([]cre.RawAttestation, error) {
	type txGroup struct {
		accepted *Log
		meta     acceptedEvent
		// pending collects per-item solvency events until the ReportAccepted that closes them.
		pending  []cre.ItemOutcome
		outcomes []cre.ItemOutcome
	}
	groups := map[common.Hash]*txGroup{}
	var order []common.Hash
	var kindTopic common.Hash
	kindTopic[31] = kind.Code()
	for i := range logs {
		l := logs[i]
		if len(l.Topics) < 2 || l.Removed {
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
			pending := g.pending
			g.pending = nil
			if len(l.Topics) != 4 || l.Topics[2] != kindTopic {
				continue
			}
			var ev acceptedEvent
			if err := contract.UnpackIntoInterface(&ev, EventReportAccepted, l.Data); err != nil {
				continue
			}
			g.accepted, g.meta, g.outcomes = &logs[i], ev, pending
		case contract.Events[EventSolvency].ID, contract.Events[EventSolvencyIgnored].ID:
			if len(l.Topics) == 3 {
				g.pending = append(g.pending, cre.ItemOutcome{Key: l.Topics[2], Stored: l.Topics[0] == contract.Events[EventSolvency].ID})
			}
		}
	}
	var out []cre.RawAttestation
	for _, tx := range order {
		g := groups[tx]
		if g.accepted == nil {
			continue
		}
		l := *g.accepted
		input, err := p.reader.TransactionInput(ctx, l.TxHash)
		if err != nil {
			return out, fmt.Errorf("cre: read transaction %s: %s", l.TxHash.Hex(), cre.SanitizeError(err))
		}
		var emitter [20]byte
		copy(emitter[:], l.Address.Bytes())
		raw := cre.RawAttestation{
			Kind: kind, Outcomes: g.outcomes,
			Evidence: cre.Evidence{Emitter: emitter, TxHash: l.TxHash.Bytes(), BlockNumber: l.BlockNumber, LogIndex: l.Index, HeadBlock: bound, Final: l.BlockNumber <= bound, ReportHash: g.meta.ReportHash},
		}
		receiver, metadata, report, err := DecodeForwarderCall(input)
		if err != nil || receiver != p.cfg.Consumer {
			// Not a forwarder delivery to our consumer: the verifier refuses it as forged, and the row records why.
			raw.Metadata = (&cre.Metadata{WorkflowID: l.Topics[3], Owner: [20]byte(g.meta.WorkflowOwner), WorkflowName: g.meta.WorkflowName, ReportID: g.meta.ReportID}).Encode()
			raw.Evidence.ReportHash = [32]byte{}
			out = append(out, raw)
			continue
		}
		raw.Metadata, raw.Report = metadata, report
		out = append(out, raw)
	}
	return out, nil
}

func (p *Provider) Health(ctx context.Context) cre.Health {
	h := cre.Health{Status: cre.HealthOK, Message: "chainlink provider"}
	if addr, err := p.signer.Address(ctx, p.cfg.KeyRef); err != nil {
		h.Status, h.Message = cre.HealthDegraded, "trigger signer unavailable: "+cre.SanitizeError(err)
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
		h.Status, h.Message = cre.HealthDown, "attestation chain RPC unreachable: "+cre.SanitizeError(err)
	}
	return h
}
