package modules

import (
	"fmt"
	"time"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/links"
)

// LinksModule is the wired payment links module (internal/links/README.md).
type LinksModule struct {
	Port        links.Port
	Service     *links.Service
	Environment links.Environment
}

// WireLinks needs the fee port, a PaymentCreator and the environment module; links take the process environment.
func WireLinks(deps Deps) (*LinksModule, error) {
	if deps.DB == nil || deps.Config == nil || deps.FeePort == nil || deps.LinkPayments == nil || deps.Environment == nil || deps.Environment.Guard == nil {
		return nil, fmt.Errorf("modules: links needs DB, Config, FeePort, LinkPayments and Environment")
	}
	precision, err := fees.ParsePrecision(deps.Config.Fees.AssetPrecision)
	if err != nil {
		return nil, err
	}
	lc := deps.Config.Links
	if lc == (config.LinksConfig{}) {
		lc = config.LinksConfig{LeaseSeconds: int(links.DefaultLease / time.Second), MaxOpenPayments: links.DefaultLimits.MaxOpen, MaxOpenPaymentsPerClient: links.DefaultLimits.MaxOpenPerClient}
	}
	if lc.LeaseSeconds < 60 || lc.MaxOpenPayments < 0 || lc.MaxOpenPaymentsPerClient < 0 {
		return nil, fmt.Errorf("modules: LINKS_LEASE_SECONDS must be at least 60 and the LINKS_MAX_OPEN_PAYMENTS caps at least 0")
	}
	svc := links.NewService(links.NewPGStore(deps.DB), deps.FeePort, deps.LinkPayments,
		links.WithPrecision(precision), links.WithGuard(deps.Environment.Guard),
		links.WithLease(time.Duration(lc.LeaseSeconds)*time.Second),
		links.WithLimits(links.ReserveLimits{MaxOpen: lc.MaxOpenPayments, MaxOpenPerClient: lc.MaxOpenPaymentsPerClient}),
		links.WithCheckoutBaseURL(deps.Config.Server.CheckoutBaseURL))
	return &LinksModule{Port: svc, Service: svc, Environment: deps.Environment.Guard.Current()}, nil
}
