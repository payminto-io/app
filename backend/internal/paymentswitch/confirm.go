package paymentswitch

import (
	"context"
	"fmt"

	"github.com/payminto/payminto/backend/internal/connectors"
	"gorm.io/gorm"
)

// Confirm runs one attempt at the selected connector. The claim (intent -> processing, attempt row created)
// is one compare-and-set transaction; only one concurrent confirm gets RowsAffected == 1. An intent whose
// active attempt has an unknown outcome is not confirmable: nothing opens a second attempt until Sync resolves it.
func (s *Service) Confirm(ctx context.Context, merchantID, intentID string, cmd ConfirmCommand) (Intent, error) {
	db := s.db.WithContext(ctx)
	intent, err := loadIntent(db, merchantID, intentID)
	if err != nil {
		return Intent{}, err
	}
	pm, ok := paymentMethodFrom(intent.PaymentMethod)
	if cmd.PaymentMethod != nil {
		if cmd.PaymentMethod.Type == "" {
			return Intent{}, fmt.Errorf("%w: payment method type required", ErrInvalid)
		}
		pm, ok = *cmd.PaymentMethod, true
	}
	if !ok {
		return Intent{}, fmt.Errorf("%w: intent %s has no payment method", ErrInvalid, intentID)
	}
	if !confirmable(intent.Status) {
		return Intent{}, fmt.Errorf("%w: intent %s is %s", ErrInvalidTransition, intentID, intent.Status)
	}

	sel, err := s.selector.Select(ctx, SelectionRequest{MerchantID: merchantID, Money: Money{Amount: intent.Amount, Asset: intent.Asset}, Method: pm.Type, Metadata: stringMap(intent.Metadata)})
	if err != nil {
		return Intent{}, err
	}
	conn, err := s.connector(sel.Code)
	if err != nil {
		return Intent{}, err
	}
	caps := conn.Capabilities()
	if !caps.Supports(pm.Type) {
		return Intent{}, fmt.Errorf("%w: connector %s does not take %s", ErrInvalid, sel.Code, pm.Type)
	}
	if intent.CaptureMethod == connectors.CaptureManual && !caps.ManualCapture {
		return Intent{}, fmt.Errorf("%w: connector %s cannot authorize without capturing", ErrInvalid, sel.Code)
	}

	now := s.now()
	lease := now.Add(s.lease)
	attempt := AttemptRow{
		ID:              s.newID("pa"),
		IntentID:        intent.ID,
		MerchantID:      merchantID,
		ConnectorCode:   sel.Code,
		Status:          AttemptStarted,
		Amount:          intent.Amount,
		Asset:           intent.Asset,
		SelectionReason: sel.Reason,
		ClaimedUntil:    &lease,
		NextSyncAt:      &lease,
		StatusChangedAt: now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if intent.CaptureMethod == connectors.CaptureAutomatic {
		attempt.AmountToCapture = intent.Amount
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		from := intent.Status
		res := tx.Model(&IntentRow{}).
			Where("id = ? AND status = ? AND version = ?", intent.ID, from, intent.Version).
			Updates(map[string]any{
				"status": IntentProcessing, "payment_method": paymentMethodJSON(pm), "payment_method_type": pm.Type,
				"connector_code": sel.Code, "active_attempt_id": attempt.ID, "next_action": nil,
				"last_error_code": "", "last_error_message": "",
				"version": intent.Version + 1, "updated_at": s.now(),
			})
		if res.Error != nil {
			return fmt.Errorf("paymentswitch: claim intent: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			current, err := loadIntent(tx, merchantID, intentID)
			if err != nil {
				return err
			}
			return fmt.Errorf("%w: intent %s is %s", ErrInvalidTransition, intentID, current.Status)
		}
		if err := tx.Create(&attempt).Error; err != nil {
			return fmt.Errorf("paymentswitch: insert attempt: %w", err)
		}
		if err := s.snapshotFee(ctx, tx, intent, attempt, pm); err != nil {
			return err
		}
		if err := s.recordTransition(tx, "attempt", attempt.ID, "", string(AttemptStarted), "confirm"); err != nil {
			return err
		}
		return s.recordTransition(tx, "intent", intent.ID, string(from), string(IntentProcessing), "confirm")
	})
	if err != nil {
		return Intent{}, err
	}

	resp, callErr := conn.Authorize(ctx, connectors.AuthorizeRequest{
		AttemptID:      attempt.ID,
		IntentID:       intent.ID,
		MerchantID:     merchantID,
		PlatformID:     intent.PlatformID,
		Money:          Money{Amount: intent.Amount, Asset: intent.Asset},
		CaptureMethod:  intent.CaptureMethod,
		PaymentMethod:  pm,
		ReturnURL:      intent.ReturnURL,
		Description:    intent.Description,
		IdempotencyKey: attempt.ID,
		Metadata:       stringMap(intent.Metadata),
	})
	update, err := s.authorizeOutcome(sel.Code, resp, callErr)
	if err != nil {
		return Intent{}, err
	}
	out, _, err := s.applyToActiveAttempt(ctx, merchantID, intentID, attempt.ID, update)
	return out, err
}

// snapshotFee pins the fee rule version to the attempt in its creating transaction (fees README "Payment path").
func (s *Service) snapshotFee(ctx context.Context, tx *gorm.DB, intent IntentRow, attempt AttemptRow, pm connectors.PaymentMethod) error {
	if s.fees == nil {
		return nil
	}
	if intent.PaymentRecordID == 0 {
		return fmt.Errorf("%w: intent %s has no payment record", ErrPaymentRecord, intent.ID)
	}
	q := FeeQuery{Method: feeMethod(pm.Type), Connector: string(attempt.ConnectorCode), Currency: intent.Asset}
	if pm.Type == connectors.MethodChain {
		q.Chain, q.Currency = pm.Details["chain"], pm.Details["asset"]
	}
	return s.fees.Snapshot(ctx, tx, FeeRef{PaymentRecordID: intent.PaymentRecordID, AttemptID: attempt.ID}, q)
}

// feeMethod maps the connector method family onto the fees vocabulary.
func feeMethod(m connectors.Method) string {
	switch m {
	case connectors.MethodChain:
		return "crypto"
	default:
		return string(m)
	}
}

// confirmable lists where a confirm may start; processing is excluded even though processing -> processing is a no-op.
func confirmable(s IntentStatus) bool {
	return s == IntentRequiresPaymentMethod || s == IntentRequiresConfirmation || s == IntentFailed
}

// authorizeOutcome turns the connector's answer, or its error, into the attempt update to apply (I2): a typed
// decline is terminal; any other error leaves the attempt pending for Sync, with the raw text in the audit reason.
func (s *Service) authorizeOutcome(code connectors.Code, resp connectors.AuthorizeResponse, callErr error) (attemptUpdate, error) {
	switch {
	case callErr == nil:
		status, err := MapAttemptStatus(code, resp.RawStatus)
		if err != nil {
			return attemptUpdate{}, err
		}
		return attemptUpdate{
			status: status, raw: resp.RawStatus, connectorTransactionID: resp.ConnectorTransactionID,
			amountReceived: resp.AmountReceived, receivedAsset: resp.ReceivedAsset, nextAction: resp.NextAction,
			errorCode: resp.ErrorCode, errorMessage: resp.ErrorMessage, reason: "authorize",
		}, nil
	case connectors.Definitive(callErr):
		ec, em := redact(callErr)
		return attemptUpdate{status: AttemptAuthorizationFailed, errorCode: ec, errorMessage: em, reason: "authorize declined: " + callErr.Error()}, nil
	default:
		ec, em := redact(callErr)
		return attemptUpdate{status: AttemptPending, errorCode: ec, errorMessage: em, reason: "authorize outcome unknown: " + callErr.Error()}, nil
	}
}

// applyToActiveAttempt loads fresh rows in lock order (intent, then attempt) and applies one update in its own
// transaction; events are emitted after commit.
func (s *Service) applyToActiveAttempt(ctx context.Context, merchantID, intentID, attemptID string, u attemptUpdate) (Intent, applyResult, error) {
	var out IntentRow
	var result applyResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		intent, err := loadIntent(lockIfPostgres(tx), merchantID, intentID)
		if err != nil {
			return err
		}
		attempt, err := loadAttempt(lockIfPostgres(tx), attemptID)
		if err != nil {
			return err
		}
		result, err = s.applyAttempt(tx, &intent, &attempt, u)
		if err != nil {
			return err
		}
		if !result.changed {
			intent, err = loadIntent(tx, merchantID, intentID)
			if err != nil {
				return err
			}
		}
		out = intent
		return nil
	})
	if err != nil {
		return Intent{}, applyResult{}, err
	}
	s.emit(ctx, result.events)
	return out.toIntent(), result, nil
}

// FirstEnabledSelector is the default ConnectorSelector: the merchant's first enabled connector that supports
// the method. Ticket 06 replaces it with the routing engine.
type FirstEnabledSelector struct {
	Merchants  MerchantConnectors
	Connectors connectors.Lookup
}

func (f FirstEnabledSelector) Select(ctx context.Context, req SelectionRequest) (Selection, error) {
	codes, err := f.Merchants.EnabledConnectors(ctx, req.MerchantID)
	if err != nil {
		return Selection{}, err
	}
	for _, code := range codes {
		c, ok := f.Connectors.Get(code)
		if !ok {
			continue
		}
		if c.Capabilities().Supports(req.Method) {
			return Selection{Code: code, Reason: "first enabled connector supporting " + string(req.Method)}, nil
		}
	}
	return Selection{}, fmt.Errorf("%w: method %s for merchant %s", ErrNoConnector, req.Method, req.MerchantID)
}

// StaticMerchantConnectors enables the same connectors, in order, for every merchant (configuration default).
type StaticMerchantConnectors []connectors.Code

func (s StaticMerchantConnectors) EnabledConnectors(context.Context, string) ([]connectors.Code, error) {
	return []connectors.Code(s), nil
}
