package modules

import (
	"context"
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
	"gorm.io/gorm"
)

type nopCreator struct{}

func (nopCreator) Connectors(context.Context, links.Environment, string, links.MethodSpec) ([]string, error) {
	return nil, nil
}
func (nopCreator) CreatePayment(context.Context, links.PaymentRequest) (links.CreatedPayment, error) {
	return links.CreatedPayment{}, nil
}
func (nopCreator) FencePayment(context.Context, links.PaymentRequest) (links.CreatedPayment, bool, error) {
	return links.CreatedPayment{}, false, nil
}
func (nopCreator) CancelPayment(context.Context, string) error { return nil }
func (nopCreator) OpenPayments(context.Context, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

func linksDeps(t *testing.T, server string, env environment.Environment, fc config.FeesConfig) Deps {
	t.Helper()
	guard, err := environment.NewGuard(env)
	if err != nil {
		t.Fatal(err)
	}
	return Deps{
		DB:      &gorm.DB{},
		Config:  &config.Config{Server: config.ServerConfig{Environment: server}, Fees: fc, Links: config.LinksConfig{LeaseSeconds: 300, MaxOpenPayments: 100, MaxOpenPaymentsPerClient: 3}},
		FeePort: fees.NewService(&gorm.DB{}, nil, fees.DefaultPolicy()), LinkPayments: nopCreator{},
		Environment: &EnvironmentModule{Environment: env, Guard: guard},
	}
}

func TestWireLinksTakesTheProcessEnvironmentNotSERVER(t *testing.T) {
	for _, tc := range []struct {
		server string
		env    environment.Environment
	}{
		{config.EnvironmentStaging, environment.Live},
		{config.EnvironmentProduction, environment.Test},
		{config.EnvironmentDevelopment, environment.Test},
	} {
		m, err := WireLinks(linksDeps(t, tc.server, tc.env, config.FeesConfig{}))
		if err != nil || m.Port == nil || m.Service == nil || m.Environment != tc.env {
			t.Errorf("SERVER=%s GATEWAY_ENVIRONMENT=%s: %+v %v", tc.server, tc.env, m, err)
		}
	}
}

func TestWireLinksRefusesMissingDepsAndBadConfig(t *testing.T) {
	for name, edit := range map[string]func(*Deps){
		"no creator":     func(d *Deps) { d.LinkPayments = nil },
		"no fees":        func(d *Deps) { d.FeePort = nil },
		"no environment": func(d *Deps) { d.Environment = nil },
		"bad precision":  func(d *Deps) { d.Config.Fees.AssetPrecision = "XRP:x" },
		"short lease":    func(d *Deps) { d.Config.Links.LeaseSeconds = 10 },
		"negative cap":   func(d *Deps) { d.Config.Links.MaxOpenPayments = -1 },
	} {
		d := linksDeps(t, config.EnvironmentDevelopment, environment.Test, config.FeesConfig{})
		edit(&d)
		if _, err := WireLinks(d); err == nil {
			t.Errorf("%s: wired", name)
		}
	}
}
