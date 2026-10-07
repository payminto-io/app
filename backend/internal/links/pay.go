package links

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

// requestHash fingerprints a pay body so an idempotency key cannot be replayed with different contents.
func requestHash(req PayRequest) string {
	raw, _ := json.Marshal(struct {
		Method   MethodSpec        `json:"m"`
		Amount   string            `json:"a"`
		Customer CustomerInput     `json:"c"`
		Billing  *Address          `json:"b"`
		Shipping *Address          `json:"s"`
		Answers  map[string]string `json:"q"`
	}{req.Method, decString(req.Amount), req.Customer, req.BillingAddress, req.ShippingAddress, req.Answers})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func decString(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}

// clientKey hashes the payer's IP; the raw address is never stored.
func clientKey(ip string) string {
	if ip == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("payminto/links/client/" + ip))
	return hex.EncodeToString(sum[:])
}

const maxIdempotencyKey = 128

// creatorBudget bounds a CreatePayment call well inside the lease, so the resolver never races a live call.
func (s *Service) creatorBudget() time.Duration { return s.lease / 2 }

func inProgress(until, now time.Time) error {
	e := newErr(CodePaymentInProgress, "Idempotency-Key", "a payment with this key is still being created; retry shortly")
	e.RetryAfter = max(until.Sub(now), time.Second)
	return e
}

// Pay validates the payer's input, reserves one use under a lock, and creates the payment through PaymentCreator.
// A use is released only when the creator promises nothing was created (README "Paying a link").
func (s *Service) Pay(ctx context.Context, code string, req PayRequest) (PayResult, error) {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" || len(key) > maxIdempotencyKey {
		return PayResult{}, newErr(CodeIdempotencyKeyRequired, "Idempotency-Key", "send a unique Idempotency-Key header of 1-%d characters", maxIdempotencyKey)
	}
	if req.Amount != nil && !sane(*req.Amount) {
		return PayResult{}, newErr(CodeAmountInvalid, "amount", "out of range")
	}
	l, err := s.publicLink(ctx, code)
	if err != nil {
		return PayResult{}, err
	}
	if err := s.guard.Require(ctx, l.Environment); err != nil {
		if errors.Is(err, environment.ErrMismatch) {
			return PayResult{}, newErr(CodeEnvironmentMismatch, "", "this link belongs to the %s environment", l.Environment)
		}
		return PayResult{}, err
	}
	req.Method.Method = fees.Method(strings.ToLower(strings.TrimSpace(string(req.Method.Method))))
	req.Method.Chain, req.Method.Asset = normCode(req.Method.Chain), normCode(req.Method.Asset)
	hash := requestHash(req)

	existing, err := s.store.FindPayment(ctx, l.ID, key)
	if err != nil {
		return PayResult{}, err
	}
	if existing != nil {
		res, done, err := s.replay(ctx, l, *existing, hash)
		if done || err != nil {
			return res, err
		}
		if l, err = s.publicLink(ctx, code); err != nil {
			return PayResult{}, err
		}
	}
	if err := availability(l, s.now()); err != nil {
		return PayResult{}, err
	}
	p, err := s.quote(ctx, l, req)
	if err != nil {
		return PayResult{}, err
	}
	now := s.now()
	p.IdempotencyKey, p.RequestHash, p.ClientKey = key, hash, clientKey(req.ClientIP)
	p.ReservedUntil = now.Add(s.lease)
	p.OpenUntil = p.ReservedUntil

	reserved, existing, err := s.store.Reserve(ctx, p, now, s.limits)
	if err != nil {
		return PayResult{}, storeErr(err)
	}
	if existing != nil {
		res, done, err := s.replay(ctx, l, *existing, hash)
		if done || err != nil {
			return res, err
		}
		return PayResult{}, inProgress(existing.ReservedUntil, s.now())
	}

	cctx, cancel := context.WithTimeout(ctx, s.creatorBudget())
	defer cancel()
	created, err := s.creator.CreatePayment(cctx, s.paymentRequest(l, reserved))
	if err != nil {
		var typed *Error
		if errors.Is(err, ErrNotCreated) || errors.As(err, &typed) {
			if rerr := s.store.Release(ctx, reserved.ID); rerr != nil {
				slog.Error("links: release a use the creator refused", "link_payment_id", reserved.ID, "error", rerr)
			}
			if typed != nil {
				return PayResult{}, typed
			}
			return PayResult{}, newErr(CodePaymentCreationFailed, "", "the payment could not be created; retry with the same key")
		}
		slog.Error("links: payment creation outcome unknown; the use stays reserved until resolved",
			"link_payment_id", reserved.ID, "error", err)
		return PayResult{}, inProgress(reserved.ReservedUntil, s.now())
	}
	if err := s.store.Complete(ctx, reserved.ID, created, s.openUntil(l, created)); err != nil {
		slog.Error("links: payment created but use not completed; the resolver will finish it",
			"link_payment_id", reserved.ID, "payment_reference", created.Reference, "error", err)
		return PayResult{}, inProgress(reserved.ReservedUntil, s.now())
	}
	reserved.Status, reserved.PaymentReference, reserved.Processor = paymentCreated, created.Reference, &created
	return s.result(l, reserved, false), nil
}

func (s *Service) paymentRequest(l Link, p LinkPayment) PaymentRequest {
	return PaymentRequest{
		LinkID: l.ID, LinkPaymentID: p.ID, MemberID: l.MemberID, PlatformID: l.ExternalPlatformID,
		Environment: l.Environment, Method: p.Method, Connector: p.Connector,
		Amount: p.Amount, Currency: p.Currency, CustomerTotal: p.CustomerTotal, FeeBearer: p.FeeBearer,
		FeeRuleID: derefUint(p.FeeRuleID), FeeRuleVersion: derefInt(p.FeeRuleVersion),
		CustomerName: p.CustomerName, CustomerEmail: p.CustomerEmail, CustomerPhone: p.CustomerPhone,
		BillingAddress: p.BillingAddress, ShippingAddress: p.ShippingAddress,
		ReferenceID: l.ReferenceID, Metadata: l.Metadata,
		CaptureMode: l.CaptureMode, ThreeDSPolicy: l.ThreeDSPolicy,
		ChainToleranceBps: l.ChainToleranceBps, QuoteExpirySeconds: l.QuoteExpirySeconds, WebhookID: l.WebhookID,
	}
}

// openUntil is when a created payment stops counting as open: its expiry, else the link's quote expiry.
func (s *Service) openUntil(l Link, c CreatedPayment) time.Time {
	if c.ExpiresAt != nil {
		return c.ExpiresAt.UTC()
	}
	return s.now().Add(time.Duration(l.QuoteExpirySeconds) * time.Second)
}

func derefUint(v *uint) uint {
	if v == nil {
		return 0
	}
	return *v
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

// replay answers a key that already has a use. done=false means the use was released and Pay should start afresh.
func (s *Service) replay(ctx context.Context, l Link, p LinkPayment, hash string) (PayResult, bool, error) {
	if p.RequestHash != hash {
		return PayResult{}, true, newErr(CodeIdempotencyKeyReused, "Idempotency-Key", "this key was used for a different payment")
	}
	if p.Status == paymentCreated {
		return s.result(l, p, true), true, nil
	}
	if now := s.now(); now.Before(p.ReservedUntil) {
		return PayResult{}, true, inProgress(p.ReservedUntil, now)
	}
	outcome, resolved, err := s.resolve(ctx, l, p)
	switch {
	case err != nil:
		slog.Error("links: resolve an expired reservation", "link_payment_id", p.ID, "error", err)
		return PayResult{}, true, inProgress(s.now().Add(time.Minute), s.now())
	case outcome == outcomeCompleted:
		return s.result(l, resolved, true), true, nil
	}
	return PayResult{}, false, nil
}

type outcome int

const (
	outcomeCompleted outcome = iota + 1
	outcomeReleased
)

// resolve settles a pending use whose lease ended: a payment found by LinkPaymentID completes it,
// a definitively absent one releases it, anything else leaves it pending.
func (s *Service) resolve(ctx context.Context, l Link, p LinkPayment) (outcome, LinkPayment, error) {
	created, found, err := s.creator.FindPayment(ctx, p.ID)
	if err != nil {
		return 0, p, fmt.Errorf("links: look up payment for %s: %w", p.ID, err)
	}
	if found {
		err = s.store.Complete(ctx, p.ID, created, s.openUntil(l, created))
	} else {
		err = s.store.Release(ctx, p.ID)
	}
	if errors.Is(err, ErrStale) {
		// Someone else settled it first; read what they decided.
		cur, ferr := s.store.FindPayment(ctx, p.LinkID, p.IdempotencyKey)
		switch {
		case ferr != nil:
			return 0, p, ferr
		case cur == nil:
			return outcomeReleased, p, nil
		case cur.Status == paymentCreated:
			return outcomeCompleted, *cur, nil
		}
		return 0, p, fmt.Errorf("links: reservation %s is still pending after a concurrent settle", p.ID)
	}
	if err != nil {
		return 0, p, err
	}
	if !found {
		return outcomeReleased, p, nil
	}
	p.Status, p.PaymentReference, p.Processor = paymentCreated, created.Reference, &created
	return outcomeCompleted, p, nil
}

// ResolveExpired settles up to limit pending uses whose lease ended; the worker calls it on a timer.
func (s *Service) ResolveExpired(ctx context.Context, limit int) (completed, released int, err error) {
	pending, err := s.store.ExpiredPending(ctx, s.now(), limit)
	if err != nil {
		return 0, 0, err
	}
	for _, p := range pending {
		out, _, rerr := s.resolve(ctx, Link{Input: Input{QuoteExpirySeconds: DefaultInput().QuoteExpirySeconds}}, p)
		switch {
		case rerr != nil:
			slog.Error("links: resolve an expired reservation", "link_payment_id", p.ID, "error", rerr)
			err = errors.Join(err, rerr)
		case out == outcomeCompleted:
			completed++
		case out == outcomeReleased:
			released++
		}
	}
	return completed, released, err
}

func (s *Service) result(l Link, p LinkPayment, replayed bool) PayResult {
	r := PayResult{
		PaymentReference: p.PaymentReference, Amount: p.Amount, Currency: p.Currency, Fee: p.Fee, Tax: p.Tax,
		CustomerTotal: p.CustomerTotal, FeeBearer: p.FeeBearer, Method: p.Method, Replayed: replayed,
	}
	if p.Processor != nil {
		r.CheckoutURL, r.DepositAddress, r.ExpiresAt = p.Processor.CheckoutURL, p.Processor.DepositAddress, p.Processor.ExpiresAt
	}
	if l.SuccessMode == SuccessRedirect && l.SuccessURL != "" {
		r.SuccessRedirectURL = withReference(l.SuccessURL, l.ReferenceID)
	}
	return r
}

// withReference appends reference_id to the merchant's success URL, keeping its own query.
func withReference(raw, ref string) string {
	u, err := url.Parse(raw)
	if err != nil || ref == "" {
		return raw
	}
	q := u.Query()
	q.Set("reference_id", ref)
	u.RawQuery = q.Encode()
	return u.String()
}

// quote validates everything the payer sent and prices it, producing the reservation to store.
func (s *Service) quote(ctx context.Context, l Link, req PayRequest) (LinkPayment, error) {
	var errs []*Error
	if !slices.Contains(l.Methods, req.Method) {
		errs = append(errs, newErr(CodeMethodNotEnabled, "method", "%s is not enabled on this link", req.Method))
	}
	places := s.places(l.Currency)
	amount, err := payAmount(l, req.Amount, places)
	if err != nil {
		errs = append(errs, err.(*Error))
	}
	name, email, phone, cerrs := resolveCustomer(l.CustomerFields, req.Customer)
	errs = append(errs, cerrs...)

	var billing, shipping *Address
	for _, a := range []struct {
		field    string
		required bool
		in       *Address
		out      **Address
	}{{"billing_address", l.BillingRequired, req.BillingAddress, &billing}, {"shipping_address", l.ShippingRequired, req.ShippingAddress, &shipping}} {
		if a.in == nil {
			if a.required {
				errs = append(errs, newErr(CodeAddressRequired, a.field, "is required"))
			}
			continue
		}
		addr := *a.in
		addr.Country = normCode(addr.Country)
		if e := validateAddress(a.field, addr); e != nil {
			errs = append(errs, e)
			continue
		}
		*a.out = &addr
	}

	answers, aerrs := checkAnswers(l, req.Answers)
	errs = append(errs, aerrs...)
	if err := joinErrs(errs); err != nil {
		return LinkPayment{}, err
	}

	p, e, err := s.price(ctx, l, req.Method, amount, true)
	if err != nil {
		return LinkPayment{}, err
	}
	if e != nil {
		if e.Code == CodeMethodNoConnector || e.Code == CodeMethodNoFeeRule || e.Code == CodeFeeRuleAmbiguous {
			e.Code = CodeMethodUnavailable
		}
		e.Field = "method"
		return LinkPayment{}, e
	}
	ruleID, version := p.rule.ID, p.rule.Version
	out := LinkPayment{
		LinkID: l.ID, Environment: l.Environment, Method: req.Method, Connector: p.connector,
		Amount: amount, Currency: l.Currency, CustomerTotal: amount, FeeBearer: l.FeeBearer,
		FeeRuleID: &ruleID, FeeRuleVersion: &version,
		CustomerName: name, CustomerEmail: email, CustomerPhone: phone,
		BillingAddress: billing, ShippingAddress: shipping, Answers: answers,
	}
	if b := p.breakdown; b != nil {
		out.Fee, out.Tax, out.CustomerTotal = &b.Fee, &b.Tax, b.CustomerTotal
		out.FeeRuleID, out.FeeRuleVersion = &b.RuleID, &b.Version
	}
	if !out.CustomerTotal.LessThan(maxMoney) {
		return LinkPayment{}, newErr(CodeAmountInvalid, "amount", "the total is out of range")
	}
	return out, nil
}

// resolveCustomer applies the field policy: hidden fields may not be sent, required ones must be present or prefilled.
func resolveCustomer(policy CustomerFieldPolicy, in CustomerInput) (name, email, phone string, errs []*Error) {
	vals := [3]string{}
	for i, f := range []struct {
		name string
		rule FieldRule
		v    *string
	}{{"name", policy.Name, in.Name}, {"email", policy.Email, in.Email}, {"phone", policy.Phone, in.Phone}} {
		field := "customer." + f.name
		if f.v != nil && f.rule.Mode == FieldHidden {
			errs = append(errs, newErr(CodeCustomerFieldHidden, field, "this link does not ask for %s", f.name))
			continue
		}
		v := ""
		if f.v != nil {
			v = strings.TrimSpace(*f.v)
		}
		if v == "" {
			v = f.rule.Prefill
		}
		if v == "" {
			if f.rule.Mode == FieldRequired {
				errs = append(errs, newErr(CodeCustomerFieldRequired, field, "is required"))
			}
			continue
		}
		if e := checkCustomerValue(f.name, v); e != nil {
			errs = append(errs, e)
			continue
		}
		vals[i] = v
	}
	return vals[0], strings.ToLower(vals[1]), vals[2], errs
}

// checkAnswers enforces every required question on every payment; per_order only says how the answer is
// attributed, never whether it is asked, so the answer set reveals nothing about earlier payers.
func checkAnswers(l Link, in map[string]string) ([]Answer, []*Error) {
	var errs []*Error
	byKey := map[string]Question{}
	for _, q := range l.Questions {
		byKey[q.Key] = q
	}
	for k := range in {
		if _, ok := byKey[k]; !ok {
			errs = append(errs, newErr(CodeAnswerUnknownQuestion, "answers."+k, "this link has no question %q", k))
		}
	}
	var out []Answer
	for _, q := range l.Questions {
		field := "answers." + q.Key
		v := strings.TrimSpace(in[q.Key])
		if v == "" {
			if q.Required {
				errs = append(errs, newErr(CodeAnswerRequired, field, "%s is required", q.Label))
			}
			continue
		}
		switch q.Type {
		case QuestionText:
			if len([]rune(v)) > maxText {
				errs = append(errs, newErr(CodeAnswerInvalid, field, "at most %d characters", maxText))
				continue
			}
		case QuestionSelect:
			if !slices.Contains(q.Options, v) {
				errs = append(errs, newErr(CodeAnswerInvalid, field, "must be one of the options"))
				continue
			}
		case QuestionCheckbox:
			if v != "true" && v != "false" {
				errs = append(errs, newErr(CodeAnswerInvalid, field, "must be true or false"))
				continue
			}
			if v == "false" && q.Required {
				errs = append(errs, newErr(CodeAnswerRequired, field, "%s must be checked", q.Label))
				continue
			}
		}
		out = append(out, Answer{QuestionKey: q.Key, QuestionLabel: q.Label, Value: v})
	}
	return out, errs
}
