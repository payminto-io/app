package modules

import (
	"context"
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
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

func linksDeps(env string, fc config.FeesConfig) Deps {
	return Deps{
		DB: &gorm.DB{}, Config: &config.Config{Server: config.ServerConfig{Environment: env}, Fees: fc},
		Fees: fees.NewService(&gorm.DB{}, nil, fees.DefaultPolicy()), LinkPayments: nopCreator{},
	}
}

func TestWireLinksTagsTheEnvironment(t *testing.T) {
	for env, want := range map[string]links.Environment{
		config.EnvironmentDevelopment: links.EnvTest, config.EnvironmentTest: links.EnvTest,
		config.EnvironmentStaging: links.EnvTest, config.EnvironmentProduction: links.EnvLive,
	} {
		m, err := WireLinks(linksDeps(env, config.FeesConfig{}))
		if err != nil || m.Port == nil || m.Environment != want {
			t.Errorf("%s: %+v %v, want %s", env, m, err, want)
		}
	}
}

func TestWireLinksRefusesMissingDepsAndBadPrecision(t *testing.T) {
	d := linksDeps(config.EnvironmentDevelopment, config.FeesConfig{})
	d.LinkPayments = nil
	if _, err := WireLinks(d); err == nil {
		t.Fatal("wired without a payment creator")
	}
	d = linksDeps(config.EnvironmentDevelopment, config.FeesConfig{})
	d.Fees = nil
	if _, err := WireLinks(d); err == nil {
		t.Fatal("wired without fees")
	}
	if _, err := WireLinks(linksDeps(config.EnvironmentDevelopment, config.FeesConfig{AssetPrecision: "XRP:x"})); err == nil {
		t.Fatal("wired with bad precision")
	}
}
