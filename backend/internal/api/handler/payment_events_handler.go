package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/constants"
	"github.com/payminto/payminto/backend/internal/metrics"
	"github.com/payminto/payminto/backend/internal/realtime"
	"github.com/payminto/payminto/backend/internal/service"
)

// PaymentEventsHandler streams payment-status changes to a checkout client over
// Server-Sent Events (SSE). The reference_id acts as the capability token, the
// same model used by the rest of the public checkout API.
type PaymentEventsHandler struct {
	broker     realtime.Broker
	paymentSvc *service.PaymentService
}

// NewPaymentEventsHandler wires the SSE handler.
func NewPaymentEventsHandler(broker realtime.Broker, paymentSvc *service.PaymentService) *PaymentEventsHandler {
	return &PaymentEventsHandler{broker: broker, paymentSvc: paymentSvc}
}

// Stream handles GET /api/v1/public/payment/:reference_id/events.
//
// It immediately emits the payment's current state (so a client that connects
// after a change is already in sync), then forwards every subsequent event
// published on the payment's topic until the client disconnects. A periodic
// heartbeat keeps idle connections and intermediary proxies alive.
func (h *PaymentEventsHandler) Stream(c *gin.Context) {
	referenceID := c.Param("reference_id")
	if referenceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reference_id is required"})
		return
	}

	// Validate the payment exists before holding the connection open.
	pr, err := h.paymentSvc.GetByReferenceIDPublic(referenceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	// Subscribe before sending the snapshot to avoid missing an event that
	// fires in the gap between snapshot and subscription.
	topic := h.broker.PaymentTopic(referenceID)
	events, unsubscribe := h.broker.Subscribe(topic)
	defer unsubscribe()

	metrics.SSEConnectionOpened()
	defer metrics.SSEConnectionClosed()

	setSSEHeaders(c)

	snapshot := realtime.PaymentEvent{ReferenceID: pr.ReferenceID, State: pr.State}
	heartbeat := time.NewTicker(constants.SSEKeepAliveInterval)
	defer heartbeat.Stop()

	c.Stream(func(w io.Writer) bool {
		// Send the current-state snapshot first, exactly once.
		if snapshot.State != "" {
			writeSSEEvent(w, snapshot)
			snapshot = realtime.PaymentEvent{}
			return true
		}
		select {
		case <-c.Request.Context().Done():
			return false
		case ev, ok := <-events:
			if !ok {
				return false
			}
			writeSSEEvent(w, ev)
			return true
		case <-heartbeat.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			return true
		}
	})
}

func setSSEHeaders(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	// Disable proxy buffering (e.g. nginx) so events flush immediately.
	h.Set("X-Accel-Buffering", "no")
}

func writeSSEEvent(w io.Writer, ev realtime.PaymentEvent) {
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: payment\ndata: %s\n\n", payload)
}
