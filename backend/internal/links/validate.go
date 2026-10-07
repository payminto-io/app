package links

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

const (
	maxTitle       = 200
	maxDescription = 5000
	maxReference   = 100
	maxCategory    = 64
	maxMetaKeys    = 20
	maxMetaKey     = 40
	maxMetaValue   = 500
	maxText        = 1000
	maxLineItems   = 100
	maxQuantity    = 100000
	maxQuestions   = 20
	maxOptions     = 50
	maxOption      = 100
	maxMethods     = 20
	maxURL         = 2048
	maxToleranceBp = 1000
	minQuoteExpiry = 60
	maxQuoteExpiry = 86400
)

var (
	questionKey = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)
	accentColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	language    = regexp.MustCompile(`^[a-z]{2,3}(-[A-Z]{2})?$`)
	chainCode   = regexp.MustCompile(`^[A-Z0-9_]{2,20}$`)
	phoneE164   = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
	country     = regexp.MustCompile(`^[A-Z]{2}$`)
)

// normalizeInput canonicalises codes and fills defaults for empty enums, so stored links compare equal.
func normalizeInput(in Input) Input {
	in.Title = strings.TrimSpace(in.Title)
	in.Currency = normCode(in.Currency)
	if in.AmountMode == "" {
		in.AmountMode = AmountFixed
	}
	if in.Metadata == nil {
		in.Metadata = map[string]string{}
	}
	for i := range in.Methods {
		in.Methods[i].Method = fees.Method(strings.ToLower(strings.TrimSpace(string(in.Methods[i].Method))))
		in.Methods[i].Chain = normCode(in.Methods[i].Chain)
		in.Methods[i].Asset = normCode(in.Methods[i].Asset)
	}
	if in.Methods == nil {
		in.Methods = []MethodSpec{}
	}
	if in.LineItems == nil {
		in.LineItems = []LineItem{}
	}
	if in.Questions == nil {
		in.Questions = []Question{}
	}
	d := DefaultInput()
	for _, p := range []struct {
		v   *string
		def string
	}{
		{&in.CaptureMode, d.CaptureMode}, {&in.ThreeDSPolicy, d.ThreeDSPolicy}, {&in.SuccessMode, d.SuccessMode},
		{&in.SettlementTiming, d.SettlementTiming}, {&in.Language, d.Language},
	} {
		if *p.v == "" {
			*p.v = p.def
		}
	}
	if in.FeeBearer == "" {
		in.FeeBearer = d.FeeBearer
	}
	for _, r := range []*FieldRule{&in.CustomerFields.Name, &in.CustomerFields.Email, &in.CustomerFields.Phone} {
		if r.Mode == "" {
			r.Mode = FieldOptional
		}
		r.Prefill = strings.TrimSpace(r.Prefill)
	}
	if o := in.SettlementOverride; o != nil {
		o.Kind = strings.ToLower(strings.TrimSpace(o.Kind))
		o.Chain, o.Asset = normCode(o.Chain), normCode(o.Asset)
	}
	return in
}

// validateShape refuses contradictions and malformed values; a draft may still be incomplete.
func validateShape(in Input, p fees.Precision) []*Error {
	var errs []*Error
	add := func(code Code, field, format string, args ...any) {
		errs = append(errs, newErr(code, field, format, args...))
	}

	if utf8.RuneCountInString(in.Title) > maxTitle {
		add(CodeTitleTooLong, "title", "at most %d characters", maxTitle)
	}
	if utf8.RuneCountInString(in.Description) > maxDescription {
		add(CodeDescriptionTooLong, "description", "at most %d characters", maxDescription)
	}

	places, currencyOK := int32(0), false
	if in.Currency != "" {
		if places, currencyOK = p.MinorUnits(in.Currency); !currencyOK {
			add(CodeCurrencyUnsupported, "currency", "%s is not a supported currency", in.Currency)
		}
	}
	money := func(field string, v *decimal.Decimal) {
		if v != nil && (!sane(*v) || !v.Abs().LessThan(maxMoney)) {
			add(CodeAmountInvalid, field, "out of range")
			return
		}
		if v == nil || !currencyOK {
			return
		}
		if err := checkMoney(field, *v, places); err != nil {
			errs = append(errs, err.(*Error))
		}
	}

	switch in.AmountMode {
	case AmountFixed:
		money("amount", in.Amount)
		if in.AmountMin != nil || in.AmountMax != nil {
			add(CodeAmountBoundsNotAllowed, "amount_min", "minimum and maximum apply only to customer-entered amounts")
		}
		if len(in.LineItems) > 0 {
			add(CodeLineItemsNotAllowed, "line_items", "line items need amount_mode line_items")
		}
	case AmountCustomer:
		if in.Amount != nil {
			add(CodeAmountNotAllowed, "amount", "the customer enters the amount")
		}
		money("amount_min", in.AmountMin)
		money("amount_max", in.AmountMax)
		if in.AmountMin != nil && in.AmountMax != nil && in.AmountMin.GreaterThan(*in.AmountMax) {
			add(CodeAmountRangeInvalid, "amount_min", "minimum is above maximum")
		}
		if len(in.LineItems) > 0 {
			add(CodeLineItemsNotAllowed, "line_items", "line items need amount_mode line_items")
		}
	case AmountLineItems:
		if in.Amount != nil {
			add(CodeAmountNotAllowed, "amount", "the amount is the sum of the line items")
		}
		if in.AmountMin != nil || in.AmountMax != nil {
			add(CodeAmountBoundsNotAllowed, "amount_min", "minimum and maximum apply only to customer-entered amounts")
		}
		lineErrs := validateLineItems(in.LineItems, places, currencyOK)
		errs = append(errs, lineErrs...)
		if len(lineErrs) == 0 && len(in.LineItems) > 0 && !itemsTotals(in.LineItems, places).Total.LessThan(maxMoney) {
			add(CodeAmountInvalid, "line_items", "the line items sum to more than the largest amount")
		}
	default:
		add(CodeAmountModeInvalid, "amount_mode", "must be fixed, customer or line_items")
	}

	if utf8.RuneCountInString(in.ReferenceID) > maxReference {
		add(CodeReferenceTooLong, "reference_id", "at most %d characters", maxReference)
	}
	if utf8.RuneCountInString(in.Category) > maxCategory {
		add(CodeCategoryTooLong, "category", "at most %d characters", maxCategory)
	}
	if len(in.Metadata) > maxMetaKeys {
		add(CodeMetadataInvalid, "metadata", "at most %d keys", maxMetaKeys)
	}
	for k, v := range in.Metadata {
		if k == "" || utf8.RuneCountInString(k) > maxMetaKey || utf8.RuneCountInString(v) > maxMetaValue {
			add(CodeMetadataInvalid, "metadata", "keys are 1-%d characters and values at most %d", maxMetaKey, maxMetaValue)
			break
		}
	}

	errs = append(errs, validatePolicy(in)...)
	errs = append(errs, validateQuestions(in.Questions)...)

	if in.UseLimit != nil && *in.UseLimit < 1 {
		add(CodeUseLimitInvalid, "use_limit", "must be at least 1")
	}
	if in.ExpiresAfterPayments != nil && *in.ExpiresAfterPayments < 1 {
		add(CodeUseLimitInvalid, "expires_after_payments", "must be at least 1")
	}
	if !in.MultiUse && in.UseLimit != nil {
		add(CodeUseLimitNeedsMultiUse, "use_limit", "a single-use link takes one payment")
	}
	if !in.MultiUse && in.ExpiresAfterPayments != nil {
		add(CodeUseLimitNeedsMultiUse, "expires_after_payments", "a single-use link takes one payment")
	}

	errs = append(errs, validateMethods(in.Methods, p)...)

	if !slices.Contains([]string{CaptureAutomatic, CaptureManual}, in.CaptureMode) {
		add(CodeCaptureModeInvalid, "capture_mode", "must be automatic or manual")
	}
	if !slices.Contains([]string{ThreeDSInherit, ThreeDSForce}, in.ThreeDSPolicy) {
		add(CodeThreeDSInvalid, "three_ds_policy", "must be inherit or force")
	}
	if in.ChainToleranceBps < 0 || in.ChainToleranceBps > maxToleranceBp {
		add(CodeChainToleranceInvalid, "chain_tolerance_bps", "must be 0-%d", maxToleranceBp)
	}
	if in.QuoteExpirySeconds < minQuoteExpiry || in.QuoteExpirySeconds > maxQuoteExpiry {
		add(CodeQuoteExpiryInvalid, "quote_expiry_seconds", "must be %d-%d", minQuoteExpiry, maxQuoteExpiry)
	}
	if in.FeeBearer != fees.BearerMerchant && in.FeeBearer != fees.BearerCustomer {
		add(CodeFeeBearerInvalid, "fee_bearer", "must be merchant or customer")
	}

	if !slices.Contains([]string{SuccessMessage, SuccessRedirect}, in.SuccessMode) {
		add(CodeSuccessModeInvalid, "success_mode", "must be message or redirect")
	}
	if in.SuccessURL != "" && !validURL(in.SuccessURL, false) {
		add(CodeSuccessURLInvalid, "success_url", "must be an absolute http or https URL")
	}
	for _, f := range []struct{ name, v string }{
		{"success_message", in.SuccessMessage}, {"receipt_note", in.ReceiptNote}, {"failure_message", in.FailureMessage},
	} {
		if utf8.RuneCountInString(f.v) > maxText {
			add(CodeTextTooLong, f.name, "at most %d characters", maxText)
		}
	}
	if in.ReceiptEmail && in.CustomerFields.Email.Mode == FieldHidden && in.CustomerFields.Email.Prefill == "" {
		add(CodeReceiptNeedsEmail, "receipt_email", "a receipt needs the customer's email: ask for it or prefill it")
	}

	if o := in.SettlementOverride; o != nil {
		switch {
		case o.Kind != SettleFiat && o.Kind != SettleCrypto:
			add(CodeSettlementInvalid, "settlement_override.kind", "must be fiat or crypto")
		case strings.TrimSpace(o.DestinationID) == "" || len(o.DestinationID) > 128:
			add(CodeSettlementInvalid, "settlement_override.destination_id", "is required")
		case o.Kind == SettleCrypto && (!chainCode.MatchString(o.Chain) || o.Asset == ""):
			add(CodeSettlementInvalid, "settlement_override.chain", "crypto settlement needs chain and asset")
		case o.Kind == SettleFiat && (o.Chain != "" || o.Asset != ""):
			add(CodeSettlementInvalid, "settlement_override.chain", "fiat settlement has no chain or asset")
		}
	}
	if !slices.Contains([]string{TimingCycle, TimingImmediate}, in.SettlementTiming) {
		add(CodeSettlementTimingInvalid, "settlement_timing", "must be cycle or immediate")
	}

	if in.LogoURL != "" && !validURL(in.LogoURL, true) {
		add(CodeLogoURLInvalid, "logo_url", "must be an absolute https URL")
	}
	if in.AccentColor != "" && !accentColor.MatchString(in.AccentColor) {
		add(CodeAccentColorInvalid, "accent_color", "must be #RRGGBB")
	}
	if !language.MatchString(in.Language) {
		add(CodeLanguageInvalid, "language", "must be a language tag like en or pt-BR")
	}
	return errs
}

func validateLineItems(items []LineItem, places int32, currencyOK bool) []*Error {
	var errs []*Error
	if len(items) > maxLineItems {
		return []*Error{newErr(CodeLineItemInvalid, "line_items", "at most %d line items", maxLineItems)}
	}
	for i, li := range items {
		field := func(f string) string { return fmt.Sprintf("line_items[%d].%s", i, f) }
		name := strings.TrimSpace(li.Name)
		switch {
		case name == "" || utf8.RuneCountInString(name) > maxTitle:
			errs = append(errs, newErr(CodeLineItemInvalid, field("name"), "is required, at most %d characters", maxTitle))
		case li.Quantity < 1 || li.Quantity > maxQuantity:
			errs = append(errs, newErr(CodeLineItemInvalid, field("quantity"), "must be 1-%d", maxQuantity))
		case !sane(li.UnitPrice) || li.UnitPrice.IsNegative() || !li.UnitPrice.Abs().LessThan(maxMoney):
			errs = append(errs, newErr(CodeLineItemInvalid, field("unit_price"), "must be zero or positive"))
		case currencyOK && !li.UnitPrice.Equal(li.UnitPrice.Truncate(places)):
			errs = append(errs, newErr(CodeLineItemInvalid, field("unit_price"), "has more than %d decimal places", places))
		case !sane(li.TaxRate) || li.TaxRate.IsNegative() || li.TaxRate.GreaterThan(hundred) || !li.TaxRate.Equal(li.TaxRate.Truncate(6)):
			errs = append(errs, newErr(CodeLineItemInvalid, field("tax_rate"), "must be 0-100 percent with at most 6 decimals"))
		}
	}
	return errs
}

func validatePolicy(in Input) []*Error {
	var errs []*Error
	for _, f := range []struct {
		name string
		r    FieldRule
	}{{"name", in.CustomerFields.Name}, {"email", in.CustomerFields.Email}, {"phone", in.CustomerFields.Phone}} {
		field := "customer_field_policy." + f.name
		if !slices.Contains([]FieldMode{FieldRequired, FieldOptional, FieldHidden}, f.r.Mode) {
			errs = append(errs, newErr(CodeCustomerPolicyInvalid, field+".mode", "must be required, optional or hidden"))
			continue
		}
		if f.r.Prefill == "" {
			continue
		}
		if err := checkCustomerValue(f.name, f.r.Prefill); err != nil {
			errs = append(errs, newErr(CodeCustomerPolicyInvalid, field+".prefill", "%s", err.Message))
		}
	}
	return errs
}

// checkCustomerValue validates one customer field value; it is shared by prefills and payers.
func checkCustomerValue(name, v string) *Error {
	field := "customer." + name
	switch name {
	case "email":
		a, err := mail.ParseAddress(v)
		if err != nil || a.Address != v || len(v) > 254 {
			return newErr(CodeCustomerEmailInvalid, field, "must be a plain email address")
		}
	case "phone":
		if !phoneE164.MatchString(v) {
			return newErr(CodeCustomerPhoneInvalid, field, "must be E.164, like +14155550123")
		}
	default:
		if utf8.RuneCountInString(v) > maxTitle {
			return newErr(CodeCustomerNameInvalid, field, "at most %d characters", maxTitle)
		}
	}
	return nil
}

func validateQuestions(qs []Question) []*Error {
	if len(qs) > maxQuestions {
		return []*Error{newErr(CodeQuestionInvalid, "questions", "at most %d questions", maxQuestions)}
	}
	var errs []*Error
	seen := map[string]bool{}
	for i, q := range qs {
		field := func(f string) string { return fmt.Sprintf("questions[%d].%s", i, f) }
		if !questionKey.MatchString(q.Key) {
			errs = append(errs, newErr(CodeQuestionInvalid, field("key"), "must be 1-64 of a-z, 0-9 and _"))
			continue
		}
		if seen[q.Key] {
			errs = append(errs, newErr(CodeQuestionKeyDuplicate, field("key"), "key %q is used twice", q.Key))
			continue
		}
		seen[q.Key] = true
		label := strings.TrimSpace(q.Label)
		if label == "" || utf8.RuneCountInString(label) > maxTitle {
			errs = append(errs, newErr(CodeQuestionInvalid, field("label"), "is required, at most %d characters", maxTitle))
			continue
		}
		switch q.Type {
		case QuestionSelect:
			if len(q.Options) == 0 || len(q.Options) > maxOptions {
				errs = append(errs, newErr(CodeQuestionInvalid, field("options"), "a select needs 1-%d options", maxOptions))
				continue
			}
			opts := map[string]bool{}
			for _, o := range q.Options {
				if strings.TrimSpace(o) == "" || utf8.RuneCountInString(o) > maxOption || opts[o] {
					errs = append(errs, newErr(CodeQuestionInvalid, field("options"), "options are unique and 1-%d characters", maxOption))
					break
				}
				opts[o] = true
			}
		case QuestionText, QuestionCheckbox:
			if len(q.Options) > 0 {
				errs = append(errs, newErr(CodeQuestionInvalid, field("options"), "only a select has options"))
			}
		default:
			errs = append(errs, newErr(CodeQuestionInvalid, field("type"), "must be text, select or checkbox"))
		}
	}
	return errs
}

func validateMethods(ms []MethodSpec, p fees.Precision) []*Error {
	if len(ms) > maxMethods {
		return []*Error{newErr(CodeMethodInvalid, "methods", "at most %d methods", maxMethods)}
	}
	var errs []*Error
	seen := map[string]bool{}
	for i, m := range ms {
		field := fmt.Sprintf("methods[%d]", i)
		switch {
		case !slices.Contains(fees.Methods, m.Method):
			errs = append(errs, newErr(CodeMethodUnknown, field+".method", "must be one of card, upi, bank, crypto"))
			continue
		case m.Method == fees.MethodCrypto && (m.Chain == "" || m.Asset == ""):
			errs = append(errs, newErr(CodeMethodChainRequired, field+".chain", "crypto needs a chain and an asset"))
			continue
		case m.Method == fees.MethodCrypto && !chainCode.MatchString(m.Chain):
			errs = append(errs, newErr(CodeMethodInvalid, field+".chain", "chain code must be 2-20 of A-Z, 0-9 and _"))
			continue
		case m.Method == fees.MethodCrypto && (p.IsFiat(m.Asset) || !isKnown(p, m.Asset)):
			errs = append(errs, newErr(CodeMethodAssetUnsupported, field+".asset", "%s is not a supported on-chain asset", m.Asset))
			continue
		case m.Method != fees.MethodCrypto && (m.Chain != "" || m.Asset != ""):
			errs = append(errs, newErr(CodeMethodInvalid, field, "%s has no chain or asset", m.Method))
			continue
		}
		if seen[m.String()] {
			errs = append(errs, newErr(CodeMethodDuplicate, field, "%s is listed twice", m))
			continue
		}
		seen[m.String()] = true
	}
	return errs
}

func isKnown(p fees.Precision, code string) bool {
	_, ok := p.MinorUnits(code)
	return ok
}

func validURL(raw string, httpsOnly bool) bool {
	if len(raw) > maxURL {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return false
	}
	if httpsOnly {
		return u.Scheme == "https"
	}
	return u.Scheme == "https" || u.Scheme == "http"
}

func validateAddress(field string, a Address) *Error {
	bad := func(f string) *Error {
		return newErr(CodeAddressInvalid, field+"."+f, "is required, at most 200 characters")
	}
	for _, f := range []struct{ name, v string }{{"line1", a.Line1}, {"city", a.City}, {"postal_code", a.PostalCode}} {
		if strings.TrimSpace(f.v) == "" || utf8.RuneCountInString(f.v) > maxTitle {
			return bad(f.name)
		}
	}
	for _, f := range []struct{ name, v string }{{"line2", a.Line2}, {"state", a.State}} {
		if utf8.RuneCountInString(f.v) > maxTitle {
			return bad(f.name)
		}
	}
	if !country.MatchString(a.Country) {
		return newErr(CodeAddressInvalid, field+".country", "must be an ISO 3166-1 alpha-2 code")
	}
	return nil
}

// immutableDiff names the first field a published link may not change; duplicate the link instead.
func immutableDiff(old, next Input) string {
	switch {
	case old.AmountMode != next.AmountMode:
		return "amount_mode"
	case !decEq(old.Amount, next.Amount):
		return "amount"
	case !decEq(old.AmountMin, next.AmountMin):
		return "amount_min"
	case !decEq(old.AmountMax, next.AmountMax):
		return "amount_max"
	case old.Currency != next.Currency:
		return "currency"
	case old.FeeBearer != next.FeeBearer:
		return "fee_bearer"
	case !slices.Equal(old.Methods, next.Methods):
		return "methods"
	case !slices.EqualFunc(old.LineItems, next.LineItems, func(a, b LineItem) bool {
		return a.Name == b.Name && a.Quantity == b.Quantity && a.UnitPrice.Equal(b.UnitPrice) && a.TaxRate.Equal(b.TaxRate)
	}):
		return "line_items"
	}
	return ""
}

func decEq(a, b *decimal.Decimal) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
