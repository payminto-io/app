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
}
