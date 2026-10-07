package service

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/payminto/payminto/backend/internal/blockchain"
	btcAdapter "github.com/payminto/payminto/backend/internal/blockchain/bitcoin"
	ethAdapter "github.com/payminto/payminto/backend/internal/blockchain/ethereum"
	tronAdapter "github.com/payminto/payminto/backend/internal/blockchain/tron"
	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/email/transport"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	jwtAccessTTL  = 24 * time.Hour
	jwtRefreshTTL = 30 * 24 * time.Hour
)

// ServiceRegistry is the central DI container. It owns every repository and
// every service, constructs them in dependency order, and exposes typed
// accessors. Mirrors PayRam's service_registry.go.
type ServiceRegistry struct {
	db    *gorm.DB
	redis *redis.Client
	cfg   *config.Config

	// network mode, read from env var BLOCKCHAIN_NETWORK_TYPE at boot
	networkType string

	// repositories
	memberRepo             repository.MemberRepository
	apiKeyRepo             repository.APIKeyRepository
	externalPlatformRepo   repository.ExternalPlatformRepository
	roleRepo               repository.RoleRepository
	permissionRepo         repository.PermissionRepository
	paymentRepo            repository.PaymentRepository
	depositRepo            repository.DepositRepository
	depositAddressRepo     repository.DepositAddressRepository
	walletRepo             repository.WalletRepository
	addressPoolRepo        repository.AddressPoolRepository
	blockchainRepo         repository.BlockchainRepository
	blockchainFamilyRepo   repository.BlockchainFamilyRepository
	currencyRepo           repository.CurrencyRepository
	blockchainCurrencyRepo repository.BlockchainCurrencyRepository
	webhookRepo            repository.WebhookRepository
	webhookDeliveryLogRepo repository.WebhookDeliveryLogRepository
	configurationRepo      repository.ConfigurationRepository

	// Phase B.3: RPCNode repository
	rpcNodeRepo repository.RPCNodeRepository

	// Phase B.2: SecretsVault repositories
	secretsVaultRepo         repository.SecretsVaultRepository
	secretsVaultActivityRepo repository.SecretsVaultActivityRepository

	// Phase D.1: Wallet extras + Address repositories
	walletXpubRepo     repository.WalletXpubRepository
	walletFunctionRepo repository.WalletFunctionRepository
	walletSCWRepo      repository.WalletSCWRepository
	addressRepo        repository.AddressRepository

	// Phase E: MissedDeposit repository
	missedDepositRepo repository.MissedDepositRepository

	// Phase F: Sweep, UTXO, InternalBlockchainTx, AddressDeployment repos
	sweepRepo             repository.SweepRepository
	sweepTxRepo           repository.SweepTransactionRepository
	utxoRepo              repository.UTXORepository
	internalBCTxRepo      repository.InternalBlockchainTransactionRepository
	addressDeploymentRepo repository.AddressDeploymentRepository
	accountRepo           repository.AccountRepository

	// Phase G: Auth refresh token, OTP, EEEvent, Withdrawal repos
	authRefreshTokenRepo repository.AuthRefreshTokenRepository
	otpRepo              repository.OTPRepository
	eeEventRepo          repository.EEEventRepository
	withdrawalRepo       repository.WithdrawalRepository
	withdrawRepo         repository.WithdrawRepository

	// blockchain adapter registry
	adapterRegistry *blockchain.AdapterRegistry

	// services (existing — Phase A.9 will refactor these to take repos)
	authService         *AuthService
	paymentService      *PaymentService
	webhookService      *WebhookService
	onrampService       *OnrampService
	secretsVaultService *SecretsVaultService
	keyResolver         *KeyResolver
	evmBroadcaster      *EVMBroadcaster
	evmSweepService     *EVMSweepService
	evmSweepConfirmer   *EVMSweepConfirmer
	hotWalletSource     *HotWalletSource

	// Phase D.2: HD wallet service
	walletService *WalletService

	// Phase D.3: AddressPoolService
	addressPoolService *AddressPoolService

	// Phase D.4: AddressService
	addressService *AddressService

	// Phase D.5: DepositAddressService
	depositAddressService *DepositAddressService

	// Phase D.6: DepositService
	depositService *DepositService

	// Modules (docs/architecture/MODULES.md): one field per wired module.
	environmentModule *modules.EnvironmentModule
	// journal is the one ledger handle bound to the process guard; every money module posts through it.
	journal *ledger.Service

	// Phase F: Sweep + ledger services
	ledgerService               *LedgerService
	feesModule                  *modules.FeesModule
	sweepService                *SweepService
	sweepTransactionService     *SweepTransactionService
	utxoService                 *UTXOService
	sweepUTXOService            *SweepUTXOService
	internalBlockchainTxService *InternalBlockchainTransactionService
	accountRewardService        *AccountRewardService
	withdrawalProcessingService *WithdrawalProcessingService

	// Phase G: JWT refresh, OTP, EventEmitter, Email, Withdrawal services
	jwtTokenService     *JWTTokenService
	otpService          *OTPService
	eventEmitterService *EventEmitterService
	emailService        *EmailService
	withdrawalService   *WithdrawalService

	// Phase H: RBAC + admin + data services
	memberExternalPlatformRoleRepo repository.MemberExternalPlatformRoleRepository
	activityLogRepo                repository.ActivityLogRepository
	genericDataStoreRepo           repository.GenericDataStoreRepository
	recipientRepo                  repository.RecipientRepository

	memberService           *MemberService
	roleService             *RoleService
	permissionService       *PermissionService
	mepRoleService          *MemberExternalPlatformRoleService
	configurationService    *ConfigurationService
	genericDataStoreService *GenericDataStoreService
	nonceService            *NonceService
	recipientService        *RecipientService

	// Phase I: Webhook CRUD, Analytics, Ticker, Referral repos
	analyticsRepo         repository.AnalyticsRepository
	referralCampaignRepo  repository.ReferralCampaignRepository
	referralMemberRepo    repository.ReferralMemberRepository
	referralRewardRepo    repository.ReferralRewardRepository
	analyticsReferralRepo repository.AnalyticsReferralRepository

	// Phase I: services
	webhookManagementService *WebhookManagementService
	analyticsService         *AnalyticsService
	tickerService            *TickerService
	referralService          *ReferralService
	referralCampaignService  *ReferralCampaignService
	analyticsReferralService *AnalyticsReferralService

	// Phase J: ExternalPlatform extras + Onramper + MissedDeposit
	epbcRepo                repository.ExternalPlatformBlockchainCurrencyRepository
	onramperPaymentsRepo    repository.OnramperPaymentsRepository
	externalPlatformService *ExternalPlatformService
	epbcService             *ExternalPlatformBlockchainCurrencyService
	missedDepositService    *MissedDepositService
	onramperPaymentsService *OnramperPaymentsService
	addressBlacklistService *AddressBlacklistService
}

// NewServiceRegistry constructs the registry in three phases:
//  1. Initialize base dependencies (db, redis, cfg)
//  2. Pass 1: construct all repositories and all services
//  3. Pass 2: resolve circular deps via Set*Service setters (no-op for now)
//
// If pass 2 ever fails, return a detailed error so deployment catches the
// misconfiguration early.
// RegistryOption adjusts construction; main passes the modules it already wired.
type RegistryOption func(*ServiceRegistry)

// WithEnvironmentModule reuses the module main wired at the boot gate instead of wiring a second one.
func WithEnvironmentModule(m *modules.EnvironmentModule) RegistryOption {
	return func(r *ServiceRegistry) { r.environmentModule = m }
}

func NewServiceRegistry(db *gorm.DB, rdb *redis.Client, cfg *config.Config, opts ...RegistryOption) (*ServiceRegistry, error) {
	if db == nil {
		return nil, fmt.Errorf("service registry: db is nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("service registry: config is nil")
	}

	r := &ServiceRegistry{
		db:          db,
		redis:       rdb,
		cfg:         cfg,
		networkType: cfg.Blockchain.NetworkType,
	}

	for _, opt := range opts {
		opt(r)
	}
	var err error
	if r.environmentModule == nil {
		if r.environmentModule, err = modules.WireEnvironment(modules.Deps{Config: cfg, DB: db}); err != nil {
			return nil, err
		}
	}

	// Construct adapter registry and register one adapter per active blockchain.
	r.adapterRegistry = blockchain.NewAdapterRegistry()

	// Pass 1: construct repositories
	r.memberRepo = repository.NewMemberRepository(db)
	r.apiKeyRepo = repository.NewAPIKeyRepository(db)
	r.externalPlatformRepo = repository.NewExternalPlatformRepository(db)
	r.roleRepo = repository.NewRoleRepository(db)
	r.permissionRepo = repository.NewPermissionRepository(db)
	r.paymentRepo = repository.NewPaymentRepository(db)
	r.depositRepo = repository.NewDepositRepository(db)
	r.depositAddressRepo = repository.NewDepositAddressRepository(db)
	r.walletRepo = repository.NewWalletRepository(db)
	r.addressPoolRepo = repository.NewAddressPoolRepository(db)
	r.blockchainRepo = repository.NewBlockchainRepository(db)
	r.blockchainFamilyRepo = repository.NewBlockchainFamilyRepository(db)
	r.currencyRepo = repository.NewCurrencyRepository(db)
	r.blockchainCurrencyRepo = repository.NewBlockchainCurrencyRepository(db)
	r.webhookRepo = repository.NewWebhookRepository(db)
	r.webhookDeliveryLogRepo = repository.NewWebhookDeliveryLogRepository(db)
	r.configurationRepo = repository.NewConfigurationRepository(db)
	r.rpcNodeRepo = repository.NewRPCNodeRepository(db)
	r.secretsVaultRepo = repository.NewSecretsVaultRepository(db)
	r.secretsVaultActivityRepo = repository.NewSecretsVaultActivityRepository(db)
	r.walletXpubRepo = repository.NewWalletXpubRepository(db)
	r.walletFunctionRepo = repository.NewWalletFunctionRepository(db)
	r.walletSCWRepo = repository.NewWalletSCWRepository(db)
	r.addressRepo = repository.NewAddressRepository(db)
	r.missedDepositRepo = repository.NewMissedDepositRepository(db)

	// Phase F: Sweep, UTXO, InternalBlockchainTx, AddressDeployment, Account repos
	r.sweepRepo = repository.NewSweepRepository(db)
	r.sweepTxRepo = repository.NewSweepTransactionRepository(db)
	r.utxoRepo = repository.NewUTXORepository(db)
	r.internalBCTxRepo = repository.NewInternalBlockchainTransactionRepository(db)
	r.addressDeploymentRepo = repository.NewAddressDeploymentRepository(db)
	r.accountRepo = repository.NewAccountRepository(db)

	// Phase G: Auth refresh token, OTP, EEEvent, Withdrawal repos
	r.authRefreshTokenRepo = repository.NewAuthRefreshTokenRepository(db)
	r.otpRepo = repository.NewOTPRepository(db)
	r.eeEventRepo = repository.NewEEEventRepository(db)
	r.withdrawalRepo = repository.NewWithdrawalRepository(db)
	r.withdrawRepo = repository.NewWithdrawRepository(db)

	// Phase H: RBAC + admin repos
	r.memberExternalPlatformRoleRepo = repository.NewMemberExternalPlatformRoleRepository(db)
	r.activityLogRepo = repository.NewActivityLogRepository(db)
	r.genericDataStoreRepo = repository.NewGenericDataStoreRepository(db)
	r.recipientRepo = repository.NewRecipientRepository(db)

	// Phase I: Analytics, Referral repos
	r.analyticsRepo = repository.NewAnalyticsRepository(db)
	r.referralCampaignRepo = repository.NewReferralCampaignRepository(db)
	r.referralMemberRepo = repository.NewReferralMemberRepository(db)
	r.referralRewardRepo = repository.NewReferralRewardRepository(db)
	r.analyticsReferralRepo = repository.NewAnalyticsReferralRepository(db)

	// Register blockchain adapters from active chains + their RPC nodes.
	// This populates the adapterRegistry so block processors can be started.
	if chains, err := r.blockchainRepo.ListActive(); err == nil {
		for i := range chains {
			chain := &chains[i]
			pool := blockchain.NewRPCPool(chain.ID, r.rpcNodeRepo)
			if err := pool.Refresh(); err != nil {
				log.Printf("[registry] RPC pool refresh for %s: %v (no adapter registered)", chain.Code, err)
				continue
			}
			if pool.Len() == 0 {
				log.Printf("[registry] no RPC nodes for %s, skipping adapter", chain.Code)
				continue
			}
			switch chain.Family {
			case "ETH_Family":
				chainID := int64(0)
				if chain.ChainID != nil {
					chainID = *chain.ChainID
				}
				adapter := ethAdapter.NewAdapter(chain.Code, chain.Name, chainID, pool)
				_ = r.adapterRegistry.Register(adapter)
				log.Printf("[registry] registered ETH adapter for %s (chainID=%d, %d RPC nodes)", chain.Code, chainID, pool.Len())
			case "BTC_Family":
				netParams := &chaincfg.MainNetParams
				if r.networkType != "mainnet" {
					netParams = &chaincfg.TestNet3Params
				}
				adapter := btcAdapter.NewAdapter(pool, netParams)
				_ = r.adapterRegistry.Register(adapter)
				log.Printf("[registry] registered BTC adapter for %s (%s, %d RPC nodes)", chain.Code, netParams.Name, pool.Len())
			case "TRX_Family":
				tronNet := "mainnet"
				if r.networkType != "mainnet" {
					tronNet = "nile"
				}
				adapter := tronAdapter.NewAdapter(pool, tronNet)
				_ = r.adapterRegistry.Register(adapter)
				log.Printf("[registry] registered TRX adapter for %s (%s, %d RPC nodes)", chain.Code, tronNet, pool.Len())
			default:
				log.Printf("[registry] no adapter impl for family %s (chain %s), skipping", chain.Family, chain.Code)
			}
		}
	}

	// Pass 1: construct services (now take repository interfaces).
	r.authService = NewAuthService(r.memberRepo, r.apiKeyRepo, cfg.Security.JWTSecret)
	r.authService.SetEnvironment(r.environmentModule.Environment)
	r.paymentService = NewPaymentService(r.paymentRepo)
	r.webhookService = NewWebhookService(r.webhookRepo, r.webhookDeliveryLogRepo)
	r.webhookService.SetEnvironment(r.environmentModule.Environment)
	r.onrampService = NewOnrampService("", "")
	r.secretsVaultService = NewSecretsVaultService(r.secretsVaultRepo, r.secretsVaultActivityRepo)

	// Phase D.2: HD wallet service (depends on vault being constructed above)
	r.walletService = NewWalletService(
		r.walletRepo,
		r.walletXpubRepo,
		r.walletFunctionRepo,
		r.blockchainFamilyRepo,
		r.secretsVaultService,
	)

	// M1: KeyResolver — resolves the signing private key for any managed
	// address (deposit/hot) by reproducing its HD derivation from the vault.
	// Used by the withdrawal and sweep broadcast paths.
	r.keyResolver = NewKeyResolver(
		r.addressPoolRepo,
		r.walletRepo,
		r.blockchainFamilyRepo,
		r.secretsVaultService,
		r.networkType == "mainnet",
	)

	// M1: EVMBroadcaster — composes the key resolver + EVM chain adapters to
	// sign and broadcast native/ERC-20 transfers. Shared by the sweep and
	// withdrawal broadcast paths.
	r.evmBroadcaster = NewEVMBroadcaster(r.keyResolver, r.adapterRegistry)

	// Phase D.3: AddressPoolService (depends on walletService constructed above)
	r.addressPoolService = NewAddressPoolService(
		r.addressPoolRepo,
		r.walletRepo,
		r.blockchainFamilyRepo,
		r.walletService,
	)

	// Phase D.4: AddressService
	r.addressService = NewAddressService(r.addressRepo, r.blockchainCurrencyRepo)

	// Phase D.5: DepositAddressService (depends on walletService + addressPoolService)
	r.depositAddressService = NewDepositAddressService(
		r.depositAddressRepo,
		r.walletRepo,
		r.blockchainCurrencyRepo,
		r.walletService,
		r.addressPoolService,
	)

	// Phase D.6: DepositService
	r.depositService = NewDepositService(
		r.depositRepo,
		r.depositAddressRepo,
		r.paymentRepo,
		r.blockchainCurrencyRepo,
		r.blockchainRepo,
	)

	// Phase F.1: LedgerService (depends on accountRepo); dual-writes into internal/ledger
	journal := ledger.New(db, ledger.WithEnvironment(r.environmentModule.Environment), ledger.WithGuard(r.environmentModule.Guard))
	r.journal = journal
	r.ledgerService = NewLedgerService(r.accountRepo, WithJournal(journal, blockchainCurrencyAssetResolver()))
	feesModule, err := modules.WireFees(modules.Deps{DB: db, Config: cfg, Ledger: journal, LedgerAsset: LedgerAssetResolver()})
	if err != nil {
		return nil, fmt.Errorf("wire fees: %w", err)
	}
	r.feesModule = feesModule

	// Phase F.2: SweepTransactionService (depends on ledgerService)
	r.sweepTransactionService = NewSweepTransactionService(
		r.sweepTxRepo,
		r.sweepRepo,
		r.ledgerService,
	)

	// Phase F.3: SweepService (depends on sweepTxService + ledgerService)
	r.sweepService = NewSweepService(
		db,
		r.sweepRepo,
		r.sweepTxRepo,
		r.blockchainRepo,
		r.ledgerService,
	)

	// M1: EVMSweepService — orchestrates native-EVM deposit sweeps to cold
	// storage via the EVMBroadcaster. minSweep guards against sweeping dust.
	r.evmSweepService = NewEVMSweepService(
		r.depositRepo,
		r.blockchainCurrencyRepo,
		r.sweepService,
		r.sweepTransactionService,
		r.evmBroadcaster,
		cfg.Blockchain.ColdWalletETH,
		evmMinSweepWei,
	)

	// M1: EVMSweepConfirmer — advances broadcast sweep txs to confirmed and
	// completes the sweep batch (ledger) once they reach the chain's depth.
	r.evmSweepConfirmer = NewEVMSweepConfirmer(
		r.sweepTxRepo,
		r.blockchainCurrencyRepo,
		r.sweepService,
		NewAdapterConfirmationChecker(r.adapterRegistry),
	)

	// Phase F.4: UTXOService
	r.utxoService = NewUTXOService(r.utxoRepo)

	// Phase F.5: SweepUTXOService
	r.sweepUTXOService = NewSweepUTXOService(
		db,
		r.utxoRepo,
		r.sweepTransactionService,
		r.ledgerService,
	)

	// Phase F.6: InternalBlockchainTransactionService
	r.internalBlockchainTxService = NewInternalBlockchainTransactionService(
		db,
		r.internalBCTxRepo,
		r.ledgerService,
	)

	// Phase F.7: AccountRewardService
	r.accountRewardService = NewAccountRewardService(r.accountRepo)

	// Phase G.1: JWTTokenService
	r.jwtTokenService = NewJWTTokenService(
		r.authRefreshTokenRepo,
		r.memberRepo,
		cfg.Security.JWTSecret,
		cfg.Security.JWTSecret, // refresh secret (same key for now; Phase K can split)
		jwtAccessTTL,
		jwtRefreshTTL,
	)
	r.jwtTokenService.SetEnvironment(r.environmentModule.Environment)

	// Phase G.2: OTPService
	r.otpService = NewOTPService(r.otpRepo)
	r.otpService.SetEnvironment(r.environmentModule.Environment)

	// Phase G.3: EventEmitterService
	r.eventEmitterService = NewEventEmitterService(r.eeEventRepo)

	// Phase G.4: EmailService (loads templates at startup). Delivery transport
	// is SMTP when configured, else a no-op logger.
	r.emailService = NewEmailService(transport.New(transport.Config{
		Host:     cfg.Email.SMTPHost,
		Port:     cfg.Email.SMTPPort,
		Username: cfg.Email.Username,
		Password: cfg.Email.Password,
		From:     cfg.Email.From,
	}))

	// Phase G.5: WithdrawalService
	r.withdrawalService = NewWithdrawalService(
		r.withdrawalRepo,
		r.blockchainCurrencyRepo,
		r.otpService,
		r.eventEmitterService,
	)

	// Phase F.8 (now real): WithdrawalProcessingService
	// ConfigurationService is needed here (hot-wallet address lookup) — construct
	// it before its first use rather than later in the Phase H grouping.
	r.configurationService = NewConfigurationService(r.configurationRepo)
	r.hotWalletSource = NewHotWalletSource(
		r.walletRepo,
		r.blockchainFamilyRepo,
		r.configurationService,
		r.secretsVaultService,
	)
	r.withdrawalProcessingService = NewWithdrawalProcessingService(
		r.withdrawalRepo,
		r.withdrawRepo,
		r.ledgerService,
		r.adapterRegistry,
		r.hotWalletSource,
		r.blockchainCurrencyRepo,
	)

	// Phase H: RBAC + admin services
	r.mepRoleService = NewMemberExternalPlatformRoleService(r.memberExternalPlatformRoleRepo)
	r.memberService = NewMemberService(r.memberRepo, r.mepRoleService)
	r.roleService = NewRoleService(r.roleRepo, r.permissionRepo)
	r.permissionService = NewPermissionService(r.permissionRepo)
	r.genericDataStoreService = NewGenericDataStoreService(r.genericDataStoreRepo)
	r.nonceService = NewNonceService(r.genericDataStoreService)
	r.recipientService = NewRecipientService(r.recipientRepo)

	// Phase I: Webhook CRUD, Analytics, Ticker, Referral services
	r.webhookManagementService = NewWebhookManagementService(r.webhookRepo, r.webhookDeliveryLogRepo)
	r.analyticsService = NewAnalyticsService(r.analyticsRepo)
	r.tickerService = NewTickerService(r.configurationService, WithCoinGecko())
	r.referralCampaignService = NewReferralCampaignService(r.referralCampaignRepo)
	r.referralService = NewReferralService(
		r.referralMemberRepo,
		r.referralRewardRepo,
		r.referralCampaignRepo,
		r.ledgerService,
	)
	r.analyticsReferralService = NewAnalyticsReferralService(r.analyticsReferralRepo)

	// Phase J: ExternalPlatform full + per-currency limits + Onramper + MissedDeposit
	r.epbcRepo = repository.NewExternalPlatformBlockchainCurrencyRepository(db)
	r.onramperPaymentsRepo = repository.NewOnramperPaymentsRepository(db)
	r.externalPlatformService = NewExternalPlatformService(r.externalPlatformRepo, r.apiKeyRepo)
	r.externalPlatformService.SetEnvironment(r.environmentModule.Environment)
	r.epbcService = NewExternalPlatformBlockchainCurrencyService(r.epbcRepo)
	r.missedDepositService = NewMissedDepositService(r.missedDepositRepo)
	r.onramperPaymentsService = NewOnramperPaymentsService(r.onramperPaymentsRepo, "", "", "")
	r.addressBlacklistService = NewAddressBlacklistService(r.configurationService)
	if err := r.addressBlacklistService.Load(); err != nil {
		return nil, fmt.Errorf("load address blacklist: %w", err)
	}

	// Pass 2: resolve circular deps via setter injection.
	// PaymentService needs DepositAddressService to auto-assign addresses on
	// payment creation. Both are already constructed above so we wire them here.
	r.paymentService.SetDepositAddressService(r.depositAddressService)
	r.paymentService.SetMemberRepo(r.memberRepo)

	// Wire JWTTokenService + EventEmitter into AuthService so Signup/Signin
	// can issue refresh tokens and emit welcome emails.
	r.authService.SetJWTTokenService(r.jwtTokenService)
	r.authService.SetEventEmitter(r.eventEmitterService)
	// Wire ExternalPlatformService + MEPRole repo so Signup can auto-create
	// a default project for root members and Signin can resolve platformID.
	r.authService.SetExternalPlatformService(r.externalPlatformService)
	r.authService.SetMEPRoleRepo(r.memberExternalPlatformRoleRepo)

	// Phase D: Wire ExternalPlatformService dependencies for wallet auto-init.
	r.externalPlatformService.SetWalletService(r.walletService)
	r.externalPlatformService.SetAddressPoolService(r.addressPoolService)
	r.externalPlatformService.SetBlockchainFamilyRepo(r.blockchainFamilyRepo)
	r.externalPlatformService.SetBlockchainCurrencyRepo(r.blockchainCurrencyRepo)
	r.externalPlatformService.SetEPBCRepo(r.epbcRepo)

	// Phase J: WithdrawalService consults EPBC for per-platform per-currency
	// limits before authorising every payout. Wired via Pass 2 setter.
	r.withdrawalService.SetExternalPlatformBlockchainCurrencyService(r.epbcService)

	// Phase L: destination blacklist check during withdrawal creation.
	r.withdrawalService.SetAddressBlacklistService(r.addressBlacklistService)

	return r, nil
}

// NetworkType returns the current boot-mode network (testnet|mainnet).
func (r *ServiceRegistry) NetworkType() string { return r.networkType }

// FeesModule returns the wired fee rules module.
func (r *ServiceRegistry) FeesModule() *modules.FeesModule { return r.feesModule }

// DB returns the underlying *gorm.DB for services that need it directly.
// Should be used sparingly — prefer repositories.
func (r *ServiceRegistry) DB() *gorm.DB { return r.db }

// Redis returns the underlying redis client for middleware that needs it
// (rate limiting, cache).
func (r *ServiceRegistry) Redis() *redis.Client { return r.redis }

// Config returns the immutable config snapshot loaded at boot.
func (r *ServiceRegistry) Config() *config.Config { return r.cfg }

// ----- Repository accessors -----

func (r *ServiceRegistry) MemberRepo() repository.MemberRepository { return r.memberRepo }
func (r *ServiceRegistry) APIKeyRepo() repository.APIKeyRepository { return r.apiKeyRepo }
func (r *ServiceRegistry) ExternalPlatformRepo() repository.ExternalPlatformRepository {
	return r.externalPlatformRepo
}
func (r *ServiceRegistry) RoleRepo() repository.RoleRepository             { return r.roleRepo }
func (r *ServiceRegistry) PermissionRepo() repository.PermissionRepository { return r.permissionRepo }
func (r *ServiceRegistry) PaymentRepo() repository.PaymentRepository       { return r.paymentRepo }
func (r *ServiceRegistry) DepositRepo() repository.DepositRepository       { return r.depositRepo }
func (r *ServiceRegistry) DepositAddressRepo() repository.DepositAddressRepository {
	return r.depositAddressRepo
}
func (r *ServiceRegistry) WalletRepo() repository.WalletRepository { return r.walletRepo }
func (r *ServiceRegistry) AddressPoolRepo() repository.AddressPoolRepository {
	return r.addressPoolRepo
}
func (r *ServiceRegistry) BlockchainRepo() repository.BlockchainRepository { return r.blockchainRepo }
func (r *ServiceRegistry) BlockchainFamilyRepo() repository.BlockchainFamilyRepository {
	return r.blockchainFamilyRepo
}
func (r *ServiceRegistry) CurrencyRepo() repository.CurrencyRepository { return r.currencyRepo }
func (r *ServiceRegistry) BlockchainCurrencyRepo() repository.BlockchainCurrencyRepository {
	return r.blockchainCurrencyRepo
}
func (r *ServiceRegistry) WebhookRepo() repository.WebhookRepository { return r.webhookRepo }
func (r *ServiceRegistry) WebhookDeliveryLogRepo() repository.WebhookDeliveryLogRepository {
	return r.webhookDeliveryLogRepo
}
func (r *ServiceRegistry) ConfigurationRepo() repository.ConfigurationRepository {
	return r.configurationRepo
}

// ----- Service accessors -----

func (r *ServiceRegistry) AuthService() *AuthService       { return r.authService }
func (r *ServiceRegistry) PaymentService() *PaymentService { return r.paymentService }
func (r *ServiceRegistry) WebhookService() *WebhookService { return r.webhookService }
func (r *ServiceRegistry) OnrampService() *OnrampService   { return r.onrampService }
func (r *ServiceRegistry) SecretsVaultService() *SecretsVaultService {
	return r.secretsVaultService
}

// RPCNodeRepo returns the RPCNode repository.
func (r *ServiceRegistry) RPCNodeRepo() repository.RPCNodeRepository { return r.rpcNodeRepo }

// AdapterRegistry returns the blockchain adapter registry.
func (r *ServiceRegistry) AdapterRegistry() *blockchain.AdapterRegistry {
	return r.adapterRegistry
}

// ----- SecretsVault repository accessors -----

func (r *ServiceRegistry) SecretsVaultRepo() repository.SecretsVaultRepository {
	return r.secretsVaultRepo
}
func (r *ServiceRegistry) SecretsVaultActivityRepo() repository.SecretsVaultActivityRepository {
	return r.secretsVaultActivityRepo
}

// WalletService returns the HD wallet service.
func (r *ServiceRegistry) WalletService() *WalletService { return r.walletService }

// AddressPoolService returns the pre-generated address pool service.
func (r *ServiceRegistry) AddressPoolService() *AddressPoolService { return r.addressPoolService }

// KeyResolver returns the address→private-key resolver used by the withdrawal
// and sweep broadcast paths.
func (r *ServiceRegistry) KeyResolver() *KeyResolver { return r.keyResolver }

// EVMBroadcaster returns the shared EVM sign-and-broadcast primitive.
func (r *ServiceRegistry) EVMBroadcaster() *EVMBroadcaster { return r.evmBroadcaster }

// EVMSweepService returns the native-EVM sweep orchestrator.
func (r *ServiceRegistry) EVMSweepService() *EVMSweepService { return r.evmSweepService }

// EVMSweepConfirmer returns the sweep confirmation-tracking service.
func (r *ServiceRegistry) EVMSweepConfirmer() *EVMSweepConfirmer { return r.evmSweepConfirmer }

// AddressService returns the address query/facade service.
func (r *ServiceRegistry) AddressService() *AddressService { return r.addressService }

// DepositAddressService returns the pool-backed deposit address assignment service.
func (r *ServiceRegistry) DepositAddressService() *DepositAddressService {
	return r.depositAddressService
}

// DepositService returns the deposit recording and confirmation service.
func (r *ServiceRegistry) DepositService() *DepositService { return r.depositService }

// ----- Phase D.1: Wallet extras + Address repository accessors -----

func (r *ServiceRegistry) WalletXpubRepo() repository.WalletXpubRepository {
	return r.walletXpubRepo
}
func (r *ServiceRegistry) WalletFunctionRepo() repository.WalletFunctionRepository {
	return r.walletFunctionRepo
}
func (r *ServiceRegistry) WalletSCWRepo() repository.WalletSCWRepository {
	return r.walletSCWRepo
}
func (r *ServiceRegistry) AddressRepo() repository.AddressRepository {
	return r.addressRepo
}

// MissedDepositRepo returns the MissedDeposit repository.
func (r *ServiceRegistry) MissedDepositRepo() repository.MissedDepositRepository {
	return r.missedDepositRepo
}

// blockchainCurrencyAssetResolver maps a blockchain_currencies id to chain-qualified ledger assets.
// Every Record* caller carries a blockchain_currencies id, so this is the only table it may read.
// The native asset comes from the chain's own native row; without one Native is empty and only a
// journal that books gas fails (gasLines), never a gas-free one.
func blockchainCurrencyAssetResolver() AssetResolver {
	return func(tx *gorm.DB, blockchainCurrencyID uint) (Assets, error) {
		var bc models.BlockchainCurrency
		if err := tx.First(&bc, blockchainCurrencyID).Error; err != nil {
			return Assets{}, err
		}
		if bc.CurrencyCode == "" || bc.BlockchainCode == "" {
			return Assets{}, fmt.Errorf("blockchain currency %d has no currency or chain code", blockchainCurrencyID)
		}
		asset := chainAsset(bc.CurrencyCode, bc.BlockchainCode)
		if strings.EqualFold(bc.Standard, "native") {
			return Assets{Asset: asset, Native: asset}, nil
		}
		var native models.BlockchainCurrency
		err := tx.Where("blockchain_id = ? AND LOWER(standard) = 'native'", bc.BlockchainID).First(&native).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Assets{Asset: asset}, nil
		}
		if err != nil {
			return Assets{}, fmt.Errorf("chain %s native currency row: %w", bc.BlockchainCode, err)
		}
		return Assets{Asset: asset, Native: chainAsset(native.CurrencyCode, native.BlockchainCode)}, nil
	}
}

// LedgerAssetResolver exposes the ledger's asset for one blockchain_currencies row to modules that post journals.
func LedgerAssetResolver() func(tx *gorm.DB, blockchainCurrencyID uint) (string, error) {
	resolve := blockchainCurrencyAssetResolver()
	return func(tx *gorm.DB, blockchainCurrencyID uint) (string, error) {
		a, err := resolve(tx, blockchainCurrencyID)
		return a.Asset, err
	}
}

func chainAsset(currencyCode, blockchainCode string) string {
	return strings.ToUpper(currencyCode) + "." + strings.ToUpper(blockchainCode)
}

// ----- Phase F: Sweep + Ledger repository accessors -----

// SweepRepo returns the Sweep repository.
func (r *ServiceRegistry) SweepRepo() repository.SweepRepository { return r.sweepRepo }

// SweepTxRepo returns the SweepTransaction repository.
func (r *ServiceRegistry) SweepTxRepo() repository.SweepTransactionRepository {
	return r.sweepTxRepo
}

// UTXORepo returns the UTXO repository.
func (r *ServiceRegistry) UTXORepo() repository.UTXORepository { return r.utxoRepo }

// InternalBCTxRepo returns the InternalBlockchainTransaction repository.
func (r *ServiceRegistry) InternalBCTxRepo() repository.InternalBlockchainTransactionRepository {
	return r.internalBCTxRepo
}

// AddressDeploymentRepo returns the AddressDeployment repository.
func (r *ServiceRegistry) AddressDeploymentRepo() repository.AddressDeploymentRepository {
	return r.addressDeploymentRepo
}

// AccountRepo returns the double-entry ledger account repository.
func (r *ServiceRegistry) AccountRepo() repository.AccountRepository { return r.accountRepo }

// ----- Phase F: Service accessors -----

// EnvironmentModule returns the process environment and its guard.
func (r *ServiceRegistry) EnvironmentModule() *modules.EnvironmentModule { return r.environmentModule }

// Journal is the guarded ledger handle; modules wired outside the registry (the switch) must post through it.
func (r *ServiceRegistry) Journal() *ledger.Service { return r.journal }

// LedgerService returns the double-entry ledger facade.
func (r *ServiceRegistry) LedgerService() *LedgerService { return r.ledgerService }

// SweepService returns the sweep batch management service.
func (r *ServiceRegistry) SweepService() *SweepService { return r.sweepService }

// SweepTransactionService returns the on-chain sweep tx service.
func (r *ServiceRegistry) SweepTransactionService() *SweepTransactionService {
	return r.sweepTransactionService
}

// UTXOService returns the UTXO CRUD service.
func (r *ServiceRegistry) UTXOService() *UTXOService { return r.utxoService }

// SweepUTXOService returns the Bitcoin UTXO aggregation + sweep submission service.
func (r *ServiceRegistry) SweepUTXOService() *SweepUTXOService { return r.sweepUTXOService }

// InternalBlockchainTxService returns the internal blockchain transaction service.
func (r *ServiceRegistry) InternalBlockchainTxService() *InternalBlockchainTransactionService {
	return r.internalBlockchainTxService
}

// AccountRewardService returns the referral reward service.
func (r *ServiceRegistry) AccountRewardService() *AccountRewardService {
	return r.accountRewardService
}

// WithdrawalProcessingService returns the withdrawal processing service.
func (r *ServiceRegistry) WithdrawalProcessingService() *WithdrawalProcessingService {
	return r.withdrawalProcessingService
}

// ----- Phase G: service accessors -----

// JWTTokenService returns the JWT access+refresh token service.
func (r *ServiceRegistry) JWTTokenService() *JWTTokenService { return r.jwtTokenService }

// OTPService returns the one-time password service.
func (r *ServiceRegistry) OTPService() *OTPService { return r.otpService }

// EventEmitterService returns the domain-event emitter.
func (r *ServiceRegistry) EventEmitterService() *EventEmitterService {
	return r.eventEmitterService
}

// EmailService returns the email rendering + delivery service.
func (r *ServiceRegistry) EmailService() *EmailService { return r.emailService }

// WithdrawalService returns the merchant withdrawal service.
func (r *ServiceRegistry) WithdrawalService() *WithdrawalService { return r.withdrawalService }

// ----- Phase G: repository accessors -----

// AuthRefreshTokenRepo returns the auth refresh token repository.
func (r *ServiceRegistry) AuthRefreshTokenRepo() repository.AuthRefreshTokenRepository {
	return r.authRefreshTokenRepo
}

// OTPRepo returns the OTP repository.
func (r *ServiceRegistry) OTPRepo() repository.OTPRepository { return r.otpRepo }

// EEEventRepo returns the ee_events queue repository.
func (r *ServiceRegistry) EEEventRepo() repository.EEEventRepository { return r.eeEventRepo }

// WithdrawalRepo returns the Withdrawal repository.
func (r *ServiceRegistry) WithdrawalRepo() repository.WithdrawalRepository {
	return r.withdrawalRepo
}

// WithdrawRepo returns the Withdraw (on-chain tx) repository.
func (r *ServiceRegistry) WithdrawRepo() repository.WithdrawRepository { return r.withdrawRepo }

// ----- Phase H: RBAC + admin repository accessors -----

// MemberExternalPlatformRoleRepo returns the MEP role join repository.
func (r *ServiceRegistry) MemberExternalPlatformRoleRepo() repository.MemberExternalPlatformRoleRepository {
	return r.memberExternalPlatformRoleRepo
}

// ActivityLogRepo returns the activity log repository.
func (r *ServiceRegistry) ActivityLogRepo() repository.ActivityLogRepository {
	return r.activityLogRepo
}

// GenericDataStoreRepo returns the generic KV store repository.
func (r *ServiceRegistry) GenericDataStoreRepo() repository.GenericDataStoreRepository {
	return r.genericDataStoreRepo
}

// RecipientRepo returns the recipient payee-book repository.
func (r *ServiceRegistry) RecipientRepo() repository.RecipientRepository { return r.recipientRepo }

// ----- Phase H: service accessors -----

// MemberService returns the member CRUD service.
func (r *ServiceRegistry) MemberService() *MemberService { return r.memberService }

// RoleService returns the role management service.
func (r *ServiceRegistry) RoleService() *RoleService { return r.roleService }

// PermissionService returns the permission management service.
func (r *ServiceRegistry) PermissionService() *PermissionService { return r.permissionService }

// MEPRoleService returns the member-external-platform-role service (RBAC hot path).
func (r *ServiceRegistry) MEPRoleService() *MemberExternalPlatformRoleService {
	return r.mepRoleService
}

// ConfigurationService returns the runtime configuration service.
func (r *ServiceRegistry) ConfigurationService() *ConfigurationService {
	return r.configurationService
}

// GenericDataStoreService returns the namespaced KV store service.
func (r *ServiceRegistry) GenericDataStoreService() *GenericDataStoreService {
	return r.genericDataStoreService
}

// NonceService returns the anti-replay nonce service.
func (r *ServiceRegistry) NonceService() *NonceService { return r.nonceService }

// RecipientService returns the merchant payee-book service.
func (r *ServiceRegistry) RecipientService() *RecipientService { return r.recipientService }

// ----- Phase I: accessors -----

// WebhookManagementService returns the webhook CRUD service.
func (r *ServiceRegistry) WebhookManagementService() *WebhookManagementService {
	return r.webhookManagementService
}

// AnalyticsService returns the analytics aggregation service.
func (r *ServiceRegistry) AnalyticsService() *AnalyticsService { return r.analyticsService }

// TickerService returns the price-feed service.
func (r *ServiceRegistry) TickerService() *TickerService { return r.tickerService }

// ReferralService returns the referral tracking service.
func (r *ServiceRegistry) ReferralService() *ReferralService { return r.referralService }

// ReferralCampaignService returns the campaign management service.
func (r *ServiceRegistry) ReferralCampaignService() *ReferralCampaignService {
	return r.referralCampaignService
}

// AnalyticsReferralService returns the referral analytics service.
func (r *ServiceRegistry) AnalyticsReferralService() *AnalyticsReferralService {
	return r.analyticsReferralService
}

// ExternalPlatformService returns the admin platform CRUD service (Phase J).
func (r *ServiceRegistry) ExternalPlatformService() *ExternalPlatformService {
	return r.externalPlatformService
}

// EPBCService returns the per-platform per-currency limit service (Phase J).
func (r *ServiceRegistry) EPBCService() *ExternalPlatformBlockchainCurrencyService {
	return r.epbcService
}

// MissedDepositService returns the operator-facing missed-deposit service (Phase J).
func (r *ServiceRegistry) MissedDepositService() *MissedDepositService {
	return r.missedDepositService
}

// OnramperPaymentsService returns the Onramper card-to-crypto session service (Phase J).
func (r *ServiceRegistry) OnramperPaymentsService() *OnramperPaymentsService {
	return r.onramperPaymentsService
}

// EPBCRepo returns the EPBC repository (Phase J).
func (r *ServiceRegistry) EPBCRepo() repository.ExternalPlatformBlockchainCurrencyRepository {
	return r.epbcRepo
}

// OnramperPaymentsRepo returns the OnramperPayments repository (Phase J).
func (r *ServiceRegistry) OnramperPaymentsRepo() repository.OnramperPaymentsRepository {
	return r.onramperPaymentsRepo
}
