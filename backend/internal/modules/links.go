package modules

import (
	"fmt"
	"strings"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
)

// LinksModule is the wired payment links module (internal/links/README.md).
type LinksModule struct {
	Port        links.Port
	Environment links.Environment
}

// WireLinks needs the fee port and a PaymentCreator; links are tagged live only in PRODUCTION until ticket 13.
func WireLinks(deps Deps) (*LinksModule, error) {
	if deps.DB == nil || deps.Config == nil || deps.Fees == nil || deps.LinkPayments == nil {
		return nil, fmt.Errorf("modules: links needs DB, Config, Fees and LinkPayments")
	}
	precision, err := fees.ParsePrecision(deps.Config.Fees.AssetPrecision)
	if err != nil {
		return nil, err
	}
	env := links.EnvTest
	if strings.ToUpper(strings.TrimSpace(deps.Config.Server.Environment)) == config.EnvironmentProduction {
		env = links.EnvLive
	}
	svc := links.NewService(links.NewPGStore(deps.DB), deps.Fees, deps.LinkPayments,
		links.WithPrecision(precision), links.WithEnvironment(env),
		links.WithCheckoutBaseURL(deps.Config.Server.CheckoutBaseURL))
	return &LinksModule{Port: svc, Environment: env}, nil
}
