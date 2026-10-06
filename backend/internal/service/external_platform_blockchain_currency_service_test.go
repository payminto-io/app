package service

import (
	"errors"
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupEPBCDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&models.ExternalPlatformBlockchainCurrency{},
		&models.ExternalPlatform{},
		&models.BlockchainCurrency{},
		&models.Currency{},
		&models.Blockchain{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func newEPBCService(t *testing.T) *ExternalPlatformBlockchainCurrencyService {
	t.Helper()
	return NewExternalPlatformBlockchainCurrencyService(
		repository.NewExternalPlatformBlockchainCurrencyRepository(setupEPBCDB(t)),
	)
}

func TestEPBCService_EnableThenGetByPair(t *testing.T) {
	svc := newEPBCService(t)
	max := decimal.NewFromInt(1000)
	min := decimal.NewFromInt(10)

	row, err := svc.Enable(1, 2, EPBCConfig{
		MinAmount: &min,
		MaxAmount: &max,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !row.Enabled {
		t.Error("expected Enabled=true")
	}

	got, err := svc.GetByPair(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.MaxAmount == nil || !got.MaxAmount.Equal(max) {
		t.Errorf("retrieved row mismatch: %+v", got)
	}
}

func TestEPBCService_GetByPair_Missing_ReturnsErrCurrencyNotEnabled(t *testing.T) {
	svc := newEPBCService(t)
	_, err := svc.GetByPair(99, 99)
	if !errors.Is(err, ErrCurrencyNotEnabledForPlatform) {
		t.Errorf("expected ErrCurrencyNotEnabledForPlatform, got %v", err)
	}
}

func TestEPBCService_Disable(t *testing.T) {
	svc := newEPBCService(t)
	_, _ = svc.Enable(1, 2, EPBCConfig{})

	if err := svc.Disable(1, 2); err != nil {
		t.Fatal(err)
	}

	row, _ := svc.GetByPair(1, 2)
	if row.Enabled {
		t.Error("expected Enabled=false after Disable")
	}
}

func TestEPBCService_UpdateLimits_PreservesEnabled(t *testing.T) {
	svc := newEPBCService(t)
	_, _ = svc.Enable(1, 2, EPBCConfig{})

	cap := decimal.NewFromInt(500)
	updated, err := svc.UpdateLimits(1, 2, EPBCConfig{HourlyCap: &cap})
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Enabled {
		t.Error("UpdateLimits must preserve Enabled state")
	}
	if updated.HourlyCap == nil || !updated.HourlyCap.Equal(cap) {
		t.Errorf("expected hourly cap %s, got %v", cap, updated.HourlyCap)
	}
}

func TestEPBCService_ListByPlatform(t *testing.T) {
	svc := newEPBCService(t)
	_, _ = svc.Enable(1, 100, EPBCConfig{})
	_, _ = svc.Enable(1, 200, EPBCConfig{})
	_, _ = svc.Enable(2, 100, EPBCConfig{})

	rows, err := svc.ListByPlatform(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows for platform 1, got %d", len(rows))
	}
}
