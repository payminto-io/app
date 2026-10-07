package paymentswitch

import (
	"fmt"
	"slices"
)

// Transition tables are explicit so every allowed and disallowed edge is tested (statemachine_test.go).

var intentTransitions = map[IntentStatus][]IntentStatus{
	IntentRequiresPaymentMethod: {IntentRequiresConfirmation, IntentProcessing, IntentCancelled},
	IntentRequiresConfirmation:  {IntentProcessing, IntentCancelled},
	IntentRequiresAction:        {IntentProcessing, IntentRequiresCapture, IntentSucceeded, IntentPartiallyCaptured, IntentPartiallyPaid, IntentFailed, IntentCancelled},
	IntentProcessing:            {IntentRequiresAction, IntentRequiresCapture, IntentSucceeded, IntentPartiallyCaptured, IntentPartiallyPaid, IntentFailed, IntentCancelled},
	IntentRequiresCapture:       {IntentProcessing, IntentSucceeded, IntentPartiallyCaptured, IntentFailed, IntentCancelled},
	IntentPartiallyCaptured:     {},
	IntentPartiallyPaid:         {},
	IntentSucceeded:             {},
	// A failed intent may be retried with a new attempt (Hyperswitch: "can be retried manually") or closed.
	IntentFailed:    {IntentRequiresPaymentMethod, IntentProcessing, IntentCancelled},
	IntentCancelled: {},
}

// Connector evidence wins: a failed or initiated operation may still turn out to have landed (C1, I1 in the review).
var attemptTransitions = map[AttemptStatus][]AttemptStatus{
	AttemptStarted:               {AttemptPending, AttemptAuthenticationPending, AttemptAuthorized, AttemptCharged, AttemptPartiallyPaid, AttemptAuthorizationFailed, AttemptFailure},
	AttemptPending:               {AttemptAuthenticationPending, AttemptAuthorized, AttemptCharged, AttemptPartiallyPaid, AttemptOverpaid, AttemptAuthorizationFailed, AttemptVoided, AttemptFailure},
	AttemptAuthenticationPending: {AttemptPending, AttemptAuthorized, AttemptCharged, AttemptPartiallyPaid, AttemptOverpaid, AttemptAuthorizationFailed, AttemptVoidInitiated, AttemptVoided, AttemptFailure},
	AttemptAuthorized:            {AttemptCaptureInitiated, AttemptCharged, AttemptPartialCharged, AttemptVoidInitiated, AttemptVoided, AttemptFailure},
	AttemptCaptureInitiated:      {AttemptAuthorized, AttemptCharged, AttemptPartialCharged, AttemptCaptureFailed, AttemptVoided},
	AttemptCaptureFailed:         {AttemptCaptureInitiated, AttemptCharged, AttemptPartialCharged, AttemptVoidInitiated, AttemptVoided, AttemptFailure},
	AttemptPartiallyPaid:         {AttemptCharged, AttemptOverpaid, AttemptVoidInitiated, AttemptUnderpaid, AttemptFailure},
	AttemptVoidInitiated:         {AttemptAuthorized, AttemptVoided, AttemptVoidFailed, AttemptUnderpaid, AttemptCharged, AttemptPartialCharged},
	AttemptVoidFailed:            {AttemptVoidInitiated, AttemptVoided, AttemptCharged, AttemptPartialCharged, AttemptFailure},
	AttemptCharged:               {},
	AttemptPartialCharged:        {},
	AttemptUnderpaid:             {},
	AttemptOverpaid:              {},
	AttemptAuthorizationFailed:   {},
	AttemptVoided:                {},
	AttemptFailure:               {},
}

var refundTransitions = map[RefundStatus][]RefundStatus{
	RefundInitiated: {RefundPending, RefundSucceeded, RefundFailed},
	RefundPending:   {RefundSucceeded, RefundFailed},
	RefundSucceeded: {},
	RefundFailed:    {},
}

// AllIntentStatuses, AllAttemptStatuses and AllRefundStatuses are the closed vocabularies, in declaration order.
var (
	AllIntentStatuses = []IntentStatus{
		IntentRequiresPaymentMethod, IntentRequiresConfirmation, IntentRequiresAction, IntentProcessing,
		IntentRequiresCapture, IntentPartiallyCaptured, IntentPartiallyPaid, IntentSucceeded, IntentFailed, IntentCancelled,
	}
	AllAttemptStatuses = []AttemptStatus{
		AttemptStarted, AttemptPending, AttemptAuthenticationPending, AttemptAuthorized, AttemptCaptureInitiated,
		AttemptCharged, AttemptPartialCharged, AttemptPartiallyPaid, AttemptUnderpaid, AttemptOverpaid, AttemptCaptureFailed,
		AttemptAuthorizationFailed, AttemptVoidInitiated, AttemptVoided, AttemptVoidFailed, AttemptFailure,
	}
	AllRefundStatuses = []RefundStatus{RefundInitiated, RefundPending, RefundSucceeded, RefundFailed}
)

func (s IntentStatus) CanTransitionTo(to IntentStatus) bool {
	return slices.Contains(intentTransitions[s], to)
}

func (s IntentStatus) IsTerminal() bool { return len(intentTransitions[s]) == 0 }

func (s IntentStatus) Valid() bool { return slices.Contains(AllIntentStatuses, s) }

func (s AttemptStatus) CanTransitionTo(to AttemptStatus) bool {
	return slices.Contains(attemptTransitions[s], to)
}

func (s AttemptStatus) IsTerminal() bool { return len(attemptTransitions[s]) == 0 }

func (s AttemptStatus) Valid() bool { return slices.Contains(AllAttemptStatuses, s) }

func (s RefundStatus) CanTransitionTo(to RefundStatus) bool {
	return slices.Contains(refundTransitions[s], to)
}

func (s RefundStatus) Valid() bool { return slices.Contains(AllRefundStatuses, s) }

func (s RefundStatus) IsTerminal() bool { return len(refundTransitions[s]) == 0 }

// InFlight reports whether a connector operation was claimed on this attempt and its outcome is still unknown.
// started is the authorize claim before any answer, pending an authorize that got no definitive answer.
func (s AttemptStatus) InFlight() bool {
	return s == AttemptStarted || s == AttemptPending || s == AttemptCaptureInitiated || s == AttemptVoidInitiated
}

// InFlightStatuses is the reconciler's set.
var InFlightStatuses = []AttemptStatus{AttemptStarted, AttemptPending, AttemptCaptureInitiated, AttemptVoidInitiated}

// IsRollback reports an edge that only means "the claimed operation never landed"; such evidence is applied only
// by the reconciler after the claim lease expired, never while the call may still be in flight (R1).
func IsRollback(from, to AttemptStatus) bool {
	return (from == AttemptCaptureInitiated || from == AttemptVoidInitiated) && to == AttemptAuthorized
}

func transitionIntent(from, to IntentStatus) error {
	if from == to {
		return nil
	}
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("%w: intent %s -> %s", ErrInvalidTransition, from, to)
	}
	return nil
}

func transitionAttempt(from, to AttemptStatus) error {
	if from == to {
		return nil
	}
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("%w: attempt %s -> %s", ErrInvalidTransition, from, to)
	}
	return nil
}

func transitionRefund(from, to RefundStatus) error {
	if from == to {
		return nil
	}
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("%w: refund %s -> %s", ErrInvalidTransition, from, to)
	}
	return nil
}

// IntentStatusFor derives the intent status from its active attempt, as Hyperswitch derives IntentStatus
// from AttemptStatus. Deviations: capture_failed and void_failed keep the authorization, so they stay in
// requires_capture rather than failing the intent; partially_paid is a chain deposit still waiting on the customer.
func IntentStatusFor(a AttemptStatus) IntentStatus {
	switch a {
	case AttemptStarted, AttemptPending, AttemptCaptureInitiated, AttemptVoidInitiated:
		return IntentProcessing
	case AttemptAuthenticationPending, AttemptPartiallyPaid:
		return IntentRequiresAction
	case AttemptUnderpaid:
		return IntentPartiallyPaid
	case AttemptAuthorized, AttemptCaptureFailed, AttemptVoidFailed:
		return IntentRequiresCapture
	case AttemptCharged, AttemptOverpaid:
		return IntentSucceeded
	case AttemptPartialCharged:
		return IntentPartiallyCaptured
	case AttemptVoided:
		return IntentCancelled
	case AttemptAuthorizationFailed, AttemptFailure:
		return IntentFailed
	}
	return IntentFailed
}

// MoneyIn reports whether an attempt in this status holds received funds the ledger must know about.
func (s AttemptStatus) MoneyIn() bool {
	return s == AttemptCharged || s == AttemptPartialCharged || s == AttemptOverpaid || s == AttemptPartiallyPaid || s == AttemptUnderpaid
}
