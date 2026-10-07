package fees

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func sp(s string) *string { return &s }

func ct(c CardType) *CardType { return &c }

var t0 = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func rule(id uint, connector *string, card *CardType, region *string) Rule {
	return Rule{
		ID:            id,
		Version:       1,
		Scope:         Scope{Method: MethodCard, Connector: connector, CardType: card, Region: region, Currency: "USD"},
		FeeBearer:     BearerMerchant,
		EffectiveFrom: t0.Add(-time.Hour),
	}
}

func cardQuery() Query {
	return Query{Method: MethodCard, Connector: "stripe", CardType: CardCredit, Region: "IN", Currency: "USD", At: t0}
}

func TestResolveSpecificityOrder(t *testing.T) {
	methodDefault := rule(1, nil, nil, nil)
	connector := rule(2, sp("stripe"), nil, nil)
	connectorCard := rule(3, sp("stripe"), ct(CardCredit), nil)
	full := rule(4, sp("stripe"), ct(CardCredit), sp("IN"))
	all := []Rule{methodDefault, connector, connectorCard, full}

	steps := []struct {
		name   string
		rules  []Rule
		wantID uint
	}{
		{"connector+card_type+region wins over everything", all, 4},
		{"connector+card_type wins without region rule", []Rule{methodDefault, connector, connectorCard}, 3},
		{"connector wins over method default", []Rule{methodDefault, connector}, 2},
		{"method default is the fallback", []Rule{methodDefault}, 1},
	}
	for _, s := range steps {
		got, err := resolve(s.rules, cardQuery())
		if err != nil || got.ID != s.wantID {
			t.Errorf("%s: got rule %d err %v, want %d", s.name, got.ID, err, s.wantID)
		}
	}
	// Input order must not matter.
	reversed := slices.Clone(all)
	slices.Reverse(reversed)
	if got, err := resolve(reversed, cardQuery()); err != nil || got.ID != 4 {
		t.Errorf("reversed input: got %d err %v", got.ID, err)
	}
}

func TestResolveSkipsRulesThatDoNotMatch(t *testing.T) {
	rules := []Rule{
		rule(1, nil, nil, nil),
		rule(2, sp("adyen"), nil, nil),                  // other connector
		rule(3, sp("stripe"), ct(CardDebit), nil),       // other card type
		rule(4, sp("stripe"), ct(CardCredit), sp("US")), // other region
		{ID: 5, Scope: Scope{Method: MethodUPI, Currency: "USD"}, EffectiveFrom: t0.Add(-time.Hour)},  // other method
		{ID: 6, Scope: Scope{Method: MethodCard, Currency: "EUR"}, EffectiveFrom: t0.Add(-time.Hour)}, // other currency
	}
	got, err := resolve(rules, cardQuery())
	if err != nil || got.ID != 1 {
		t.Fatalf("got %d err %v, want the method default", got.ID, err)
	}
}

func TestResolveTieIsATypedConfigurationError(t *testing.T) {
	rules := []Rule{rule(1, nil, nil, nil), rule(9, sp("stripe"), nil, nil), rule(5, sp("stripe"), nil, nil)}
	_, err := resolve(rules, cardQuery())
	var amb *AmbiguousRuleError
	if !errors.As(err, &amb) {
		t.Fatalf("err = %v, want *AmbiguousRuleError", err)
	}
	if amb.Specificity != SpecConnector || !slices.Equal(amb.RuleIDs, []uint{5, 9}) {
		t.Errorf("ambiguity = %+v", amb)
	}
}

func TestResolveTieBelowTheWinnerIsNotAnError(t *testing.T) {
	rules := []Rule{rule(1, nil, nil, nil), rule(2, nil, nil, nil), rule(3, sp("stripe"), nil, nil)}
	got, err := resolve(rules, cardQuery())
	if err != nil || got.ID != 3 {
		t.Fatalf("got %d err %v, want 3", got.ID, err)
	}
}

func TestResolveEffectiveWindows(t *testing.T) {
	expired := rule(1, sp("stripe"), nil, nil)
	expired.EffectiveTo = ptrTime(t0) // [from, to) excludes to itself
	future := rule(2, sp("stripe"), ct(CardCredit), nil)
	future.EffectiveFrom = t0.Add(time.Second)
	current := rule(3, nil, nil, nil)
	startsNow := rule(4, sp("stripe"), nil, nil)
	startsNow.EffectiveFrom = t0 // from is inclusive

	got, err := resolve([]Rule{expired, future, current}, cardQuery())
	if err != nil || got.ID != 3 {
		t.Fatalf("got %d err %v, want 3 (expired and future rules ignored)", got.ID, err)
	}
	got, err = resolve([]Rule{expired, future, current, startsNow}, cardQuery())
	if err != nil || got.ID != 4 {
		t.Fatalf("got %d err %v, want 4", got.ID, err)
	}
	q := cardQuery()
	q.At = t0.Add(time.Second)
	got, err = resolve([]Rule{expired, future, current}, q)
	if err != nil || got.ID != 2 {
		t.Fatalf("at the future rule's start: got %d err %v, want 2", got.ID, err)
	}
}

func TestResolveNoMatch(t *testing.T) {
	_, err := resolve([]Rule{rule(1, sp("adyen"), nil, nil)}, cardQuery())
	if !errors.Is(err, ErrNoRule) {
		t.Fatalf("err = %v, want ErrNoRule", err)
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
