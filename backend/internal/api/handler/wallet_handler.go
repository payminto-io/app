package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// coldWalletKeyPrefix is the configurations-table key prefix used to persist
// merchant-configured cold-storage destinations, one row per blockchain code.
// Example: "cold_wallet.ETH".
const coldWalletKeyPrefix = "cold_wallet."

// WalletHandler exposes merchant-facing wallet management endpoints: listing
// HD wallets, paginating their address pools, configuring cold-storage
// destinations, and registering externally-owned hot wallets per blockchain
// family. All endpoints operate on the currently-authenticated member.
type WalletHandler struct {
	walletRepo           repository.WalletRepository
	addressPoolRepo      repository.AddressPoolRepository
	blockchainFamilyRepo repository.BlockchainFamilyRepository
	blockchainRepo       repository.BlockchainRepository
	configSvc            *service.ConfigurationService
	walletSvc            *service.WalletService
	vaultSvc             *service.SecretsVaultService
	adapterReg           *blockchain.AdapterRegistry
}

// NewWalletHandler constructs a WalletHandler. All dependencies are required
// when the corresponding routes are mounted by the router; callers should nil-
// check upstream before constructing it.
func NewWalletHandler(
	walletRepo repository.WalletRepository,
	addressPoolRepo repository.AddressPoolRepository,
	blockchainFamilyRepo repository.BlockchainFamilyRepository,
	blockchainRepo repository.BlockchainRepository,
	configSvc *service.ConfigurationService,
	walletSvc *service.WalletService,
	vaultSvc *service.SecretsVaultService,
	adapterReg *blockchain.AdapterRegistry,
) *WalletHandler {
	return &WalletHandler{
		walletRepo:           walletRepo,
		addressPoolRepo:      addressPoolRepo,
		blockchainFamilyRepo: blockchainFamilyRepo,
		blockchainRepo:       blockchainRepo,
		configSvc:            configSvc,
		walletSvc:            walletSvc,
		vaultSvc:             vaultSvc,
		adapterReg:           adapterReg,
	}
}

// walletDTO is the JSON projection for listing wallets in the dashboard. It
// enriches each wallet with a live count of available addresses in the pool
// so the UI can show capacity without a second round-trip.
type walletDTO struct {
	ID                 uint                     `json:"id"`
	Name               string                   `json:"name"`
	Kind               string                   `json:"kind"`
	Status             string                   `json:"status"`
	BlockchainFamilyID uint                     `json:"blockchainFamilyID"`
	BlockchainFamily   *models.BlockchainFamily `json:"blockchainFamily,omitempty"`
	AddressCount       int64                    `json:"addressCount"`
	CreatedAt          string                   `json:"createdAt"`
}

// ListWallets handles GET /api/v1/wallets — returns every HD/hot wallet
// belonging to the authenticated member along with its current available
// address-pool count.
func (h *WalletHandler) ListWallets(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	wallets, err := h.walletRepo.ListByMember(memberID, repository.WithPreload("BlockchainFamily"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]walletDTO, 0, len(wallets))
	for i := range wallets {
		w := &wallets[i]
		count, cErr := h.addressPoolRepo.CountAvailableByWallet(w.ID)
		if cErr != nil {
			// Non-fatal: fall back to zero so the UI still renders.
			count = 0
		}
		out = append(out, walletDTO{
			ID:                 w.ID,
			Name:               w.Name,
			Kind:               w.Kind,
			Status:             w.Status,
			BlockchainFamilyID: w.BlockchainFamilyID,
			BlockchainFamily:   w.BlockchainFamily,
			AddressCount:       count,
			CreatedAt:          w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"wallets": out})
}

// addressDTO is the JSON projection for listing pooled deposit addresses.
// The encrypted private key is never returned — it's stripped by the
// json:"-" tag on the model itself, but we also project explicitly to
// avoid leaking any future fields accidentally.
type addressDTO struct {
	ID                 uint   `json:"id"`
	Address            string `json:"address"`
	PathIndex          uint   `json:"pathIndex"`
	Status             string `json:"status"`
	WalletID           uint   `json:"walletID"`
	BlockchainFamilyID uint   `json:"blockchainFamilyID"`
	CreatedAt          string `json:"createdAt"`
}

// ListWalletAddresses handles GET /api/v1/wallets/:id/addresses. Supports
// optional status filter ("available" | "used" | "locked") plus limit/offset
// pagination. The caller must own the wallet (member-scoped).
func (h *WalletHandler) ListWalletAddresses(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	walletID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || walletID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet id"})
		return
	}

	// Ownership check: reject access to wallets owned by other members.
	wallet, err := h.walletRepo.GetByID(uint(walletID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if wallet.MemberID != memberID {
		c.JSON(http.StatusNotFound, gin.H{"error": "wallet not found"})
		return
	}

	status := c.Query("status")
	limit := parseIntDefault(c.Query("limit"), 50)
	if limit > 500 {
		limit = 500
	}
	offset := parseIntDefault(c.Query("offset"), 0)

	var (
		rows  []models.AddressPool
		total int64
	)

	if status != "" {
		// Fetch the paginated slice from the repo in a single query.
		opts := []repository.QueryOption{
			repository.WithAscendingOrder("path_index"),
			repository.WithLimit(limit),
			repository.WithOffset(offset),
		}
		rows, err = h.addressPoolRepo.GetByWalletAndStatus(uint(walletID), status, opts...)
	} else {
		// No status filter: merge the three canonical statuses into one
		// sorted slice, then apply limit/offset in-memory. Address pools
		// per wallet are small (tens to hundreds), so the extra queries
		// are cheap and the repo interface stays tight.
		merged, mErr := h.listAllAddressesForWallet(uint(walletID))
		if mErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": mErr.Error()})
			return
		}
		rows = paginateAddresses(merged, limit, offset)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total, err = h.countAddressesForWallet(uint(walletID), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]addressDTO, 0, len(rows))
	for i := range rows {
		a := &rows[i]
		out = append(out, addressDTO{
			ID:                 a.ID,
			Address:            a.Address,
			PathIndex:          a.PathIndex,
			Status:             a.Status,
			WalletID:           a.WalletID,
			BlockchainFamilyID: a.BlockchainFamilyID,
			CreatedAt:          a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"addresses": out,
		"total":     total,
	})
}

// listAllAddressesForWallet returns every pool entry for a wallet across the
// canonical statuses, sorted by path_index ascending. Address pools per
// wallet are small enough that a single in-memory merge is cheaper than
// expanding the repository interface with another bespoke query.
func (h *WalletHandler) listAllAddressesForWallet(walletID uint) ([]models.AddressPool, error) {
	merged := make([]models.AddressPool, 0, 128)
	for _, st := range []string{"available", "used", "locked"} {
		rows, err := h.addressPoolRepo.GetByWalletAndStatus(walletID, st,
			repository.WithAscendingOrder("path_index"),
		)
		if err != nil {
			return nil, err
		}
		merged = append(merged, rows...)
	}
	// Second pass: stable-sort by path_index so the merged slice honours the
	// UI's expected ordering regardless of which status bucket a row came
	// from. Simple insertion sort is fine for the small slice sizes here.
	for i := 1; i < len(merged); i++ {
		for j := i; j > 0 && merged[j-1].PathIndex > merged[j].PathIndex; j-- {
			merged[j-1], merged[j] = merged[j], merged[j-1]
		}
	}
	return merged, nil
}

// paginateAddresses applies limit/offset in memory to a pre-sorted slice.
func paginateAddresses(in []models.AddressPool, limit, offset int) []models.AddressPool {
	if offset >= len(in) {
		return []models.AddressPool{}
	}
	end := offset + limit
	if end > len(in) {
		end = len(in)
	}
	return in[offset:end]
}

// countAddressesForWallet returns the total row count for the given wallet,
// optionally filtered by status. Used to populate the paginated response.
func (h *WalletHandler) countAddressesForWallet(walletID uint, status string) (int64, error) {
	// We leverage the one available counting method for "available" status;
	// for "used"/"locked" or "all" we approximate by listing and len()-ing.
	// This keeps the repo interface untouched while still returning a
	// correct total for UI pagination.
	if status == "available" {
		return h.addressPoolRepo.CountAvailableByWallet(walletID)
	}
	if status != "" {
		rows, err := h.addressPoolRepo.GetByWalletAndStatus(walletID, status)
		if err != nil {
			return 0, err
		}
		return int64(len(rows)), nil
	}
	var total int64
	for _, st := range []string{"available", "used", "locked"} {
		rows, err := h.addressPoolRepo.GetByWalletAndStatus(walletID, st)
		if err != nil {
			return 0, err
		}
		total += int64(len(rows))
	}
	return total, nil
}

// coldWalletEntry is the persisted JSON shape for a cold-wallet configuration
// row, stored under key "cold_wallet.{blockchainCode}" in the configurations
// table. Keeping it in a single JSON value keeps the schema free of a bespoke
// cold-wallet table while still letting us evolve fields.
type coldWalletEntry struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type configureColdWalletRequest struct {
	BlockchainCode string `json:"blockchainCode" binding:"required"`
	Address        string `json:"address" binding:"required"`
	Name           string `json:"name"`
}

// ConfigureColdWallet handles POST /api/v1/wallets/cold. It validates the
// blockchain exists and persists the cold-storage destination under a
// "cold_wallet.{code}" key in the configurations table.
func (h *WalletHandler) ConfigureColdWallet(c *gin.Context) {
	if _, ok := callerMemberID(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req configureColdWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	code := strings.ToUpper(strings.TrimSpace(req.BlockchainCode))
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "blockchainCode is required"})
		return
	}

	// Best-effort validation that the blockchain exists — skip if the
	// registry is not wired in tests.
	if h.blockchainRepo != nil {
		if _, err := h.blockchainRepo.GetByCode(code); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown blockchain code"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	entry := coldWalletEntry{Address: req.Address, Name: req.Name}
	payload, err := json.Marshal(entry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	key := coldWalletKeyPrefix + code
	if err := h.configSvc.Set(c.Request.Context(), key, string(payload), fmt.Sprintf("Cold wallet destination for %s", code)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configured":     true,
		"blockchainCode": code,
		"address":        req.Address,
	})
}

// coldWalletDTO is the wire shape of a single cold-wallet entry in the
// ListColdWallets response.
type coldWalletDTO struct {
	BlockchainCode string `json:"blockchainCode"`
	Address        string `json:"address"`
	Name           string `json:"name"`
}

// ListColdWallets handles GET /api/v1/wallets/cold. It scans the
// configurations table for entries prefixed with "cold_wallet." and
// unmarshals each row into a structured response.
func (h *WalletHandler) ListColdWallets(c *gin.Context) {
	if _, ok := callerMemberID(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	all, err := h.configSvc.ListAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]coldWalletDTO, 0)
	for i := range all {
		cfg := &all[i]
		if !strings.HasPrefix(cfg.Key, coldWalletKeyPrefix) {
			continue
		}
		code := strings.TrimPrefix(cfg.Key, coldWalletKeyPrefix)

		var entry coldWalletEntry
		if err := json.Unmarshal([]byte(cfg.Value), &entry); err != nil {
			// Tolerate legacy rows where the value is the raw address.
			entry = coldWalletEntry{Address: cfg.Value}
		}
		out = append(out, coldWalletDTO{
			BlockchainCode: code,
			Address:        entry.Address,
			Name:           entry.Name,
		})
	}

	c.JSON(http.StatusOK, gin.H{"coldWallets": out})
}

type registerHotWalletRequest struct {
	BlockchainFamilyCode string `json:"blockchainFamilyCode" binding:"required"`
	PrivateKey           string `json:"privateKey" binding:"required"`
	Address              string `json:"address" binding:"required"`
	Name                 string `json:"name"`
}

// RegisterHotWallet handles POST /api/v1/wallets/hot. It creates a Wallet row
// with kind="hot" bound to the member and target blockchain family, then
// stores the provided private key in the SecretsVault under the label
// "hot_wallet.{walletID}". The private key is never persisted in plaintext
// and is never returned by any GET endpoint.
func (h *WalletHandler) RegisterHotWallet(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	if h.vaultSvc == nil || !h.vaultSvc.IsUnlocked() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "secrets vault is locked"})
		return
	}

	var req registerHotWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	family, err := h.blockchainFamilyRepo.GetByCode(req.BlockchainFamilyCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown blockchain family code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = fmt.Sprintf("Hot Wallet (%s)", family.Code)
	}

	wallet := &models.Wallet{
		Name:               name,
		Kind:               "hot",
		Status:             "active",
		BlockchainFamilyID: family.ID,
		MemberID:           memberID,
	}
	if err := h.walletRepo.Create(wallet); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	label := fmt.Sprintf("hot_wallet.%d", wallet.ID)
	mid := memberID
	if err := h.vaultSvc.StoreKey(label, req.PrivateKey, models.SecretTypePrivateKey, &mid); err != nil {
		// Roll back the wallet row so we don't leave an orphan reference.
		_ = h.walletRepo.Delete(wallet.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("store key: %v", err)})
		return
	}

	// Persist the public address as a configuration row keyed by walletID so
	// list endpoints can surface it without a dedicated column migration.
	addrKey := fmt.Sprintf("hot_wallet_address.%d", wallet.ID)
	if err := h.configSvc.Set(c.Request.Context(), addrKey, req.Address, fmt.Sprintf("Hot wallet address for wallet %d", wallet.ID)); err != nil {
		// Non-fatal: the vault entry and wallet row are still good. Log via
		// response header so the UI can surface it without a hard failure.
		c.Header("X-Warning", "address metadata not persisted")
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                   wallet.ID,
		"kind":                 wallet.Kind,
		"address":              req.Address,
		"blockchainFamilyCode": family.Code,
	})
}

// hotWalletDTO is the wire shape for the ListHotWallets response. The private
// key is never included.
type hotWalletDTO struct {
	ID                   uint   `json:"id"`
	Name                 string `json:"name"`
	Kind                 string `json:"kind"`
	Status               string `json:"status"`
	BlockchainFamilyID   uint   `json:"blockchainFamilyID"`
	BlockchainFamilyCode string `json:"blockchainFamilyCode,omitempty"`
	Address              string `json:"address,omitempty"`
	CreatedAt            string `json:"createdAt"`
}

// ListHotWallets handles GET /api/v1/wallets/hot. It returns every wallet row
// for the authenticated member with kind="hot", enriched with the public
// address pulled from the configurations table. Private keys are never
// returned — they live only in the SecretsVault.
func (h *WalletHandler) ListHotWallets(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	wallets, err := h.walletRepo.ListByMember(memberID, repository.WithPreload("BlockchainFamily"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]hotWalletDTO, 0)
	for i := range wallets {
		w := &wallets[i]
		if w.Kind != "hot" {
			continue
		}

		dto := hotWalletDTO{
			ID:                 w.ID,
			Name:               w.Name,
			Kind:               w.Kind,
			Status:             w.Status,
			BlockchainFamilyID: w.BlockchainFamilyID,
			CreatedAt:          w.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
		if w.BlockchainFamily != nil {
			dto.BlockchainFamilyCode = w.BlockchainFamily.Code
		}

		// Best-effort address lookup — tolerate missing config rows.
		addrKey := fmt.Sprintf("hot_wallet_address.%d", w.ID)
		if cfg, err := h.configSvc.Get(c.Request.Context(), addrKey); err == nil && cfg != nil {
			dto.Address = cfg.Value
		}

		out = append(out, dto)
	}

	c.JSON(http.StatusOK, gin.H{"hotWallets": out})
}

// nativeChainForFamily maps a blockchain family to its native chain code,
// decimals, and symbol for balance display.
func nativeChainForFamily(familyCode string) (chain string, decimals int32, symbol string, ok bool) {
	switch familyCode {
	case "ETH_Family":
		return "ETH", 18, "ETH", true
	case "BTC_Family":
		return "BTC", 8, "BTC", true
	case "TRX_Family":
		return "TRX", 6, "TRX", true
	}
	return "", 0, "", false
}

// GetHotWalletBalance handles GET /api/v1/wallets/hot/:id/balance. It returns the
// live on-chain native balance of a hot wallet's address by querying the chain
// adapter. Fetched lazily per wallet by the dashboard so the wallet list itself
// stays fast and does not depend on RPC availability.
func (h *WalletHandler) GetHotWalletBalance(c *gin.Context) {
	memberID, ok := callerMemberID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet id"})
		return
	}

	wallet, err := h.walletRepo.GetByID(uint(id))
	if err != nil || wallet.MemberID != memberID || wallet.Kind != "hot" {
		c.JSON(http.StatusNotFound, gin.H{"error": "hot wallet not found"})
		return
	}

	family, err := h.blockchainFamilyRepo.GetByID(wallet.BlockchainFamilyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load family: " + err.Error()})
		return
	}
	chain, decimals, symbol, ok := nativeChainForFamily(family.Code)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported family " + family.Code})
		return
	}

	addrKey := fmt.Sprintf("hot_wallet_address.%d", wallet.ID)
	cfg, err := h.configSvc.Get(c.Request.Context(), addrKey)
	if err != nil || cfg == nil || cfg.Value == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "hot wallet address not configured"})
		return
	}
	address := cfg.Value

	if h.adapterReg == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "blockchain adapters unavailable"})
		return
	}
	adapter, err := h.adapterReg.Get(chain)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no active adapter for " + chain})
		return
	}

	raw, err := adapter.GetBalance(c.Request.Context(), address, "")
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "balance lookup failed: " + err.Error()})
		return
	}

	balance := decimal.NewFromBigInt(raw, -decimals)
	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"chain":   chain,
		"symbol":  symbol,
		"balance": balance.String(),
	})
}

// parseIntDefault returns the parsed int from s, or def when s is empty or
// cannot be parsed.
func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}
