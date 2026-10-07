package paymentswitch

import (
	"fmt"
	"sync"

	"github.com/payminto/payminto/backend/internal/connectors"
)

// StatusMap translates one connector's raw statuses into the switch vocabulary. An unmapped raw status is an
// error, never a guess (CLAUDE.md: no inferred statuses).
type StatusMap struct {
	Payment map[connectors.RawStatus]AttemptStatus
	Refund  map[connectors.RawStatus]RefundStatus
}

var (
	statusMapsMu sync.RWMutex
	statusMaps   = map[connectors.Code]StatusMap{
		"mock": {
			Payment: map[connectors.RawStatus]AttemptStatus{
				"authorized":         AttemptAuthorized,
				"captured":           AttemptCharged,
				"partially_captured": AttemptPartialCharged,
				"declined":           AttemptAuthorizationFailed,
				"action_required":    AttemptAuthenticationPending,
				"pending":            AttemptPending,
				"voided":             AttemptVoided,
				"failed":             AttemptFailure,
			},
			Refund: map[connectors.RawStatus]RefundStatus{
				"refund_pending": RefundPending,
				"refunded":       RefundSucceeded,
				"refund_failed":  RefundFailed,
			},
		},
		// Payminto payment_requests.state, lower-cased by the chaindeposit connector; an open request waits on the customer.
		"chaindeposit": {
			Payment: map[connectors.RawStatus]AttemptStatus{
				"open":                AttemptAuthenticationPending,
				"partially_filled":    AttemptPartiallyPaid,
				"filled":              AttemptCharged,
				"over_filled":         AttemptOverpaid,
				"cancelled":           AttemptVoided,
				"cancelled_underpaid": AttemptUnderpaid,
			},
			Refund: map[connectors.RawStatus]RefundStatus{},
		},
	}
)

// RegisterStatusMap adds a map for a connector that is not built in (enterprise providers register this way).
func RegisterStatusMap(code connectors.Code, m StatusMap) error {
	statusMapsMu.Lock()
	defer statusMapsMu.Unlock()
	if _, exists := statusMaps[code]; exists {
		return fmt.Errorf("%w: status map for %q already registered", ErrInvalid, code)
	}
	statusMaps[code] = m
	return nil
}

func statusMapFor(code connectors.Code) (StatusMap, bool) {
	statusMapsMu.RLock()
	defer statusMapsMu.RUnlock()
	m, ok := statusMaps[code]
	return m, ok
}

func MapAttemptStatus(code connectors.Code, raw connectors.RawStatus) (AttemptStatus, error) {
	m, ok := statusMapFor(code)
	if !ok {
		return "", fmt.Errorf("%w: connector %q", ErrUnmappedStatus, code)
	}
	s, ok := m.Payment[raw]
	if !ok {
		return "", fmt.Errorf("%w: connector %q payment status %q", ErrUnmappedStatus, code, raw)
	}
	return s, nil
}

func MapRefundStatus(code connectors.Code, raw connectors.RawStatus) (RefundStatus, error) {
	m, ok := statusMapFor(code)
	if !ok {
		return "", fmt.Errorf("%w: connector %q", ErrUnmappedStatus, code)
	}
	s, ok := m.Refund[raw]
	if !ok {
		return "", fmt.Errorf("%w: connector %q refund status %q", ErrUnmappedStatus, code, raw)
	}
	return s, nil
}

// CheckStatusMap verifies every raw status a connector declares is mapped to a valid status; wiring calls it
// so a connector with a gap never starts.
func CheckStatusMap(c connectors.Connector) error {
	m, ok := statusMapFor(c.Code())
	if !ok {
		return fmt.Errorf("%w: connector %q", ErrUnmappedStatus, c.Code())
	}
	caps := c.Capabilities()
	for _, raw := range caps.RawStatuses {
		s, ok := m.Payment[raw]
		if !ok {
			return fmt.Errorf("%w: connector %q payment status %q", ErrUnmappedStatus, c.Code(), raw)
		}
		if !s.Valid() {
			return fmt.Errorf("%w: connector %q maps %q to unknown attempt status %q", ErrInvalid, c.Code(), raw, s)
		}
	}
	for _, raw := range caps.RawRefundStatuses {
		s, ok := m.Refund[raw]
		if !ok {
			return fmt.Errorf("%w: connector %q refund status %q", ErrUnmappedStatus, c.Code(), raw)
		}
		if !s.Valid() {
			return fmt.Errorf("%w: connector %q maps %q to unknown refund status %q", ErrInvalid, c.Code(), raw, s)
		}
	}
	if caps.Refund && len(caps.RawRefundStatuses) == 0 {
		return fmt.Errorf("%w: connector %q supports refunds but declares no refund statuses", ErrInvalid, c.Code())
	}
	return nil
}
