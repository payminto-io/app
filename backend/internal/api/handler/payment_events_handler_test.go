package handler

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/payminto/payminto/backend/internal/realtime"
)

type disconnectedSSEWriter struct{}

func (disconnectedSSEWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestWriteSSEEventReturnsWriteFailure(t *testing.T) {
	err := writeSSEEvent(disconnectedSSEWriter{}, realtime.PaymentEvent{ReferenceID: "ref", State: "FILLED"})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write error = %v, want disconnected stream", err)
	}
}

func TestWriteSSEEventPreservesPaymentFrame(t *testing.T) {
	var buf bytes.Buffer
	if err := writeSSEEvent(&buf, realtime.PaymentEvent{ReferenceID: "ref", State: "FILLED"}); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "event: payment\ndata: {\"referenceID\":\"ref\",\"state\":\"FILLED\"}\n\n"; got != want {
		t.Fatalf("frame = %q, want %q", got, want)
	}
}
