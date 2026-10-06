package quote

import (
	"context"

	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
)

type AssetCatalog interface {
	Resolve(context.Context, paymentlifecycle.TenantID, paymentlifecycle.ChainID, paymentlifecycle.AssetID) (Asset, error)
}

type PriceSource interface {
	Price(context.Context, string, paymentlifecycle.ChainID, paymentlifecycle.AssetID) (Price, error)
}
