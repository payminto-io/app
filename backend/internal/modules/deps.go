// Package modules wires each capability module from configuration (docs/architecture/MODULES.md).
package modules

import (
	"github.com/payminto/payminto/backend/internal/config"
	"gorm.io/gorm"
)

// Deps is what every Wire<Module> function may draw on; a module uses only the fields it needs.
type Deps struct {
	Config *config.Config
	DB     *gorm.DB
}
