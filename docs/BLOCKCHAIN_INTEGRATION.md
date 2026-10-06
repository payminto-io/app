# Payminto Blockchain Integration Specification

**Version:** 1.0
**Date:** April 7, 2026
**Status:** Pre-implementation (reverse-engineered from PayRam reference)
**Purpose:** Complete blockchain integration specification for Payminto, a self-hosted non-custodial cryptocurrency payment gateway.

---

## Table of Contents

1. [Supported Networks](#1-supported-networks)
2. [Token Support Matrix](#2-token-support-matrix)
3. [Blockchain Client Architecture](#3-blockchain-client-architecture)
4. [HD Wallet and Key Management](#4-hd-wallet-and-key-management)
5. [SmartSweep Contract System](#5-smartsweep-contract-system)
6. [Block Monitoring System](#6-block-monitoring-system)
7. [RPC Node Management](#7-rpc-node-management)
8. [Payment Detection Flow](#8-payment-detection-flow)
9. [UTXO Management (Bitcoin)](#9-utxo-management-bitcoin)
10. [Transaction Signing and Broadcasting](#10-transaction-signing-and-broadcasting)
11. [Card-to-Crypto Onramp Integration](#11-card-to-crypto-onramp-integration)
12. [MCP/AI Agent Integration](#12-mcpai-agent-integration)

---

## 1. Supported Networks

### Live Networks

| Network | Code | Family | Chain ID (Mainnet) | Chain ID (Testnet) | Block Time | Status |
|---------|------|--------|--------------------|--------------------|------------|--------|
| Bitcoin | BTC | bitcoin | N/A | N/A | ~10 min | Live |
| Ethereum | ETH | ethereum | 1 | 11155111 (Sepolia) | ~12 sec | Live |
| Base | BASE | ethereum | 8453 | 84532 (Sepolia) | ~2 sec | Live |
| Polygon | POLYGON | ethereum | 137 | 80001 (Mumbai) | ~2 sec | Live |
| Tron | TRX | tron | N/A | Nile testnet | ~3 sec | Live |

### Planned Networks

| Network | Code | Family | Status | Priority |
|---------|------|--------|--------|----------|
| TON | TON | ton | Coming Soon | High (Telegram integration) |
| Solana | SOL | solana | Coming Soon | High (high-speed payments) |
| BNB Chain | BNB | ethereum | Planned | Medium (BSC ecosystem) |
| Ripple | XRP | ripple | Planned | Medium (cross-border) |
| Monero | XMR | monero | Planned | Low (privacy payments) |

### Confirmation Requirements

These are the minimum confirmations required before a deposit is considered confirmed and a webhook is fired to the merchant:

| Network | Required Confirmations | Approximate Time | Rationale |
|---------|----------------------|-------------------|-----------|
| Bitcoin | 3 | ~30 minutes | High value per block, deep reorg resistance |
| Ethereum | 12 | ~2.5 minutes | Post-merge PoS finality characteristics |
| Base | 12 | ~24 seconds | L2 with L1 settlement, lower finality risk |
| Polygon | 128 | ~4.3 minutes | Frequent minor reorgs on PoS sidechain |
| Tron | 19 | ~57 seconds | DPoS consensus with 27 super representatives |

Confirmation counts are configurable per blockchain via the `blockchain_currencies` table. The `min_confirmation` column stores the threshold for each blockchain-currency pair.

### Network Type Configuration

The system supports both mainnet and testnet operation, controlled by the `BLOCKCHAIN_NETWORK_TYPE` environment variable:

```
BLOCKCHAIN_NETWORK_TYPE=mainnet   # Production
BLOCKCHAIN_NETWORK_TYPE=testnet   # Development/testing (Sepolia, Nile, etc.)
```

Each blockchain client's `IsMainnet()` method checks this configuration to select the appropriate RPC endpoints, chain IDs, and contract addresses.

---

## 2. Token Support Matrix

### Native Currencies

| Token | Network | Precision (Decimals) | Monetary Storage | Notes |
|-------|---------|---------------------|------------------|-------|
| BTC | Bitcoin | 8 | `numeric(38,18)` | UTXO model, satoshi-level precision |
| ETH | Ethereum | 18 | `numeric(38,18)` | Account model, wei-level precision |
| TRX | Tron | 6 | `numeric(38,18)` | Account model, sun-level precision |
| MATIC | Polygon | 18 | `numeric(38,18)` | Account model |
| ETH (Base) | Base | 18 | `numeric(38,18)` | L2 native, used for gas |

### ERC-20 / TRC-20 Tokens

| Token | Network | Contract Standard | Decimals | Primary Use Case |
|-------|---------|-------------------|----------|------------------|
| USDT | Ethereum | ERC-20 | 6 | Stablecoin payments (high volume) |
| USDT | Tron | TRC-20 | 6 | Stablecoin payments (low fees, very high volume) |
| USDC | Ethereum | ERC-20 | 6 | Stablecoin payments |
| USDC | Base | ERC-20 | 6 | Preferred for card-to-crypto (low gas) |
| DAI | Ethereum | ERC-20 | 18 | Decentralized stablecoin |
| WBTC | Ethereum | ERC-20 | 8 | Wrapped Bitcoin on Ethereum |
| WETH | Base | ERC-20 | 18 | Wrapped Ether on Base |
| LINK | Ethereum | ERC-20 | 18 | Chainlink token |
| UNI | Ethereum | ERC-20 | 18 | Uniswap token |
| SHIB | Ethereum | ERC-20 | 18 | Meme token |
| 10+ others | Various | ERC-20/TRC-20 | Varies | Extended token support |

### Token Detection Strategy

- **Native transfers**: Extracted from block transaction data via `GetNativeTransfersInBlock()`
- **ERC-20 transfers**: Detected via `Transfer(address,address,uint256)` event logs using `GetTokenTransfersInBlock()`
- **TRC-20 transfers**: Detected via TronGrid API event queries using `GetTokenTransfersInBlock()`

### Database Representation

Tokens are stored in the `blockchain_currencies` table with foreign keys to `blockchains` and `currencies`:

```
blockchain_currencies:
  id                    BIGINT PRIMARY KEY
  blockchain_id         BIGINT FK -> blockchains.id
  currency_id           BIGINT FK -> currencies.id
  contract_address      VARCHAR        -- ERC-20/TRC-20 contract (NULL for native)
  min_confirmation      INTEGER        -- Required confirmations
  is_active             BOOLEAN        -- Enable/disable token
  precision             INTEGER        -- Token decimals
  min_deposit_amount    NUMERIC(38,18) -- Minimum deposit threshold
  approval_fee_amount   NUMERIC(38,18) -- ERC-20 approval gas cost
  sweep_batch_size      INTEGER        -- Batch size for sweep operations
  auto_sweep_interval   INTEGER        -- Auto-sweep timer in seconds
  min_amount_in_sweep_tx NUMERIC(38,18) -- Minimum amount for sweep transaction
  deleted_at            TIMESTAMP      -- Soft delete (GORM convention)
```

---

## 3. Blockchain Client Architecture

### Overview

The blockchain client layer follows a polymorphic design pattern where an abstract base class defines the full interface (40+ methods), and chain-specific implementations override methods as needed. This lives in the `internal/jobs/bclient` package.

```
┌─────────────────────────────────────────────────────────┐
│              AbstractBlockchainClient                    │
│  (40+ abstract methods defining the blockchain interface)│
├─────────────────────────────────────────────────────────┤
│  TestConnection()          GetLatestBlockNumber()        │
│  TestNodeConnection()      GetBlockByNumber()            │
│  Close()                   GetBlockByHash()              │
│  IsMainnet()               GetBlockDetails()             │
│  GetChainID()              GetNativeTransfersInBlock()   │
│  GetFamilyName()           GetTokenTransfersInBlock()    │
│  GetNativeCurrencyPrec()   GetUTXOsInBlock()             │
│  GetBalance()              GetContractEventsInBlock()     │
│  GetTokenBalance()         GetTransactionReceipt()       │
│  GetTransactionDetails()   GetTransactionToAddress()     │
│  GetTransactionFee()       GetEventLogs()                │
│  DecodeEventLogs()         GetEstimatedFeesForContract() │
│  GetRawTx()                GetTxOut()                    │
│  GetWithdrawalAuditData()  GetPublicKeyFromPrivateKey()  │
│  DeriveKeys()              DeriveXPUB()                  │
│  SignTransaction()         SignSmartContractTransaction() │
│  BroadcastRawTransaction() DeserializeTransaction()      │
│  Transfer()                Transaction()                 │
│  ValidateAddress()         VerifyTransaction()           │
│  EnsureTxConfirmed()       ProcessBlock()                │
│  ProcessBlockRange()       SubscribeToBlocks()           │
│  SubscribeToLogs()         FeeSatoshiPerByte()           │
│  EstimateTransactionSize() IsContract()                  │
│  AddressToHex()            ConvertHexToAddress()         │
│  ConvertAddressToEthereumFormat()                        │
└──────────────┬──────────────┬──────────────┬────────────┘
               │              │              │
    ┌──────────▼──┐  ┌───────▼──────┐  ┌───▼──────────────┐
    │ETHBlockchain│  │BTCBlockchain │  │TRXBlockchain     │
    │Client       │  │Client        │  │Client            │
    │             │  │              │  │                  │
    │ EVM chains: │  │ UTXO model:  │  │ Tron-specific:   │
    │ - Ethereum  │  │ - Blockstream│  │ - TronGrid API   │
    │ - Base      │  │ - Mempool    │  │ - Nile testnet   │
    │ - Polygon   │  │ - btcd RPC   │  │ - TRC-20 events  │
    │             │  │ - Fee/byte   │  │ - Energy/bandwidth│
    │ Gas estim.  │  │ - UTXO mgmt  │  │ - Base58 addrs   │
    │ ABI encoding│  │              │  │                  │
    └─────────────┘  └──────────────┘  └──────────────────┘
```

### AbstractBlockchainClient

The abstract base class at `internal/jobs/bclient/blockchain_client.go` (lines 240-403) defines the interface that all chain-specific clients must implement. It provides:

- **Connection management**: `TestConnection()`, `TestNodeConnection()`, `Close()`
- **Chain metadata**: `IsMainnet()`, `GetChainID()`, `GetFamilyName()`, `GetNativeCurrencyPrecision()`
- **Block reading**: `GetLatestBlockNumber()`, `GetBlockByNumber()`, `GetBlockByHash()`, `GetBlockDetails()`
- **Transaction extraction**: `GetNativeTransfersInBlock()`, `GetTokenTransfersInBlock()`, `GetUTXOsInBlock()`, `GetContractEventsInBlock()`
- **Transaction details**: `GetTransactionReceipt()`, `GetTransactionDetails()`, `GetTransactionFee()`, `GetTransactionToAddress()`, `GetEventLogs()`, `DecodeEventLogs()`
- **Balance queries**: `GetBalance()`, `GetTokenBalance()`
- **Key derivation**: `DeriveKeys()`, `DeriveXPUB()`, `GetPublicKeyFromPrivateKey()`
- **Transaction lifecycle**: `SignTransaction()`, `SignSmartContractTransaction()`, `BroadcastRawTransaction()`, `DeserializeTransaction()`
- **Transfer operations**: `Transfer()`, `Transaction()`
- **Verification**: `ValidateAddress()`, `VerifyTransaction()`, `EnsureTxConfirmed()`
- **Block processing**: `ProcessBlock()`, `ProcessBlockRange()`
- **Subscriptions**: `SubscribeToBlocks()`, `SubscribeToLogs()`
- **UTXO-specific**: `FeeSatoshiPerByte()`, `EstimateTransactionSize()`, `GetTxOut()`
- **Utilities**: `IsContract()`, `AddressToHex()`, `ConvertHexToAddress()`, `ConvertAddressToEthereumFormat()`
- **Audit**: `GetWithdrawalAuditData()`
- **Gas estimation**: `GetEstimatedFeesForContractAndMethod()`

### ETHBlockchainClient

The EVM implementation handles Ethereum, Base, and Polygon through a shared client with chain-specific configuration. Key implementation details:

**Connection management:**
- `conn()` / `connPreferred()` — connection pooling with preferred node selection
- `ethExec()` / `ethExecPreferred()` — RPC call execution with automatic failover
- Uses go-ethereum (`ethclient`) for JSON-RPC communication

**Block processing:**
- `GetNativeTransfersInBlock()` — iterates block transactions, extracts ETH value transfers
- `GetTokenTransfersInBlock()` — parses `Transfer(address,address,uint256)` event logs from receipts
- `GetContractEventsInBlock()` — extracts contract events (sweep events, factory events)
- `ProcessBlock()` / `ProcessBlockRange()` — orchestrates full block scanning with range support

**Transaction operations:**
- `SignTransaction()` — builds and signs EIP-1559 transactions with dynamic gas pricing (6 nested functions for complex signing logic)
- `SignSmartContractTransaction()` — ABI-encodes contract calls and signs (3 nested functions)
- `BroadcastRawTransaction()` — submits signed transaction bytes to the network
- `GetEstimatedFeesForContractAndMethod()` — simulates contract calls to estimate gas (2 nested functions)

**Base L2 specifics:**
- `fetchL1FeeAndCalculateTxFeeForBase()` — calculates combined L1 data + L2 execution fee for Base transactions
- `baseNetworkFee()` — fetches Base-specific network fee parameters
- `getBlockRaw()` — raw block fetching for L2 compatibility

**EVM utilities:**
- `decodeCalldataFast()` — fast ABI calldata decoding without full ABI parsing
- `IsContract()` — checks if an address is a contract (2 nested functions for retry logic)
- `GetChainID()` — fetches and validates chain ID from connected node

**Methods that are no-ops for EVM** (inherited from abstract but not applicable):
- `EstimateTransactionSize()` — BTC-only concept
- `FeeSatoshiPerByte()` — BTC-only concept
- `GetTxOut()` — UTXO-only concept
- `GetUTXOsInBlock()` — UTXO-only concept
- `GetWithdrawalAuditData()` — BTC-specific withdrawal audit
- `SubscribeToBlocks()` / `SubscribeToLogs()` — not used (polling model preferred)
- `Transaction()` / `Transfer()` — simplified transfer wrappers

### BTCBlockchainClient

The Bitcoin implementation handles the UTXO model with multiple API backends. Key implementation details:

**API backends** (dual-source for redundancy):
- `fetchFromBlockstreamAPI()` — Blockstream.info REST API for block/transaction data
- `fetchFromMempoolAPI()` — Mempool.space REST API as fallback
- `btcExec()` — direct Bitcoin Core RPC via btcd library
- `conn()` — btcd RPC connection management

**Block and transaction processing:**
- `GetBlockByNumber()` — fetches block by height (2 nested functions for API selection)
- `GetBlockByHash()` — fetches block by hash
- `GetBlockDetails()` — full block details with transaction list (2 nested functions)
- `GetLatestBlockNumber()` — current chain tip height
- `GetNativeTransfersInBlock()` — parses UTXO outputs to identify BTC transfers (2 nested functions)
- `GetUTXOsInBlock()` — extracts all UTXOs from a block (2 nested functions)
- `processUTXODeposits()` — processes raw UTXOs into deposit records

**Transaction operations:**
- `GetTransactionDetails()` — full transaction data including inputs and outputs
- `GetTransactionReceipt()` — confirmation count and block inclusion (2 nested functions)
- `GetTransactionFee()` — calculates fee from input/output difference
- `calculateTransactionFee()` — detailed fee calculation with input resolution
- `GetRawTx()` — raw transaction hex
- `GetTxOut()` — specific UTXO output data

**Fee estimation:**
- `FeeSatoshiPerByte()` — current network fee rate in satoshi/byte
- `EstimateTransactionSize()` — estimates transaction size based on input/output count
- `calculateTransactionFee()` — combines fee rate and size for total fee estimate

**Bitcoin-specific operations:**
- `DeriveKeys()` — BIP-44 key derivation at m/44'/0'/0'/0/N
- `DeriveXPUB()` — extended public key derivation
- `ValidateAddress()` — validates Bitcoin address format (P2PKH, P2SH, Bech32)
- `IsMainnet()` — determines network from address prefix or configuration

**Sweep detection:**
- `detectBTCSweepTransactions()` — identifies sweep transactions in blocks
- `detectBTCSweepTransactionsBatch()` — batch processing for sweep detection

**Withdrawal audit:**
- `GetWithdrawalAuditData()` — retrieves full audit trail for BTC withdrawals (3 nested functions for complex data assembly)

**Methods that are no-ops for Bitcoin** (contract concepts not applicable):
- `CallContractFunction()` — no smart contracts on Bitcoin
- `DecodeEventLogs()` — no event logs on Bitcoin
- `GetContractEventsInBlock()` — no contract events
- `GetEstimatedFeesForContractAndMethod()` — no contracts
- `SignSmartContractTransaction()` — no contracts
- `IsContract()` — always false
- `GetTokenBalance()` / `GetTokenTransfersInBlock()` — no native token standard

### TRXBlockchainClient

The Tron implementation communicates via TronGrid/TronGrid-compatible APIs and gRPC. Key implementation details:

**Connection management:**
- Uses TronGrid HTTP API for most operations
- gRPC connection for transaction broadcasting
- `TestNodeConnection()` — connection validation with transport credentials

**Block processing:**
- `GetBlockByNumber()` / `GetBlockByHash()` — block fetching via TronGrid API
- `GetBlockDetails()` — full block with internal transactions
- `GetNativeTransfersInBlock()` — extracts TRX value transfers from transactions
- `GetTokenTransfersInBlock()` — parses TRC-20 transfer events (2 nested functions)
- `GetContractEventsInBlock()` — extracts smart contract events (2 nested functions)

**Transaction operations:**
- `SignTransaction()` — Tron transaction signing using tron-specific protobuf format
- `SignSmartContractTransaction()` — TRC-20/contract call signing (4 nested functions including error handling)
- `BroadcastRawTransaction()` — submits via TronGrid/gRPC
- `GetTransactionDetails()` / `GetTransactionReceipt()` — transaction lookup via TronGrid
- `GetTransactionFee()` — Tron uses bandwidth/energy model, not gas

**Fee estimation:**
- `GetEstimatedFeesForContractAndMethod()` — estimates energy cost for contract calls
- Tron uses a bandwidth/energy model rather than gas: simple TRX transfers consume bandwidth (free daily allocation), while TRC-20 transfers consume energy (paid in TRX)

**Address handling:**
- `AddressToHex()` — converts T-prefix Base58 address to hex
- `ConvertHexToAddress()` — converts hex to T-prefix Base58
- `ConvertAddressToEthereumFormat()` — converts Tron address to 0x-prefix format for internal processing
- `ValidateAddress()` — validates Tron Base58Check address format

**Key derivation:**
- `DeriveKeys()` — BIP-44 at m/44'/195'/0'/0/N
- `DeriveXPUB()` — extended public key for Tron derivation path
- `GetPublicKeyFromPrivateKey()` — secp256k1 public key extraction

**Tron-specific methods:**
- `GetBalance()` — TRX balance via TronGrid
- `GetEventLogs()` — contract event logs (2 nested functions)
- `DecodeEventLogs()` — ABI-compatible event log parsing
- `IsContract()` — checks if address is a smart contract
- `GetChainID()` — Tron chain identifier

**Methods that are no-ops for Tron:**
- `EstimateTransactionSize()` — not applicable (bandwidth model)
- `FeeSatoshiPerByte()` — BTC-only
- `GetTxOut()` — UTXO-only
- `GetUTXOsInBlock()` — UTXO-only
- `GetTokenBalance()` — handled differently via TronGrid
- `GetWithdrawalAuditData()` — BTC-specific
- `SubscribeToBlocks()` / `SubscribeToLogs()` — not used (polling model)

---

## 4. HD Wallet and Key Management

### BIP-39/32/44 Implementation

Payminto uses hierarchical deterministic (HD) wallet technology to generate unique deposit addresses for each payment without storing private keys on the server.

```
┌──────────────────────────────────────────────────────────────┐
│                    HD WALLET HIERARCHY                         │
│                                                              │
│  BIP-39 Mnemonic (12-24 words)                               │
│  ⚠️  PERMANENTLY OFFLINE — never touches the server           │
│       │                                                      │
│       ▼                                                      │
│  BIP-32 Master Key                                           │
│  ⚠️  PERMANENTLY OFFLINE — derived from mnemonic              │
│       │                                                      │
│       ├── XPUB (Extended Public Key)                         │
│       │   ✅ Stored on server — can only DERIVE addresses     │
│       │   ❌ Cannot sign transactions                         │
│       │   ❌ Cannot extract private keys                      │
│       │                                                      │
│       ├── m/44'/60'/0'/0/N   → Ethereum deposit addresses    │
│       │   (also used for Base and Polygon — same key)        │
│       │                                                      │
│       ├── m/44'/0'/0'/0/N    → Bitcoin deposit addresses     │
│       │   (P2WPKH Bech32 format)                             │
│       │                                                      │
│       └── m/44'/195'/0'/0/N  → Tron deposit addresses        │
│           (T-prefix Base58Check format)                       │
│                                                              │
│  Where N = incrementing address index (0, 1, 2, ...)         │
└──────────────────────────────────────────────────────────────┘
```

### Derivation Paths

| Chain Family | BIP-44 Coin Type | Full Path | Address Format |
|-------------|-----------------|-----------|----------------|
| Ethereum (ETH, BASE, POLYGON) | 60 | `m/44'/60'/0'/0/N` | 0x-prefix hex, EIP-55 checksum |
| Bitcoin | 0 | `m/44'/0'/0'/0/N` | Bech32 (bc1...) for mainnet, tb1 for testnet |
| Tron | 195 | `m/44'/195'/0'/0/N` | T-prefix Base58Check |

EVM chains (Ethereum, Base, Polygon) share the same derivation path and therefore the same deposit addresses. The system assigns the same address on all EVM chains and monitors each chain's block monitor independently.

### Zero Key Exposure Model

```
┌──────────────────────────────────────────────────────────┐
│                  SERVER (Merchant VPS)                     │
│                                                          │
│  ✅ XPUB (extended public key)                            │
│     → Used to derive deposit addresses deterministically  │
│     → Cannot derive private keys from XPUB alone          │
│                                                          │
│  ✅ Address index counter                                 │
│     → Tracks which addresses have been generated          │
│     → Stored in deposit_addresses table                   │
│                                                          │
│  ✅ Hot wallet private key (AES-256 encrypted)            │
│     → Contains MINIMAL funds for gas fees only            │
│     → Used to pay gas for sweep transactions              │
│     → RSA encrypted in database                           │
│                                                          │
│  ❌ Master seed / mnemonic — NEVER on server              │
│  ❌ Deposit wallet private keys — NEVER on server         │
│  ❌ Cold storage private keys — NEVER on server           │
└──────────────────────────────────────────────────────────┘
```

### Key Derivation Functions

The blockchain clients implement key derivation through the abstract interface:

```
DeriveKeys(xpub string, index uint32) → (address string, err error)
  - Takes the extended public key and child index
  - Returns the derived address for the specific chain
  - ETH: Uses secp256k1 → Keccak256 → 0x prefix
  - BTC: Uses secp256k1 → HASH160 → Bech32 encoding
  - TRX: Uses secp256k1 → Keccak256 → Base58Check with T prefix

DeriveXPUB(mnemonic string) → (xpub string, err error)
  - Takes BIP-39 mnemonic words
  - Returns the extended public key for the chain's derivation path
  - Only called during initial wallet setup (offline)
```

### Address Pool Pre-Generation

The system maintains a pool of pre-generated addresses to avoid latency during payment creation. The `WalletServiceImpl.GenerateAddressesIfPoolLow()` function manages this:

```
GenerateAddressesIfPoolLow():
  1. Count unused addresses in deposit_addresses table
     WHERE status = 'available' AND blockchain_family_id = X
  2. If count < POOL_THRESHOLD (configurable):
     a. Get current max index from deposit_addresses
     b. For i in range(BATCH_SIZE):
        - Call DeriveKeys(xpub, max_index + i + 1)
        - Insert into deposit_addresses:
          {address, address_lower, index, status: 'available',
           blockchain_family_id, wallet_id}
  3. Log pool maintenance result
```

The pool is checked during the `runPoolMaintenance()` step of each polling cycle in the block monitor (lines 440-452 of `blockchain_processor_impl.go`).

### Private Key Encryption

For the hot wallet (gas fee payer), private keys are stored encrypted in the database:

```
encryptPrivateKey(plaintext string, rsaPublicKey *rsa.PublicKey) → (encrypted string, err error)
  - Located in internal/repository
  - Uses RSA-OAEP encryption with the server's RSA public key
  - Encrypted value stored as base64 in the database

decryptPrivateKey(encrypted string, rsaPrivateKey *rsa.PrivateKey) → (plaintext string, err error)
  - Located in internal/repository
  - Decrypts using the server's RSA private key
  - RSA private key itself is AES-256 encrypted on disk
  - AES key derived from AES_KEY environment variable (32-byte hex)
```

The double-encryption scheme:
1. Hot wallet private key encrypted with RSA public key (stored in DB)
2. RSA private key encrypted with AES-256 (stored on disk)
3. AES key provided as environment variable `AES_KEY`

This means compromising the database alone does not expose private keys. An attacker would need both database access AND the AES key from the environment.

### Wallet Database Schema

```
wallets:
  id                    BIGINT PRIMARY KEY
  blockchain_family_id  BIGINT FK -> blockchain_families.id
  xpub                  TEXT          -- Extended public key
  encrypted_private_key TEXT          -- RSA-encrypted hot wallet key
  salt                  VARCHAR       -- Encryption salt
  master_address        VARCHAR       -- Master/cold wallet address
  fund_sweeper_address  VARCHAR       -- SmartSweep contract address
  created_at            TIMESTAMP
  updated_at            TIMESTAMP
  deleted_at            TIMESTAMP     -- Soft delete

deposit_addresses:
  id                    BIGINT PRIMARY KEY
  wallet_id             BIGINT FK -> wallets.id
  blockchain_family_id  BIGINT FK -> blockchain_families.id
  address               VARCHAR       -- Derived address
  address_lower         VARCHAR       -- Lowercase for case-insensitive matching
  index                 INTEGER       -- BIP-44 derivation index
  status                VARCHAR       -- 'available', 'assigned', 'used'
  payment_request_id    BIGINT FK     -- Linked payment (NULL if available)
  created_at            TIMESTAMP
  updated_at            TIMESTAMP
  deleted_at            TIMESTAMP
```

---

## 5. SmartSweep Contract System

### Overview

SmartSweep is a smart contract system that automatically moves funds from deposit addresses to cold storage. The key security property is that the cold wallet destination is **immutable** — hardcoded at deployment time and cannot be changed, even by the contract owner.

### SmartSweep.sol Contract

```
┌─────────────────────────────────────────────────────────┐
│                  SmartSweep.sol (Solidity)                │
│                                                         │
│  State Variables:                                       │
│  ├── coldWallet: address (IMMUTABLE)                    │
│  │   Set at deployment, cannot be changed               │
│  │   All swept funds go here and ONLY here              │
│  │                                                      │
│  └── owner: address                                     │
│      Can authorize sweepers, but CANNOT change           │
│      cold wallet destination                             │
│                                                         │
│  Functions:                                             │
│  ├── sweep(token: address, amount: uint256)              │
│  │   Transfers specified amount of token from            │
│  │   deposit address → coldWallet                        │
│  │   Only callable by authorized sweeper                 │
│  │   Requires ERC-20 approval first                      │
│  │                                                      │
│  ├── sweepAll(token: address)                            │
│  │   Sweeps entire token balance to coldWallet           │
│  │   Convenience function for full balance sweep         │
│  │                                                      │
│  └── sweepNative()                                      │
│      Sweeps native currency (ETH/MATIC) to coldWallet    │
│                                                         │
│  Events:                                                │
│  ├── Swept(token, amount, from, to)                     │
│  └── NativeSwept(amount, from, to)                      │
│                                                         │
│  Security:                                              │
│  ├── Cold wallet address set at deployment               │
│  ├── Cannot be changed after deployment                  │
│  ├── Even owner cannot redirect funds                    │
│  ├── Audited by QuillAudits                              │
│  └── One contract per deposit address (SCW model)        │
└─────────────────────────────────────────────────────────┘
```

### SCW (Smart Contract Wallet) Deployment Flow

Each deposit address on EVM chains can have an associated Smart Contract Wallet (SCW) deployed for sweep operations:

```
SCW Deployment Flow:
  1. Address is assigned to a payment via deposit_addresses table
  2. BroadcastSCWDepositWalletProcessorJob checks for addresses
     needing SCW deployment
  3. For each address:
     a. BuildArgumentsForDepositWalletDeployment() constructs
        deployment calldata with:
        - coldWallet = merchant's configured cold storage address
        - owner = hot wallet address (for authorization)
     b. Factory contract deploys new SCW via CREATE2
        (deterministic address from salt + bytecode)
     c. Deployment transaction broadcasted via hot wallet
     d. address_deployments table tracks deployment status:
        pending → broadcasted → confirmed
  4. ValidatePendingAndBroadcastedAddressDeployments() monitors
     deployment confirmations
  5. ProcessAddressDeploymentsAccounting() records gas costs
```

The factory contract addresses are tracked in `blockchain_contracts` table and retrieved via `getFactoryContractAddresses()` during block processing.

### Sweep Scheduling and Execution

The AccountProcessorJob handles sweep scheduling with separate methods for each chain type:

```
AccountProcessorJob.StartAccountProcessor():
  Runs 9 goroutines on configurable intervals:
  
  1. processETHAutoSweep()
     - Checks EVM deposit address balances
     - If balance > min_amount_in_sweep_tx:
       a. Build sweep transaction calling SmartSweep.sweep()
       b. Sign with hot wallet (gas payer)
       c. Broadcast transaction
       d. Create sweep_transactions record (status: initiated)
  
  2. processBitcoinSweeps()
     - Collects UTXOs from deposit addresses
     - Creates consolidated sweep transaction
     - Handles BTC-specific fee calculation
  
  3. processERC20Sweeps()
     - For each ERC-20 token on deposit addresses:
       a. Check token balance
       b. If balance > threshold:
          - Ensure ERC-20 approval exists
          - Call SmartSweep.sweep(token, amount)
  
  4. createSweepTransactionPayload()
     - Constructs the sweep transaction data
     - Calculates gas/fee estimates
  
  5. processWithdrawals()
     - Processes merchant-initiated withdrawals
  
  6. processRewards() / processFailedRewards()
     - Handles referral reward distributions
  
  7. createAllPendingUTXOsAndAccountForSweepTransaction()
     - BTC sweep UTXO aggregation
  
  8. retryStaleBTCSweepTransactions()
     - Retries failed/stuck BTC sweeps
  
  9. processStaleInitiatedSweeps()
     - Handles sweeps stuck in initiated state
     - processConfirmingBTCSweeps() — monitors BTC sweep confirmations
```

### ERC-20 Approval Flow

Before an ERC-20 token can be swept by the SmartSweep contract, the deposit address must approve the contract to spend tokens:

```
ERC-20 Sweep Approval Flow (erc20_sweep_approval_processor.go):

  SweepApprovalProcessorJob:
  1. Query blockchain_currencies for tokens needing approval
     WHERE approval_fee_amount > 0
  2. For each deposit address with token balance:
     a. Check if approval already exists
        (address_contract_signatures table)
     b. If no approval:
        - Build ERC-20 approve() transaction:
          approve(smartSweepContract, MAX_UINT256)
        - Sign with deposit address private key
          (derived from master key — offline operation)
        - Broadcast approval transaction
        - Track in address_contract_signatures:
          {address, contract_type, status: pending}
     c. Monitor approval confirmation
     d. Update status: pending → confirmed
  3. Once approved, the SmartSweep contract can call
     transferFrom() on the token contract
```

### Post-Sweep Accounting

After a sweep completes, the `addressesBalancesAccountingAfterSweep()` function (in `sweep_approval_processor.go`) performs double-entry accounting:

```
addressesBalancesAccountingAfterSweep():
  1. Verify sweep transaction confirmed on-chain
  2. Update deposit address balance:
     account_address_balances.balance -= swept_amount
  3. Update cold wallet balance:
     cold_wallet_balance += swept_amount
  4. Record internal blockchain transaction:
     internal_blockchain_transactions {
       from_address, to_address, amount, tx_hash,
       transaction_type: 'sweep', status: 'confirmed'
     }
  5. Update sweep_transactions status:
     initiated → confirmed
  6. Deduct gas fee from hot wallet balance accounting
  7. Apply fee percentage if configured:
     sweep_transactions.fee_percentage
```

### Sweep Database Schema

```
sweep_transactions:
  id                    BIGINT PRIMARY KEY
  wallet_id             BIGINT FK -> wallets.id
  blockchain_id         BIGINT FK -> blockchains.id
  currency_id           BIGINT FK -> currencies.id
  from_address          VARCHAR
  to_address            VARCHAR       -- Always the cold wallet
  amount                NUMERIC(38,18)
  tx_hash               VARCHAR
  status                VARCHAR       -- initiated, confirming, confirmed, failed
  fee_amount            NUMERIC(38,18) -- Gas fee paid
  fee_percentage        NUMERIC(5,4)  -- Platform fee percentage
  confirming_blocknumber BIGINT       -- Block where confirming started
  created_at            TIMESTAMP
  updated_at            TIMESTAMP
  deleted_at            TIMESTAMP

sweeps:
  id                    BIGINT PRIMARY KEY
  blockchain_currency_id BIGINT FK
  deposit_address_id    BIGINT FK
  amount                NUMERIC(38,18)
  status                VARCHAR       -- pending, processing, completed, failed
  confirming_blocknumber BIGINT
  created_at            TIMESTAMP
  updated_at            TIMESTAMP

sweep_utxos:
  id                    BIGINT PRIMARY KEY
  sweep_transaction_id  BIGINT FK -> sweep_transactions.id
  utxo_id               BIGINT FK -> utxos.id
  status                VARCHAR       -- pending, spent
  created_at            TIMESTAMP
```

---

## 6. Block Monitoring System

### Architecture Overview

The block monitoring system runs as a set of per-chain goroutines within the Go backend process. Each chain has a dedicated processor that polls for new blocks, extracts relevant transactions, and processes deposits.

```
┌─────────────────────────────────────────────────────────────┐
│               BLOCK MONITORING SYSTEM                        │
│                                                             │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  blockchainProcessorImpl (shared base)                  │ │
│  │  File: blockchain_processor_impl.go (~2500 LOC)         │ │
│  │                                                        │ │
│  │  Core Loop:                                            │ │
│  │  CreateBlockchainProcessor() → initializeClient()       │ │
│  │  → Start() → startPolling() → polling() [loop]         │ │
│  │       │                                                │ │
│  │       ├── GetLatestBlockNumber()                        │ │
│  │       ├── ProcessBlockRange(lastScanned, latest)        │ │
│  │       │   └── ProcessSingleBlock(blockNumber)           │ │
│  │       │       ├── buildProcessBlockConfig()             │ │
│  │       │       ├── client.ProcessBlock(config)           │ │
│  │       │       ├── handleExtractedDeposit() [407 LOC]    │ │
│  │       │       ├── handleSweepEvent() [75 LOC]           │ │
│  │       │       └── handleFactoryEvent() [106 LOC]        │ │
│  │       ├── processConfirmingDeposits()                   │ │
│  │       ├── processConfirmingSweeps()                     │ │
│  │       ├── runPoolMaintenance()                          │ │
│  │       ├── updateBlockTimeHistory()                      │ │
│  │       └── calculatePollingInterval() [63 LOC]           │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  Chain-Specific Listeners (cmd/ entry points):              │
│  ┌──────────────────┐ ┌──────────────────┐                  │
│  │ eth_erc20_       │ │ base_erc20_      │                  │
│  │ processor.go     │ │ processor.go     │                  │
│  │ StartEtherAnd    │ │ StartBaseAnd     │                  │
│  │ ERC20Listener    │ │ ERC20Listener    │                  │
│  └──────────────────┘ └──────────────────┘                  │
│  ┌──────────────────┐ ┌──────────────────┐                  │
│  │ polygon_erc20_   │ │ trx_trc20_       │                  │
│  │ processor.go     │ │ processor.go     │                  │
│  │ StartPolygonAnd  │ │ StartTronAnd     │                  │
│  │ ERC20Listener    │ │ TRC20Listener    │                  │
│  └──────────────────┘ └──────────────────┘                  │
│  ┌──────────────────┐                                       │
│  │ bitcoin_         │                                       │
│  │ processor.go     │                                       │
│  │ StartBitcoin     │                                       │
│  │ Listener         │                                       │
│  └──────────────────┘                                       │
└─────────────────────────────────────────────────────────────┘
```

### Per-Chain Polling Loop

The core polling loop in `blockchainProcessorImpl.polling()` (lines 305-440, 135 LOC):

```
polling():
  1. defer: catch panics, log, restart after delay
  
  2. latestBlock = client.GetLatestBlockNumber()
  
  3. if latestBlock > lastScannedBlock:
     a. blockRange = min(latestBlock - lastScannedBlock, MAX_BATCH_SIZE)
     b. ProcessBlockRange(lastScannedBlock + 1, lastScannedBlock + blockRange)
     c. Update lastScannedBlock in database:
        blockchains.block_height = newBlockHeight
  
  4. processConfirmingDeposits()
     - Re-check deposits in CONFIRMING state
     - If confirmations >= threshold: promote to CONFIRMED
     - verifyAndUpsertDepositsForTxHash() [183 LOC]
  
  5. processConfirmingSweeps() [161 LOC]
     - Re-check sweep transactions in CONFIRMING state
     - If confirmed: update accounting
  
  6. runPoolMaintenance()
     - GenerateAddressesIfPoolLow()
  
  7. updateBlockTimeHistory()
     - Track recent block timestamps for interval calculation
  
  8. pollingInterval = calculatePollingInterval()
     - Dynamic based on recent block times
     - Faster when blocks are being produced quickly
     - Slower during quiet periods
  
  9. Sleep(pollingInterval)
  10. Goto 2
```

### Dynamic Polling Interval

The `calculatePollingInterval()` function (lines 1211-1274, 63 LOC) adjusts the polling frequency based on recent block production:

```
calculatePollingInterval():
  1. Collect last N block timestamps from history
  2. Calculate average block time from recent blocks
  3. If hasConfirmingDeposits():
     - Use FASTER interval (more frequent polling to update confirmation counts)
  4. If no recent blocks produced:
     - Use SLOWER interval (avoid wasting RPC calls)
  5. Clamp to MIN_INTERVAL / MAX_INTERVAL bounds
  6. Return calculated interval
```

This ensures efficient RPC usage: polling every few seconds during active payment periods, but backing off to longer intervals when the system is idle.

### handleExtractedDeposit — Core Deposit Detection (407 LOC)

The `handleExtractedDeposit()` function (lines 777-1184) is the heart of the payment detection system. It processes every transaction extracted from a block:

```
handleExtractedDeposit(extractedDeposit):
  1. FILTER: Is the to_address one of our deposit addresses?
     - identifyOurAddresses() checks against deposit_addresses table
     - Uses address_lower for case-insensitive matching
     - Skips system addresses (hot wallet, cold wallet)
     - Skips blacklisted addresses (loadBlacklistedAddresses())
  
  2. CHECK DUPLICATES:
     - Has this tx_hash + log_index already been processed?
     - If yes: skip (idempotent processing)
  
  3. BUILD DEPOSIT DATA:
     - buildDepositDataFromExtracted() [271 LOC]
     - Resolves: blockchain_currency, amount, precision
     - Determines deposit type: NATIVE, ERC20, UTXO
     - Validates amount >= min_deposit_amount
  
  4. CHECK FOR ENTRYPOINT (ERC-4337):
     - If transaction went through ERC-4337 EntryPoint contract
       (0x4337...08), mark as "payments app" transaction
     - Call PAYMENTS_APP_SERVER_URL to fetch sponsorship data
  
  5. CREATE/UPDATE DEPOSIT:
     - Insert into deposits table:
       {address, tx_hash, amount, blockchain_currency_id,
        status: 'confirming', block_number, log_index}
     - verifyAndUpsertDepositsForTxHash() handles upsert logic
  
  6. LINK TO PAYMENT:
     - linkDepositToOpenPaymentRequest()
       (in BlockchainRepositoryImpl)
     - Matches deposit address to payment_requests
     - Updates payment_request status:
       OPEN → CONFIRMING (if confirmations < threshold)
       OPEN → FILLED (if confirmations >= threshold)
  
  7. DETERMINE FILL STATE:
     - Compare deposit amount vs payment_request amount:
       - deposit >= requested: FILLED
       - deposit < requested: PARTIALLY_FILLED
       - deposit > requested: OVER_FILLED
  
  8. HANDLE SWEEP EVENTS:
     - handleSweepEvent() [75 LOC]
     - If the transaction is a sweep contract event:
       Update sweep_transactions status
  
  9. HANDLE FACTORY EVENTS:
     - handleFactoryEvent() [106 LOC]
     - If the transaction is a factory deployment:
       Update address_deployments status
  
  10. LOG AND CONTINUE
```

### Block Range Processing for Catch-Up

When the system starts or recovers from downtime, `ProcessBlockRange()` (lines 461-600, 139 LOC) processes all missed blocks:

```
ProcessBlockRange(fromBlock, toBlock):
  1. Validate range: toBlock >= fromBlock
  2. For blockNum in range(fromBlock, toBlock):
     a. ProcessSingleBlock(blockNum)
     b. If error:
        - Log error
        - Continue to next block (don't halt)
        - Track missed blocks for RecoverMissedDeposits()
  3. Update blockchain.block_height after each successful block
  4. On startup: processes ALL blocks since last shutdown
```

The `RecoverMissedDeposits()` function (lines 1288-1358, 70 LOC) provides a secondary recovery mechanism:
- Queries `missed_deposits` table for transactions that may have been missed
- Re-processes affected blocks
- Cross-references with on-chain data to verify completeness

### Listener Entry Points

Each chain has a dedicated entry point function that configures and starts the blockchain processor:

```
StartEtherAndERC20Listener (lines 2455-2461):
  - Creates ETHBlockchainClient with Ethereum mainnet/Sepolia config
  - Starts blockchainProcessorImpl with ETH-specific parameters

StartBaseAndERC20Listener (via EtherAndERC20ProcessorJob, lines 2477-2516):
  - Creates ETHBlockchainClient with Base mainnet/Sepolia config
  - Same EVM processing logic, different chain ID and RPC nodes

StartPolygonAndERC20Listener (similar pattern):
  - Creates ETHBlockchainClient with Polygon config

StartTronAndTRC20Listener (lines 2461-2467):
  - Creates TRXBlockchainClient with TronGrid/Nile config
  - Tron-specific block processing

StartBitcoinListener (lines 2467-2477):
  - Creates BTCBlockchainClient
  - Includes additional UTXO processing and BTC sweep handling
  - Starts withdrawAuditProcessor as a sub-goroutine
```

### Background Worker Process Table

All 10 worker processes launched from `cmd/root.go`:

| Worker | File | Function | Purpose |
|--------|------|----------|---------|
| Account Processor | `account_processor.go` | `startAccountProcessorCmd` | Sweep scheduling, withdrawals, rewards |
| ETH + ERC20 Listener | `eth_erc20_processor.go` | `startEtherAndERC20ListenerCmd` | Ethereum block monitoring |
| Base + ERC20 Listener | `base_erc20_processor.go` | `startBaseAndERC20ListenerCmd` | Base L2 block monitoring |
| Polygon + ERC20 Listener | `polygon_erc20_processor.go` | `startPolygonAndERC20ListenerCmd` | Polygon block monitoring |
| Bitcoin Processor | `bitcoin_processor.go` | `startBitcoinProcessorCmd` | Bitcoin block + UTXO monitoring |
| Deposit Processor | `deposit_processor.go` | `startDepositProcessorCmd` | Deposit confirmation processing |
| Webhook Processor | `webhook_processor.go` | `startWebhookProcessorCmd` | Webhook delivery + retries |
| Email Processor | `email_processor.go` | `startEmailProcessorCmd` | Email notifications |
| TRX + TRC20 Listener | `trx_trc20_processor.go` | `startTronAndTRC20ListenerCmd` | Tron block monitoring |
| ERC20 Sweep Approval | `erc20_sweep_approval_processor.go` | `startSweepApprovalListenerCmd` | ERC-20 approval management |
| Seeder/Initialiser | `seeder_intialise_processor.go` | `startSeederInitialiseCmd` | Database seeding |
| SCW Broadcaster | `broadcast_scw_deposit_wallet_processor.go` | `startBroadcastSCWDepositWalletProcessorCmd` | Smart contract wallet deployment |
| Duplicate Deposits | `accounting_duplicate_deposits_processor.go` | `startAccountingDuplicateDepositsCmd` | Duplicate deposit accounting |

---

## 7. RPC Node Management

### Multi-Provider Strategy

Payminto uses a multi-provider RPC node strategy for reliability and uptime. Each blockchain can have multiple RPC endpoints configured with priority-based selection and automatic failover.

```
┌─────────────────────────────────────────────────────────────┐
│                 RPC NODE MANAGEMENT                          │
│                                                             │
│  Per-Blockchain Node Pool:                                  │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Ethereum (10+ configurable nodes):                  │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐            │    │
│  │  │ Self-    │ │ Alchemy  │ │ Infura   │ ...        │    │
│  │  │ hosted   │ │ (backup) │ │ (backup) │            │    │
│  │  │ (primary)│ │          │ │          │            │    │
│  │  │ Priority │ │ Priority │ │ Priority │            │    │
│  │  │    1     │ │    2     │ │    3     │            │    │
│  │  └──────────┘ └──────────┘ └──────────┘            │    │
│  │                                                     │    │
│  │  Selection: Try highest priority first,             │    │
│  │  failover to next on timeout/error                  │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Bitcoin:                                            │    │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐            │    │
│  │  │ Bitcoin  │ │Blockstream│ │ Mempool  │            │    │
│  │  │ Core RPC │ │ API      │ │ API      │            │    │
│  │  │ (primary)│ │ (fallback)│ │ (fallback)│           │    │
│  │  └──────────┘ └──────────┘ └──────────┘            │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  Tron:                                               │    │
│  │  ┌──────────┐ ┌──────────┐                          │    │
│  │  │ TronGrid │ │ TronGrid │                          │    │
│  │  │ Mainnet  │ │ Nile     │                          │    │
│  │  │ (primary)│ │ (testnet)│                          │    │
│  │  └──────────┘ └──────────┘                          │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

### Per-Chain RPC Configurations

The system supports extensive RPC configuration stored in the `rpc_nodes` table:

```
rpc_nodes:
  id                    BIGINT PRIMARY KEY
  blockchain_id         BIGINT FK -> blockchains.id
  url                   VARCHAR       -- RPC endpoint URL
  name                  VARCHAR       -- Human-readable name
  is_active             BOOLEAN       -- Enable/disable this node
  priority              INTEGER       -- Selection order (lower = preferred)
  is_preferred          BOOLEAN       -- Preferred for certain operations
  auth_header           VARCHAR       -- Optional auth header (API key)
  chain_id              BIGINT        -- Expected chain ID for validation
  max_batch_size        INTEGER       -- Max batch request size
  timeout_ms            INTEGER       -- Request timeout in milliseconds
  created_at            TIMESTAMP
  updated_at            TIMESTAMP
  deleted_at            TIMESTAMP
```

Typical configurations per chain:

**Ethereum (10+ nodes possible):**
- Self-hosted Geth/Erigon (primary)
- Alchemy mainnet/Sepolia
- Infura mainnet/Sepolia
- QuickNode mainnet
- Ankr public RPC (low priority fallback)

**Base:**
- Alchemy Base mainnet/Sepolia
- Base public RPC (Coinbase-operated)
- QuickNode Base

**Polygon:**
- Alchemy Polygon mainnet
- Polygon public RPC
- QuickNode Polygon

**Bitcoin:**
- Bitcoin Core RPC (self-hosted)
- Blockstream API (REST fallback)
- Mempool.space API (REST fallback)

**Tron:**
- TronGrid mainnet API
- TronGrid Nile testnet API
- Self-hosted Tron Full Node (optional)

### CheckAllNodeConnections (508 LOC)

The `BlockchainServiceImpl.CheckAllNodeConnections()` function (lines 457-965) performs comprehensive health checks across all configured RPC nodes:

```
CheckAllNodeConnections():
  For each blockchain in database:
    1. Get all active RPC nodes for this blockchain
    
    2. For each RPC node:
       a. Create temporary blockchain client
       b. Call TestNodeConnection():
          - Attempts to fetch latest block number
          - Measures response time
          - Validates chain ID matches expected
       c. Record result: success/failure, latency, error
    
    3. checkChainMismatch():
       - Compare chain_id from node vs expected chain_id
       - Flag nodes returning wrong chain (e.g., mainnet node
         configured as testnet)
    
    4. fetchChainIdentifier():
       - Get the actual chain identifier from each node
       - Used for chain ID validation
    
    5. CheckServerURLConnection():
       - Verify the Payminto server URL is accessible
       - Used for webhook callback validation
    
    6. Aggregate results:
       - Per-node status (connected/disconnected/chain_mismatch)
       - Overall blockchain health (all_nodes_ok/some_failed/all_failed)
       - Response time percentiles
    
    7. Return comprehensive health report
```

This function is exposed via the dashboard for merchants to monitor their node health.

### ValidateActiveNodesChainID

The `validateActiveNodesChainID()` function (lines 882-932, 50 LOC) ensures all active nodes are on the correct network:

```
validateActiveNodesChainID():
  For each active RPC node:
    1. Call client.GetChainID() through the node
    2. Compare returned chain_id with expected:
       - Ethereum mainnet: 1
       - Ethereum Sepolia: 11155111
       - Base mainnet: 8453
       - Base Sepolia: 84532
       - Polygon mainnet: 137
    3. If mismatch:
       - Log warning
       - Mark node as inactive (prevent usage)
       - Alert via system notification
    4. If match:
       - Confirm node is valid
       - Update last_validated timestamp
```

### Node Priority and Selection

RPC nodes are selected using a priority-based system with automatic failover:

```
enrichRPCNodesWithPriority() (lines 932-965):
  1. Sort nodes by priority field (ascending)
  2. Assign priority slots:
     - Priority 1: Primary (used for all operations)
     - Priority 2-3: Secondary (used when primary fails)
     - Priority 4+: Tertiary (last resort fallback)

assignPriorityToNodes() (lines 969-1019):
  1. For preferred operations (connPreferred in ETH client):
     - Select nodes where is_preferred = true
     - Used for sensitive operations like broadcasting
  2. For general operations (conn in ETH client):
     - Use highest priority available node
     - On timeout/error: try next priority level
     - If all fail: return error with last failure details

reloadRPCPool() (lines 848-876):
  - Called after RPC node CRUD operations
  - Rebuilds the in-memory connection pool
  - Validates new configuration before activating
```

### Failover Behavior

When an RPC call fails, the system follows this failover pattern:

```
ethExec(operation) / btcExec(operation) / tronExec(operation):
  1. Select primary node (highest priority active node)
  2. Execute operation with timeout
  3. If success: return result
  4. If timeout or connection error:
     a. Log warning with node URL and error
     b. Mark node as temporarily unhealthy
     c. Select next priority node
     d. Retry operation
     e. If all nodes exhausted: return error
  5. If chain-level error (e.g., "block not found"):
     - Do NOT failover (error is legitimate)
     - Return error to caller
```

---

## 8. Payment Detection Flow

### End-to-End Flow

```
┌─────────┐    ┌──────────┐    ┌──────────┐    ┌──────────────┐    ┌───────────┐
│Customer  │    │Payment   │    │Go API    │    │Block Monitor │    │Merchant   │
│Browser   │    │Widget    │    │Gateway   │    │(per chain)   │    │Backend    │
└────┬─────┘    └────┬─────┘    └────┬─────┘    └──────┬───────┘    └─────┬─────┘
     │               │               │                 │                  │
     │  Select amount│               │                 │                  │
     │──────────────>│               │                 │                  │
     │               │               │                 │                  │
     │               │ POST /payment │                 │                  │
     │               │──────────────>│                 │                  │
     │               │               │                 │                  │
     │               │               │ Create payment_request             │
     │               │               │ (status: OPEN)  │                  │
     │               │               │                 │                  │
     │               │ {ref_id, url} │                 │                  │
     │               │<──────────────│                 │                  │
     │               │               │                 │                  │
     │  Select chain │               │                 │                  │
     │──────────────>│               │                 │                  │
     │               │               │                 │                  │
     │               │ POST /deposit-│                 │                  │
     │               │ address/{ref} │                 │                  │
     │               │──────────────>│                 │                  │
     │               │               │                 │                  │
     │               │               │ Assign address  │                  │
     │               │               │ from pool       │                  │
     │               │               │ (deposit_       │                  │
     │               │               │  addresses)     │                  │
     │               │               │                 │                  │
     │               │ {address, QR} │                 │                  │
     │               │<──────────────│                 │                  │
     │               │               │                 │                  │
     │  Show QR code │               │                 │                  │
     │<──────────────│               │                 │                  │
     │               │               │                 │                  │
     │  Send crypto  │               │                 │                  │
     │  from wallet  ─────────────────────────────────>│ ON-CHAIN TX      │
     │               │               │                 │                  │
     │               │               │                 │ Block monitor    │
     │               │               │                 │ detects tx       │
     │               │               │                 │                  │
     │               │               │ handleExtracted │                  │
     │               │               │ Deposit()       │                  │
     │               │               │<────────────────│                  │
     │               │               │                 │                  │
     │               │               │ Create deposit  │                  │
     │               │               │ (CONFIRMING)    │                  │
     │               │               │                 │                  │
     │               │               │ Link to payment │                  │
     │               │               │ request         │                  │
     │               │               │                 │                  │
     │  Poll /payment│               │                 │                  │
     │  /reference/  │               │                 │                  │
     │  {ref}        │               │                 │                  │
     │──────────────>│──────────────>│                 │                  │
     │               │               │ status:         │                  │
     │  "Confirming" │               │ CONFIRMING      │                  │
     │<──────────────│<──────────────│                 │                  │
     │               │               │                 │                  │
     │               │               │                 │ N confirmations  │
     │               │               │                 │ reached          │
     │               │               │                 │                  │
     │               │               │ Update deposit: │                  │
     │               │               │ CONFIRMED       │                  │
     │               │               │                 │                  │
     │               │               │ Update payment: │                  │
     │               │               │ FILLED          │                  │
     │               │               │                 │                  │
     │               │               │ Fire webhook ──────────────────────>│
     │               │               │                 │                  │
     │  Poll /payment│               │                 │                  │
     │──────────────>│──────────────>│                 │                  │
     │               │               │ status: FILLED  │                  │
     │  "Confirmed!" │               │                 │                  │
     │<──────────────│<──────────────│                 │                  │
     │               │               │                 │                  │
     │               │               │                 │ SmartSweep       │
     │               │               │                 │ → cold storage   │
```

### Payment Status State Machine

```
                    ┌──────────────┐
                    │              │
                    │    OPEN      │ ← Payment created, waiting for deposit
                    │              │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
     ┌────────────┐ ┌──────────┐ ┌──────────┐
     │            │ │          │ │          │
     │ CANCELLED  │ │CONFIRMING│ │ EXPIRED  │
     │            │ │          │ │          │
     └────────────┘ └────┬─────┘ └──────────┘
                         │
            ┌────────────┼──────────────┐
            │            │              │
            ▼            ▼              ▼
   ┌──────────────┐ ┌──────────┐ ┌───────────────┐
   │              │ │          │ │               │
   │ PARTIALLY_   │ │  FILLED  │ │  OVER_FILLED  │
   │ FILLED       │ │          │ │               │
   │              │ │          │ │               │
   └──────────────┘ └────┬─────┘ └───────┬───────┘
                         │               │
                         ▼               ▼
                    ┌──────────┐    ┌──────────┐
                    │          │    │          │
                    │  SWEPT   │    │  SWEPT   │
                    │          │    │          │
                    └──────────┘    └──────────┘
```

**State definitions:**

| State | Description | Trigger |
|-------|-------------|---------|
| OPEN | Payment created, no deposit detected | POST /api/v1/payment |
| CONFIRMING | Deposit detected, waiting for confirmations | Block monitor detects matching tx |
| CANCELLED | Payment cancelled by merchant or customer | Manual cancellation or timeout |
| EXPIRED | Payment expired without sufficient deposit | Expiration timer elapsed |
| FILLED | Deposit amount matches or exceeds requested amount, confirmed | Confirmations >= threshold |
| PARTIALLY_FILLED | Deposit amount less than requested, confirmed | Confirmations >= threshold, amount < requested |
| OVER_FILLED | Deposit amount exceeds requested amount, confirmed | Confirmations >= threshold, amount > requested |
| SWEPT | Funds moved to cold storage via SmartSweep | Sweep transaction confirmed |

### Frontend Polling

The payment page uses HTTP polling (not WebSocket) to check payment status:

```javascript
// Poll every 5 seconds while payment is OPEN or CONFIRMING
const shouldPoll = paymentState === "OPEN" || isConfirming;
useInterval(fetchPaymentStatus, shouldPoll ? 5000 : null);

// fetchPaymentStatus:
// GET /api/v1/payment/reference/{reference_id}
// Returns: { paymentState, amount, currency, deposits[], ... }
```

The payment page shows progressive status updates:
1. "Waiting for payment..." (OPEN)
2. "Transaction spotted! Waiting for confirmations..." (CONFIRMING)
3. "Payment Successful!" (FILLED)

### linkDepositToOpenPaymentRequest Logic

The `linkDepositToOpenPaymentRequest()` function in `BlockchainRepositoryImpl` connects detected deposits to existing payment requests:

```
linkDepositToOpenPaymentRequest(deposit):
  1. Query payment_requests:
     WHERE deposit_address_id = deposit.deposit_address_id
     AND status IN ('OPEN', 'CONFIRMING')
     ORDER BY created_at ASC
     LIMIT 1
  
  2. If no matching payment request:
     - Deposit is "unlinked" — recorded but not associated with a payment
     - Appears in dashboard as unmatched deposit
     - Can be manually linked later
  
  3. If matching payment request found:
     a. Link deposit to payment:
        deposit.payment_request_id = payment_request.id
     b. Calculate fill state:
        total_deposited = SUM(deposits.amount WHERE payment_request_id = X)
        requested_amount = payment_request.amount_in_crypto
        
        if total_deposited >= requested_amount:
          payment_request.status = 'FILLED'
        elif total_deposited > 0:
          payment_request.status = 'PARTIALLY_FILLED'
        
        if total_deposited > requested_amount * 1.01:
          payment_request.status = 'OVER_FILLED'
     
     c. If FILLED or OVER_FILLED and confirmations >= threshold:
        - Trigger webhook delivery
        - Queue sweep scheduling
```

---

## 9. UTXO Management (Bitcoin)

### Overview

Bitcoin uses the UTXO (Unspent Transaction Output) model instead of an account balance model. This requires specialized handling for deposit detection, balance tracking, and sweep construction.

### UTXO Identification

```
identifyOurUTXOs(utxosInBlock):
  For each UTXO output in the block:
    1. isOurUTXO(utxo):
       a. Query deposit_addresses:
          WHERE address_lower = utxo.to_address.lower()
          AND blockchain_family_id = BITCOIN_FAMILY_ID
       b. Also check: is this a known system address?
          (hot wallet, cold wallet, sweep change address)
       c. If match found: return true
       d. If no match: skip (not our UTXO)
    
    2. If isOurUTXO returns true:
       a. createUTXOEntry(utxo):
          Insert into utxos table:
          {
            tx_hash, output_index, address,
            amount (in BTC), script_pubkey,
            block_number, status: 'unspent',
            deposit_address_id
          }
       b. Process as deposit (same as other chains)
```

### UTXO Database Schema

```
utxos:
  id                    BIGINT PRIMARY KEY
  tx_hash               VARCHAR       -- Transaction hash
  output_index          INTEGER       -- Output index in transaction
  address               VARCHAR       -- Bitcoin address
  amount                NUMERIC(38,18) -- BTC amount
  script_pubkey         TEXT          -- Locking script
  block_number          BIGINT        -- Block where UTXO was created
  status                VARCHAR       -- unspent, spent, pending_sweep
  deposit_address_id    BIGINT FK     -- Link to deposit_addresses
  sweep_transaction_id  BIGINT FK     -- Link to sweep (when spent)
  created_at            TIMESTAMP
  updated_at            TIMESTAMP
  deleted_at            TIMESTAMP
```

### BTC Sweep Transaction Construction

Bitcoin sweep transactions consolidate multiple UTXOs into a single output to the cold wallet:

```
queueBTCSweepTransaction(depositAddressId):
  1. Collect all unspent UTXOs for the deposit address:
     SELECT * FROM utxos
     WHERE deposit_address_id = X
     AND status = 'unspent'
  
  2. Calculate total input amount:
     totalInput = SUM(utxo.amount)
  
  3. Estimate fee:
     feeRate = FeeSatoshiPerByte()  -- Current network fee rate
     txSize = EstimateTransactionSize(numInputs, 1)  -- 1 output (cold wallet)
     fee = feeRate * txSize
  
  4. Calculate sweep amount:
     sweepAmount = totalInput - fee
  
  5. If sweepAmount < dust threshold: skip (not worth sweeping)
  
  6. Create sweep_transactions record:
     {status: 'pending', amount: sweepAmount, fee_amount: fee}
  
  7. Mark UTXOs as pending_sweep:
     UPDATE utxos SET status = 'pending_sweep'
     WHERE id IN (selected_utxo_ids)

handleBTCSweepTransaction(sweepTx):
  1. Fetch all pending_sweep UTXOs for this sweep transaction
  
  2. Build raw Bitcoin transaction:
     Inputs: all pending_sweep UTXOs (with signing data)
     Output: cold_wallet_address, sweepAmount
     (No change output — entire balance minus fee goes to cold wallet)
  
  3. Sign transaction:
     - For each input: sign with the deposit address private key
     - Private keys derived from master seed (offline signing)
     - Or: using hot wallet if keys are available
  
  4. Broadcast: BroadcastRawTransaction(signedTxHex)
  
  5. Update sweep_transactions:
     status: 'pending' → 'initiated'
     tx_hash: broadcasted transaction hash
  
  6. Monitor confirmation:
     processConfirmingBTCSweeps() checks periodically
     When confirmed:
       - UpdateUTXOsStatusToSpent()
       - Update sweep_transactions status: 'confirmed'
       - Record accounting entries
```

### BTC-Specific Tracking Functions

```
WithdrawDepositsBTC:
  - Handles merchant-initiated BTC withdrawals
  - Selects optimal UTXOs for the withdrawal amount
  - Constructs and broadcasts withdrawal transaction
  - Tracked via WithdrawDeposits handler in BlockchainHandler

UpdateUTXOsStatusToSpent:
  - Called after a sweep or withdrawal transaction confirms
  - Marks all input UTXOs as 'spent'
  - Updates sweep_utxos join table

createAllPendingUTXOsAndAccountForSweepTransaction:
  - Aggregates UTXOs across multiple deposit addresses
  - Creates a consolidated sweep transaction
  - Handles UTXO selection optimization (minimize inputs for lower fees)

retryStaleBTCSweepTransactions:
  - Finds sweep transactions stuck in 'initiated' state
  - Checks if the transaction was actually broadcast
  - Re-broadcasts if not found in mempool
  - Escalates fee if needed (RBF — Replace-By-Fee)

processConfirmingBTCSweeps:
  - Monitors BTC sweep transactions waiting for confirmation
  - Checks confirmation count via GetTransactionReceipt
  - Promotes to 'confirmed' when threshold reached
```

---

## 10. Transaction Signing and Broadcasting

### SignTransaction (Per Chain)

Each blockchain client implements transaction signing differently:

**Ethereum (ETH/Base/Polygon):**
```
SignTransaction(params):
  1. Build transaction object:
     - To: destination address
     - Value: amount in wei
     - GasLimit: estimated gas
     - GasFeeCap: max fee per gas (EIP-1559)
     - GasTipCap: max priority fee (EIP-1559)
     - Nonce: from NonceService
     - ChainID: chain-specific
  
  2. ABI-encode calldata (if contract call)
  
  3. Sign with private key:
     - Decrypt hot wallet key (decryptPrivateKey)
     - Use go-ethereum's crypto.Sign()
     - Produce v, r, s signature components
  
  4. Return signed transaction bytes (RLP-encoded)
  
  Note: 6 nested functions handle complex signing with
  retry logic, fee escalation, and error recovery
```

**Bitcoin:**
```
SignTransaction(params):
  1. Build unsigned transaction:
     - Inputs: UTXO references (txid + vout)
     - Outputs: destination + amount, change address + change amount
  
  2. For each input:
     - Derive private key for the input's address
     - Create signature hash (SIGHASH_ALL)
     - Sign with secp256k1
     - Attach witness data (SegWit)
  
  3. Serialize signed transaction to hex
  
  4. Return raw transaction hex
```

**Tron:**
```
SignTransaction(params):
  1. Build Tron transaction protobuf:
     - Contract: TransferContract (TRX) or TriggerSmartContract (TRC-20)
     - Expiration: current time + 10 minutes
     - Timestamp: current time
     - FeeLimit: estimated energy cost in sun
  
  2. Compute transaction hash
  
  3. Sign with secp256k1 (same curve as Ethereum)
  
  4. Return signed transaction protobuf
```

### SignSmartContractTransaction

For contract calls (sweep, ERC-20 approval, factory deployment):

```
SignSmartContractTransaction(contractAddress, methodName, params):
  ETH/Base/Polygon:
    1. ABI-encode the method call:
       - Function selector: keccak256(methodSignature)[:4]
       - Encoded parameters per ABI spec
    2. Estimate gas via eth_estimateGas
    3. Build and sign transaction with calldata
    4. 3 nested functions handle: ABI encoding, gas estimation, signing

  Tron:
    1. Build TriggerSmartContract:
       - contract_address: hex address
       - function_selector: method signature
       - parameter: ABI-encoded params
    2. Set energy/bandwidth limits
    3. Sign and return
    4. 4 nested functions including error recovery
```

### BroadcastRawTransaction

```
BroadcastRawTransaction(signedTxBytes):
  ETH:
    1. Call eth_sendRawTransaction via preferred RPC node
    2. Return transaction hash
    3. If node returns error:
       - If "nonce too low": nonce conflict, retry with new nonce
       - If "replacement transaction underpriced": bump gas
       - If "already known": transaction already in mempool (OK)

  BTC:
    1. Call sendrawtransaction via Bitcoin Core RPC
    2. Or: POST to Blockstream/Mempool API
    3. Return transaction hash

  TRX:
    1. Call wallet/broadcasttransaction via TronGrid
    2. Or: broadcast via gRPC
    3. Return transaction hash
```

### Nonce Management

The `NonceServiceImpl` manages Ethereum transaction nonces to prevent conflicts:

```
NonceServiceImpl:
  GenerateNonceWithKeysForWallet(walletAddress, chainID):
    1. Lock mutex for (walletAddress, chainID)
    2. Get pending nonce from chain: eth_getTransactionCount(address, "pending")
    3. Get local tracked nonce from cache
    4. Use max(chainNonce, localNonce)
    5. Increment local nonce
    6. Unlock mutex
    7. Return nonce
  
  Init():
    - Initialize per-wallet nonce tracking
    - Sync with on-chain state on startup
```

This prevents the common issue of two transactions using the same nonce, which would cause one to be rejected.

### Gas Estimation

```
GetEstimatedFeesForContractAndMethod(contractAddr, method, params):
  ETH/Base/Polygon:
    1. Simulate contract call via eth_estimateGas
    2. Fetch current gas prices:
       - baseFee from latest block
       - maxPriorityFeePerGas from eth_maxPriorityFeePerGas
    3. Calculate total fee:
       fee = gasLimit * (baseFee + maxPriorityFee)
    4. For Base: add L1 data fee component
       fetchL1FeeAndCalculateTxFeeForBase()
    5. Return estimated fee in native currency
  
  BTC:
    fee = FeeSatoshiPerByte() * EstimateTransactionSize(numInputs, numOutputs)
  
  TRX:
    energyCost = estimateEnergy(contractAddr, method, params)
    fee = energyCost * energyPrice (in TRX)
```

---

## 11. Card-to-Crypto Onramp Integration

### Overview

Payminto supports card-to-crypto payments through two distinct channels, allowing customers to pay with traditional payment methods (credit/debit cards, Apple Pay, Google Pay, bank transfers) while merchants receive cryptocurrency.

### Two Payment Channels

```
┌────────────────────��────────────────────────────────────────┐
│               CARD-TO-CRYPTO CHANNELS                        │
│                                                             │
│  Channel 1: "Cards" (channelType: app)                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                                                        │ │
│  │  Card → Onramp Provider → Customer PayRam Wallet       │ │
│  │                           (on Base)                    │ │
│  │                              │                         │ │
│  │                              ▼                         │ │
│  │                    Merchant Deposit Address             │ │
│  │                                                        │ │
│  │  TWO-STEP: Customer gets a reusable wallet first,      │ │
│  │  then wallet sends to merchant.                        │ │
│  │                                                        │ │
│  │  Wallet: wallet.payram.com (popup, 360x750)            │ │
│  │  Uses ERC-4337 Account Abstraction via Pimlico         │ │
│  │  EntryPoint: 0x4337084d9e255ff0702461cf8895ce9e3b5ff108│ │
│  └────────────────────────────────────────────────────────┘ │
│                                                             │
│  Channel 2: "TransFi" (channelType: onramp)                 │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                                                        │ │
│  │  Card → TransFi → Merchant Deposit Address (DIRECT)    │ │
│  │                                                        │ │
│  │  ONE-STEP: TransFi sends crypto directly to merchant.  │ │
│  │  No intermediate wallet needed.                        │ │
│  │                                                        │ │
│  │  TransFi iframe: buy.transfi.com                       │ │
│  │  Embedded directly in the payment page                 │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Cards Flow (channelType: app) — Detailed

**Prerequisites:**
- Cards channel is ONLY enabled when USDC on BASE chain is available
- Base chain required due to low gas fees making the two-hop economically viable

**Step-by-step flow:**

1. **Payment page loads** — Frontend calls APIs to get available channels, currencies, and merchant info

2. **Customer selects "Cards"** — Frontend automatically sets token to USDC, blockchain to BASE, and calls `POST /api/v1/deposit-address/{ref}` with `{blockchain_code: "BASE"}`

3. **Wallet popup opens** — Desktop: popup window (360x750) to `wallet.payram.com` with URL parameters:
   ```
   recipientAddress=0x...     ← Merchant deposit address
   amount=1000000             ← USDC amount (6 decimals)
   chainID=8453               ← Base mainnet
   tokenAddress=0x833589...   ← USDC contract on Base
   fiatAmount=100             ← USD display amount
   referenceID=abc-123        ← Payment reference
   host=http://merchant:8080  ← Merchant API URL
   ```

4. **Customer creates wallet** — Email + OTP verification, ERC-4337 smart account created via Pimlico

5. **Customer funds wallet** — Onramp widget loads inside wallet, card payment processed, USDC deposited to customer's smart account on Base

6. **Customer sends payment** — Wallet sends USDC to merchant deposit address via ERC-4337 UserOperation

7. **Payment detection** — Base ERC20 listener detects USDC deposit at merchant address, same as regular crypto flow

**ERC-4337 detection on merchant server:**
- When the block monitor detects a deposit, it checks if the transaction went through the ERC-4337 EntryPoint contract (0x4337...08)
- If yes: marks as "payments app transaction" and calls `PAYMENTS_APP_SERVER_URL` (x.payram.com) to fetch gas sponsorship data

### TransFi Flow (channelType: onramp) — Detailed

**Simpler one-step flow:**

1. **Customer selects "TransFi Payments"** on payment page

2. **Deposit address assigned** via same `POST /api/v1/deposit-address/{ref}` API

3. **TransFi iframe loads** directly in the payment page:
   ```html
   <iframe src="https://buy.transfi.com/?view=buy
     &apiKey=l8ooOivL2Hx5qZbu
     &redirectUrl={current_page_url}
     &walletAddress=0xMerchantDepositAddress
     &cryptoTicker=USDC
     &cryptoNetwork=BASE
     &partnerContext={referenceId:xxx}
   " />
   ```

4. **Customer pays** with card inside the iframe

5. **TransFi converts** fiat to USDC and sends **directly** to merchant deposit address

6. **Payment detection** — Standard Base ERC20 listener detects deposit

**Key difference:** No intermediate wallet, no popup, no wallet creation. TransFi handles everything and sends directly to merchant.

### Gas Fee Sponsorship

Merchants can sponsor gas fees for customer wallet transactions to reduce friction:

```
Database: payments_apps table
  sponsorship_percentage  NUMERIC(5,4)   -- % of gas to sponsor (0-100%)
  sponsorship_cut_off     NUMERIC(38,18) -- Max sponsorship amount in USD

When enabled:
  1. Merchant pre-deposits ETH on Base for gas
  2. ERC-4337 Paymaster (Pimlico) pays gas on behalf of customer
  3. Customer's wallet-to-merchant tx gas is subsidized
  4. Reduces friction (customer doesn't need Base ETH)
  5. On Base: gas ~$0.01, making sponsorship economically trivial
```

### Payment Notification — No Magic, Just Blockchain

There is NO direct callback from wallet.payram.com to the merchant server. There is NO postMessage between the popup and the payment page. There is NO webhook from the wallet.

The flow is entirely on-chain:
1. Wallet sends USDC on-chain to merchant deposit address
2. Merchant's Base ERC20 listener detects the transaction
3. Creates deposit record, links to payment request
4. Payment page polls every 5 seconds until status changes
5. Webhook fires to merchant backend

### Why Two Channels Exist

| Aspect | TransFi (onramp) | Cards (app) |
|--------|-------------------|-------------|
| Customer steps | 1 (pay in iframe) | 3 (create wallet, fund, send) |
| Wallet needed | No | Yes (ERC-4337 smart account) |
| Provider | TransFi only | Any onramp in the wallet |
| Gas fees | Included by TransFi | Can be sponsored by merchant |
| Reusable | No (one-time) | Yes (wallet persists) |
| Provider dependency | High (single provider) | Lower (multiple onramps) |
| Regulatory model | B2B partnership | Customer self-custody |

### Compliance Architecture

The two-hop Cards flow exists for regulatory reasons:

```
TransFi direct (onramp):
  TransFi charges card → sends USDC to MERCHANT
  = TransFi acts as payment processor for merchant
  = Requires merchant KYB + partnership agreements

Cards wallet (app):
  Onramp charges card → sends USDC to CUSTOMER'S wallet
  Customer sends USDC → MERCHANT
  = Onramp only serves the customer (not the merchant)
  = Peer-to-peer transfer from customer to merchant
  = No merchant KYB required from PayRam/Payminto
```

This is why PayRam docs state: "Merchants do not need to complete KYC/KYB to enable this Onramp method."

---

## 12. MCP/AI Agent Integration

### Overview

Payminto includes a Model Context Protocol (MCP) server that enables AI agents to autonomously create payments, manage recipients, and query financial data without human intervention.

### Architecture

```
┌──────────────────────────────────────────────────────────┐
│  AI Agent (Claude, GPT, Copilot, any MCP client)          │
│                                                          │
│  Auto-discovers tools via MCP handshake:                  │
│  → 10 tools available                                     │
│  → No API keys needed for MCP connection                  │
└───────────────────┬──────────────────────────────────────┘
                    │
                    │ StreamableHTTP (POST /mcp)
                    │ Server-Sent Events (GET /mcp/sse)
                    │
                    ▼
┌──────────────────────────────────────────────────────────┐
│  MCP Server (TypeScript / Node.js)                        │
│  Port: 3333                                              │
│                                                          │
│  ┌───────────────────────────────────────────────────┐   │
│  │  Protocol Handler                                  │   │
│  │  - Tool discovery (list available tools)           │   │
│  │  - Request routing                                 │   │
│  │  - SSE event streaming                             │   │
│  │  - Error handling                                  │   │
│  └───────────────────┬───────────────────────────────┘   │
│                      │                                    │
│  ┌───────────────────▼───────────────────────────────┐   │
│  │  Tool Handlers                                     │   │
│  │  - PaymentTools (create, lookup, search)           │   │
│  │  - WalletTools (balance, unswept)                  │   │
│  │  - AnalyticsTools (volume, summary)                │   │
│  │  - ConnectionTools (test)                          │   │
│  │  - PayeeTools (create payee)                       │   │
│  │  - InvoiceTools (generate invoice)                 │   │
│  └───────────────────┬───────────────────────────────┘   │
│                      │                                    │
│  ┌───────────────────▼───────────────────────────────┐   │
│  │  Internal API Client                               │   │
│  │  → Calls Go API Gateway (localhost:8080)           │   │
│  │  → Uses internal authentication                    │   │
│  └───────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────┘
```

### MCP Tools (10 Tools)

| Tool | Purpose | Parameters | Returns |
|------|---------|------------|---------|
| `create-payee` | Create a payment recipient | email, name, wallet_address, blockchain_code | payee_id, status |
| `send-payment` | Execute a payment transaction | payee_id, amount, currency_code, blockchain_code | payment_id, tx_hash, status |
| `get-balance` | Query account liquidity | blockchain_code (optional), currency_code (optional) | balances per chain/token |
| `generate-invoice` | Create a payment request | amount_in_usd, customer_email, customer_id, expiration | reference_id, payment_url |
| `test-connection` | Verify connectivity to Payminto | none | status, version, uptime |
| `lookup-payment` | Get transaction details | reference_id or payment_id | full payment object with deposits |
| `search-payments` | Query payment history | date_range, status, currency, pagination | payment list with totals |
| `get-daily-volume` | Aggregate daily metrics | date (optional, defaults to today) | volume_usd, transaction_count, by_currency |
| `get-payment-summary` | Overview reporting | date_range (optional) | total_volume, avg_payment, top_currencies |
| `get-unswept-balances` | Track pending sweeps | blockchain_code (optional) | unswept balances per address/token |

### MCP Configuration

Agents connect to the MCP server with minimal configuration:

```json
{
  "mcpServers": {
    "payminto": {
      "url": "http://localhost:3333/mcp"
    }
  }
}
```

No API keys are needed for the MCP connection. The MCP server communicates with the Go API gateway internally.

### MCP Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/mcp` | POST | StreamableHTTP MCP interface (tool calls) |
| `/mcp/sse` | GET | Server-Sent Events streaming (real-time updates) |
| `/healthz` | GET | Health check for MCP server |

### Supported Protocols

| Protocol | Purpose | Description |
|----------|---------|-------------|
| **MCP Standard** | AI agent integration | Model Context Protocol for tool discovery and execution |
| **x402** | Machine-to-machine payments | HTTP 402-based protocol where APIs can request payment before serving content |
| **ERC-8004** | Agent identity | Trustless agent identity, reputation, and validation standard |

### AI Agent Payment Flow

```
AI Agent                    MCP Server              Go API              Blockchain
    │                           │                     │                    │
    │  MCP Handshake            │                     │                    │
    │  (discover tools)         │                     │                    │
    │──────────────────────────>│                     │                    │
    │  [10 tools available]     │                     │                    │
    │<──────────────────────────│                     │                    │
    │                           │                     │                    │
    │  create-payee             │                     │                    │
    │  {email, address}         │                     │                    │
    │──────────────────────────>│  POST /payee        │                    │
    │                           │────────────────────>│                    │
    │                           │  {payee_id}         │                    │
    │  {payee_id}               │<────────────────────│                    │
    │<──────────────────────────│                     │                    │
    │                           │                     │                    │
    │  send-payment             │                     │                    │
    │  {payee_id, amount}       │                     │                    │
    │──────────────────────────>│  POST /withdrawal   │                    │
    │                           │────────────────────>│                    │
    │                           │                     │  Sign & broadcast  │
    │                           │                     │───────────────────>│
    │                           │                     │  tx_hash           │
    │                           │  {payment_id}       │<───────────────────│
    │  {payment_id, tx_hash}    │<────────────────────│                    │
    │<──────────────────────────│                     │                    │
    │                           │                     │                    │
    │  lookup-payment           │                     │                    │
    │  {payment_id}             │                     │                    │
    │──────────────────────────>│  GET /payment/{id}  │                    │
    │                           │────────────────────>│                    │
    │  {status: confirmed}      │  {status, details}  │                    │
    │<──────────────────────────│<────────────────────│                    │
    │                           │                     │                    │
    │  Autonomous payment       │                     │                    │
    │  completed!               │                     │                    │
```

---

## Appendix A: Source Code File Map

### Blockchain Client Package (`internal/jobs/bclient/`)

| File | Purpose | Key Types/Functions |
|------|---------|-------------------|
| `blockchain_client.go` | Abstract base class | `AbstractBlockchainClient` (40+ methods) |
| `eth_blockchain_client.go` | EVM implementation | `ETHBlockchainClient` |
| `btc_blockchain_client.go` | Bitcoin implementation | `BTCBlockchainClient` |
| `trx_blockchain_client.go` | Tron implementation | `TRXBlockchainClient` |
| `blockchain_events.go` | Event type definitions | `SweepEventType`, `FactoryEventType`, `DepositType` |

### Block Processing (`internal/service/`)

| File | Purpose | Key Functions |
|------|---------|--------------|
| `blockchain_processor_impl.go` | Core block monitor (~2500 LOC) | `handleExtractedDeposit` (407 LOC), `polling` (135 LOC), `ProcessBlockRange` (139 LOC), `calculatePollingInterval` (63 LOC) |
| `blockchain_service_impl.go` | Blockchain CRUD + node management | `CheckAllNodeConnections` (508 LOC), `validateActiveNodesChainID` (50 LOC) |
| `deposit_processor_job.go` | Deposit confirmation processing | Deposit status transitions |
| `sweep_approval_processor.go` | ERC-20 approval management | `addressesBalancesAccountingAfterSweep` |
| `account_processor_job.go` | Sweep scheduling + withdrawals | `processETHAutoSweep`, `processBitcoinSweeps`, `processERC20Sweeps` |
| `broadcast_scw_deposit_wallets_processor.go` | SCW deployment | `BroadcastSCWDepositWalletProcessorJob` (184 LOC) |
| `nonce_service_impl.go` | Nonce management | `GenerateNonceWithKeysForWallet` |
| `wallet_service_impl.go` | Wallet + address pool | `GenerateAddressesIfPoolLow` |

### Repository Layer (`internal/repository/`)

| File | Purpose |
|------|---------|
| `blockchain_repo_impl.go` | Blockchain CRUD |
| `blockchain_currency_repo_impl.go` | Token configuration |
| `blockchain_contract_repo_impl.go` | Contract address management |
| `deposit_addresses_repo_impl.go` | Address pool management |
| `deposit_repo_impl.go` | Deposit records |
| `sweep_repo_impl.go` | Sweep records |
| `sweep_transaction_repo_impl.go` | Sweep transaction tracking |
| `sweep_utxo_repo_impl.go` | UTXO-to-sweep mapping |
| `internal_blockchain_transaction_repo_impl.go` | Internal transaction accounting |

### Worker Entry Points (`cmd/`)

| File | Purpose |
|------|---------|
| `eth_erc20_processor.go` | Ethereum listener entry point |
| `base_erc20_processor.go` | Base listener entry point |
| `polygon_erc20_processor.go` | Polygon listener entry point |
| `trx_trc20_processor.go` | Tron listener entry point |
| `bitcoin_processor.go` | Bitcoin listener entry point |
| `deposit_processor.go` | Deposit processor entry point |
| `account_processor.go` | Account processor (sweeps) entry point |
| `webhook_processor.go` | Webhook delivery entry point |
| `erc20_sweep_approval_processor.go` | ERC-20 approval entry point |
| `broadcast_scw_deposit_wallet_processor.go` | SCW deployment entry point |

---

## Appendix B: Key Database Tables for Blockchain Integration

| Table | Records | Purpose |
|-------|---------|---------|
| `blockchains` | Per-chain config | Chain code, name, block_height, is_active |
| `blockchain_families` | Chain family grouping | ethereum, bitcoin, tron |
| `blockchain_currencies` | Token config per chain | Contract address, confirmations, precision |
| `blockchain_contracts` | Smart contract addresses | SmartSweep, Factory per chain |
| `rpc_nodes` | RPC endpoint config | URL, priority, auth, chain_id |
| `wallets` | HD wallet config | XPUB, encrypted keys, cold wallet address |
| `deposit_addresses` | Address pool | Address, index, status, wallet_id |
| `deposits` | Detected deposits | Amount, tx_hash, status, block_number |
| `payment_requests` | Payment records | Amount, status, reference_id |
| `sweep_transactions` | Sweep records | Amount, tx_hash, from/to, status |
| `sweeps` | Sweep scheduling | Status, blockchain_currency_id |
| `sweep_utxos` | BTC UTXO-sweep mapping | Links UTXOs to sweep transactions |
| `utxos` | Bitcoin UTXOs | tx_hash, output_index, amount, status |
| `internal_blockchain_transactions` | Internal accounting | All internal transfers (sweeps, gas) |
| `address_deployments` | SCW deployment tracking | Address, status, tx_hash |
| `address_contract_signatures` | ERC-20 approvals | Address, contract_type, status |
| `missed_deposits` | Recovery tracking | Potentially missed deposits for re-processing |
| `account_address_balances` | Balance tracking | Per-address balance accounting |

---

## Appendix C: Environment Variables for Blockchain Configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `BLOCKCHAIN_NETWORK_TYPE` | mainnet or testnet | `mainnet` |
| `AES_KEY` | 32-byte hex key for wallet encryption | `openssl rand -hex 32` |
| `PAYMENTS_APP_SERVER_URL` | PayRam wallet server URL | `https://x.payram.com` |
| `SERVER` | Environment mode | `PRODUCTION` or `DEVELOPMENT` |
| `POSTGRES_HOST` | Database host | `localhost` |
| `POSTGRES_PORT` | Database port | `5432` |
| `POSTGRES_DATABASE` | Database name | `payminto` |
| `POSTGRES_USERNAME` | Database user | `payminto` |
| `POSTGRES_PASSWORD` | Database password | (user-defined) |

RPC node URLs are stored in the database (`rpc_nodes` table) rather than environment variables, allowing runtime configuration without restart.
