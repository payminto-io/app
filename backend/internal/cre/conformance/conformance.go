// Package conformance is the suite every Attester must pass (MODULES.md rule 10).
package conformance

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/cre"
)

// Harness is how the suite drives one provider.
type Harness struct {
	Attester cre.Attester
	// Input returns the JSON a run of kind would be given; the gateway serves the same over HTTP.
	Input func(kind cre.Kind) []byte
	// Settle, when set, makes a triggered run observable to Poll (a fake chain writing the log).
	Settle func(kind cre.Kind, executionID string)
	// Expect says what Trigger must do for this provider.
	ExpectTriggerErr error
}

// Run executes the suite against one provider.
func Run(t *testing.T, h Harness) {
	t.Helper()
	ctx := context.Background()
	a := h.Attester

	t.Run("name", func(t *testing.T) {
		if a.Name() == "" {
			t.Fatal("Name() is empty")
		}
	})

	t.Run("health", func(t *testing.T) {
		got := a.Health(ctx)
		switch got.Status {
		case cre.HealthOK, cre.HealthDegraded, cre.HealthDown, cre.HealthOff:
		default:
			t.Fatalf("Health().Status = %q", got.Status)
		}
	})

	t.Run("poll_from_zero", func(t *testing.T) {
		for _, kind := range cre.Kinds {
			raws, next, err := a.Poll(ctx, kind, cre.Cursor{})
			if err != nil {
				t.Fatalf("%s: Poll error %v", kind, err)
			}
			if len(raws) != 0 {
				t.Fatalf("%s: a fresh cursor replayed %d records", kind, len(raws))
			}
			again, _, err := a.Poll(ctx, kind, next)
			if err != nil || len(again) != 0 {
				t.Fatalf("%s: second idle poll = %d records, err %v", kind, len(again), err)
			}
		}
	})

	for _, kind := range cre.Kinds {
		kind := kind
		t.Run("trigger_"+string(kind), func(t *testing.T) {
			_, cursor, _ := a.Poll(ctx, kind, cre.Cursor{})
			input := []byte("{}")
			if h.Input != nil {
				input = h.Input(kind)
			}
			if !json.Valid(input) {
				t.Fatalf("harness input for %s is not JSON", kind)
			}
			execID, err := a.Trigger(ctx, kind, input)
			if h.ExpectTriggerErr != nil {
				if !errors.Is(err, h.ExpectTriggerErr) {
					t.Fatalf("Trigger err = %v, want %v", err, h.ExpectTriggerErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Trigger: %v", err)
			}
			if execID == "" {
				t.Fatal("Trigger returned an empty execution id")
			}
			if h.Settle != nil {
				h.Settle(kind, execID)
			}
			raws, next, err := a.Poll(ctx, kind, cursor)
			if err != nil {
				t.Fatalf("Poll after trigger: %v", err)
			}
			if len(raws) == 0 {
				t.Fatal("Poll after trigger returned nothing")
			}
			if next == cursor {
				t.Fatal("cursor did not advance")
			}
			for _, raw := range raws {
				if raw.Kind != kind {
					t.Errorf("raw.Kind = %s, want %s", raw.Kind, kind)
				}
				meta, err := cre.DecodeMetadata(raw.Metadata)
				if err != nil {
					t.Fatalf("metadata: %v", err)
				}
				if meta.WorkflowID == ([32]byte{}) || meta.Owner == ([20]byte{}) {
					t.Error("metadata has an empty workflow id or owner")
				}
				report, err := cre.DecodeReport(raw.Report)
				if err != nil {
					t.Fatalf("report: %v", err)
				}
				if report.Kind != kind {
					t.Errorf("report kind %s under %s", report.Kind, kind)
				}
				if time.Since(report.ObservedAt) > 24*time.Hour || report.ObservedAt.After(time.Now().Add(time.Hour)) {
					t.Errorf("observed_at %s is not recent", report.ObservedAt)
				}
				ev := raw.Evidence
				if len(ev.Signature) != 65 && len(ev.TxHash) == 0 {
					t.Error("raw attestation carries neither a signature nor a transaction")
				}
				if len(ev.TxHash) != 0 && ev.Emitter == ([20]byte{}) {
					t.Error("on-chain evidence has no emitter")
				}
				if len(ev.TxHash) != 0 && ev.ReportHash != [32]byte(cre.PayloadHash(raw.Report)) {
					t.Error("on-chain evidence: rebuilt report does not hash to the contract's reportHash")
				}
			}
			replay, _, err := a.Poll(ctx, kind, next)
			if err != nil || len(replay) != 0 {
				t.Fatalf("poll past the new cursor replayed %d records, err %v", len(replay), err)
			}
		})
	}
}

// Inputs is a ready-made Input function with one subject per kind.
func Inputs(gatewayID [32]byte) func(kind cre.Kind) []byte {
	return func(kind cre.Kind) []byte {
		var v any
		switch kind {
		case cre.KindSolvency:
			v = map[string]any{"checkpoint_id": "cp-conformance", "checkpoint_hash": "0x" + hexOf(cre.SubjectKey("cp-conformance")), "taken_at": time.Now().UTC().Format(time.RFC3339), "assets": []map[string]any{{"asset": "USDC.SOLANA", "liabilities_minor": "1000000", "decimals": 6}}}
		case cre.KindDepositFinality:
			v = map[string]any{"deposits": []map[string]any{{"deposit_id": "dep-conformance", "chain": "solana", "tx": "sig1", "log_index_or_signature": "sig1", "token": "USDC", "expected_amount_minor": "5000000", "destination": "Dest111"}}}
		case cre.KindConversionReference:
			v = map[string]any{"conversions": []map[string]any{{"conversion_id": "cv-conformance", "executed_at": time.Now().UTC().Format(time.RFC3339), "base": "USDC", "quote": "USD", "executed_rate_decimal": "0.9998", "amount_minor": "100"}}}
		}
		b, _ := json.Marshal(v)
		return b
	}
}

func hexOf(b [32]byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 64)
	for i, c := range b {
		out[i*2], out[i*2+1] = digits[c>>4], digits[c&0xf]
	}
	return string(out)
}
