package quote

import (
	"errors"
	"time"

	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
)

// Asset is one Tenant's capability to accept an exact chain-scoped asset.
type Asset struct {
	TenantID paymentlifecycle.TenantID
	ChainID  paymentlifecycle.ChainID
	AssetID  paymentlifecycle.AssetID
	Decimals uint8
	Enabled  bool
}

// Price is an exact rational number of Invoice minor units per Asset atomic
// unit. For example, 3/20000 means one atomic unit is worth 0.00015 minor units.
type Price struct {
	Currency                       string
	ChainID                        paymentlifecycle.ChainID
	AssetID                        paymentlifecycle.AssetID
	MinorUnitsPerAtomicNumerator   string
	MinorUnitsPerAtomicDenominator string
	Source                         string
	ObservedAt                     time.Time
}

type Config struct {
	Now         func() time.Time
	QuoteTTL    time.Duration
	MaxPriceAge time.Duration
}

var (
	ErrAssetUnknown     = errors.New("quote asset unknown")
	ErrAssetDisabled    = errors.New("quote asset disabled")
	ErrPriceUnavailable = errors.New("quote price unavailable")
)
