package links

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/fees"
)

func TestCreateRefusesEveryShapeRule(t *testing.T) {
	cases := []struct {
		name  string
		code  Code
		field string
		edit  func(*Input)
	}{
		{"title too long", CodeTitleTooLong, "title", func(in *Input) { in.Title = strings.Repeat("x", 201) }},
		{"description too long", CodeDescriptionTooLong, "description", func(in *Input) { in.Description = strings.Repeat("x", 5001) }},
		{"unknown amount mode", CodeAmountModeInvalid, "amount_mode", func(in *Input) { in.AmountMode = "tip" }},
		{"unknown currency", CodeCurrencyUnsupported, "currency", func(in *Input) { in.Currency = "XYZ" }},
		{"zero amount", CodeAmountInvalid, "amount", func(in *Input) { in.Amount = decp("0") }},
		{"negative amount", CodeAmountInvalid, "amount", func(in *Input) { in.Amount = decp("-1") }},
		{"amount finer than cents", CodeAmountInvalid, "amount", func(in *Input) { in.Amount = decp("1.001") }},
		{"absurd amount", CodeAmountInvalid, "amount", func(in *Input) { in.Amount = decp("1e30") }},
		{"yen with decimals", CodeAmountInvalid, "amount", func(in *Input) { in.Currency, in.Amount = "JPY", decp("100.5") }},
		{"fixed with bounds", CodeAmountBoundsNotAllowed, "amount_min", func(in *Input) { in.AmountMin = decp("1") }},
		{"fixed with line items", CodeLineItemsNotAllowed, "line_items", func(in *Input) {
			in.LineItems = []LineItem{{Name: "a", Quantity: 1, UnitPrice: dec("1")}}
		}},
		{"customer with amount", CodeAmountNotAllowed, "amount", func(in *Input) { in.AmountMode = AmountCustomer }},
		{"customer min above max", CodeAmountRangeInvalid, "amount_min", func(in *Input) {
			in.AmountMode, in.Amount, in.AmountMin, in.AmountMax = AmountCustomer, nil, decp("10"), decp("5")
		}},
		{"customer min finer than cents", CodeAmountInvalid, "amount_min", func(in *Input) {
			in.AmountMode, in.Amount, in.AmountMin = AmountCustomer, nil, decp("0.001")
		}},
		{"line items with client amount", CodeAmountNotAllowed, "amount", func(in *Input) {
			in.AmountMode = AmountLineItems
			in.LineItems = []LineItem{{Name: "a", Quantity: 1, UnitPrice: dec("1")}}
		}},
		{"line items with bounds", CodeAmountBoundsNotAllowed, "amount_min", func(in *Input) {
			in.AmountMode, in.Amount, in.AmountMin = AmountLineItems, nil, decp("1")
		}},
		{"line item without name", CodeLineItemInvalid, "line_items[0].name", func(in *Input) {
			in.AmountMode, in.Amount = AmountLineItems, nil
			in.LineItems = []LineItem{{Name: " ", Quantity: 1, UnitPrice: dec("1")}}
		}},
		{"line item zero quantity", CodeLineItemInvalid, "line_items[0].quantity", func(in *Input) {
			in.AmountMode, in.Amount = AmountLineItems, nil
			in.LineItems = []LineItem{{Name: "a", Quantity: 0, UnitPrice: dec("1")}}
		}},
		{"line item negative price", CodeLineItemInvalid, "line_items[0].unit_price", func(in *Input) {
			in.AmountMode, in.Amount = AmountLineItems, nil
			in.LineItems = []LineItem{{Name: "a", Quantity: 1, UnitPrice: dec("-1")}}
		}},
		{"line item price finer than cents", CodeLineItemInvalid, "line_items[0].unit_price", func(in *Input) {
			in.AmountMode, in.Amount = AmountLineItems, nil
			in.LineItems = []LineItem{{Name: "a", Quantity: 1, UnitPrice: dec("1.005")}}
		}},
		{"line item tax over 100", CodeLineItemInvalid, "line_items[0].tax_rate", func(in *Input) {
			in.AmountMode, in.Amount = AmountLineItems, nil
			in.LineItems = []LineItem{{Name: "a", Quantity: 1, UnitPrice: dec("1"), TaxRate: dec("101")}}
		}},
		{"reference too long", CodeReferenceTooLong, "reference_id", func(in *Input) { in.ReferenceID = strings.Repeat("r", 101) }},
		{"category too long", CodeCategoryTooLong, "category", func(in *Input) { in.Category = strings.Repeat("c", 65) }},
		{"metadata empty key", CodeMetadataInvalid, "metadata", func(in *Input) { in.Metadata = map[string]string{"": "v"} }},
		{"metadata too many keys", CodeMetadataInvalid, "metadata", func(in *Input) {
			for i := range 21 {
				in.Metadata[strings.Repeat("k", i+1)] = "v"
			}
		}},
		{"customer field mode", CodeCustomerPolicyInvalid, "customer_field_policy.phone.mode", func(in *Input) { in.CustomerFields.Phone.Mode = "maybe" }},
		{"prefilled email invalid", CodeCustomerPolicyInvalid, "customer_field_policy.email.prefill", func(in *Input) { in.CustomerFields.Email.Prefill = "not-an-email" }},
		{"prefilled phone invalid", CodeCustomerPolicyInvalid, "customer_field_policy.phone.prefill", func(in *Input) { in.CustomerFields.Phone.Prefill = "12345" }},
		{"question bad key", CodeQuestionInvalid, "questions[0].key", func(in *Input) {
			in.Questions = []Question{{Key: "Bad Key", Label: "x", Type: QuestionText}}
		}},
		{"question duplicate key", CodeQuestionKeyDuplicate, "questions[1].key", func(in *Input) {
			in.Questions = []Question{{Key: "a", Label: "x", Type: QuestionText}, {Key: "a", Label: "y", Type: QuestionText}}
		}},
		{"question without label", CodeQuestionInvalid, "questions[0].label", func(in *Input) {
			in.Questions = []Question{{Key: "a", Type: QuestionText}}
		}},
		{"question unknown type", CodeQuestionInvalid, "questions[0].type", func(in *Input) {
			in.Questions = []Question{{Key: "a", Label: "x", Type: "radio"}}
		}},
		{"select without options", CodeQuestionInvalid, "questions[0].options", func(in *Input) {
			in.Questions = []Question{{Key: "a", Label: "x", Type: QuestionSelect}}
		}},
		{"select duplicate options", CodeQuestionInvalid, "questions[0].options", func(in *Input) {
			in.Questions = []Question{{Key: "a", Label: "x", Type: QuestionSelect, Options: []string{"S", "S"}}}
		}},
		{"text with options", CodeQuestionInvalid, "questions[0].options", func(in *Input) {
			in.Questions = []Question{{Key: "a", Label: "x", Type: QuestionText, Options: []string{"S"}}}
		}},
		{"use limit zero", CodeUseLimitInvalid, "use_limit", func(in *Input) { in.MultiUse, in.UseLimit = true, intp(0) }},
		{"expires after zero payments", CodeUseLimitInvalid, "expires_after_payments", func(in *Input) {
			in.MultiUse, in.ExpiresAfterPayments = true, intp(0)
		}},
		{"use limit on single use", CodeUseLimitNeedsMultiUse, "use_limit", func(in *Input) { in.UseLimit = intp(5) }},
		{"payment cap on single use", CodeUseLimitNeedsMultiUse, "expires_after_payments", func(in *Input) { in.ExpiresAfterPayments = intp(5) }},
		{"unknown method", CodeMethodUnknown, "methods[0].method", func(in *Input) { in.Methods = []MethodSpec{{Method: "cash"}} }},
		{"crypto without chain", CodeMethodChainRequired, "methods[0].chain", func(in *Input) {
			in.Methods = []MethodSpec{{Method: fees.MethodCrypto, Asset: "USDC"}}
		}},
		{"crypto with fiat asset", CodeMethodAssetUnsupported, "methods[0].asset", func(in *Input) {
			in.Methods = []MethodSpec{{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USD"}}
		}},
		{"crypto bad chain", CodeMethodInvalid, "methods[0].chain", func(in *Input) {
			in.Methods = []MethodSpec{{Method: fees.MethodCrypto, Chain: "s", Asset: "USDC"}}
		}},
		{"card with chain", CodeMethodInvalid, "methods[0]", func(in *Input) { in.Methods = []MethodSpec{{Method: fees.MethodCard, Chain: "SOL"}} }},
		{"duplicate method", CodeMethodDuplicate, "methods[1]", func(in *Input) { in.Methods = []MethodSpec{card, card} }},
		{"capture mode", CodeCaptureModeInvalid, "capture_mode", func(in *Input) { in.CaptureMode = "later" }},
		{"3ds policy", CodeThreeDSInvalid, "three_ds_policy", func(in *Input) { in.ThreeDSPolicy = "never" }},
		{"tolerance negative", CodeChainToleranceInvalid, "chain_tolerance_bps", func(in *Input) { in.ChainToleranceBps = -1 }},
		{"tolerance over 10%", CodeChainToleranceInvalid, "chain_tolerance_bps", func(in *Input) { in.ChainToleranceBps = 1001 }},
		{"quote expiry short", CodeQuoteExpiryInvalid, "quote_expiry_seconds", func(in *Input) { in.QuoteExpirySeconds = 59 }},
		{"quote expiry long", CodeQuoteExpiryInvalid, "quote_expiry_seconds", func(in *Input) { in.QuoteExpirySeconds = 86401 }},
		{"fee bearer", CodeFeeBearerInvalid, "fee_bearer", func(in *Input) { in.FeeBearer = "platform" }},
		{"success mode", CodeSuccessModeInvalid, "success_mode", func(in *Input) { in.SuccessMode = "confetti" }},
		{"success url not absolute", CodeSuccessURLInvalid, "success_url", func(in *Input) { in.SuccessURL = "/thanks" }},
		{"success url javascript", CodeSuccessURLInvalid, "success_url", func(in *Input) { in.SuccessURL = "javascript:alert(1)" }},
		{"receipt note too long", CodeTextTooLong, "receipt_note", func(in *Input) { in.ReceiptNote = strings.Repeat("n", 1001) }},
		{"receipt without email", CodeReceiptNeedsEmail, "receipt_email", func(in *Input) {
			in.ReceiptEmail, in.CustomerFields.Email.Mode = true, FieldHidden
		}},
		{"webhook of another platform", CodeWebhookNotFound, "webhook_id", func(in *Input) { in.WebhookID = uintp(99) }},
		{"settlement kind", CodeSettlementInvalid, "settlement_override.kind", func(in *Input) {
			in.SettlementOverride = &SettlementOverride{Kind: "cash", DestinationID: "d"}
		}},
		{"settlement without destination", CodeSettlementInvalid, "settlement_override.destination_id", func(in *Input) {
			in.SettlementOverride = &SettlementOverride{Kind: SettleFiat}
		}},
		{"crypto settlement without chain", CodeSettlementInvalid, "settlement_override.chain", func(in *Input) {
			in.SettlementOverride = &SettlementOverride{Kind: SettleCrypto, DestinationID: "d"}
		}},
		{"fiat settlement with chain", CodeSettlementInvalid, "settlement_override.chain", func(in *Input) {
			in.SettlementOverride = &SettlementOverride{Kind: SettleFiat, DestinationID: "d", Chain: "SOL"}
		}},
		{"settlement timing", CodeSettlementTimingInvalid, "settlement_timing", func(in *Input) { in.SettlementTiming = "weekly" }},
		{"logo over http", CodeLogoURLInvalid, "logo_url", func(in *Input) { in.LogoURL = "http://cdn.test/logo.png" }},
		{"accent colour", CodeAccentColorInvalid, "accent_color", func(in *Input) { in.AccentColor = "red" }},
		{"language", CodeLanguageInvalid, "language", func(in *Input) { in.Language = "english" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			in := validInput()
			tc.edit(&in)
			_, err := f.svc.Create(context.Background(), merchant, in)
			if !hasCode(err, tc.code) {
				t.Fatalf("codes %v, want %s (%v)", Codes(err), tc.code, err)
			}
			var found bool
			for _, e := range allErrs(err) {
				found = found || (e.Code == tc.code && e.Field == tc.field)
			}
			if !found {
				t.Fatalf("%s not reported on field %s: %v", tc.code, tc.field, err)
			}
		})
	}
}

func allErrs(err error) []*Error {
	e, ok := err.(*Error)
	if !ok {
		return nil
	}
	if len(e.Errors) == 0 {
		return []*Error{e}
	}
	return e.Errors
}

func TestDraftMayBeIncomplete(t *testing.T) {
	f := newFixture(t)
	l := f.create(DefaultInput())
	if l.Status != StatusDraft || l.ShortCode != "" || l.Environment != EnvTest {
		t.Fatalf("draft = %+v", l)
	}
	if l.AmountMode != AmountFixed || l.FeeBearer != fees.BearerMerchant || l.CaptureMode != CaptureAutomatic {
		t.Fatalf("defaults not applied: %+v", l.Input)
	}
}

func TestPublishRefusesEveryCompletenessRule(t *testing.T) {
	cases := []struct {
		name  string
		code  Code
		edit  func(*Input)
		setup func(*fixture)
	}{
		{"no title", CodeTitleRequired, func(in *Input) { in.Title = "" }, nil},
		{"no currency", CodeCurrencyRequired, func(in *Input) { in.Currency, in.Amount = "", nil }, nil},
		{"fixed without amount", CodeAmountRequired, func(in *Input) { in.Amount = nil }, nil},
		{"line items empty", CodeLineItemsRequired, func(in *Input) { in.AmountMode, in.Amount = AmountLineItems, nil }, nil},
		{"line items sum to zero", CodeAmountInvalid, func(in *Input) {
			in.AmountMode, in.Amount = AmountLineItems, nil
			in.LineItems = []LineItem{{Name: "free", Quantity: 2, UnitPrice: dec("0")}}
		}, nil},
		{"no methods", CodeMethodsRequired, func(in *Input) { in.Methods = nil }, nil},
		{"message without text", CodeSuccessMessageRequired, func(in *Input) { in.SuccessMessage = " " }, nil},
		{"redirect without url", CodeSuccessURLRequired, func(in *Input) { in.SuccessMode = SuccessRedirect }, nil},
		{"expiry in the past", CodeExpiresAtInPast, func(in *Input) {
			past := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			in.ExpiresAt = &past
		}, nil},
		{"manual capture without card", CodeCaptureModeNeedsCard, func(in *Input) {
			in.Methods, in.CaptureMode = []MethodSpec{upi}, CaptureManual
		}, nil},
		{"hold in asset without crypto", CodeHoldInAssetNeedsCrypto, func(in *Input) { in.HoldInAsset = true }, nil},
		{"unverified destination", CodeDestinationUnverified, func(in *Input) {
			in.SettlementOverride = &SettlementOverride{Kind: SettleFiat, DestinationID: "dest_pending"}
		}, nil},
		{"method without connector", CodeMethodNoConnector, func(in *Input) {
			in.Methods = []MethodSpec{{Method: fees.MethodBank}}
		}, nil},
		{"method without fee rule", CodeMethodNoFeeRule, func(in *Input) {
			in.Methods = []MethodSpec{{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USDT"}}
		}, func(f *fixture) {
			f.creator.offer(MethodSpec{Method: fees.MethodCrypto, Chain: "SOL", Asset: "USDT"}, "")
		}},
		{"surcharge forbidden on upi", CodeSurchargeForbidden, func(in *Input) {
			in.Methods, in.FeeBearer = []MethodSpec{upi}, fees.BearerCustomer
		}, nil},
		{"surcharge across currencies", CodeSurchargeNeedsQuote, func(in *Input) {
			in.Methods, in.FeeBearer = []MethodSpec{usdcSol}, fees.BearerCustomer
		}, nil},
		{"fee larger than amount", CodeFeeExceedsAmount, func(in *Input) { in.Amount = decp("0.01") }, func(f *fixture) {
			f.fees.add(fees.MethodCard, "USD", "", "200", fees.BearerMerchant)
		}},
		{"fee larger than the customer minimum", CodeFeeExceedsAmount, func(in *Input) {
			in.AmountMode, in.Amount, in.AmountMin = AmountCustomer, nil, decp("0.01")
		}, func(f *fixture) { f.fees.add(fees.MethodCard, "USD", "", "200", fees.BearerMerchant) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.setup != nil {
				tc.setup(f)
			}
			in := validInput()
			tc.edit(&in)
			l := f.create(in)
			_, err := f.svc.Publish(context.Background(), merchant.PlatformID, l.ID)
			if !hasCode(err, tc.code) {
				t.Fatalf("codes %v, want %s (%v)", Codes(err), tc.code, err)
			}
			got, _ := f.svc.Get(context.Background(), merchant.PlatformID, l.ID)
			if got.Status != StatusDraft || got.ShortCode != "" {
				t.Fatalf("refused publish changed the link: %s %q", got.Status, got.ShortCode)
			}
		})
	}
}

func TestPublishReportsEveryFailureAtOnce(t *testing.T) {
	f := newFixture(t)
	l := f.create(DefaultInput())
	_, err := f.svc.Publish(context.Background(), merchant.PlatformID, l.ID)
	for _, c := range []Code{CodeTitleRequired, CodeCurrencyRequired, CodeAmountRequired, CodeMethodsRequired, CodeSuccessMessageRequired} {
		if !hasCode(err, c) {
			t.Errorf("missing %s in %v", c, Codes(err))
		}
	}
}

func TestPublishPicksAConnectorScopedRule(t *testing.T) {
	f := newFixture(t)
	bank := MethodSpec{Method: fees.MethodBank}
	f.creator.offer(bank, "wise", "achco")
	f.fees.add(fees.MethodBank, "USD", "achco", "0.5", fees.BearerMerchant)
	in := validInput()
	in.Methods = []MethodSpec{bank}
	if l := f.published(in); l.Status != StatusActive {
		t.Fatalf("status %s", l.Status)
	}
}

func TestVerifiedDestinationAndCryptoOptionsPublish(t *testing.T) {
	f := newFixture(t)
	in := validInput()
	in.Methods = []MethodSpec{card, usdcSol}
	in.HoldInAsset = true
	in.CaptureMode = CaptureManual
	in.SettlementOverride = &SettlementOverride{Kind: SettleCrypto, DestinationID: "dest_ok", Chain: "sol", Asset: "usdc"}
	l := f.published(in)
	if l.SettlementOverride.Chain != "SOL" || l.Methods[1] != usdcSol {
		t.Fatalf("codes not normalised: %+v %+v", l.SettlementOverride, l.Methods)
	}
}
