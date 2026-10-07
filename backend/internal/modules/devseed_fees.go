package modules

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/shopspring/decimal"
)

// ErrDevSeedLive is returned when the development fee seed is asked to run outside the test environment.
var ErrDevSeedLive = errors.New("modules: development fee rules are seeded in the test environment only")

// DevFeeCurrency is the currency a development default rule prices for each fees method.
func DevFeeCurrency(method fees.Method) string {
	if method == fees.MethodCrypto {
		return "USDC"
	}
	return "USD"
}

// SeedDevelopmentFeeRules creates one zero-percent, merchant-borne default rule per method that has no active
// rule yet, so a fresh development install can confirm a payment. It runs only for the test environment and only
// from cmd/devseed; migrations never seed fee rules.
func SeedDevelopmentFeeRules(ctx context.Context, env environment.Environment, port fees.Port, methods []fees.Method) ([]fees.Rule, error) {
	if env != environment.Test {
		return nil, fmt.Errorf("%w: environment is %s", ErrDevSeedLive, env)
	}
	var created []fees.Rule
	for _, method := range methods {
		currency := DevFeeCurrency(method)
		existing, err := port.ListRules(ctx, fees.RuleFilter{Method: method, Currency: currency})
		if err != nil {
			return created, fmt.Errorf("list %s rules: %w", method, err)
		}
		active := false
		for _, r := range existing {
			if r.EffectiveTo == nil {
				active = true
				break
			}
		}
		if active {
			continue
		}
		rule, err := port.CreateRule(ctx, fees.RuleInput{
			Scope:   fees.Scope{Method: method, Currency: currency},
			Pricing: fees.Pricing{Percent: decimal.Zero, Flat: decimal.Zero, FeeBearer: fees.BearerMerchant},
		}, "devseed")
		if err != nil {
			return created, fmt.Errorf("create %s default rule: %w", method, err)
		}
		created = append(created, rule)
	}
	return created, nil
}

// DevFeeMethods maps the enabled connector codes onto the fees methods a development install must price.
func DevFeeMethods(connectorCodes []string) []fees.Method {
	seen := map[fees.Method]bool{}
	var out []fees.Method
	add := func(m fees.Method) {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	for _, code := range connectorCodes {
		switch code {
		case "mock":
			add(fees.MethodCard)
			add(fees.MethodBank)
		case "chaindeposit":
			add(fees.MethodCrypto)
		}
	}
	return out
}
