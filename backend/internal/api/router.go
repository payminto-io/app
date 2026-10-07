package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/payminto/payminto/backend/internal/api/handler"
	"github.com/payminto/payminto/backend/internal/api/middleware"
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/constants"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/payminto/payminto/backend/internal/realtime"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RouterConfig holds all dependencies needed to build the router.
type RouterConfig struct {
	DB                *gorm.DB
	AuthSvc           *service.AuthService
	JWTTokenSvc       *service.JWTTokenService
	OTPSvc            *service.OTPService
	WithdrawalSvc     *service.WithdrawalService
	PaymentSvc        *service.PaymentService
	DepositAddressSvc *service.DepositAddressService
	Host              string

	// AllowedOrigins lists origins permitted to make credentialed cross-origin
	// requests. Empty → any origin without credentials (safe public default).
	AllowedOrigins []string

	// Broker streams real-time payment-status events to checkout clients (SSE).
	Broker realtime.Broker

	// Phase H: RBAC + admin services
	MemberSvc     *service.MemberService
	RoleSvc       *service.RoleService
	PermissionSvc *service.PermissionService
	MEPRoleSvc    *service.MemberExternalPlatformRoleService
	RecipientSvc  *service.RecipientService
	ConfigSvc     *service.ConfigurationService
	SystemSvc     *service.SystemService

	// Phase I: Webhook CRUD, Analytics, Ticker, Referral
	WebhookMgmtSvc       *service.WebhookManagementService
	AnalyticsSvc         *service.AnalyticsService
	TickerSvc            *service.TickerService
	ReferralSvc          *service.ReferralService
	ReferralCampaignSvc  *service.ReferralCampaignService
	AnalyticsReferralSvc *service.AnalyticsReferralService

	// Phase K: Public API dependencies
	BlockchainCurRepo repository.BlockchainCurrencyRepository

	// Phase J: ExternalPlatform admin, MissedDeposit reconciliation, Onramper
	ExternalPlatformSvc *service.ExternalPlatformService
	EPBCSvc             *service.ExternalPlatformBlockchainCurrencyService
	MissedDepositSvc    *service.MissedDepositService
	OnramperSvc         *service.OnramperPaymentsService

	// Wallet management — merchant-facing HD wallet listing, address pool
	// pagination, cold-storage destination config, and hot-wallet registration.
	WalletRepo           repository.WalletRepository
	AddressPoolRepo      repository.AddressPoolRepository
	BlockchainFamilyRepo repository.BlockchainFamilyRepository
	BlockchainRepo       repository.BlockchainRepository
	WalletSvc            *service.WalletService
	VaultSvc             *service.SecretsVaultService

	// Merchant API key management.
	APIKeyRepo repository.APIKeyRepository

	// MetricsEnabled exposes the Prometheus /metrics endpoint and records
	// per-request HTTP metrics when true.
	MetricsEnabled bool

	// AdapterReg gives handlers read access to chain adapters (e.g. hot-wallet
	// balance lookups).
	AdapterReg *blockchain.AdapterRegistry

	// Environment is the process environment module; NewRouter refuses to build without one (ticket 13).
	Environment *modules.EnvironmentModule

	// Fees is the fee rules module (internal/fees); nil leaves its routes unmounted.
	Fees *modules.FeesModule

	// Links is the payment links module (internal/links); nil leaves its /api/v2 routes unmounted.
	Links *modules.LinksModule

	// Redis backs the public rate limits; nil disables them (middleware.RateLimit).
	Redis *redis.Client
}

// processEnvironment is the environment every request is tagged with; there is no default.
func (cfg RouterConfig) processEnvironment() environment.Environment {
	if cfg.Environment == nil || !cfg.Environment.Environment.Valid() {
		panic("api: RouterConfig.Environment is not wired; the router never assumes an environment")
	}
	return cfg.Environment.Environment
}

// NewRouter constructs and returns a configured Gin engine.
func NewRouter(cfg RouterConfig) *gin.Engine {
	r := gin.Default()

	r.Use(middleware.RequestID())
	r.Use(middleware.Environment(cfg.processEnvironment()))
	r.Use(middleware.CORS(cfg.AllowedOrigins...))

	// Observability: record per-request metrics and expose the Prometheus
	// scrape endpoint when enabled.
	if cfg.MetricsEnabled {
		r.Use(middleware.Metrics())
		r.GET(constants.MetricsPath, gin.WrapH(promhttp.Handler()))
	}

	healthH := handler.NewHealthHandler(cfg.DB)
	r.GET("/healthz", healthH.Health)
	r.GET("/livez", func(c *gin.Context) { c.JSON(200, gin.H{"status": "alive"}) })
	r.GET("/readyz", healthH.Health)

	v1 := r.Group("/api/v1")

	// ---- v2: environment indicator (session or API key) ----
	v2 := r.Group("/v2")
	v2.Use(middleware.JWTOrAPIKey(cfg.AuthSvc))
	RegisterEnvironmentRoutes(v2, cfg.Environment)

	// ---- Public auth routes ----
	authH := handler.NewAuthHandler(cfg.AuthSvc, cfg.JWTTokenSvc)
	auth := v1.Group("/auth")
	{
		auth.POST("/signup", authH.Signup)
		auth.POST("/signin", authH.Signin)
		auth.POST("/refresh", authH.Refresh)
		auth.POST("/signout", authH.SignOut)
		auth.POST("/forgot-password", authH.ForgotPassword)
		auth.POST("/reset-password", authH.ResetPassword)
	}

	// ---- JWT-protected routes ----
	jwtProtected := v1.Group("")
	jwtProtected.Use(middleware.JWTAuth(cfg.AuthSvc))
	{
		jwtProtected.POST("/auth/signout-all", authH.SignOutAll)

		if cfg.OTPSvc != nil {
			otpH := handler.NewOTPHandler(cfg.OTPSvc)
			jwtProtected.POST("/otp/generate", otpH.Generate)
			jwtProtected.POST("/otp/verify", otpH.Verify)
		}
	}

	// ---- Merchant-facing routes (JWT or API key) ----
	// The same route surface is used by the human dashboard (JWT Bearer) and
	// by server-to-server integrations (X-API-Key). Admin routes keep their
	// own APIKeyAuth chain below so RBAC is evaluated exclusively against the
	// API-key identity.
	protected := v1.Group("")
	protected.Use(middleware.JWTOrAPIKey(cfg.AuthSvc))
	{
		paymentH := handler.NewPaymentHandler(cfg.PaymentSvc, cfg.Host)
		depositAddrH := handler.NewDepositAddressHandler(cfg.PaymentSvc, cfg.DepositAddressSvc)

		protected.POST("/payment", paymentH.CreatePayment)
		protected.GET("/payment/reference/:reference_id", paymentH.GetPayment)
		protected.GET("/payments", paymentH.ListPayments)

		protected.POST("/deposit-address/reference/:reference_id", depositAddrH.AssignForReference)
		protected.GET("/deposit-address/reference/:reference_id", depositAddrH.ListForReference)

		if cfg.WithdrawalSvc != nil {
			withdrawalH := handler.NewWithdrawalHandler(cfg.WithdrawalSvc)
			protected.POST("/withdrawal/merchant", withdrawalH.Create)
			protected.POST("/withdrawal/:id/otp/verify", withdrawalH.VerifyOTP)
			protected.POST("/withdrawal/:id/approve", withdrawalH.Approve)
			protected.POST("/withdrawal/:id/cancel", withdrawalH.Cancel)
			protected.GET("/withdrawal/:id", withdrawalH.GetByID)
			protected.GET("/withdrawal/merchant", withdrawalH.List)
		}

		// Recipient (payee book) — tenant-isolated, no extra permission beyond API key.
		if cfg.RecipientSvc != nil {
			recipientH := handler.NewRecipientHandler(cfg.RecipientSvc)
			protected.GET("/recipients", recipientH.ListRecipients)
			protected.GET("/recipients/:id", recipientH.GetRecipient)
			protected.POST("/recipients", recipientH.CreateRecipient)
			protected.PUT("/recipients/:id", recipientH.UpdateRecipient)
			protected.DELETE("/recipients/:id", recipientH.DeleteRecipient)
		}

		// Self-service member endpoints — available to anyone on the merchant
		// group. Admin CRUD of other members lives under /admin/members below.
		if cfg.MemberSvc != nil {
			meH := handler.NewMemberHandler(cfg.MemberSvc)
			protected.GET("/members/me", meH.GetMe)
			protected.PUT("/members/me/password", meH.ChangeOwnPassword)

			// Customer list — returns members with member_type = "customer"
			// auto-created during payment creation.
			protected.GET("/customers", meH.ListCustomers)
		}

		// Merchant-facing API key management — scoped to caller's platform.
		if cfg.APIKeyRepo != nil {
			apiKeyH := handler.NewAPIKeyHandler(cfg.APIKeyRepo, cfg.processEnvironment())
			protected.GET("/api-keys", apiKeyH.ListAPIKeys)
			protected.POST("/api-keys", apiKeyH.CreateAPIKey)
			protected.POST("/api-keys/:id/revoke", apiKeyH.RevokeAPIKey)
		}

		// Wallet management — HD wallets, address pool, cold & hot wallets.
		// Mounted under the JWTOrAPIKey group so both dashboard users and
		// server-to-server integrations can administer merchant wallets.
		if cfg.WalletRepo != nil && cfg.AddressPoolRepo != nil && cfg.ConfigSvc != nil {
			walletH := handler.NewWalletHandler(
				cfg.WalletRepo,
				cfg.AddressPoolRepo,
				cfg.BlockchainFamilyRepo,
				cfg.BlockchainRepo,
				cfg.ConfigSvc,
				cfg.WalletSvc,
				cfg.VaultSvc,
				cfg.AdapterReg,
			)
			protected.GET("/wallets", walletH.ListWallets)
			protected.GET("/wallets/cold", walletH.ListColdWallets)
			protected.POST("/wallets/cold", walletH.ConfigureColdWallet)
			protected.GET("/wallets/hot", walletH.ListHotWallets)
			protected.GET("/wallets/hot/:id/balance", walletH.GetHotWalletBalance)
			protected.POST("/wallets/hot", walletH.RegisterHotWallet)
			protected.GET("/wallets/:id/addresses", walletH.ListWalletAddresses)
		}
	}

	// ---- Admin routes (JWT or API-key + RBAC) ----
	// All admin routes are under /api/v1/admin and guarded by JWTOrAPIKey auth +
	// individual RequirePermission checks per route group. Dashboard users
	// authenticate via JWT; server-to-server integrations use API keys.
	if cfg.MEPRoleSvc != nil {
		admin := v1.Group("/admin")
		admin.Use(middleware.JWTOrAPIKey(cfg.AuthSvc))

		// Roles management (requires roles.manage)
		if cfg.RoleSvc != nil {
			roleH := handler.NewRoleHandler(cfg.RoleSvc)
			rolesGrp := admin.Group("/roles")
			rolesGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "roles.manage"))
			{
				rolesGrp.GET("", roleH.ListRoles)
				rolesGrp.GET("/:id", roleH.GetRole)
				rolesGrp.POST("", roleH.CreateRole)
				rolesGrp.PUT("/:id/permissions", roleH.UpdateRolePermissions)
			}
		}

		// Permissions list (any authenticated caller).
		if cfg.PermissionSvc != nil {
			permH := handler.NewPermissionHandler(cfg.PermissionSvc)
			admin.GET("/permissions", permH.ListPermissions)
		}

		// Member management.
		if cfg.MemberSvc != nil {
			memberH := handler.NewMemberHandler(cfg.MemberSvc)
			// Read requires members.read.
			membersReadGrp := admin.Group("/members")
			membersReadGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "members.read"))
			// List all members on the caller's external platform.
			membersReadGrp.GET("", memberH.ListMembers)
			membersReadGrp.GET("/:id", memberH.GetMember)

			// Invite requires members.invite.
			membersInviteGrp := admin.Group("/members")
			membersInviteGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "members.invite"))
			membersInviteGrp.POST("", memberH.InviteMember)

			// Remove requires members.remove.
			membersRemoveGrp := admin.Group("/members")
			membersRemoveGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "members.remove"))
			membersRemoveGrp.DELETE("/:id", memberH.RemoveMember)
		}

		// Configuration (requires settings.write to mutate, settings.read to view).
		if cfg.ConfigSvc != nil {
			configH := handler.NewConfigurationHandler(cfg.ConfigSvc)
			configReadGrp := admin.Group("/configurations")
			configReadGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "settings.read"))
			configReadGrp.GET("", configH.ListConfigurations)
			configReadGrp.GET("/:key", configH.GetConfiguration)

			configWriteGrp := admin.Group("/configurations")
			configWriteGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "settings.write"))
			configWriteGrp.PUT("/:key", configH.SetConfiguration)
		}

		// System worker control (requires system.admin).
		if cfg.SystemSvc != nil {
			systemH := handler.NewSystemHandler(cfg.SystemSvc)
			sysGrp := admin.Group("/system")
			sysGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "system.admin"))
			{
				sysGrp.GET("/workers", systemH.ListWorkers)
				sysGrp.POST("/workers/stop-all", systemH.StopAllWorkers)
				sysGrp.GET("/health", systemH.GetHealth)
			}
		}
	}

	// ---- Phase I: Ticker (public, no auth) ----
	if cfg.TickerSvc != nil {
		tickerH := handler.NewTickerHandler(cfg.TickerSvc)
		v1.GET("/ticker/:symbol", tickerH.GetPrice)
		v1.GET("/ticker", tickerH.GetPrices)
	}

	// ---- Phase I: Webhook CRUD (API-key + webhooks.manage) ----
	if cfg.WebhookMgmtSvc != nil && cfg.MEPRoleSvc != nil {
		webhookH := handler.NewWebhookHandler(cfg.WebhookMgmtSvc)
		webhookGrp := protected.Group("/webhooks")
		webhookGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "webhooks.manage"))
		{
			webhookGrp.POST("", webhookH.CreateWebhook)
			webhookGrp.GET("", webhookH.ListWebhooks)
			webhookGrp.GET("/:id", webhookH.GetWebhook)
			webhookGrp.PUT("/:id", webhookH.UpdateWebhook)
			webhookGrp.DELETE("/:id", webhookH.DeleteWebhook)
			webhookGrp.POST("/:id/regenerate-secret", webhookH.RegenerateSecret)
			webhookGrp.GET("/:id/deliveries", webhookH.ListDeliveries)
		}
	}

	// ---- Phase I: Analytics (API-key + analytics.read) ----
	if cfg.AnalyticsSvc != nil && cfg.MEPRoleSvc != nil {
		analyticsH := handler.NewAnalyticsHandler(cfg.AnalyticsSvc)
		analyticsGrp := protected.Group("/analytics")
		analyticsGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "analytics.read"))
		{
			analyticsGrp.GET("/volume", analyticsH.GetVolume)
			analyticsGrp.GET("/customers/top", analyticsH.GetTopCustomers)
			analyticsGrp.GET("/revenue", analyticsH.GetRevenueBreakdown)
			analyticsGrp.GET("/withdrawals", analyticsH.GetWithdrawalStats)
			analyticsGrp.GET("/summary", analyticsH.GetSummary)
		}
	}

	// ---- Phase I: Referral (API-key, member's own data) ----
	if cfg.ReferralSvc != nil && cfg.AnalyticsReferralSvc != nil {
		referralH := handler.NewReferralHandler(cfg.ReferralSvc, cfg.AnalyticsReferralSvc)
		referralGrp := protected.Group("/referrals")
		{
			referralGrp.GET("/code", referralH.GetMyCode)
			referralGrp.POST("/register", referralH.Register)
			referralGrp.GET("/rewards", referralH.ListMyRewards)
			referralGrp.GET("/stats", referralH.GetMyStats)
			referralGrp.POST("/record-payment", referralH.RecordPayment)
		}
	}

	// ---- Phase I: Referral Admin (API-key + system.admin) ----
	if cfg.ReferralCampaignSvc != nil && cfg.AnalyticsReferralSvc != nil && cfg.MEPRoleSvc != nil {
		adminReferralH := handler.NewReferralAdminHandler(cfg.ReferralCampaignSvc, cfg.AnalyticsReferralSvc)
		adminReferralGrp := v1.Group("/admin/referrals")
		adminReferralGrp.Use(middleware.JWTOrAPIKey(cfg.AuthSvc))
		adminReferralGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "system.admin"))
		{
			adminReferralGrp.POST("/campaigns", adminReferralH.CreateCampaign)
			adminReferralGrp.GET("/campaigns", adminReferralH.ListCampaigns)
			adminReferralGrp.GET("/campaigns/:id", adminReferralH.GetCampaign)
			adminReferralGrp.POST("/campaigns/:id/activate", adminReferralH.ActivateCampaign)
			adminReferralGrp.POST("/campaigns/:id/deactivate", adminReferralH.DeactivateCampaign)
			adminReferralGrp.DELETE("/campaigns/:id", adminReferralH.DeleteCampaign)
			adminReferralGrp.GET("/campaigns/:id/performance", adminReferralH.GetCampaignPerformance)
			adminReferralGrp.GET("/top", adminReferralH.GetTopReferrers)
		}
	}

	// ---- Phase K: Public API (no auth, narrowed projection) ----
	if cfg.PaymentSvc != nil {
		publicH := handler.NewPublicAPIHandler(cfg.PaymentSvc, cfg.DepositAddressSvc, cfg.ExternalPlatformSvc, cfg.TickerSvc, cfg.BlockchainCurRepo)
		pub := v1.Group("/public")
		pub.GET("/payment/:reference_id", publicH.GetPaymentByReference)
		pub.POST("/deposit-address/reference/:reference_id", publicH.AssignDepositAddress)
		pub.GET("/blockchain-currencies", publicH.ListBlockchainCurrencies)
		if cfg.TickerSvc != nil {
			pub.GET("/ticker", publicH.GetTicker)
		}
		// Real-time payment status stream (SSE) for the hosted checkout.
		// Mounted under a distinct "events" prefix (sharing no radix prefix with
		// "/payment/:reference_id") to avoid gin/httprouter's rule that a node
		// cannot hold both a param child and a static sibling.
		if cfg.Broker != nil {
			eventsH := handler.NewPaymentEventsHandler(cfg.Broker, cfg.PaymentSvc)
			pub.GET("/events/:reference_id", eventsH.Stream)
		}
	}

	// ---- Phase J: Onramper (merchant + webhook) ----
	if cfg.OnramperSvc != nil {
		onramperH := handler.NewOnramperHandler(cfg.OnramperSvc)
		// Public webhook — signature-verified inside the handler.
		v1.POST("/onramper/webhook", onramperH.Webhook)
		// Merchant endpoints require API key.
		protected.POST("/onramper/session", onramperH.CreateSession)
		protected.GET("/onramper/payments", onramperH.List)
	}

	// ---- Phase J: Admin — external platforms + missed deposits ----
	if cfg.MEPRoleSvc != nil {
		adminJ := v1.Group("/admin")
		adminJ.Use(middleware.JWTOrAPIKey(cfg.AuthSvc))

		if cfg.ExternalPlatformSvc != nil && cfg.EPBCSvc != nil {
			epH := handler.NewExternalPlatformHandler(cfg.ExternalPlatformSvc, cfg.EPBCSvc)
			epGrp := adminJ.Group("/external-platforms")
			epGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "system.admin"))
			{
				epGrp.POST("", epH.Create)
				epGrp.GET("", epH.List)
				epGrp.GET("/:id", epH.Get)
				epGrp.PUT("/:id", epH.Update)
				epGrp.DELETE("/:id", epH.Delete)
				epGrp.POST("/:id/regenerate-key", epH.RegenerateKey)
				epGrp.GET("/:id/currencies", epH.ListCurrencies)
				epGrp.POST("/:id/currencies", epH.EnableCurrency)
				epGrp.PUT("/:id/currencies/:currencyID", epH.UpdateCurrencyLimits)
				epGrp.DELETE("/:id/currencies/:currencyID", epH.DisableCurrency)
			}
		}

		if cfg.MissedDepositSvc != nil {
			mdH := handler.NewMissedDepositHandler(cfg.MissedDepositSvc)
			mdGrp := adminJ.Group("/missed-deposits")
			mdGrp.Use(middleware.RequirePermission(cfg.MEPRoleSvc, "system.admin"))
			{
				mdGrp.GET("", mdH.ListPending)
				mdGrp.GET("/:id", mdH.Get)
				mdGrp.POST("/:id/resolve", mdH.Resolve)
			}
		}
	}

	// ---- Fee rules: preview for merchants; management needs a dashboard session and system.admin ----
	if cfg.Fees != nil && cfg.AuthSvc != nil {
		auth := FeesAuth{Merchant: middleware.JWTOrAPIKey(cfg.AuthSvc), Session: middleware.JWTAuth(cfg.AuthSvc)}
		if cfg.MEPRoleSvc != nil {
			auth.Admin = middleware.RequirePermission(cfg.MEPRoleSvc, "system.admin")
		}
		RegisterFeesRoutes(v1, cfg.Fees, auth)
	}

	// ---- Payment links: merchant CRUD (session or API key) and the public checkout, rate limited per IP ----
	if cfg.Links != nil {
		auth := LinksAuth{
			PublicRead: middleware.RateLimitScoped(cfg.Redis, "links:read", linksPublicReadPerMinute, time.Minute),
			PublicPay:  middleware.RateLimitScoped(cfg.Redis, "links:pay", linksPublicPayPerMinute, time.Minute),
		}
		if cfg.AuthSvc != nil {
			auth.Merchant = middleware.JWTOrAPIKey(cfg.AuthSvc)
		}
		RegisterLinksRoutes(r.Group("/api/v2"), cfg.Links, auth)
	}

	return r
}

// Public link limits per IP per minute; a checkout loads the link a few times and pays once or twice.
const (
	linksPublicReadPerMinute = 120
	linksPublicPayPerMinute  = 20
)
