package paymentswitch

import (
	"errors"
	"fmt"
	"testing"
)

// The expected edges are written out independently of the production table so a change to either is caught.
type intentEdge struct{ from, to IntentStatus }
type attemptEdge struct{ from, to AttemptStatus }

var expectedIntentEdges = map[intentEdge]bool{
	{IntentRequiresPaymentMethod, IntentRequiresConfirmation}: true,
	{IntentRequiresPaymentMethod, IntentProcessing}:           true,
	{IntentRequiresPaymentMethod, IntentCancelled}:            true,
	{IntentRequiresConfirmation, IntentProcessing}:            true,
	{IntentRequiresConfirmation, IntentCancelled}:             true,
	{IntentRequiresAction, IntentProcessing}:                  true,
	{IntentRequiresAction, IntentRequiresCapture}:             true,
	{IntentRequiresAction, IntentSucceeded}:                   true,
	{IntentRequiresAction, IntentPartiallyCaptured}:           true,
	{IntentRequiresAction, IntentPartiallyPaid}:               true,
	{IntentRequiresAction, IntentFailed}:                      true,
	{IntentRequiresAction, IntentCancelled}:                   true,
	{IntentProcessing, IntentRequiresAction}:                  true,
	{IntentProcessing, IntentRequiresCapture}:                 true,
	{IntentProcessing, IntentSucceeded}:                       true,
	{IntentProcessing, IntentPartiallyCaptured}:               true,
	{IntentProcessing, IntentPartiallyPaid}:                   true,
	{IntentProcessing, IntentFailed}:                          true,
	{IntentProcessing, IntentCancelled}:                       true,
	{IntentRequiresCapture, IntentProcessing}:                 true,
	{IntentRequiresCapture, IntentSucceeded}:                  true,
	{IntentRequiresCapture, IntentPartiallyCaptured}:          true,
	{IntentRequiresCapture, IntentFailed}:                     true,
	{IntentRequiresCapture, IntentCancelled}:                  true,
	{IntentFailed, IntentRequiresPaymentMethod}:               true,
	{IntentFailed, IntentProcessing}:                          true,
	{IntentFailed, IntentCancelled}:                           true,
}

var expectedAttemptEdges = map[attemptEdge]bool{
	{AttemptStarted, AttemptPending}:                           true,
	{AttemptStarted, AttemptAuthenticationPending}:             true,
	{AttemptStarted, AttemptAuthorized}:                        true,
	{AttemptStarted, AttemptCharged}:                           true,
	{AttemptStarted, AttemptPartiallyPaid}:                     true,
	{AttemptStarted, AttemptAuthorizationFailed}:               true,
	{AttemptStarted, AttemptFailure}:                           true,
	{AttemptPending, AttemptAuthenticationPending}:             true,
	{AttemptPending, AttemptAuthorized}:                        true,
	{AttemptPending, AttemptCharged}:                           true,
	{AttemptPending, AttemptPartiallyPaid}:                     true,
	{AttemptPending, AttemptOverpaid}:                          true,
	{AttemptPending, AttemptAuthorizationFailed}:               true,
	{AttemptPending, AttemptVoided}:                            true,
	{AttemptPending, AttemptFailure}:                           true,
	{AttemptAuthenticationPending, AttemptPending}:             true,
	{AttemptAuthenticationPending, AttemptAuthorized}:          true,
	{AttemptAuthenticationPending, AttemptCharged}:             true,
	{AttemptAuthenticationPending, AttemptAuthorizationFailed}: true,
	{AttemptAuthenticationPending, AttemptVoided}:              true,
	{AttemptAuthenticationPending, AttemptPartiallyPaid}:       true,
	{AttemptAuthenticationPending, AttemptOverpaid}:            true,
	{AttemptAuthenticationPending, AttemptVoidInitiated}:       true,
	{AttemptAuthenticationPending, AttemptFailure}:             true,
	{AttemptAuthorized, AttemptCaptureInitiated}:               true,
	{AttemptAuthorized, AttemptCharged}:                        true,
	{AttemptAuthorized, AttemptPartialCharged}:                 true,
	{AttemptAuthorized, AttemptVoidInitiated}:                  true,
	{AttemptAuthorized, AttemptVoided}:                         true,
	{AttemptAuthorized, AttemptFailure}:                        true,
	{AttemptCaptureInitiated, AttemptCharged}:                  true,
	{AttemptCaptureInitiated, AttemptPartialCharged}:           true,
	{AttemptCaptureInitiated, AttemptCaptureFailed}:            true,
	{AttemptCaptureInitiated, AttemptAuthorized}:               true,
	{AttemptCaptureInitiated, AttemptVoided}:                   true,
	{AttemptCaptureFailed, AttemptCharged}:                     true,
	{AttemptCaptureFailed, AttemptPartialCharged}:              true,
	{AttemptCaptureFailed, AttemptCaptureInitiated}:            true,
	{AttemptCaptureFailed, AttemptVoidInitiated}:               true,
	{AttemptCaptureFailed, AttemptVoided}:                      true,
	{AttemptCaptureFailed, AttemptFailure}:                     true,
	{AttemptPartiallyPaid, AttemptCharged}:                     true,
	{AttemptPartiallyPaid, AttemptOverpaid}:                    true,
	{AttemptPartiallyPaid, AttemptVoidInitiated}:               true,
	{AttemptPartiallyPaid, AttemptUnderpaid}:                   true,
	{AttemptPartiallyPaid, AttemptFailure}:                     true,
	{AttemptVoidInitiated, AttemptVoided}:                      true,
	{AttemptVoidInitiated, AttemptVoidFailed}:                  true,
	{AttemptVoidInitiated, AttemptAuthorized}:                  true,
	{AttemptVoidInitiated, AttemptUnderpaid}:                   true,
	{AttemptVoidInitiated, AttemptCharged}:                     true,
	{AttemptVoidInitiated, AttemptPartialCharged}:              true,
	{AttemptVoidFailed, AttemptCharged}:                        true,
	{AttemptVoidFailed, AttemptPartialCharged}:                 true,
	{AttemptVoidFailed, AttemptVoidInitiated}:                  true,
	{AttemptVoidFailed, AttemptVoided}:                         true,
	{AttemptVoidFailed, AttemptFailure}:                        true,
}

func TestIntentTransitions_EveryPairMatchesTheExpectedTable(t *testing.T) {
	if len(AllIntentStatuses) != 10 {
		t.Fatalf("vocabulary changed: %d intent statuses; update the expected edges", len(AllIntentStatuses))
	}
	seen := 0
	for _, from := range AllIntentStatuses {
		for _, to := range AllIntentStatuses {
			if from == to {
				continue
			}
			want := expectedIntentEdges[intentEdge{from, to}]
			if got := from.CanTransitionTo(to); got != want {
				t.Errorf("intent %s -> %s: allowed=%v want %v", from, to, got, want)
			}
			err := transitionIntent(from, to)
			if want && err != nil {
				t.Errorf("transitionIntent(%s, %s) = %v, want nil", from, to, err)
			}
			if !want && !errors.Is(err, ErrInvalidTransition) {
				t.Errorf("transitionIntent(%s, %s) = %v, want ErrInvalidTransition", from, to, err)
			}
			if want {
				seen++
			}
		}
	}
	if seen != len(expectedIntentEdges) {
		t.Fatalf("expected table lists %d edges but %d were exercised", len(expectedIntentEdges), seen)
	}
	for _, s := range AllIntentStatuses {
		if err := transitionIntent(s, s); err != nil {
			t.Errorf("self transition %s must be a no-op, got %v", s, err)
		}
	}
}

func TestAttemptTransitions_EveryPairMatchesTheExpectedTable(t *testing.T) {
	if len(AllAttemptStatuses) != 16 {
		t.Fatalf("vocabulary changed: %d attempt statuses; update the expected edges", len(AllAttemptStatuses))
	}
	seen := 0
	for _, from := range AllAttemptStatuses {
		for _, to := range AllAttemptStatuses {
			if from == to {
				continue
			}
			want := expectedAttemptEdges[attemptEdge{from, to}]
			if got := from.CanTransitionTo(to); got != want {
				t.Errorf("attempt %s -> %s: allowed=%v want %v", from, to, got, want)
			}
			err := transitionAttempt(from, to)
			if want && err != nil {
				t.Errorf("transitionAttempt(%s, %s) = %v, want nil", from, to, err)
			}
			if !want && !errors.Is(err, ErrInvalidTransition) {
				t.Errorf("transitionAttempt(%s, %s) = %v, want ErrInvalidTransition", from, to, err)
			}
			if want {
				seen++
			}
		}
	}
	if seen != len(expectedAttemptEdges) {
		t.Fatalf("expected table lists %d edges but %d were exercised", len(expectedAttemptEdges), seen)
	}
}

func TestRefundTransitions(t *testing.T) {
	cases := map[[2]RefundStatus]bool{
		{RefundPending, RefundSucceeded}: true,
		{RefundPending, RefundFailed}:    true,
		{RefundSucceeded, RefundPending}: false,
		{RefundSucceeded, RefundFailed}:  false,
		{RefundFailed, RefundPending}:    false,
		{RefundFailed, RefundSucceeded}:  false,
	}
	for pair, want := range cases {
		if got := pair[0].CanTransitionTo(pair[1]); got != want {
			t.Errorf("refund %s -> %s: allowed=%v want %v", pair[0], pair[1], got, want)
		}
	}
}

func TestTerminalStatuses(t *testing.T) {
	for _, s := range AllIntentStatuses {
		want := s == IntentSucceeded || s == IntentPartiallyCaptured || s == IntentPartiallyPaid || s == IntentCancelled
		if s.IsTerminal() != want {
			t.Errorf("intent %s terminal=%v want %v", s, s.IsTerminal(), want)
		}
	}
	terminalAttempts := map[AttemptStatus]bool{
		AttemptCharged: true, AttemptPartialCharged: true, AttemptOverpaid: true, AttemptUnderpaid: true,
		AttemptAuthorizationFailed: true, AttemptVoided: true, AttemptFailure: true,
	}
	for _, s := range AllAttemptStatuses {
		if s.IsTerminal() != terminalAttempts[s] {
			t.Errorf("attempt %s terminal=%v want %v", s, s.IsTerminal(), terminalAttempts[s])
		}
	}
}

func TestIntentStatusFor_CoversEveryAttemptStatusWithAValidIntentStatus(t *testing.T) {
	want := map[AttemptStatus]IntentStatus{
		AttemptStarted:               IntentProcessing,
		AttemptPending:               IntentProcessing,
		AttemptAuthenticationPending: IntentRequiresAction,
		AttemptAuthorized:            IntentRequiresCapture,
		AttemptCaptureInitiated:      IntentProcessing,
		AttemptCharged:               IntentSucceeded,
		AttemptPartialCharged:        IntentPartiallyCaptured,
		AttemptPartiallyPaid:         IntentRequiresAction,
		AttemptUnderpaid:             IntentPartiallyPaid,
		AttemptOverpaid:              IntentSucceeded,
		AttemptCaptureFailed:         IntentRequiresCapture,
		AttemptAuthorizationFailed:   IntentFailed,
		AttemptVoidInitiated:         IntentProcessing,
		AttemptVoided:                IntentCancelled,
		AttemptVoidFailed:            IntentRequiresCapture,
		AttemptFailure:               IntentFailed,
	}
	for _, a := range AllAttemptStatuses {
		got := IntentStatusFor(a)
		if got != want[a] {
			t.Errorf("IntentStatusFor(%s) = %s, want %s", a, got, want[a])
		}
		if !got.Valid() {
			t.Errorf("IntentStatusFor(%s) = %q is not in the vocabulary", a, got)
		}
	}
}

// Every attempt edge must derive an intent edge the intent table allows, otherwise applying a
// connector result would fail mid-flight.
func TestAttemptEdgesDeriveAllowedIntentEdges(t *testing.T) {
	for edge := range expectedAttemptEdges {
		from, to := IntentStatusFor(edge.from), IntentStatusFor(edge.to)
		if err := transitionIntent(from, to); err != nil {
			t.Errorf("attempt %s -> %s derives intent %s -> %s: %v", edge.from, edge.to, from, to, err)
		}
	}
}

func TestMoneyInAndInFlight(t *testing.T) {
	moneyIn := map[AttemptStatus]bool{AttemptCharged: true, AttemptPartialCharged: true, AttemptOverpaid: true, AttemptPartiallyPaid: true, AttemptUnderpaid: true}
	inFlight := map[AttemptStatus]bool{AttemptStarted: true, AttemptPending: true, AttemptCaptureInitiated: true, AttemptVoidInitiated: true}
	for _, s := range AllAttemptStatuses {
		if s.MoneyIn() != moneyIn[s] {
			t.Errorf("%s MoneyIn = %v", s, s.MoneyIn())
		}
		if s.InFlight() != inFlight[s] {
			t.Errorf("%s InFlight = %v", s, s.InFlight())
		}
	}
}

func TestIsRollback(t *testing.T) {
	for _, from := range AllAttemptStatuses {
		for _, to := range AllAttemptStatuses {
			want := (from == AttemptCaptureInitiated || from == AttemptVoidInitiated) && to == AttemptAuthorized
			if IsRollback(from, to) != want {
				t.Errorf("IsRollback(%s, %s) = %v", from, to, !want)
			}
		}
	}
}

func ExampleIntentStatusFor() {
	fmt.Println(IntentStatusFor(AttemptAuthorized), IntentStatusFor(AttemptCharged))
	// Output: requires_capture succeeded
}
