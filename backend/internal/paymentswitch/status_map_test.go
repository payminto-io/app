package paymentswitch

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/connectors"
)

func TestMapAttemptStatus_MockAndChainDeposit(t *testing.T) {
	cases := []struct {
		code connectors.Code
		raw  connectors.RawStatus
		want AttemptStatus
	}{
		{"mock", "authorized", AttemptAuthorized},
		{"mock", "captured", AttemptCharged},
		{"mock", "partially_captured", AttemptPartialCharged},
		{"mock", "declined", AttemptAuthorizationFailed},
		{"mock", "action_required", AttemptAuthenticationPending},
		{"mock", "pending", AttemptPending},
		{"mock", "voided", AttemptVoided},
		{"mock", "failed", AttemptFailure},
		{"chaindeposit", "open", AttemptPending},
		{"chaindeposit", "partially_filled", AttemptPartiallyPaid},
		{"chaindeposit", "filled", AttemptCharged},
		{"chaindeposit", "over_filled", AttemptOverpaid},
		{"chaindeposit", "cancelled", AttemptVoided},
	}
	for _, tc := range cases {
		got, err := MapAttemptStatus(tc.code, tc.raw)
		if err != nil || got != tc.want {
			t.Errorf("MapAttemptStatus(%s, %s) = %s, %v; want %s", tc.code, tc.raw, got, err, tc.want)
		}
	}
}

func TestMapStatus_UnknownIsAnErrorNotAGuess(t *testing.T) {
	if _, err := MapAttemptStatus("mock", "SUCCEEDED"); !errors.Is(err, ErrUnmappedStatus) {
		t.Fatalf("case-different raw status must not map, err = %v", err)
	}
	if _, err := MapAttemptStatus("stripe", "succeeded"); !errors.Is(err, ErrUnmappedStatus) {
		t.Fatalf("unknown connector err = %v", err)
	}
	if _, err := MapRefundStatus("chaindeposit", "refunded"); !errors.Is(err, ErrUnmappedStatus) {
		t.Fatalf("chaindeposit has no refunds, err = %v", err)
	}
	if got, err := MapRefundStatus("mock", "refunded"); err != nil || got != RefundSucceeded {
		t.Fatalf("MapRefundStatus(mock, refunded) = %s, %v", got, err)
	}
}

type capsOnly struct {
	connectors.Connector
	code connectors.Code
	caps connectors.Capabilities
}

func (c capsOnly) Code() connectors.Code                 { return c.code }
func (c capsOnly) Capabilities() connectors.Capabilities { return c.caps }

func TestCheckStatusMap(t *testing.T) {
	ok := capsOnly{code: "mock", caps: connectors.Capabilities{
		Refund: true, RawStatuses: []connectors.RawStatus{"authorized", "captured"}, RawRefundStatuses: []connectors.RawStatus{"refunded"},
	}}
	if err := CheckStatusMap(ok); err != nil {
		t.Fatalf("CheckStatusMap(ok) = %v", err)
	}
	gap := capsOnly{code: "mock", caps: connectors.Capabilities{RawStatuses: []connectors.RawStatus{"authorized", "requires_capture"}}}
	if err := CheckStatusMap(gap); !errors.Is(err, ErrUnmappedStatus) {
		t.Fatalf("gap err = %v", err)
	}
	refundGap := capsOnly{code: "chaindeposit", caps: connectors.Capabilities{Refund: true, RawStatuses: []connectors.RawStatus{"open"}}}
	if err := CheckStatusMap(refundGap); !errors.Is(err, ErrInvalid) {
		t.Fatalf("refund without refund statuses err = %v", err)
	}
	unknown := capsOnly{code: "stripe", caps: connectors.Capabilities{RawStatuses: []connectors.RawStatus{"succeeded"}}}
	if err := CheckStatusMap(unknown); !errors.Is(err, ErrUnmappedStatus) {
		t.Fatalf("unknown connector err = %v", err)
	}
}

func TestRegisterStatusMap(t *testing.T) {
	m := StatusMap{Payment: map[connectors.RawStatus]AttemptStatus{"ok": AttemptCharged}}
	if err := RegisterStatusMap("test_only", m); err != nil {
		t.Fatalf("register = %v", err)
	}
	if err := RegisterStatusMap("test_only", m); !errors.Is(err, ErrInvalid) {
		t.Fatalf("second register err = %v", err)
	}
	if got, err := MapAttemptStatus("test_only", "ok"); err != nil || got != AttemptCharged {
		t.Fatalf("map after register = %s, %v", got, err)
	}
}

// Every mapped value must be a member of the vocabulary; a typo in the table would otherwise ship.
func TestBuiltInStatusMapsOnlyUseVocabularyStatuses(t *testing.T) {
	statusMapsMu.RLock()
	defer statusMapsMu.RUnlock()
	for code, m := range statusMaps {
		for raw, s := range m.Payment {
			if !s.Valid() {
				t.Errorf("%s: %q -> %q is not an attempt status", code, raw, s)
			}
		}
		for raw, s := range m.Refund {
			if !s.Valid() {
				t.Errorf("%s: %q -> %q is not a refund status", code, raw, s)
			}
		}
	}
}
