package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newWithdrawalHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Withdrawal{},
		&models.Withdraw{},
		&models.OTP{},
		&models.EEEvent{},
		&models.Blockchain{},
		&models.BlockchainCurrency{},
		&models.ExternalPlatform{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Seed the ETH/USDT pair the withdrawal flow validates against, with
	// withdrawal enabled and a 24h limit high enough that the happy-path test
	// (50) passes while the high-value test (above the OTP threshold) still
	// prompts for OTP.
	withdrawFee := decimal.NewFromFloat(1)
	minWithdraw := decimal.NewFromFloat(1)
	limit24 := decimal.NewFromInt(1000000)
	if err := db.Create(&models.BlockchainCurrency{
		CurrencyCode:      "USDT",
		BlockchainCode:    "ETH",
		Standard:          "ERC20",
		WithdrawFee:       &withdrawFee,
		MinWithdrawAmount: &minWithdraw,
		WithdrawLimit24hr: &limit24,
		WithdrawalEnabled: true,
		Visible:           true,
		WalletPrecision:   6,
		CurrencyID:        1,
		BlockchainID:      1,
	}).Error; err != nil {
		t.Fatalf("seed blockchain_currency: %v", err)
	}
	return db
}

func newWithdrawalHandler(db *gorm.DB) *WithdrawalHandler {
	withdrawalRepo := repository.NewWithdrawalRepository(db)
	bcCcyRepo := repository.NewBlockchainCurrencyRepository(db)
	otpRepo := repository.NewOTPRepository(db)
	eeRepo := repository.NewEEEventRepository(db)
	otpSvc := service.NewOTPService(otpRepo)
	emitter := service.NewEventEmitterService(eeRepo)
	withdrawalSvc := service.NewWithdrawalService(withdrawalRepo, bcCcyRepo, otpSvc, emitter)
	return NewWithdrawalHandler(withdrawalSvc)
}

func authInjector(memberID, platformID uint) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("memberID", memberID)
		c.Set("externalPlatformID", platformID)
		c.Next()
	}
}

func TestWithdrawalHandler_Create_HappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newWithdrawalHandlerDB(t)
	h := newWithdrawalHandler(db)

	r := gin.New()
	r.POST("/withdrawal/merchant", authInjector(1, 1), h.Create)

	body, _ := json.Marshal(map[string]string{
		"blockchainCode": "ETH",
		"currencyCode":   "USDT",
		"amount":         "50",
		"toAddress":      "0xdeadbeef",
	})
	req, _ := http.NewRequest("POST", "/withdrawal/merchant", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("create withdrawal: got %d, want 201; body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["withdrawal"]; !ok {
		t.Error("expected 'withdrawal' in response")
	}
}

func TestWithdrawalHandler_Create_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newWithdrawalHandlerDB(t)
	h := newWithdrawalHandler(db)

	r := gin.New()
	r.POST("/withdrawal/merchant", authInjector(1, 1), h.Create)

	// Missing toAddress.
	body, _ := json.Marshal(map[string]string{
		"blockchainCode": "ETH",
		"currencyCode":   "USDT",
		"amount":         "50",
	})
	req, _ := http.NewRequest("POST", "/withdrawal/merchant", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing toAddress, got %d", w.Code)
	}
}

func TestWithdrawalHandler_Create_HighAmount_OTPPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newWithdrawalHandlerDB(t)
	h := newWithdrawalHandler(db)

	r := gin.New()
	r.POST("/withdrawal/merchant", authInjector(1, 1), h.Create)

	body, _ := json.Marshal(map[string]string{
		"blockchainCode": "ETH",
		"currencyCode":   "USDT",
		"amount":         "1000",
		"toAddress":      "0xdeadbeef",
	})
	req, _ := http.NewRequest("POST", "/withdrawal/merchant", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("create high-value withdrawal: got %d, want 201; body=%s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	otp, ok := resp["otp"].(map[string]any)
	if !ok {
		t.Fatal("expected 'otp' object in response")
	}
	if otpRequired, _ := otp["otpRequired"].(bool); !otpRequired {
		t.Error("expected otpRequired=true for high-value withdrawal")
	}
}

func TestWithdrawalHandler_GetByID_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newWithdrawalHandlerDB(t)
	h := newWithdrawalHandler(db)

	r := gin.New()
	r.GET("/withdrawal/:id", authInjector(1, 1), h.GetByID)

	req, _ := http.NewRequest("GET", "/withdrawal/9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing withdrawal, got %d", w.Code)
	}
}

func seedWithdrawalForTenant(t *testing.T, db *gorm.DB, platformID uint, state string) models.Withdrawal {
	t.Helper()
	w := models.Withdrawal{
		ReferenceID:          "wd-tenant-scope-" + string(rune('0'+platformID)),
		State:                state,
		BlockchainCode:       "ETH",
		CurrencyCode:         "USDT",
		Amount:               decimal.NewFromInt(5),
		ToAddress:            "0xdeadbeef",
		MemberID:             platformID,
		ExternalPlatformID:   platformID,
		BlockchainCurrencyID: 1,
	}
	if err := db.Create(&w).Error; err != nil {
		t.Fatalf("seed withdrawal: %v", err)
	}
	return w
}

func TestWithdrawalHandler_TenantCannotGetApproveOrCancelAnotherTenantWithdrawal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		method string
		path   func(uint) string
		state  string
		route  func(*gin.Engine, *WithdrawalHandler)
	}{
		{
			name: "get", method: http.MethodGet,
			path:  func(id uint) string { return "/withdrawal/" + strconv.FormatUint(uint64(id), 10) },
			state: models.WithdrawalStatePendingApproval,
			route: func(r *gin.Engine, h *WithdrawalHandler) { r.GET("/withdrawal/:id", authInjector(2, 2), h.GetByID) },
		},
		{
			name: "approve", method: http.MethodPost,
			path:  func(id uint) string { return "/withdrawal/" + strconv.FormatUint(uint64(id), 10) + "/approve" },
			state: models.WithdrawalStatePendingApproval,
			route: func(r *gin.Engine, h *WithdrawalHandler) {
				r.POST("/withdrawal/:id/approve", authInjector(2, 2), h.Approve)
			},
		},
		{
			name: "cancel", method: http.MethodPost,
			path:  func(id uint) string { return "/withdrawal/" + strconv.FormatUint(uint64(id), 10) + "/cancel" },
			state: models.WithdrawalStatePendingApproval,
			route: func(r *gin.Engine, h *WithdrawalHandler) {
				r.POST("/withdrawal/:id/cancel", authInjector(2, 2), h.Cancel)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newWithdrawalHandlerDB(t)
			h := newWithdrawalHandler(db)
			ownedByTenantOne := seedWithdrawalForTenant(t, db, 1, tc.state)
			r := gin.New()
			tc.route(r, h)

			req := httptest.NewRequest(tc.method, tc.path(ownedByTenantOne.ID), nil)
			response := httptest.NewRecorder()
			r.ServeHTTP(response, req)

			if response.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404; body=%s", response.Code, response.Body.String())
			}
			var stored models.Withdrawal
			if err := db.First(&stored, ownedByTenantOne.ID).Error; err != nil {
				t.Fatalf("reload withdrawal: %v", err)
			}
			if stored.State != tc.state {
				t.Fatalf("cross-tenant %s changed state to %q", tc.name, stored.State)
			}
		})
	}
}
