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

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

// Service is the links core: validation, lifecycle and paying a link. Storage is behind Store.
type Service struct {
	store        Store
	fees         FeeQuoter
	creator      PaymentCreator
	destinations DestinationVerifier
	precision    fees.Precision
	env          Environment
	checkoutBase string
	newCode      func() (string, error)
	now          func() time.Time
}

var _ Port = (*Service)(nil)

type Option func(*Service)

func WithDestinations(d DestinationVerifier) Option { return func(s *Service) { s.destinations = d } }
func WithPrecision(p fees.Precision) Option         { return func(s *Service) { s.precision = p } }
func WithEnvironment(e Environment) Option          { return func(s *Service) { s.env = e } }
func WithClock(now func() time.Time) Option         { return func(s *Service) { s.now = now } }

// WithCheckoutBaseURL is the hosted checkout origin; a link's URL is <base>/l/<short code>.
func WithCheckoutBaseURL(base string) Option {
	return func(s *Service) { s.checkoutBase = strings.TrimRight(base, "/") }
}

// WithShortCodes replaces the generator, for collision tests.
func WithShortCodes(gen func() (string, error)) Option { return func(s *Service) { s.newCode = gen } }

func NewService(store Store, quoter FeeQuoter, creator PaymentCreator, opts ...Option) *Service {
	s := &Service{
		store: store, fees: quoter, creator: creator, destinations: NoDestinations{},
		precision: fees.DefaultPrecision(), env: EnvTest, newCode: NewShortCode,
		now: func() time.Time { return time.Now().UTC() },
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Service) URL(code string) string {
	if code == "" {
		return ""
	}
	return s.checkoutBase + "/l/" + code
}

func notFound() error { return newErr(CodeNotFound, "", "payment link not found") }

func storeErr(err error) error {
	switch {
	case errors.Is(err, errStoreNotFound):
		return notFound()
	case errors.Is(err, errStale):
		return newErr(CodeConflict, "", "the link changed while this request ran; reload and retry")
	}
	return err
}

func (s *Service) places(currency string) int32 {
	p, _ := s.precision.MinorUnits(currency)
	return p
}

// prepare normalises, validates shape and computes the stored total.
func (s *Service) prepare(ctx context.Context, platformID uint, in Input) (Input, *decimal.Decimal, error) {
	in = normalizeInput(in)
	errs := validateShape(in, s.precision)
	if e := s.checkWebhook(ctx, platformID, in); e != nil {
		errs = append(errs, e)
	} else if ctx.Err() != nil {
		return in, nil, ctx.Err()
	}
	if err := joinErrs(errs); err != nil {
		return in, nil, err
	}
	return in, storedAmount(in, s.places(in.Currency)), nil
}

func (s *Service) checkWebhook(ctx context.Context, platformID uint, in Input) *Error {
	if in.WebhookID == nil {
		return nil
	}
	ok, err := s.store.WebhookExists(ctx, platformID, *in.WebhookID)
	if err != nil || !ok {
		return newErr(CodeWebhookNotFound, "webhook_id", "no webhook %d on this platform", *in.WebhookID)
	}
	return nil
}

func (s *Service) Create(ctx context.Context, actor Actor, in Input) (Link, error) {
	in, total, err := s.prepare(ctx, actor.PlatformID, in)
	if err != nil {
		return Link{}, err
	}
	l, err := s.store.Insert(ctx, Link{
		Input: in, Total: total, MemberID: actor.MemberID, ExternalPlatformID: actor.PlatformID,
		Environment: s.env, Status: StatusDraft,
	})
	return l, storeErr(err)
}

func (s *Service) Get(ctx context.Context, platformID uint, id string) (Link, error) {
	l, err := s.store.Get(ctx, platformID, id)
	return l, storeErr(err)
}

func (s *Service) List(ctx context.Context, platformID uint, f ListFilter) ([]Link, int64, error) {
	if f.Status != "" && !slices.Contains([]Status{StatusDraft, StatusActive, StatusPaused, StatusArchived}, f.Status) {
		return nil, 0, newErr(CodeInvalidRequest, "status", "must be draft, active, paused or archived")
	}
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 25
	}
	f.Offset = max(f.Offset, 0)
	return s.store.List(ctx, platformID, f)
}

// Update replaces the editable fields. A published link keeps amount, currency, methods and line items and is re-validated in full.
func (s *Service) Update(ctx context.Context, platformID uint, id string, in Input) (Link, error) {
	cur, err := s.Get(ctx, platformID, id)
	if err != nil {
		return Link{}, err
	}
	if cur.Status == StatusArchived {
		return Link{}, newErr(CodeNotEditable, "", "an archived link cannot be edited; duplicate it")
	}
	in, total, err := s.prepare(ctx, platformID, in)
	if err != nil {
		return Link{}, err
	}
	next := cur
	next.Input, next.Total = in, total
	if cur.Status.Published() {
		if f := immutableDiff(cur.Input, in); f != "" {
			return Link{}, newErr(CodePublishedImmutable, f, "a published link cannot change %s; duplicate it instead", f)
		}
		if err := s.validatePublish(ctx, next); err != nil {
			return Link{}, err
		}
	}
	saved, err := s.store.Save(ctx, next)
	return saved, storeErr(err)
}

func (s *Service) Delete(ctx context.Context, platformID uint, id string) error {
	cur, err := s.Get(ctx, platformID, id)
	if err != nil {
		return err
	}
	if cur.Status != StatusDraft {
		return newErr(CodeNotDeletable, "", "only drafts can be deleted; archive a published link")
	}
	return storeErr(s.store.DeleteDraft(ctx, platformID, id, cur.Revision))
}

// transition loads the link and refuses a move that the lifecycle (README "Lifecycle") does not allow.
func (s *Service) transition(ctx context.Context, platformID uint, id string, from []Status, to Status) (Link, error) {
	cur, err := s.Get(ctx, platformID, id)
	if err != nil {
		return Link{}, err
	}
	if !slices.Contains(from, cur.Status) {
		return Link{}, newErr(CodeInvalidTransition, "status", "cannot go from %s to %s", cur.Status, to)
	}
	return cur, nil
}

// Publish validates everything and makes the link payable; it also resumes a paused link. The short code is minted once.
func (s *Service) Publish(ctx context.Context, platformID uint, id string) (Link, error) {
	cur, err := s.transition(ctx, platformID, id, []Status{StatusDraft, StatusPaused}, StatusActive)
	if err != nil {
		return Link{}, err
	}
	if err := s.validatePublish(ctx, cur); err != nil {
		return Link{}, err
	}
	if cur.ShortCode != "" {
		l, err := s.store.SetStatus(ctx, platformID, id, cur.Revision, StatusActive, "", s.now())
		return l, storeErr(err)
	}
	for range shortCodeRetries {
		code, err := s.newCode()
		if err != nil {
			return Link{}, err
		}
		l, err := s.store.SetStatus(ctx, platformID, id, cur.Revision, StatusActive, code, s.now())
		if errors.Is(err, errShortCodeTaken) {
			continue
		}
		return l, storeErr(err)
	}
	return Link{}, newErr(CodeShortCodeExhausted, "", "could not mint a unique short code; retry")
}

func (s *Service) Pause(ctx context.Context, platformID uint, id string) (Link, error) {
	cur, err := s.transition(ctx, platformID, id, []Status{StatusActive}, StatusPaused)
	if err != nil {
		return Link{}, err
	}
	l, err := s.store.SetStatus(ctx, platformID, id, cur.Revision, StatusPaused, "", s.now())
	return l, storeErr(err)
}

func (s *Service) Archive(ctx context.Context, platformID uint, id string) (Link, error) {
	cur, err := s.transition(ctx, platformID, id, []Status{StatusDraft, StatusActive, StatusPaused}, StatusArchived)
	if err != nil {
		return Link{}, err
	}
	l, err := s.store.SetStatus(ctx, platformID, id, cur.Revision, StatusArchived, "", s.now())
	return l, storeErr(err)
}

// Duplicate copies the form into a new draft: the way to change what a published link may not.
func (s *Service) Duplicate(ctx context.Context, actor Actor, id string) (Link, error) {
	cur, err := s.Get(ctx, actor.PlatformID, id)
	if err != nil {
		return Link{}, err
	}
	c := cloneLink(cur)
	l, err := s.store.Insert(ctx, Link{
		Input: c.Input, Total: c.Total, MemberID: actor.MemberID, ExternalPlatformID: actor.PlatformID,
		Environment: s.env, Status: StatusDraft,
	})
	return l, storeErr(err)
}

// validatePublish is the full check: shape, completeness, and every method, fee rule and destination it names.
func (s *Service) validatePublish(ctx context.Context, l Link) error {
	errs := validateShape(l.Input, s.precision)
	add := func(code Code, field, format string, args ...any) {
		errs = append(errs, newErr(code, field, format, args...))
	}
	if l.Title == "" {
		add(CodeTitleRequired, "title", "is required")
	}
	if l.Currency == "" {
		add(CodeCurrencyRequired, "currency", "is required")
	}
	switch l.AmountMode {
	case AmountFixed:
		if l.Amount == nil {
			add(CodeAmountRequired, "amount", "a fixed link needs an amount")
		}
	case AmountLineItems:
		if len(l.LineItems) == 0 {
			add(CodeLineItemsRequired, "line_items", "add at least one line item")
		} else if l.Total == nil || !l.Total.IsPositive() {
			add(CodeAmountInvalid, "line_items", "the line items sum to zero")
		}
	}
	if len(l.Methods) == 0 {
		add(CodeMethodsRequired, "methods", "enable at least one way to pay")
	}
	switch {
	case l.SuccessMode == SuccessMessage && strings.TrimSpace(l.SuccessMessage) == "":
		add(CodeSuccessMessageRequired, "success_message", "is required when success_mode is message")
	case l.SuccessMode == SuccessRedirect && l.SuccessURL == "":
		add(CodeSuccessURLRequired, "success_url", "is required when success_mode is redirect")
	}
	if l.ExpiresAt != nil && !l.ExpiresAt.After(s.now()) {
		add(CodeExpiresAtInPast, "expires_at", "must be in the future")
	}
	hasCard := slices.ContainsFunc(l.Methods, func(m MethodSpec) bool { return m.Method == fees.MethodCard })
	hasCrypto := slices.ContainsFunc(l.Methods, func(m MethodSpec) bool { return m.Method == fees.MethodCrypto })
	if l.CaptureMode == CaptureManual && !hasCard {
		add(CodeCaptureModeNeedsCard, "capture_mode", "authorise-then-capture applies to card only")
	}
	if l.HoldInAsset && !hasCrypto {
		add(CodeHoldInAssetNeedsCrypto, "hold_in_asset", "holding in asset needs a crypto method")
	}
	if e := s.checkWebhook(ctx, l.ExternalPlatformID, l.Input); e != nil {
		errs = append(errs, e)
	}
	if o := l.SettlementOverride; o != nil && o.Kind != "" {
		ok, err := s.destinations.Verified(ctx, l.ExternalPlatformID, *o)
		if err != nil {
			return fmt.Errorf("links: verify settlement destination: %w", err)
		}
		if !ok {
			add(CodeDestinationUnverified, "settlement_override.destination_id", "destination %s is not verified", o.DestinationID)
		}
	}
	if len(errs) == 0 {
		for i, m := range l.Methods {
			e, err := s.checkMethod(ctx, l, m)
			if err != nil {
				return err
			}
			if e != nil {
				e.Field = fmt.Sprintf("methods[%d]", i)
				errs = append(errs, e)
			}
		}
	}
	return joinErrs(errs)
}

// priced is a method resolved for one amount: the connector, the rule, and the breakdown when computable.
type priced struct {
	connector string
	rule      fees.Rule
	breakdown *fees.Breakdown
}

// price finds a connector with an active fee rule for m and, when the fee currency is the link's, previews amount.
// errs are refusals (typed); err is an infrastructure failure.
func (s *Service) price(ctx context.Context, l Link, m MethodSpec, amount decimal.Decimal, preview bool) (priced, *Error, error) {
	conns, err := s.creator.Connectors(ctx, l.Environment, l.Currency, m)
	if err != nil {
		return priced{}, nil, fmt.Errorf("links: connectors for %s: %w", m, err)
	}
	if len(conns) == 0 {
		return priced{}, newErr(CodeMethodNoConnector, "", "no connector takes %s for %s in %s", m, l.Currency, l.Environment), nil
	}
	feeCur := m.FeeCurrency(l.Currency)
	var p priced
	found := false
	for _, c := range conns {
		r, err := s.fees.Resolve(ctx, fees.Query{Method: m.Method, Connector: c, Currency: feeCur, Chain: m.Chain})
		var amb *fees.AmbiguousRuleError
		switch {
		case errors.Is(err, fees.ErrNoRule):
			continue
		case errors.As(err, &amb):
			return priced{}, newErr(CodeFeeRuleAmbiguous, "", "fee rules %v tie for %s", amb.RuleIDs, m), nil
		case err != nil:
			return priced{}, nil, fmt.Errorf("links: resolve fee rule for %s: %w", m, err)
		}
		p, found = priced{connector: c, rule: r}, true
		break
	}
	if !found {
		return priced{}, newErr(CodeMethodNoFeeRule, "", "no active fee rule for %s in %s", m, feeCur), nil
	}
	if feeCur != l.Currency {
		if l.FeeBearer == fees.BearerCustomer {
			return priced{}, newErr(CodeSurchargeNeedsQuote, "", "a surcharge on %s needs a %s to %s quote, which is not available yet", m, l.Currency, feeCur), nil
		}
		return p, nil, nil
	}
	bearer := l.FeeBearer
	b, err := s.fees.Preview(ctx, fees.PreviewRequest{
		Query:     fees.Query{Method: m.Method, Connector: p.connector, Currency: feeCur, Chain: m.Chain},
		Amount:    amount,
		FeeBearer: &bearer,
	})
	switch {
	case errors.Is(err, fees.ErrSurchargeForbidden):
		return priced{}, newErr(CodeSurchargeForbidden, "fee_bearer", "%s does not allow a customer surcharge", m.Method), nil
	case errors.Is(err, fees.ErrFeeExceedsAmount):
		if preview {
			return priced{}, newErr(CodeFeeExceedsAmount, "", "the %s fee is larger than %s %s", m, amount, l.Currency), nil
		}
		return p, nil, nil
	case errors.Is(err, fees.ErrNoRule):
		return priced{}, newErr(CodeMethodNoFeeRule, "", "no active fee rule for %s in %s", m, feeCur), nil
	case err != nil:
		return priced{}, nil, fmt.Errorf("links: preview fee for %s: %w", m, err)
	}
	if preview {
		p.breakdown = &b
	}
	return p, nil, nil
}

// representative is the amount publish previews fees on, and whether it is a real amount a payer could pay.
func (s *Service) representative(l Link) (decimal.Decimal, bool) {
	switch {
	case l.Total != nil:
		return *l.Total, true
	case l.AmountMin != nil:
		return *l.AmountMin, true
	}
	return decimal.New(1, 0), false
}

func (s *Service) checkMethod(ctx context.Context, l Link, m MethodSpec) (*Error, error) {
	amount, real := s.representative(l)
	_, e, err := s.price(ctx, l, m, amount, real)
	return e, err
}

func (s *Service) publicLink(ctx context.Context, code string) (Link, error) {
	if !ValidShortCode(code) {
		return Link{}, notFound()
	}
	l, err := s.store.GetByShortCode(ctx, code)
	if err != nil {
		return Link{}, storeErr(err)
	}
	switch l.Status {
	case StatusDraft:
		return Link{}, notFound()
	case StatusArchived:
		return Link{}, newErr(CodeArchived, "", "this link is no longer available")
	}
	return l, nil
}

// Render is the public checkout view; it lists only the methods payable right now.
func (s *Service) Render(ctx context.Context, code string) (RenderModel, error) {
	l, err := s.publicLink(ctx, code)
	if err != nil {
		return RenderModel{}, err
	}
	merchant, err := s.store.MerchantName(ctx, l.ExternalPlatformID)
	if err != nil {
		return RenderModel{}, err
	}
	methods := []RenderMethod{}
	for _, m := range l.Methods {
		amount, known := decimal.Zero, l.Total != nil
		if known {
			amount = *l.Total
		} else {
			amount, _ = s.representative(l)
		}
		p, e, err := s.price(ctx, l, m, amount, known)
		if err != nil {
			return RenderModel{}, err
		}
		if e != nil {
			continue
		}
		rm := RenderMethod{Method: m.Method, Chain: strOrNil(m.Chain), Asset: strOrNil(m.Asset)}
		if p.breakdown != nil && l.FeeBearer == fees.BearerCustomer {
			rm.Fee, rm.Tax, rm.CustomerTotal = &p.breakdown.Fee, &p.breakdown.Tax, &p.breakdown.CustomerTotal
		}
		methods = append(methods, rm)
	}
	return s.renderModel(l, merchant, methods, availability(l, s.now())), nil
}

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

const maxIdempotencyKey = 128

// Pay validates the payer's input, reserves one use under a lock, and creates the payment through PaymentCreator.
func (s *Service) Pay(ctx context.Context, code string, req PayRequest) (PayResult, error) {
	key := strings.TrimSpace(req.IdempotencyKey)
	if key == "" || len(key) > maxIdempotencyKey {
		return PayResult{}, newErr(CodeIdempotencyKeyRequired, "Idempotency-Key", "send a unique Idempotency-Key header of 1-%d characters", maxIdempotencyKey)
	}
	l, err := s.publicLink(ctx, code)
	if err != nil {
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
		return s.replay(l, *existing, hash)
	}
	if err := availability(l, s.now()); err != nil {
		return PayResult{}, err
	}
	p, err := s.quote(ctx, l, req)
	if err != nil {
		return PayResult{}, err
	}
	p.IdempotencyKey, p.RequestHash = key, hash

	reserved, existing, err := s.store.Reserve(ctx, p, s.now())
	if err != nil {
		return PayResult{}, storeErr(err)
	}
	if existing != nil {
		return s.replay(l, *existing, hash)
	}

	created, err := s.creator.CreatePayment(ctx, PaymentRequest{
		LinkID: l.ID, LinkPaymentID: reserved.ID, MemberID: l.MemberID, PlatformID: l.ExternalPlatformID,
		Environment: l.Environment, Method: p.Method, Connector: p.Connector,
		Amount: p.Amount, Currency: p.Currency, CustomerTotal: p.CustomerTotal, FeeBearer: p.FeeBearer,
		FeeRuleID: derefUint(p.FeeRuleID), FeeRuleVersion: derefInt(p.FeeRuleVersion),
		CustomerName: p.CustomerName, CustomerEmail: p.CustomerEmail, CustomerPhone: p.CustomerPhone,
		BillingAddress: p.BillingAddress, ShippingAddress: p.ShippingAddress,
		ReferenceID: l.ReferenceID, Metadata: l.Metadata,
		CaptureMode: l.CaptureMode, ThreeDSPolicy: l.ThreeDSPolicy,
		ChainToleranceBps: l.ChainToleranceBps, QuoteExpirySeconds: l.QuoteExpirySeconds, WebhookID: l.WebhookID,
	})
	if err != nil {
		if rerr := s.store.Release(ctx, reserved.ID); rerr != nil {
			slog.Error("links: release reservation after failed payment creation", "link_payment_id", reserved.ID, "error", rerr)
		}
		var e *Error
		if errors.As(err, &e) {
			return PayResult{}, e
		}
		slog.Error("links: payment creation failed", "link_payment_id", reserved.ID, "error", err)
		return PayResult{}, newErr(CodePaymentCreationFailed, "", "the payment could not be created; retry with the same key")
	}
	if err := s.store.Complete(ctx, reserved.ID, created); err != nil {
		slog.Error("links: payment created but reservation not completed", "link_payment_id", reserved.ID, "payment_reference", created.Reference, "error", err)
		return PayResult{}, fmt.Errorf("links: complete reservation %s: %w", reserved.ID, err)
	}
	reserved.Status, reserved.PaymentReference, reserved.Processor = paymentCreated, created.Reference, &created
	return s.result(l, reserved, false), nil
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

func (s *Service) replay(l Link, p LinkPayment, hash string) (PayResult, error) {
	switch {
	case p.RequestHash != hash:
		return PayResult{}, newErr(CodeIdempotencyKeyReused, "Idempotency-Key", "this key was used for a different payment")
	case p.Status != paymentCreated:
		return PayResult{}, newErr(CodePaymentInProgress, "Idempotency-Key", "a payment with this key is still being created; retry shortly")
	}
	return s.result(l, p, true), nil
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

	answers, aerrs, err := s.checkAnswers(ctx, l, email, req.Answers)
	if err != nil {
		return LinkPayment{}, err
	}
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

func (s *Service) checkAnswers(ctx context.Context, l Link, email string, in map[string]string) ([]Answer, []*Error, error) {
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
	var perCustomer []string
	for _, q := range l.Questions {
		if !q.PerOrder && q.Required {
			perCustomer = append(perCustomer, q.Key)
		}
	}
	answered := map[string]bool{}
	if email != "" && len(perCustomer) > 0 {
		var err error
		if answered, err = s.store.AnsweredBefore(ctx, l.ID, email, perCustomer); err != nil {
			return nil, nil, err
		}
	}
	var out []Answer
	for _, q := range l.Questions {
		field := "answers." + q.Key
		v := strings.TrimSpace(in[q.Key])
		if v == "" {
			if q.Required && !answered[q.Key] {
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
			if v == "false" && q.Required && !answered[q.Key] {
				errs = append(errs, newErr(CodeAnswerRequired, field, "%s must be checked", q.Label))
				continue
			}
		}
		out = append(out, Answer{QuestionKey: q.Key, QuestionLabel: q.Label, Value: v})
	}
	return out, errs, nil
}
