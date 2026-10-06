// Payment statuses
export type PaymentStatus = "created" | "confirming" | "confirmed" | "cancelled" | "expired"

// Blockchain networks
export type BlockchainNetwork = "ethereum" | "bitcoin" | "base" | "polygon" | "tron"

// Currencies
export type CryptoCurrency = "BTC" | "ETH" | "USDT" | "USDC" | "TRX" | "MATIC" | "DAI"

// Sweep statuses
export type SweepStatus = "pending" | "processing" | "completed" | "failed"

// Webhook statuses
export type WebhookStatus = "pending" | "delivered" | "failed"

// Wallet types
export type WalletType = "master" | "hot" | "cold" | "deposit"

// Sweep contract statuses
export type SweepContractStatus = "deployed" | "pending" | "failed"

// Member/User
export interface Member {
  id: number
  email: string
  firstName: string
  lastName: string
  role: "root" | "admin" | "project_lead" | "project_manager" | "operations" | "referral_admin"
  avatarUrl?: string
  twoFactorEnabled: boolean
  createdAt: string
  lastActiveAt: string
}

// Project (External Platform)
export interface Project {
  id: number
  name: string
  slug: string
  websiteUrl?: string
  logoUrl?: string
  brandColor?: string
  createdAt: string
}

// Payment Request
export interface PaymentRequest {
  id: number
  refId?: string
  referenceId?: string
  projectId: number
  amount: string
  currency: CryptoCurrency
  blockchain: BlockchainNetwork
  status: PaymentStatus
  customerEmail?: string
  depositAddress?: string
  walletAddress?: string
  merchantAddress?: string
  txHash?: string
  blockNumber?: number
  gasUsed?: string
  confirmations: number
  requiredConfirmations: number
  webhookStatus?: WebhookStatus
  description?: string
  createdAt: string
  updatedAt: string
  confirmedAt?: string
  expiresAt?: string
  metadata?: Record<string, string>
}

// Payment Link
export interface PaymentLink {
  id: number
  refId: string
  url: string
  amount: string
  currency: CryptoCurrency
  description: string | null
  customerEmail: string | null
  createdAt: string
}

// Payment Timeline Event
export interface PaymentTimelineEvent {
  id: number
  paymentId: number
  event: string
  description: string
  createdAt: string
}

// Wallet keyholder model — who controls the private keys
export type WalletKeyholderModel = "self_custody" | "platform_managed" | "mpc_shared"

// Wallet creation wizard form (frontend-only state)
export interface WalletWizardForm {
  blockchain: BlockchainNetwork
  name: string
  keyholderModel: WalletKeyholderModel
  type: WalletType
}

// Public payment session lookup (used by /pay)
export interface PublicPaymentLookup {
  refId: string
  amount: string
  currency: CryptoCurrency
  blockchain: BlockchainNetwork
  depositAddress: string
  status: PaymentStatus
  expiresAt: string
  merchantName: string
  merchantLogoUrl?: string
}

// Wallet
export interface Wallet {
  id: number
  memberId: number
  name: string
  type: WalletType
  blockchain: BlockchainNetwork
  address: string
  balance: string
  status: "active" | "inactive"
  createdAt: string
}

// Deposit
export interface Deposit {
  id: number
  memberId: number
  amount: string
  currency: CryptoCurrency
  blockchain: BlockchainNetwork
  txHash: string
  fromAddress: string
  toAddress: string
  status: "pending" | "confirming" | "confirmed"
  confirmations: number
  createdAt: string
}

// Sweep
export interface Sweep {
  id: number
  walletId: number
  amount: string
  currency: CryptoCurrency
  blockchain: BlockchainNetwork
  txHash?: string
  status: SweepStatus
  gasUsed?: string
  createdAt: string
}

// Sweep Contract
export interface SweepContract {
  id: number
  type: "ERC20" | "TRC20"
  network: BlockchainNetwork
  contractAddress: string | null
  collectorAddress: string
  txHash: string | null
  status: SweepContractStatus
  deployedAt: string | null
  createdAt: string
}

// Webhook
export interface WebhookConfig {
  id: number
  projectId: number
  url: string
  secret: string
  events: string[]
  active: boolean
  createdAt: string
}

export interface WebhookDelivery {
  id: number
  webhookId: number
  event: string
  payload: string
  responseCode?: number
  responseBody?: string
  attempts: number
  status: WebhookStatus
  nextRetryAt?: string
  createdAt: string
}

// API Key
export interface ApiKey {
  id: number
  projectId: number
  name: string
  keyPrefix: string
  active: boolean
  createdAt: string
  lastUsedAt?: string
}

// Withdrawal/Payout
export interface Withdrawal {
  id: number
  memberId: number
  projectId: number
  amount: string
  currency: CryptoCurrency
  blockchain: BlockchainNetwork
  toAddress: string
  txHash?: string
  status: "pending" | "processing" | "completed" | "failed"
  createdAt: string
}

// Activity Log
export interface ActivityLog {
  id: number
  event: string
  category: string
  userId: number
  userName: string
  userRole: string
  projectId?: number
  projectName?: string
  action: string
  source: string
  status: "success" | "failure"
  details?: string
  createdAt: string
}

// Campaign
export interface Campaign {
  id: number
  projectId: number
  name: string
  status: "active" | "paused" | "archived"
  budget: string
  spent: string
  expiresAt?: string
  createdAt: string
}

// Customer
export interface Customer {
  id: number
  email: string
  totalPayments: number
  totalAmount: string
  lastPaymentAt?: string
  createdAt: string
}

// Dashboard metrics
export interface DashboardMetrics {
  allTimePayments: {
    amount: string
    currency: string
    count: number
  }
  todayVolume: {
    amount: string
    currency: string
    count: number
  }
  activeWallets: number
  pendingSweeps: number
  paymentVolumeChart: Array<{
    date: string
    amount: number
  }>
  paymentDistribution: Array<{
    blockchain: string
    amount: number
    percentage: number
  }>
}

// Blockchain sync status
export interface BlockchainSyncStatus {
  blockchain: BlockchainNetwork
  currentBlock: number
  latestBlock: number
  isSynced: boolean
  lastSyncedAt: string
  connectionStatus: "connected" | "syncing" | "disconnected"
}

// Promoter
export interface Promoter {
  id: number
  name: string
  email: string
  referrals: number
  activeReferrals: number
  commissionEarned: string
  status: "active" | "inactive"
  joinedAt: string
}

// Referral
export interface Referral {
  id: number
  promoterId: number
  userName: string
  userEmail: string
  status: "active" | "pending" | "expired"
  commission: string
  createdAt: string
}

// Sweep fund (token balances across deposit addresses)
export interface SweepFund {
  id: number
  asset: string
  network: BlockchainNetwork
  balance: string
  addressCount: number
  contractDeployed: boolean
  approved: boolean
  tokenType: "native" | "ERC20" | "TRC20"
}

// Sweep transaction history
export interface SweepTransaction {
  id: number
  date: string
  type: "sweep" | "approval" | "deploy"
  asset: string
  amount: string
  fromAddress: string
  toAddress: string
  txHash: string
  status: SweepStatus
  gasUsed: string
}

// Payout (user or referral)
export interface Payout {
  id: number
  timestamp: string
  fromAddress: string
  toAddress: string
  token: CryptoCurrency
  amount: string
  network: BlockchainNetwork
  status: "pending" | "processing" | "completed" | "failed"
  projectName?: string
  type: "user" | "referral"
}

// Address book entry
export interface AddressBookEntry {
  id: number
  name: string
  userId?: string
  email?: string
  phone?: string
  walletAddress: string
  network: BlockchainNetwork
  label?: string
  createdAt: string
}

// Refund
export interface Refund {
  id: number
  date: string
  paymentRef: string
  originalAmount: string
  refundAmount: string
  currency: CryptoCurrency
  status: "pending" | "processing" | "completed" | "failed"
  toAddress: string
}

// Analytics summary
export interface AnalyticsSummary {
  totalRewards: string
  totalCommissions: string
  totalUsers: number
  rewardsDistribution: { name: string; value: number }[]
  userGrowth: { date: string; users: number }[]
}

// Cold wallet
export interface ColdWallet {
  id: number
  name: string
  blockchain: BlockchainNetwork
  address: string
  balance: string
  lastVerifiedAt: string
  createdAt: string
}

// Gas fee wallet (used to pay sweep gas / approvals)
export interface GasFeeWallet {
  id: number
  name: string
  blockchain: BlockchainNetwork
  address: string
  balance: string
  isDefault: boolean
  createdAt: string
}

// Per-chain sweep summary
export interface PerChainSweepData {
  chain: "btc" | "eth" | "usdc" | "usdt"
  network: BlockchainNetwork
  pendingBalance: string
  sweptToday: string
  nextSweepInMinutes: number
  deposits: {
    id: number
    address: string
    amount: string
    age: string
    status: SweepStatus
  }[]
}

// Sweep history entry (top-level /sweepIn page)
export interface SweepHistoryEntry {
  id: number
  date: string
  chain: BlockchainNetwork
  asset: string
  amount: string
  fromAddress: string
  toAddress: string
  status: SweepStatus
  txHash: string
}

// Hot wallet transaction (recent activity in detail page)
export interface HotWalletTransaction {
  id: number
  txHash: string
  direction: "in" | "out"
  amount: string
  asset: string
  counterparty: string
  status: SweepStatus
  createdAt: string
}

// Paginated response
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

// ---------------------------------------------------------------------------
// Phase 5C — Roles, Wallet Policies, Global Webhook, API Docs
// ---------------------------------------------------------------------------

export type PermissionDomain =
  | "payments"
  | "wallets"
  | "settings"
  | "developer"
  | "growth"

export type PermissionAction = "view" | "create" | "edit" | "delete"

export type PermissionMatrix = Record<
  PermissionDomain,
  Record<PermissionAction, boolean>
>

export interface Role {
  id: number
  name: string
  description: string
  userCount: number
  builtIn: boolean
  permissions: PermissionMatrix
  createdAt: string
}

export interface WalletPolicies {
  autoSweepEnabled: boolean
  defaultSweepThresholdUsd: string
  sweepIntervalMinutes: number
  coldWalletAllowlist: { label: string; address: string; network: BlockchainNetwork }[]
  perChainGasLimits: { network: BlockchainNetwork; gasLimit: string; maxFeeGwei: string }[]
}

export interface GlobalWebhookConfig {
  defaultRetryCount: number
  defaultRetryBackoffSeconds: number
  timeoutSeconds: number
  signingSecret: string
  signingSecretRotatedAt: string
  headerOverrides: { key: string; value: string }[]
}

export interface ApiDocSection {
  id: string
  title: string
  anchor: string
  body: string
  code?: { language: string; snippet: string }
}

// ---------------------------------------------------------------------------
// Phase 5D additions
// ---------------------------------------------------------------------------

export type InvoiceStatus = "draft" | "sent" | "paid" | "overdue" | "cancelled"

export interface InvoiceLineItem {
  description: string
  quantity: number
  unitPrice: string
}

export interface Invoice {
  id: string
  customerName: string
  customerEmail: string
  amount: string
  currency: CryptoCurrency
  status: InvoiceStatus
  dueDate: string
  createdAt: string
  lineItems: InvoiceLineItem[]
  notes?: string
}

export type MissedPaymentReason = "expired" | "cancelled" | "insufficient" | "abandoned"

export interface MissedPayment {
  id: string
  date: string
  customerEmail: string
  amount: string
  currency: CryptoCurrency
  reason: MissedPaymentReason
  paymentLinkId?: string
}

export interface CardRevenuePoint {
  date: string
  card: number
  crypto: number
}

export interface CardPaymentRow {
  id: string
  date: string
  customerEmail: string
  cardLast4: string
  amount: string
  currency: "USD" | "EUR"
  cryptoOut: string
  cryptoCurrency: CryptoCurrency
  status: "approved" | "declined" | "chargeback" | "pending"
}

export interface UserBalance {
  userId: string
  email: string
  totalUsd: string
  balances: { currency: CryptoCurrency; chain: BlockchainNetwork; amount: string }[]
  lastActivity: string
}

export interface OnrampProvider {
  id: string
  name: string
  logo?: string
  status: "active" | "degraded" | "disabled"
  volume24h: string
  successRate: number
  fees: string
}

export interface OnrampTransaction {
  id: string
  provider: string
  date: string
  fiatAmount: string
  fiatCurrency: "USD" | "EUR"
  cryptoAmount: string
  cryptoCurrency: CryptoCurrency
  status: "completed" | "pending" | "failed"
  customerEmail: string
}

export interface GlobalAddress {
  id: string
  customerEmail: string
  chain: BlockchainNetwork
  address: string
  balance: string
  currency: CryptoCurrency
  lastActivity: string
}

export interface FeatureFlag {
  id: string
  name: string
  description: string
  enabled: boolean
  category: "experimental" | "beta" | "stable"
}

export interface PaymentMethod {
  id: string
  name: string
  description: string
  icon: string
  enabled: boolean
  category: "card" | "crypto" | "bank" | "swap"
}

export interface TestnetFaucet {
  chain: BlockchainNetwork
  tokens: { symbol: CryptoCurrency; available: boolean; amount: string }[]
}
