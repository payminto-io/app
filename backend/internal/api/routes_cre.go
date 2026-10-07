package api

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/cre"
	"github.com/payminto/payminto/backend/internal/modules"
)

// CREAuth carries the dashboard guards; workflow routes authenticate with per-workflow bearer credentials.
type CREAuth struct {
	// Session authenticates dashboard reads; dashboard sessions only.
	Session gin.HandlerFunc
	// Owner is the permission check for the force-run route.
	Owner gin.HandlerFunc
	// RateLimit throttles every cre and public route; nil applies none.
	RateLimit gin.HandlerFunc
}

// RegisterCRERoutes mounts nothing when the module is off, so a disabled gateway's route table is unchanged.
func RegisterCRERoutes(rg *gin.RouterGroup, m *modules.CREModule, auth CREAuth) {
	if !m.Enabled() {
		return
	}
	h := &creHandler{svc: m.Service}
	cfg := m.Service.Config()
	limited := rg.Group("")
	if auth.RateLimit != nil {
		limited.Use(auth.RateLimit)
	}

	// Workflow-facing pulls and the report push (SPEC section 7). The liabilities read is open when public
	// verification is on, but only the solvency credential may cause a checkpoint to be published.
	if cfg.PublicVerifyEnabled {
		limited.GET("/cre/liabilities", h.liabilities)
	} else {
		limited.GET("/cre/liabilities", h.workflowAuth(cre.KindSolvency), h.liabilities)
	}
	limited.GET("/cre/pending-deposits", h.workflowAuth(cre.KindDepositFinality), h.pendingDeposits)
	limited.GET("/cre/conversions", h.workflowAuth(cre.KindConversionReference), h.conversions)
	// The kind is in the path so the credential is checked before a byte of body is read.
	limited.POST("/cre/reports/:kind", h.kindAuth, h.submitReport)

	if cfg.PublicVerifyEnabled {
		limited.GET("/public/attestations/:id", h.publicAttestation)
	}

	if auth.Session == nil {
		return
	}
	dash := limited.Group("/cre", auth.Session)
	dash.GET("/status", h.status)
	dash.GET("/attestations", h.list)
	dash.GET("/attestations/:id", h.get)
	if auth.Owner != nil {
		dash.POST("/runs/:kind", auth.Owner, h.run)
	}
}

type creHandler struct {
	svc *cre.Service
}

func creAbort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message, "code": code})
}

// bearer reads the Authorization header; the token is a per-workflow credential, never a session.
func bearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (h *creHandler) workflowAuth(kind cre.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.svc.Authorize(kind, bearer(c)) {
			creAbort(c, http.StatusUnauthorized, "unauthorized", "a valid credential for the "+string(kind)+" workflow is required")
			return
		}
		c.Next()
	}
}

func (h *creHandler) liabilities(c *gin.Context) {
	authenticated := h.svc.Authorize(cre.KindSolvency, bearer(c))
	cp, err := h.svc.Liabilities(c.Request.Context(), authenticated)
	if err != nil {
		if errors.Is(err, cre.ErrNotFound) {
			creAbort(c, http.StatusServiceUnavailable, "no_checkpoint", "no liabilities checkpoint has been published yet")
			return
		}
		creError(c, err)
		return
	}
	c.JSON(http.StatusOK, cre.CheckpointJSON(cp, authenticated))
}

func (h *creHandler) pendingDeposits(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))
	rows, err := h.svc.PendingDeposits(c.Request.Context(), limit)
	if err != nil {
		creError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deposits": cre.DepositsJSON(rows)})
}

func (h *creHandler) conversions(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	since := h.svc.Now().Add(-15 * time.Minute)
	if raw := c.Query("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			creAbort(c, http.StatusBadRequest, "invalid_request", "since must be RFC 3339")
			return
		}
		since = parsed
	}
	rows, err := h.svc.Conversions(c.Request.Context(), since, limit)
	if err != nil {
		creError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversions": cre.ConversionsJSON(rows), "next_since": h.svc.Now().Format(time.RFC3339)})
}

// reportBody is what a workflow (or the simulator) pushes. For chainlink the bytes are a hint only: the
// gateway reads the log over its own RPC and verifies that, never the body (SPEC section 6).
type reportBody struct {
	Metadata    string `json:"metadata"`
	Report      string `json:"report"`
	Signature   string `json:"signature"`
	TxHash      string `json:"tx_hash"`
	ExecutionID string `json:"execution_id"`
	Simulated   bool   `json:"simulated"`
}

// creMaxBody bounds a pushed report: a 20-asset solvency batch is about 4 KiB hex-encoded.
const creMaxBody = 64 << 10

func hexBytes(s string) ([]byte, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "0x")
	if s == "" {
		return nil, true
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return nil, false
	}
	return b, true
}

// kindAuth resolves the path kind and checks its credential before any body is read.
func (h *creHandler) kindAuth(c *gin.Context) {
	kind, ok := cre.ParseKind(c.Param("kind"))
	if !ok {
		creAbort(c, http.StatusBadRequest, "invalid_request", "kind must be solvency, deposit_finality or conversion_reference")
		return
	}
	if !h.svc.Authorize(kind, bearer(c)) {
		creAbort(c, http.StatusUnauthorized, "unauthorized", "a valid credential for the "+string(kind)+" workflow is required")
		return
	}
	c.Set("creKind", kind)
	c.Next()
}

func (h *creHandler) submitReport(c *gin.Context) {
	kind, _ := c.MustGet("creKind").(cre.Kind)
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, creMaxBody+1))
	if err != nil || len(raw) > creMaxBody {
		creAbort(c, http.StatusBadRequest, "invalid_json", "body unreadable or larger than 64 KiB")
		return
	}
	if h.svc.Provider() == cre.ProviderChainlink {
		// Nothing in the body is trusted: the gateway reads the consumer contract itself.
		n, err := h.svc.Poll(c.Request.Context(), kind)
		if err != nil {
			creError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"recorded": n, "source": "own_rpc"})
		return
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var body reportBody
	if err := dec.Decode(&body); err != nil || dec.More() {
		creAbort(c, http.StatusBadRequest, "invalid_json", "body must be one JSON object with known fields")
		return
	}
	metadata, ok1 := hexBytes(body.Metadata)
	report, ok2 := hexBytes(body.Report)
	sig, ok3 := hexBytes(body.Signature)
	tx, ok4 := hexBytes(body.TxHash)
	if !ok1 || !ok2 || !ok3 || !ok4 {
		creAbort(c, http.StatusBadRequest, "invalid_request", "metadata, report, signature and tx_hash must be hex")
		return
	}
	rows, err := h.svc.Submit(c.Request.Context(), cre.RawAttestation{
		Kind: kind, Metadata: metadata, Report: report, Evidence: cre.Evidence{Signature: sig, TxHash: tx}, ExecutionID: body.ExecutionID, Simulated: body.Simulated,
	})
	if err != nil {
		creError(c, err)
		return
	}
	ids := make([]string, 0, len(rows))
	attested := 0
	for _, r := range rows {
		ids = append(ids, r.ID)
		if r.Status == cre.StatusAttested {
			attested++
		}
	}
	c.JSON(http.StatusAccepted, gin.H{"attestation_ids": ids, "recorded": len(rows), "attested": attested})
}

func (h *creHandler) status(c *gin.Context) {
	rep, err := h.svc.Status(c.Request.Context())
	if err != nil {
		creError(c, err)
		return
	}
	c.JSON(http.StatusOK, statusJSON(rep))
}

func (h *creHandler) list(c *gin.Context) {
	var kind cre.Kind
	if raw := c.Query("kind"); raw != "" {
		k, ok := cre.ParseKind(raw)
		if !ok {
			creAbort(c, http.StatusBadRequest, "invalid_request", "unknown kind")
			return
		}
		kind = k
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	rows, err := h.svc.Attestations(c.Request.Context(), kind, limit)
	if err != nil {
		creError(c, err)
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, attestationJSON(r))
	}
	c.JSON(http.StatusOK, gin.H{"attestations": out})
}

func (h *creHandler) get(c *gin.Context) {
	a, err := h.svc.Attestation(c.Request.Context(), c.Param("id"))
	if err != nil {
		creError(c, err)
		return
	}
	c.JSON(http.StatusOK, attestationJSON(a))
}

// publicAttestation is the verification read: the stored row plus what the record does and does not prove.
func (h *creHandler) publicAttestation(c *gin.Context) {
	a, err := h.svc.Attestation(c.Request.Context(), c.Param("id"))
	if err != nil {
		creError(c, err)
		return
	}
	cfg := h.svc.Config()
	out := attestationJSON(a)
	out["consumer_address"] = addrOrNil(cfg.ConsumerAddress)
	out["forwarder_address"] = addrOrNil(cfg.ForwarderAddress)
	out["gateway_id"] = "0x" + common.Bytes2Hex(cfg.GatewayID[:])
	// A simulated record (simulator identity or simulation forwarder) is never presented as a production attestation.
	out["independently_signed"] = a.Provider == cre.ProviderChainlink && !a.Simulated
	c.JSON(http.StatusOK, out)
}

func (h *creHandler) run(c *gin.Context) {
	kind, ok := cre.ParseKind(c.Param("kind"))
	if !ok {
		creAbort(c, http.StatusBadRequest, "invalid_request", "unknown kind")
		return
	}
	run, err := h.svc.Run(c.Request.Context(), kind)
	if err != nil && run.Status == "" {
		creError(c, err)
		return
	}
	status := http.StatusAccepted
	if run.Status == cre.RunFailed {
		status = http.StatusBadGateway
	}
	c.JSON(status, runJSON(run))
}

func creError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, cre.ErrDisabled):
		creAbort(c, http.StatusNotFound, "attestations_off", err.Error())
	case errors.Is(err, cre.ErrNotFound):
		creAbort(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, cre.ErrReplayed):
		creAbort(c, http.StatusConflict, "replayed", err.Error())
	case errors.Is(err, cre.ErrUnauthorized):
		creAbort(c, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, cre.ErrNotFinal), errors.Is(err, cre.ErrUnconfirmed):
		creAbort(c, http.StatusServiceUnavailable, "retry_later", cre.Sanitize(err.Error()))
	case cre.IsRejection(err):
		creAbort(c, http.StatusUnprocessableEntity, "report_rejected", err.Error())
	default:
		creAbort(c, http.StatusInternalServerError, "internal", "internal error")
	}
}

func addrOrNil(a common.Address) any {
	if a == (common.Address{}) {
		return nil
	}
	return a.Hex()
}

func timeOrNil(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func runJSON(r cre.Run) gin.H {
	return gin.H{"kind": string(r.Kind), "provider": r.Provider, "execution_id": r.ExecutionID, "status": r.Status, "detail": cre.Sanitize(r.Detail), "started_at": timeOrNil(r.StartedAt)}
}

func attestationJSON(a cre.Attestation) gin.H {
	var tx any
	if len(a.TxHash) > 0 {
		tx = "0x" + common.Bytes2Hex(a.TxHash)
	}
	return gin.H{
		"id": a.ID, "kind": string(a.Kind), "subject_type": a.SubjectType, "subject_id": a.SubjectID, "status": string(a.Status),
		"provider": a.Provider, "simulated": a.Simulated, "chain": a.Chain, "tx_hash": tx, "block_number": a.BlockNumber,
		"workflow_id": "0x" + common.Bytes2Hex(a.WorkflowID[:]), "workflow_owner": common.BytesToAddress(a.WorkflowOwner[:]).Hex(),
		"report_id": "0x" + common.Bytes2Hex(a.ReportID[:]), "payload_hash": "0x" + common.Bytes2Hex(a.PayloadHash),
		"observed_at": timeOrNil(a.ObservedAt), "recorded_at": timeOrNil(a.RecordedAt), "reason": a.Reason, "item": cre.ItemJSON(a.Item),
	}
}

func statusJSON(rep cre.StatusReport) gin.H {
	workflows := make([]gin.H, 0, len(rep.Workflows))
	for _, w := range rep.Workflows {
		row := gin.H{
			"kind": string(w.Kind), "workflow_id": w.WorkflowID, "workflow_name": w.WorkflowName, "interval_seconds": int64(w.Interval.Seconds()),
			"credential_configured": w.CredentialConfigured, "state": w.State, "last_run": nil, "last_attestation": nil, "last_verified": nil,
		}
		if w.LastRun != nil {
			row["last_run"] = runJSON(*w.LastRun)
		}
		if w.LastAttestation != nil {
			row["last_attestation"] = attestationJSON(*w.LastAttestation)
		}
		if w.LastVerified != nil {
			row["last_verified"] = attestationJSON(*w.LastVerified)
		}
		workflows = append(workflows, row)
	}
	missing := rep.MissingKeys
	if missing == nil {
		missing = []string{}
	}
	return gin.H{
		"enabled": rep.Enabled, "provider": rep.Provider, "degraded_from": rep.DegradedFrom, "missing_keys": missing,
		"environment": string(rep.Environment), "chain": rep.Chain, "consumer_address": rep.ConsumerAddress, "forwarder_address": rep.ForwarderAddress,
		"workflow_owner": rep.WorkflowOwner, "trigger_signer": rep.TriggerSigner, "trigger_signer_address": rep.TriggerSignerAddress,
		"gateway_id": rep.GatewayID, "public_base_url": rep.PublicBaseURL, "public_verify_enabled": rep.PublicVerifyEnabled, "forwarder_simulated": rep.ForwarderSimulated,
		"health":    gin.H{"status": string(rep.Health.Status), "message": rep.Health.Message},
		"workflows": workflows,
	}
}
