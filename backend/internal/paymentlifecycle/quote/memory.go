package quote

import (
	"context"
	"sync"

	"github.com/payminto/payminto/backend/internal/paymentlifecycle"
)

type MemoryAssetCatalog struct {
	mu     sync.RWMutex
	assets map[string]Asset
}

var _ AssetCatalog = (*MemoryAssetCatalog)(nil)

func NewMemoryAssetCatalog(assets []Asset) *MemoryAssetCatalog {
	catalog := &MemoryAssetCatalog{assets: make(map[string]Asset, len(assets))}
	for _, asset := range assets {
		catalog.assets[assetKey(asset.TenantID, asset.ChainID, asset.AssetID)] = asset
	}
	return catalog
}

func (c *MemoryAssetCatalog) Resolve(ctx context.Context, tenantID paymentlifecycle.TenantID, chainID paymentlifecycle.ChainID, assetID paymentlifecycle.AssetID) (Asset, error) {
	if err := ctx.Err(); err != nil {
		return Asset{}, err
	}
	c.mu.RLock()
	asset, ok := c.assets[assetKey(tenantID, chainID, assetID)]
	c.mu.RUnlock()
	if !ok {
		return Asset{}, ErrAssetUnknown
	}
	if !asset.Enabled {
		return Asset{}, ErrAssetDisabled
	}
	return asset, nil
}

func assetKey(tenantID paymentlifecycle.TenantID, chainID paymentlifecycle.ChainID, assetID paymentlifecycle.AssetID) string {
	return string(tenantID) + "\x00" + string(chainID) + "\x00" + string(assetID)
}

type MemoryPriceSource struct {
	mu     sync.RWMutex
	prices map[string]Price
}

var _ PriceSource = (*MemoryPriceSource)(nil)

func NewMemoryPriceSource(prices []Price) *MemoryPriceSource {
	source := &MemoryPriceSource{prices: make(map[string]Price, len(prices))}
	for _, price := range prices {
		source.prices[priceKey(price.Currency, price.ChainID, price.AssetID)] = price
	}
	return source
}

func (s *MemoryPriceSource) Price(ctx context.Context, currency string, chainID paymentlifecycle.ChainID, assetID paymentlifecycle.AssetID) (Price, error) {
	if err := ctx.Err(); err != nil {
		return Price{}, err
	}
	s.mu.RLock()
	price, ok := s.prices[priceKey(currency, chainID, assetID)]
	s.mu.RUnlock()
	if !ok {
		return Price{}, ErrPriceUnavailable
	}
	return price, nil
}

func priceKey(currency string, chainID paymentlifecycle.ChainID, assetID paymentlifecycle.AssetID) string {
	return currency + "\x00" + string(chainID) + "\x00" + string(assetID)
}
