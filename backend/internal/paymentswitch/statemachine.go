package paymentswitch

import (
	"fmt"
	"slices"
)

// Transition tables are explicit so every allowed and disallowed edge is tested (statemachine_test.go).

var intentTransitions = map[IntentStatus][]IntentStatus{
	IntentRequiresPaymentMethod: {IntentRequiresConfirmation, IntentProcessing, IntentCancelled},
	IntentRequiresConfirmation:  {IntentProcessing, IntentCancelled},
	IntentRequiresAction:        {IntentProcessing, IntentRequiresCapture, IntentSucceeded, IntentPartiallyCaptured, IntentFailed, IntentCancelled},
	IntentProcessing:            {IntentRequiresAction, IntentRequiresCapture, IntentSucceeded, IntentPartiallyCaptured, IntentFailed, IntentCancelled},
	IntentRequiresCapture:       {IntentProcessing, IntentSucceeded, IntentPartiallyCaptured, IntentFailed, IntentCancelled},
	IntentPartiallyCaptured:     {},
	IntentSucceeded:             {},
	// A failed intent may be retried with a new attempt (Hyperswitch: "can be retried manually").
	IntentFailed:    {IntentRequiresPaymentMethod, IntentProcessing},
	IntentCancelled: {},
}

var attemptTransitions = map[AttemptStatus][]AttemptStatus{
	AttemptStarted:               {AttemptPending, AttemptAuthenticationPending, AttemptAuthorized, AttemptCharged, AttemptPartiallyPaid, AttemptAuthorizationFailed, AttemptFailure},
	AttemptPending:               {AttemptAuthenticationPending, AttemptAuthorized, AttemptCharged, AttemptPartiallyPaid, AttemptOverpaid, AttemptAuthorizationFailed, AttemptVoided, AttemptFailure},
	AttemptAuthenticationPending: {AttemptPending, AttemptAuthorized, AttemptCharged, AttemptAuthorizationFailed, AttemptFailure},
	AttemptAuthorized:            {AttemptCaptureInitiated, AttemptCharged, AttemptPartialCharged, AttemptVoidInitiated, AttemptVoided, AttemptFailure},
	AttemptCaptureInitiated:      {AttemptCharged, AttemptPartialCharged, AttemptCaptureFailed},
	AttemptCaptureFailed:         {AttemptCaptureInitiated, AttemptVoidInitiated, AttemptVoided, AttemptFailure},
	AttemptPartiallyPaid:         {AttemptCharged, AttemptOverpaid, AttemptVoided, AttemptFailure},
	AttemptVoidInitiated:         {AttemptVoided, AttemptVoidFailed},
	AttemptVoidFailed:            {AttemptVoidInitiated, AttemptVoided, AttemptFailure},
	AttemptCharged:               {},
	AttemptPartialCharged:        {},
	AttemptOverpaid:              {},
	AttemptAuthorizationFailed:   {},
	AttemptVoided:                {},
	AttemptFailure:               {},
}

var refundTransitions = map[RefundStatus][]RefundStatus{
	RefundPending:   {RefundSucceeded, RefundFailed},
	RefundSucceeded: {},
	RefundFailed:    {},
}

// AllIntentStatuses, AllAttemptStatuses and AllRefundStatuses are the closed vocabularies, in declaration order.
var (
	AllIntentStatuses = []IntentStatus{
		IntentRequiresPaymentMethod, IntentRequiresConfirmation, IntentRequiresAction, IntentProcessing,
		IntentRequiresCapture, IntentPartiallyCaptured, IntentSucceeded, IntentFailed, IntentCancelled,
	}
	AllAttemptStatuses = []AttemptStatus{
		AttemptStarted, AttemptPending, AttemptAuthenticationPending, AttemptAuthorized, AttemptCaptureInitiated,
		AttemptCharged, AttemptPartialCharged, AttemptPartiallyPaid, AttemptOverpaid, AttemptCaptureFailed,
		AttemptAuthorizationFailed, AttemptVoidInitiated, AttemptVoided, AttemptVoidFailed, AttemptFailure,
	}
	AllRefundStatuses = []RefundStatus{RefundPending, RefundSucceeded, RefundFailed}
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
// requires_capture rather than failing the intent; partially_paid is a chain deposit still waiting for funds.
func IntentStatusFor(a AttemptStatus) IntentStatus {
	switch a {
	case AttemptStarted, AttemptPending, AttemptCaptureInitiated, AttemptVoidInitiated, AttemptPartiallyPaid:
		return IntentProcessing
	case AttemptAuthenticationPending:
		return IntentRequiresAction
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

// MoneyIn reports whether an attempt in this status has captured funds the ledger must know about.
func (s AttemptStatus) MoneyIn() bool {
	return s == AttemptCharged || s == AttemptPartialCharged || s == AttemptOverpaid
}
