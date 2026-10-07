// Package modules wires each capability module from shared dependencies; see docs/architecture/MODULES.md.
// It must not import internal/service: service/registry.go imports this package.
package modules

import (
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/ledger"
	"gorm.io/gorm"
)

// Deps is what every Wire<Module> may draw on.
type Deps struct {
	DB     *gorm.DB
	Config *config.Config
	Ledger *ledger.Service
	// LedgerAsset is the ledger's asset for a blockchain_currencies row (service.LedgerAssetResolver).
	LedgerAsset func(tx *gorm.DB, blockchainCurrencyID uint) (string, error)
	// Environment is the process environment module; slot modules ask its guard before resolving a provider.
	Environment *EnvironmentModule
	// EmitEvent publishes a named, versioned event through the gateway's emitter (MODULES.md rule 9).
	EmitEvent func(eventType string, payload map[string]any) error
}
