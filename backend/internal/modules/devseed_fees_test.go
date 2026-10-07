package modules

import (
	"context"
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/fees"
)

type fakeFeesPort struct {
	fees.Port
	rules   []fees.Rule
	created []fees.RuleInput
}

func (f *fakeFeesPort) ListRules(_ context.Context, filter fees.RuleFilter) ([]fees.Rule, error) {
	var out []fees.Rule
	for _, r := range f.rules {
		if r.Method == filter.Method && r.Currency == filter.Currency {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeFeesPort) CreateRule(_ context.Context, in fees.RuleInput, _ string) (fees.Rule, error) {
	f.created = append(f.created, in)
	r := fees.Rule{ID: uint(len(f.created)), Scope: in.Scope, Percent: in.Percent, Flat: in.Flat, FeeBearer: in.FeeBearer}
	f.rules = append(f.rules, r)
	return r, nil
}

func TestSeedDevelopmentFeeRules_TestOnlyZeroPercentOncePerMethod(t *testing.T) {
	port := &fakeFeesPort{}
	methods := DevFeeMethods([]string{"mock", "chaindeposit", "mock"})
	created, err := SeedDevelopmentFeeRules(context.Background(), environment.Test, port, methods)
	if err != nil || len(created) != 3 {
		t.Fatalf("seed = %d rules, %v", len(created), err)
	}
	for _, in := range port.created {
		if !in.Percent.IsZero() || !in.Flat.IsZero() || in.FeeBearer != fees.BearerMerchant {
			t.Fatalf("a development default must be a zero merchant-borne fee: %+v", in)
		}
		if in.Currency != DevFeeCurrency(in.Method) {
			t.Fatalf("currency = %s for %s", in.Currency, in.Method)
		}
	}
	again, err := SeedDevelopmentFeeRules(context.Background(), environment.Test, port, methods)
	if err != nil || len(again) != 0 || len(port.created) != 3 {
		t.Fatalf("seeding twice must create nothing: %d, %v (total %d)", len(again), err, len(port.created))
	}
}

func TestSeedDevelopmentFeeRules_LiveNeverGetsIt(t *testing.T) {
	port := &fakeFeesPort{}
	_, err := SeedDevelopmentFeeRules(context.Background(), environment.Live, port, DevFeeMethods([]string{"mock"}))
	if !errors.Is(err, ErrDevSeedLive) || len(port.created) != 0 {
		t.Fatalf("live seed err = %v, created = %d", err, len(port.created))
	}
	if _, err := SeedDevelopmentFeeRules(context.Background(), "", port, nil); !errors.Is(err, ErrDevSeedLive) {
		t.Fatalf("unknown environment err = %v", err)
	}
}
