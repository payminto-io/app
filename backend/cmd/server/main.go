package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/payminto/payminto/backend/internal/api"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/payminto/payminto/backend/internal/observability"
	"github.com/payminto/payminto/backend/internal/realtime"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/payminto/payminto/backend/internal/worker"
)

// managerAdapter adapts worker.Manager to the service.WorkerManager interface,
// converting worker.WorkerStatus to service.WorkerStatus to avoid import cycles.
type managerAdapter struct {
	mgr *worker.Manager
}

func (a *managerAdapter) GetWorkerStatus() []service.WorkerStatus {
	raw := a.mgr.GetWorkerStatus()
	out := make([]service.WorkerStatus, len(raw))
	for i, s := range raw {
		out[i] = service.WorkerStatus{
			Name:          s.Name,
			Running:       s.Running,
			StartedAt:     s.StartedAt,
			LastHeartbeat: s.LastHeartbeat,
			Error:         s.Error,
		}
	}
	return out
}

func (a *managerAdapter) Stop(ctx context.Context) error {
	return a.mgr.Stop(ctx)
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Boot gate: live and test are isolated before a database is even opened (ticket 13).
	envModule, err := modules.WireEnvironment(modules.Deps{Config: cfg})
	if err != nil {
		log.Fatalf("environment: %v", err)
	}

	observability.Init(cfg.Server.Environment)
	observability.InitErrorReporting(cfg.Telemetry.SentryDSN, cfg.Server.Environment)
	defer observability.FlushErrors()

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if err := database.PrepareSchema(db, cfg.Server.Environment, cfg.Database.SchemaMode); err != nil {
		log.Fatalf("schema startup: %v", err)
	}
	if err := envModule.VerifyDatabase(context.Background(), db); err != nil {
		log.Fatalf("environment: %v", err)
	}

	reg, err := service.NewServiceRegistry(db, nil, cfg)
	if err != nil {
		log.Fatalf("service registry: %v", err)
	}

	modeRepo := service.NewConfigRepoAdapter(reg.ConfigurationRepo())
	if err := config.EnforceModeMatch(cfg.Blockchain.NetworkType, modeRepo); err != nil {
		log.Fatalf("mode enforcement: %v", err)
	}

	// Custody is an explicit capability. Config validation guarantees a strong
	// passphrase whenever it is enabled; a non-custodial deployment never
	// initializes or unlocks the vault merely because the service exists.
	if cfg.Security.CustodyEnabled {
		if err := reg.SecretsVaultService().Unlock(cfg.Security.VaultPassphrase, nil); err != nil {
			log.Fatalf("vault unlock: %v", err)
		}
		log.Println("[server] secrets vault unlocked")
	} else {
		log.Println("[server] custody disabled (CUSTODY_ENABLED=false)")
	}

	// Seed default RBAC permissions and roles (idempotent).
	ctx := context.Background()
	if err := reg.PermissionService().SeedDefaults(ctx); err != nil {
		log.Fatalf("seed permissions: %v", err)
	}
	if err := reg.RoleService().SeedDefaults(ctx); err != nil {
		log.Fatalf("seed roles: %v", err)
	}

	// Boot banner
	banner := "[TESTNET]"
	if cfg.Blockchain.NetworkType == "mainnet" {
		banner = "[MAINNET]"
	}
	log.Printf("=====================================")
	log.Printf("  Payminto %s", banner)
	log.Printf("  Network mode: %s", cfg.Blockchain.NetworkType)
	log.Printf("  Environment: %s", envModule.Environment)
	log.Printf("=====================================")

	// Real-time event broker: publishes payment-status changes to connected
	// checkout clients over SSE. Single in-process instance; swap for a
	// Redis-backed broker to scale horizontally.
	broker := realtime.NewMemoryBroker()

	// Wire the worker manager + system service.
	mgr := worker.NewManager()

	// Address derivation is a custody capability and must not run against a
	// deliberately locked vault in a non-custodial deployment.
	if cfg.Security.CustodyEnabled {
		mgr.Register(worker.NewAddressPoolWarmer(
			reg.WalletRepo(),
			reg.AddressPoolService(),
		))
	}

	// Deposit confirmation processor: promotes CONFIRMING deposits to CONFIRMED
	// and flips their payment to FILLED, publishing a real-time event.
	mgr.Register(worker.NewDepositProcessor(db, broker))

	// Native-EVM sweep: consolidates confirmed deposits to cold storage. Only
	// runs when an EVM cold wallet is configured.
	if cfg.Security.CustodyEnabled && cfg.Blockchain.ColdWalletETH != "" {
		mgr.Register(worker.NewEVMSweepWorker(reg.EVMSweepService(), reg.EVMSweepConfirmer(), 30*time.Second))
		log.Printf("[server] EVM sweep worker enabled (cold=%s)", cfg.Blockchain.ColdWalletETH)
	} else if !cfg.Security.CustodyEnabled {
		log.Printf("[server] EVM sweep worker disabled (custody disabled)")
	} else {
		log.Printf("[server] EVM sweep worker disabled (COLD_WALLET_ETH not set)")
	}

	// Register per-chain block processors for deposit detection.
	// Each active blockchain with a registered adapter gets its own processor
	// that scans blocks, detects deposits, and tracks confirmations.
	processorDeps := worker.ProcessorDeps{
		BlockchainRepo:         reg.BlockchainRepo(),
		DepositService:         reg.DepositService(),
		DepositAddressRepo:     reg.DepositAddressRepo(),
		MissedDepositRepo:      reg.MissedDepositRepo(),
		BlockchainCurrencyRepo: reg.BlockchainCurrencyRepo(),
	}
	adapterReg := reg.AdapterRegistry()
	for code, adapter := range adapterReg.All() {
		chain, err := reg.BlockchainRepo().GetByCode(code)
		if err != nil {
			log.Printf("[server] skip block processor for %s: %v", code, err)
			continue
		}
		if chain.Status != "active" {
			log.Printf("[server] skip block processor for %s: status=%s", code, chain.Status)
			continue
		}
		mgr.Register(worker.NewBlockchainProcessor(
			chain, processorDeps.BlockchainRepo, processorDeps.DepositService,
			processorDeps.DepositAddressRepo, processorDeps.MissedDepositRepo,
			processorDeps.BlockchainCurrencyRepo, adapter,
		))
		log.Printf("[server] registered block processor for %s (height=%d, confirmations=%d)",
			chain.Code, chain.Height, chain.MinConfirmations)
	}

	systemSvc := service.NewSystemService(&managerAdapter{mgr: mgr}, db)

	checkoutHost := cfg.Server.CheckoutBaseURL
	if checkoutHost == "" {
		checkoutHost = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	}
	router := api.NewRouter(api.RouterConfig{
		DB:                db,
		AuthSvc:           reg.AuthService(),
		JWTTokenSvc:       reg.JWTTokenService(),
		OTPSvc:            reg.OTPService(),
		WithdrawalSvc:     reg.WithdrawalService(),
		PaymentSvc:        reg.PaymentService(),
		DepositAddressSvc: reg.DepositAddressService(),
		Host:              checkoutHost,
		AllowedOrigins:    cfg.Server.AllowedOrigins,
		Broker:            broker,
		MetricsEnabled:    cfg.Telemetry.MetricsEnabled,
		AdapterReg:        reg.AdapterRegistry(),
		// Phase H
		MemberSvc:     reg.MemberService(),
		RoleSvc:       reg.RoleService(),
		PermissionSvc: reg.PermissionService(),
		MEPRoleSvc:    reg.MEPRoleService(),
		RecipientSvc:  reg.RecipientService(),
		ConfigSvc:     reg.ConfigurationService(),
		SystemSvc:     systemSvc,
		// Phase I
		WebhookMgmtSvc:       reg.WebhookManagementService(),
		AnalyticsSvc:         reg.AnalyticsService(),
		TickerSvc:            reg.TickerService(),
		ReferralSvc:          reg.ReferralService(),
		ReferralCampaignSvc:  reg.ReferralCampaignService(),
		AnalyticsReferralSvc: reg.AnalyticsReferralService(),
		// Phase K
		BlockchainCurRepo: reg.BlockchainCurrencyRepo(),
		// Phase J
		ExternalPlatformSvc: reg.ExternalPlatformService(),
		EPBCSvc:             reg.EPBCService(),
		MissedDepositSvc:    reg.MissedDepositService(),
		OnramperSvc:         reg.OnramperPaymentsService(),
		// Wallet management
		WalletRepo:           reg.WalletRepo(),
		AddressPoolRepo:      reg.AddressPoolRepo(),
		BlockchainFamilyRepo: reg.BlockchainFamilyRepo(),
		BlockchainRepo:       reg.BlockchainRepo(),
		WalletSvc:            reg.WalletService(),
		VaultSvc:             reg.SecretsVaultService(),
		APIKeyRepo:           reg.APIKeyRepo(),
		Environment:          reg.EnvironmentModule(),
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown: SIGINT/SIGTERM triggers 30s drain of in-flight
	// requests, then stops the worker manager.
	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start background workers.
	mgr.StartAll(context.Background())
	log.Println("[server] background workers started")

	serverErr := make(chan error, 1)
	go func() {
		observability.Logger().Info("server starting", "addr", addr, "env", cfg.Server.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("server: %v", err)
	case <-shutdownCtx.Done():
		observability.Logger().Info("shutdown signal received, draining")
	}

	drainCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(drainCtx); err != nil {
		observability.Logger().Error("server shutdown error", "err", err)
	}
	if err := mgr.Stop(drainCtx); err != nil {
		observability.Logger().Error("worker manager stop error", "err", err)
	}
	observability.Logger().Info("shutdown complete")
}
