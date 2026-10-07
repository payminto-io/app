// Package modules wires each capability module from shared dependencies; see docs/architecture/MODULES.md.
// It must not import internal/service: service/registry.go imports this package.
package modules

import (
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/links"
	"gorm.io/gorm"
)

// Deps is what every Wire<Module> may draw on.
type Deps struct {
	DB     *gorm.DB
	Config *config.Config
	Ledger *ledger.Service
	// LedgerAsset is the ledger's asset for a blockchain_currencies row (service.LedgerAssetResolver).
	LedgerAsset func(tx *gorm.DB, blockchainCurrencyID uint) (string, error)
	// Fees is the wired fee port, for modules that price (links).
	Fees fees.Port
	// LinkPayments turns a paid link into a payment (service.LinkPaymentCreator until the switch).
	LinkPayments links.PaymentCreator
}
