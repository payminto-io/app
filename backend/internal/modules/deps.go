// Package modules wires each capability module from shared dependencies; see docs/architecture/MODULES.md.
// It must not import internal/service: service/registry.go imports this package.
package modules

import (
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/fees"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"gorm.io/gorm"
)

// Deps is what every Wire<Module> may draw on.
type Deps struct {
	DB     *gorm.DB
	Config *config.Config
	Ledger *ledger.Service
	// LedgerAsset is the ledger's asset for a blockchain_currencies row (service.LedgerAssetResolver).
	LedgerAsset func(tx *gorm.DB, blockchainCurrencyID uint) (string, error)
	// FeePort is the wired fee port, for modules that quote fees (links).
	FeePort fees.Port
	// LinkPayments turns a paid link into a payment (service.LinkPaymentCreator until the switch).
	LinkPayments links.PaymentCreator

	// Payment switch (WirePaymentSwitch). ChainDeposit is the Payminto deposit flow the chaindeposit connector
	// drives, nil disables that connector; Events receives domain events, nil drops them; Fees prices every attempt,
	// nil is refused in deployment environments; Records opens the payment record fees prices against, nil means
	// PaymintoPaymentRecords on DB.
	ChainDeposit chaindeposit.Backend
	Events       paymentswitch.Events
	Fees         paymentswitch.Fees
	Records      paymentswitch.PaymentRecords
	// Environment is the process environment module (ticket 13); the switch's guard and the connectors slot check.
	Environment *EnvironmentModule
}
