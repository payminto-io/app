# Payminto Worker Specification

This document provides a comprehensive specification for all background workers and job processors in Payminto. Workers run alongside the main API server as long-lived goroutines, each responsible for a distinct domain of asynchronous processing.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Worker Management](#2-worker-management)
3. [Worker Specifications](#3-worker-specifications)
   - [A. Account Processor](#a-account-processor)
   - [B. Base ERC20 Processor](#b-base-erc20-processor)
   - [C. Bitcoin Block Processor](#c-bitcoin-block-processor)
   - [D. Ethereum Block Processor](#d-ethereum-block-processor)
   - [E. Tron Block Processor](#e-tron-block-processor)
   - [F. Polygon ERC20 Processor](#f-polygon-erc20-processor)
   - [G. Deposit Processor](#g-deposit-processor)
   - [H. Webhook Processor](#h-webhook-processor)
   - [I. Email Processor](#i-email-processor)
   - [J. ERC20 Sweep Approval Processor](#j-erc20-sweep-approval-processor)
   - [K. SCW Deposit Wallet Broadcaster](#k-scw-deposit-wallet-broadcaster)
   - [L. Seeder Initialize Processor](#l-seeder-initialize-processor)
   - [M. Accounting Duplicate Deposits Processor](#m-accounting-duplicate-deposits-processor)
4. [Shared Infrastructure](#4-shared-infrastructure)
5. [Block Monitor Flow](#5-block-monitor-flow)
6. [Scheduling Patterns](#6-scheduling-patterns)
7. [Error Handling](#7-error-handling)
8. [Graceful Shutdown](#8-graceful-shutdown)

---

## 1. Overview

### Worker Architecture

Payminto runs 13 background worker processes alongside its HTTP API server. Each worker is a standalone Go struct that encapsulates its own goroutine lifecycle, polling loop, and shutdown mechanism. Workers are instantiated during application startup via the `processor.Execute` command, which registers and launches each worker as a named subprocess.

All workers share the following architectural properties:

- **Goroutine-based concurrency**: Each worker runs one or more goroutines managed by a `sync.WaitGroup` or equivalent coordination primitive.
- **Channel-based shutdown**: Every worker holds a `stopChan` (typically `chan struct{}`) that signals goroutines to terminate.
- **Ticker-driven polling**: Most workers use `time.Ticker` or `time.Timer` to periodically execute their main processing loop.
- **Panic recovery**: Each goroutine wraps its core logic in a `defer` block that recovers from panics, logs the error, and optionally restarts the goroutine.
- **Service injection**: Workers receive their dependencies (services, repositories, blockchain clients) via constructor injection at creation time.

### Worker Process Topology

```
┌──────────────────────────────────────────────────────────────┐
│                     Payminto Process                         │
│                                                              │
│  ┌──────────────────────┐   ┌─────────────────────────┐     │
│  │   HTTP API Server    │   │   Worker Manager         │     │
│  │   (Gin, port 8080)   │   │   (SystemServiceImpl)    │     │
│  └──────────────────────┘   └─────────┬───────────────┘     │
│                                       │                      │
│  ┌────────────────────────────────────┼──────────────────┐  │
│  │              Background Workers    │                   │  │
│  │                                    ▼                   │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │  │
│  │  │ Account      │  │ ETH Block    │  │ BTC Block   │ │  │
│  │  │ Processor    │  │ Processor    │  │ Processor   │ │  │
│  │  └──────────────┘  └──────────────┘  └─────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │  │
│  │  │ BASE ERC20   │  │ Polygon      │  │ TRX Block   │ │  │
│  │  │ Processor    │  │ Processor    │  │ Processor   │ │  │
│  │  └──────────────┘  └──────────────┘  └─────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │  │
│  │  │ Deposit      │  │ Webhook      │  │ Email       │ │  │
│  │  │ Processor    │  │ Processor    │  │ Processor   │ │  │
│  │  └──────────────┘  └──────────────┘  └─────────────┘ │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐ │  │
│  │  │ Sweep        │  │ SCW Wallet   │  │ Seeder      │ │  │
│  │  │ Approval     │  │ Broadcaster  │  │ Initialize  │ │  │
│  │  └──────────────┘  └──────────────┘  └─────────────┘ │  │
│  │  ┌──────────────┐                                     │  │
│  │  │ Accounting   │                                     │  │
│  │  │ Dup Deposits │                                     │  │
│  │  └──────────────┘                                     │  │
│  └───────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

### Worker Registration (cmd/payram/processor)

The `processor.Execute` function registers all 13 workers as named subcommands. Each subcommand creates the worker's job struct, injects dependencies from the `ServiceRegistry`, and calls the worker's `Start*` method. The following subcommands are registered:

| Subcommand | Function | Worker Created |
|------------|----------|----------------|
| `startEtherAndERC20ListenerCmd` | `func1` | Ethereum Block Processor |
| `startBaseAndERC20ListenerCmd` | `func2` | Base ERC20 Processor |
| `startPolygonAndERC20ListenerCmd` | `func3` | Polygon ERC20 Processor |
| `startBitcoinProcessorCmd` | `func4` | Bitcoin Block Processor |
| `startDepositProcessorCmd` | `func5` | Deposit Processor |
| `startAccountProcessorCmd` | `func6` | Account Processor |
| `startWebhookProcessorCmd` | `func7` | Webhook Processor |
| `startEmailProcessorCmd` | `func8` | Email Processor |
| `startTronAndTRC20ListenerCmd` | `func9` | Tron Block Processor |
| `startSweepApprovalListenerCmd` | `func10` | ERC20 Sweep Approval Processor |
| `startSeederInitialiseCmd` | `func11` | Seeder Initialize Processor |
| `startAccountingDuplicateDepositsCmd` | `func12` | Accounting Duplicate Deposits |
| `startBroadcastSCWDepositWalletProcessorCmd` | `func13` | SCW Deposit Wallet Broadcaster |

---

## 2. Worker Management

### SystemServiceImpl

The `SystemServiceImpl` provides a management interface for controlling workers at runtime. It exposes methods for starting, stopping, restarting, and querying worker status. The admin dashboard and admin-only API endpoints use this service to control worker lifecycle.

#### Methods

**GetAllWorkers**
```
(*SystemServiceImpl).GetAllWorkers() []WorkerInfo
```
Returns a list of all registered workers with their names and types. This method iterates over the internal worker registry and returns metadata about each worker without querying runtime status.

**GetAllWorkerStatuses**
```
(*SystemServiceImpl).GetAllWorkerStatuses() []WorkerStatus
```
Returns the current runtime status of every registered worker (37 lines of logic). For each worker, it reports:
- Worker name (parsed via `ParseWorkerName`)
- Running/stopped state
- Last error (if any)
- Uptime since last start
- Goroutine count

**StartWorker**
```
(*SystemServiceImpl).StartWorker(workerName string) error
```
Starts a specific worker by name. Looks up the worker in the registry, verifies it is not already running, and calls the worker's `Start*` method. Returns an error if the worker name is not found or if the worker is already running.

**StopWorker**
```
(*SystemServiceImpl).StopWorker(workerName string) error
```
Stops a specific worker by name. Calls the worker's `Stop()` method, which sends a signal on `stopChan` and waits for goroutines to drain. Returns an error if the worker is not found or is already stopped.

**RestartWorker**
```
(*SystemServiceImpl).RestartWorker(workerName string) error
```
Restarts a specific worker by calling `StopWorker` followed by `StartWorker` (9 lines). Includes a brief delay between stop and start to allow goroutine cleanup to complete.

**RestartAllWorkers**
```
(*SystemServiceImpl).RestartAllWorkers() error
```
Restarts all registered workers concurrently (24 lines). Launches a goroutine per worker that calls `RestartWorker`, with a deferred panic recovery wrapper around each goroutine. Collects and returns any errors that occurred during the restart process.

**ParseWorkerName**
```
ParseWorkerName(rawName string) string
```
A standalone function (58 lines) that normalizes raw worker type names into human-readable display names. Handles mapping from Go struct type names (e.g., `*service.AccountProcessorJob`) to clean names (e.g., `account_processor`). Used by `GetAllWorkerStatuses` and the admin API handlers.

#### Admin API Endpoints

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/admin/workers` | `GetAllWorkersHandler` | List all workers |
| GET | `/admin/workers/status` | `GetAllWorkersStatus` | Get status of all workers |
| POST | `/admin/workers/:name/start` | `StartWorkerHandler` | Start a specific worker |
| POST | `/admin/workers/:name/stop` | `StopWorkerHandler` | Stop a specific worker |
| POST | `/admin/workers/:name/restart` | `RestartWorkerHandler` | Restart a specific worker |
| POST | `/admin/workers/restart` | `RestartAllWorkersHandler` | Restart all workers |

---

## 3. Worker Specifications

### A. Account Processor

**Package**: `payram/internal/service`
**File**: `account_processor_job.go` (625 lines)
**Struct**: `AccountProcessorJob`

#### Purpose

The Account Processor is the most complex worker in the system. It coordinates all post-deposit financial operations: auto-sweeping funds from deposit addresses to cold/hot wallets, processing referral rewards, handling failed rewards, managing withdrawals, and monitoring confirming sweep transactions. It runs 9 concurrent goroutines, each on its own ticker-based polling loop.

#### Constructor

```
NewAccountProcessorJob(
    depositService       DepositService,
    sweepService         SweepService,
    sweepTxService       SweepTransactionService,
    sweepUTXOService     SweepUTXOService,
    blockchainService    BlockchainService,
    accountRewardService AccountRewardService,
    withdrawalService    WithdrawalProcessingService,
    accountService       AccountService,
    addressService       AddressService,
    utxoService          UTXOService,
    configService        ConfigService,
) *AccountProcessorJob
```

Lines 27-51 (24 lines). Stores all service references and initializes the `stopChan`.

#### Dependencies

| Service | Purpose |
|---------|---------|
| `DepositService` | Query confirmed deposits for sweep eligibility |
| `SweepService` | Create and manage sweep entries |
| `SweepTransactionService` | Create, monitor, and complete sweep transactions |
| `SweepUTXOService` | Handle UTXO-based sweep accounting for Bitcoin |
| `BlockchainService` | Get blockchain configurations and parameters |
| `AccountRewardService` | Process referral program rewards |
| `WithdrawalProcessingService` | Process merchant withdrawal requests |
| `AccountService` | ERC20 sweep balance accounting |
| `AddressService` | Query eligible addresses for sweep operations |
| `UTXOService` | Query pending UTXOs for sweep transactions |
| `ConfigService` | Retrieve system-level configurations (thresholds, intervals) |

#### StartAccountProcessor

```
(*AccountProcessorJob).StartAccountProcessor()
```

Lines 51-83 (32 lines). Launches 9 goroutines, each wrapped in a panic-recovery defer:

| Goroutine | Function Called | Description |
|-----------|----------------|-------------|
| `gowrap1` | Main coordinator | Ticker loop dispatching to sub-processors |
| `gowrap2` | `processETHAutoSweep` | Auto-sweep ETH/native deposits |
| `gowrap3` | `processBitcoinSweeps` | Process BTC sweep transactions |
| `gowrap4` | `processERC20Sweeps` | Process ERC20/TRC20 token sweeps |
| `gowrap5` | `createSweepTransactionPayload` | Build sweep transaction payloads |
| `gowrap6` | `processRewards` | Process pending referral rewards |
| `gowrap7` | `processFailedRewards` | Retry failed reward distributions |
| `gowrap8` | `processWithdrawals` | Process merchant withdrawal requests |
| `gowrap9` | `retryStaleBTCSweepTransactions` | Retry stale BTC sweep transactions |

#### Sub-processor Functions

**processETHAutoSweep** (Lines 86-112, 26 lines)
Handles automatic sweeping of ETH and native currency deposits from deposit addresses to the hot wallet. Queries deposit addresses that have confirmed balances above the sweep threshold. For each eligible address, calls `SweepTransactionService.EthAutoSweepTransaction` to create and broadcast the sweep transaction. Includes deferred panic recovery (30 lines).

**processBitcoinSweeps** (Lines 127-153, 26 lines)
Processes Bitcoin sweep transactions by gathering eligible UTXO-based deposits. Coordinates with `SweepUTXOService` to aggregate UTXOs into sweep batches. Handles the BTC-specific requirement of building multi-input transactions that spend from multiple deposit addresses simultaneously.

**processERC20Sweeps** (Lines 168-200, 32 lines)
Manages sweeping of ERC20 tokens (and TRC20 on Tron) from deposit addresses. This is a two-step process: first, gas/fee funding must be sent to the deposit address, then the token transfer is executed. Uses `AccountService.ProcessERC20Sweep` which handles the fee-fund-then-sweep pattern.

**createSweepTransactionPayload** (Lines 215-243, 28 lines)
Builds the transaction payloads for pending sweep operations. Retrieves sweep transaction records that are in "pending" state and constructs the appropriate blockchain transaction data. Includes deferred recovery (25 lines).

**processRewards** (Lines 253-285, 32 lines)
Processes pending referral reward entries. Queries the `AccountRewardService` for unprocessed reward events and distributes them according to the referral program rules. Handles multi-tier referral structures where rewards cascade to multiple levels of referrers.

**processFailedRewards** (Lines 300-329, 29 lines)
Retries previously failed reward distributions. Queries for reward entries with a "failed" status and attempts to reprocess them. Includes a backoff mechanism to avoid rapid retry loops on persistently failing rewards.

**processWithdrawals** (Lines 341-376, 35 lines)
Processes merchant withdrawal requests. Picks up approved withdrawal records and initiates the blockchain transaction to send funds from the hot wallet to the merchant's designated withdrawal address. Handles both native currency and token withdrawals across all supported chains.

**retryStaleBTCSweepTransactions** (Lines 448-474, 26 lines)
Identifies BTC sweep transactions that have been in an "initiated" state for too long without confirmation. Rebroadcasts or replaces these transactions using RBF (Replace-By-Fee) if supported, or simply retries the broadcast.

**processStaleInitiatedSweeps** (Lines 485-536, 51 lines)
Broader stale sweep handler that covers all chains. Identifies sweep transactions that have been stuck in "initiated" status beyond the expected confirmation window and takes corrective action.

**processConfirmingBTCSweeps** (Lines 536-562, 26 lines)
Monitors BTC sweep transactions that are in "confirming" state (broadcasted but not yet at the required confirmation depth). Checks the current block height against the transaction's block number to determine if sufficient confirmations have been reached.

**processConfirmingSweepsOnce** (Lines 572-622, 50 lines)
A one-shot processor that checks all confirming sweep transactions across all chains. Called periodically to verify that confirming sweeps have reached the required number of block confirmations and transitions them to "completed" status.

**createAllPendingUTXOsAndAccountForSweepTransaction** (Lines 394-448, 54 lines)
Handles UTXO accounting for Bitcoin sweep transactions. Queries all pending UTXOs across deposit addresses, groups them by wallet, and creates the accounting entries needed for the sweep transaction.

#### Stop

```
(*AccountProcessorJob).Stop()
```

Lines 622-625 (3 lines). Closes the `stopChan`, signaling all 9 goroutines to exit their polling loops.

---

### B. Base ERC20 Processor

**Package**: `payram/internal/service`
**Files**: `blockchain_processor_impl.go` (2477+ lines)
**Struct**: `blockchainProcessorImpl` (configured for Base chain)

#### Purpose

Monitors the Base blockchain (an Ethereum L2) for incoming deposits of both native ETH and ERC20 tokens. Detects deposits by scanning each block for transactions targeting known deposit addresses, then creates deposit records and triggers downstream processing.

#### Constructor and Factory

```
Execute.startBaseAndERC20ListenerCmd (Lines 16-43, 27 lines in cmd/payram/processor/base_erc20_processor.go)
```

The command handler calls `CreateBlockchainProcessor` (Lines 83-142, 59 lines) with the Base chain configuration, which initializes the `blockchainProcessorImpl` struct with:
- Blockchain record (from DB, identifying Base chain)
- RPC pool configuration (multiple Base RPC endpoints)
- All necessary services (DepositService, EventEmitterService, AddressService, etc.)
- Block processing callbacks (address matching, deposit handling)

#### Key Functions

**initializeClient** (Lines 142-196, 54 lines)
Creates the blockchain client by connecting to the RPC pool. For Base (an EVM chain), this instantiates an `ETHBlockchainClient` through the `NewBlockchainClient` factory. Configures the RPC pool with all seeded Base RPC nodes and tests connectivity.

**Start** (Lines 196-241, 45 lines)
Entry point that initializes the client and starts the polling goroutine. Spawns two goroutines:
- `gowrap1`: The main polling loop
- `gowrap2`: The withdrawal audit processor

**startPolling** (Lines 252-299, 47 lines)
Sets up the polling loop with dynamic interval calculation. Initializes the last processed block number from the database and enters the main polling ticker loop.

**polling** (Lines 305-440, 135 lines)
The core polling function executed on each tick. Steps:
1. Fetch the latest block number from the RPC node
2. Calculate the range of unprocessed blocks
3. If there are blocks to process, call `ProcessBlockRange`
4. Process any confirming deposits (deposits waiting for more confirmations)
5. Process any confirming sweeps
6. Run RPC pool maintenance
7. Recalculate the polling interval based on block production rate

**ProcessBlockRange** (Lines 461-600, 139 lines)
Processes a contiguous range of blocks. For each block in the range:
1. Fetch the block data from the RPC node
2. Call `ProcessSingleBlock` to extract relevant transactions
3. Handle any extracted deposits, sweep events, or factory events
4. Update the last processed block number in the database
5. Handle errors with retry logic (skip blocks that consistently fail after multiple retries)

**ProcessSingleBlock** (Lines 600-649, 49 lines)
Processes a single block by calling the blockchain client's `ProcessBlock` method with a configuration that specifies which addresses and contract events to look for. Returns extracted deposits, sweep events, and factory events.

**handleExtractedDeposit** (Lines 777-1184, 407 lines)
The largest function in the codebase. Processes a single extracted deposit from a block scan. Steps:
1. Verify the transaction hash and extract detailed transaction data
2. Check if the deposit is for a known deposit address
3. Determine the deposit type (native transfer, ERC20 transfer, or smart contract deposit)
4. Check for duplicate deposits (same tx hash + address + currency)
5. Create or update the deposit record in the database
6. Calculate the deposit amount with proper decimal precision
7. Set the initial confirmation count and required confirmations
8. Emit events for webhook notifications
9. Handle edge cases: zero-value transfers, blacklisted addresses, dust deposits

**verifyAndUpsertDepositsForTxHash** (Lines 1429-1612, 183 lines)
Verifies a deposit by re-fetching the transaction from the blockchain and upserting the deposit record. Handles the transition from "pending" to "confirming" to "confirmed" status as block confirmations accumulate.

#### RPC Nodes (Base Chain)

| Node | Function | Network |
|------|----------|---------|
| PublicNode | `GetRPCNodeBASEPublicNode` | Mainnet |
| LlamaRPC | `GetRPCNodeBASELlamaRPC` | Mainnet |
| 1RPC | `GetRPCNodeBASE1RPC` | Mainnet |
| dRPC | `GetRPCNodeBASEdRPC` | Mainnet |
| MeowRPC | `GetRPCNodeBASEMeowRPC` | Mainnet |
| Tenderly | `GetRPCNodeBASETenderly` | Mainnet |
| Sepolia Official | `GetRPCNodeBASESepoliaOfficial` | Testnet |
| Sepolia PublicNode | `GetRPCNodeBASESepoliaPublicNode` | Testnet |
| Sepolia dRPC | `GetRPCNodeBASESepoliaddRPC` | Testnet |
| Sepolia Tenderly | `GetRPCNodeBASESepoliaTenderly` | Testnet |

---

### C. Bitcoin Block Processor

**Package**: `payram/internal/service`
**Files**: `blockchain_processor_impl.go`
**Struct**: `blockchainProcessorImpl` (configured for Bitcoin)

#### Purpose

Monitors the Bitcoin blockchain for incoming UTXO-based deposits. Unlike EVM chains, Bitcoin requires UTXO-specific processing: identifying unspent transaction outputs that belong to known deposit addresses, tracking them individually, and managing UTXO aggregation for sweep transactions.

#### Factory

```
BitcoinProcessorJob (Lines 2554-2565, 11 lines)
```

Creates a `blockchainProcessorImpl` configured with Bitcoin-specific parameters and a `BTCBlockchainClient`. The Bitcoin processor uses a different block scanning approach: instead of scanning for address matches in `to` fields, it scans all transaction outputs for matching scriptPubKey patterns.

#### Key Functions (Bitcoin-specific)

**StartBitcoinListener** (Lines 2467-2477, 10 lines)
Wrapper that configures the blockchain processor for Bitcoin and calls `Start`. Sets up the UTXO identification callbacks.

**identifyOurUTXOs** (Lines 2164-2186, 22 lines)
Scans all transaction outputs in a block and identifies those that belong to known deposit addresses. For each UTXO, it checks the scriptPubKey against the database of deposit addresses.

**isOurUTXO** (Lines 2133-2154, 21 lines)
Checks whether a single UTXO belongs to a known deposit address. Uses WHERE clauses to match the address against the `addresses` table with conditions on blockchain code and address status.

**createUTXOEntry** (Lines 2186-2202, 16 lines)
Creates a UTXO record in the database for a newly discovered deposit output. Records the transaction hash, output index, amount (in satoshis), and the associated deposit address.

**queueBTCSweepTransaction** (Lines 2202-2219, 17 lines)
Queues a Bitcoin sweep transaction for later processing by the Account Processor. Creates a sweep transaction record in "pending" state with references to the UTXOs to be spent.

**handleBTCSweepTransaction** (Lines 2219-2260, 41 lines)
Processes a Bitcoin sweep event detected in a block. This handles the case where a previously broadcasted sweep transaction has been mined. Updates the sweep transaction status and the associated UTXO statuses.

#### BTC Blockchain Client (from bclient package)

**BTCBlockchainClient** provides Bitcoin-specific blockchain interaction:

| Function | Lines | Description |
|----------|-------|-------------|
| `GetBlockByNumber` | via `func1`, `func2` | Fetches block by height with retry |
| `GetBlockByHash` | via `func1` | Fetches block by hash |
| `GetLatestBlockNumber` | via `func1` | Gets current chain tip |
| `GetUTXOsInBlock` | via `func1`, `func2` | Extracts UTXOs from block |
| `processUTXODeposits` | 1 function | Processes UTXO deposits from block data |
| `detectBTCSweepTransactions` | 67 lines | Detects sweep txs in block |
| `detectBTCSweepTransactionsBatch` | 113 lines | Batch detection of sweep txs |
| `EnsureTxConfirmed` | via `func1`, `func2` | Waits for tx confirmation |
| `calculateTransactionFee` | via `func1` | Calculates BTC tx fee |
| `GetWithdrawalAuditData` | via `func1-3` | Gets withdrawal verification data |

#### RPC Nodes (Bitcoin)

| Node | Function | Network |
|------|----------|---------|
| Preferred Node | `GetRPCNodeBTCMainnetPreferredNode` | Mainnet |
| NOWNodes | `GetRPCNodeBTCNOWNodes` | Mainnet |
| PublicNode | `GetRPCNodeBTCPublicNode` | Mainnet |
| Testnet Preferred | `GetRPCNodeBTCTestnetPreferredNode` | Testnet |

---

### D. Ethereum Block Processor

**Package**: `payram/internal/service`
**Files**: `blockchain_processor_impl.go`
**Struct**: `blockchainProcessorImpl` (configured for Ethereum)

#### Purpose

Monitors the Ethereum mainnet for native ETH transfers and ERC20 token transfers to known deposit addresses. Uses the shared `blockchainProcessorImpl` base with Ethereum-specific client configuration.

#### Factory

```
EtherAndERC20ProcessorJob (Lines 2477-2516, 39 lines)
```

Creates a `blockchainProcessorImpl` configured for Ethereum. Sets up the `ETHBlockchainClient` with the Ethereum RPC pool and configures the processor for both native ETH and ERC20 token detection.

#### Key Functions

**StartEtherAndERC20Listener** (Lines 2455-2461, 6 lines)
Wrapper that configures the blockchain processor for Ethereum and calls `Start`. Uses the same polling, block processing, and deposit handling logic as the Base processor (shared `blockchainProcessorImpl`).

#### ETH Blockchain Client Functions

The `ETHBlockchainClient` provides EVM-specific processing:

| Function | Description |
|----------|-------------|
| `processNativeTransfersInBlockRaw` | Scans block for native ETH transfers |
| `processTokenTransfersFromLogs` | Extracts ERC20 Transfer events from logs |
| `processTokenTransfersInBlock` | Processes token transfers from block |
| `processSweepEventsFromLogs` | Extracts SmartSweep contract events (82 lines) |
| `processSweepEventsInBlock` | Processes sweep events from block (105 lines) |
| `processFactoryEventsFromLogs` | Extracts SCW factory deployment events |
| `processFactoryEventsInBlock` | Processes factory events from block |
| `processBalanceDiff` | Detects deposits via balance change analysis |
| `decodeCalldataFast` | Fast calldata decoding for tx classification |
| `fetchL1FeeAndCalculateTxFeeForBase` | L1 fee calculation (Base-specific) |
| `getBlockRaw` | Raw block fetch via JSON-RPC |
| `getBlockRawByHash` | Raw block fetch by hash |

#### RPC Nodes (Ethereum)

| Node | Function | Network |
|------|----------|---------|
| PublicNode | `GetRPCNodeETHPublicNode` | Mainnet |
| LlamaRPC | `GetRPCNodeETHLlamaRPC` | Mainnet |
| 1RPC | `GetRPCNodeETH1RPC` | Mainnet |
| dRPC | `GetRPCNodeETHdRPC` | Mainnet |
| MeowRPC | `GetRPCNodeETHMeowRPC` | Mainnet |
| Tenderly | `GetRPCNodeETHTenderly` | Mainnet |
| Sepolia 1RPC | `GetRPCNodeETHSepolia1RPC` | Testnet |
| Sepolia PublicNode | `GetRPCNodeETHSepoliaPublicNode` | Testnet |
| Sepolia dRPC | `GetRPCNodeETHSepoliadRPC` | Testnet |
| Sepolia Tenderly | `GetRPCNodeETHSepoliaTenderly` | Testnet |

---

### E. Tron Block Processor

**Package**: `payram/internal/service`
**Files**: `blockchain_processor_impl.go`
**Struct**: `blockchainProcessorImpl` (configured for Tron)

#### Purpose

Monitors the Tron blockchain for TRX native transfers and TRC20 token transfers (primarily USDT-TRC20) to known deposit addresses. Tron uses a unique protocol (gRPC-based) distinct from EVM chains, requiring its own blockchain client implementation.

#### Factory

```
TronAndTRC20ProcessorJob (Lines 2516-2554, 38 lines)
```

Creates a `blockchainProcessorImpl` configured for Tron. Initializes a `TRXBlockchainClient` that connects via gRPC to Tron full nodes.

#### Key Functions

**StartTronAndTRC20Listener** (Lines 2461-2467, 6 lines)
Wrapper that configures the blockchain processor for Tron and calls `Start`.

#### TRX Blockchain Client Functions

The `TRXBlockchainClient` provides Tron-specific processing:

| Function | Lines | Description |
|----------|-------|-------------|
| `GetTokenTransfersInBlock` | 101 lines (estimated) | Extracts TRC20 Transfer events from block |
| `GetContractEventsInBlock` | 101 lines (estimated) | Extracts smart contract events |
| `processTRC20Transfer` | 61 lines | Processes a single TRC20 transfer event |
| `ProcessBlockRange` | 61 lines | Processes a range of Tron blocks |
| `processBlockData` | Multiple parts | Core block data processing |
| `processFactoryTransaction` | 1 function | Handles SCW factory deploy events on Tron |
| `processSweepTransaction` | 79 lines | Handles SmartSweep events on Tron |
| `processTransferContract` | 1 function | Handles native TRX TransferContract |
| `normalizeAddress` | 1 function | Normalizes Tron addresses (Base58 <-> Hex) |
| `getChainIDViaJSONRPC` | 1 function | Gets Tron chain ID via JSON-RPC fallback |

#### Tron-Specific Considerations

- **gRPC Protocol**: Unlike EVM chains which use JSON-RPC, Tron nodes communicate primarily via gRPC. The `TRXBlockchainClient` uses gRPC connections with optional transport credentials.
- **Address Format**: Tron uses Base58Check encoding (starting with 'T') rather than hex addresses. The client includes `Base58ToHexTRX`, `HexToBase58TRX`, and `toTron41Hex` conversion utilities.
- **Energy Model**: Tron uses an "energy" resource model instead of gas. The client includes `getEstimatedFeesForContractExecution` for energy estimation and `validateContractExecutionResult` for checking execution outcomes.
- **Transaction Serialization**: Uses `SerializeTransactionTRX` and `DeserializeTransactionTRX` / `DeserializeTransactionTronWebSerialized` for Tron-specific transaction formats.

#### RPC Nodes (Tron)

| Node | Function | Network |
|------|----------|---------|
| TronGrid | `GetRPCNodeTRXTronGrid` | Mainnet |
| Nile | `GetRPCNodeTRXNile` | Testnet (Nile) |

---

### F. Polygon ERC20 Processor

**Package**: `payram/internal/service`
**Files**: `blockchain_processor_impl.go`
**Struct**: `blockchainProcessorImpl` (configured for Polygon)

#### Purpose

Monitors the Polygon PoS blockchain for native MATIC/POL transfers and ERC20 token transfers to known deposit addresses. Uses the same `blockchainProcessorImpl` and `ETHBlockchainClient` as the Ethereum and Base processors, since Polygon is EVM-compatible.

#### Factory

```
Execute.startPolygonAndERC20ListenerCmd (Lines 17-44, 27 lines in cmd/payram/processor/polygon_erc20_processor.go)
```

Creates a `blockchainProcessorImpl` with Polygon chain configuration. The processing logic is identical to the Ethereum and Base processors since Polygon is EVM-compatible.

#### RPC Nodes (Polygon)

| Node | Function | Network |
|------|----------|---------|
| PublicNode | `GetRPCNodePOLYGONPublicNode` | Mainnet |
| 1RPC | `GetRPCNodePOLYGON1RPC` | Mainnet |
| dRPC | `GetRPCNodePOLYGONdRPC` | Mainnet |
| Tenderly | `GetRPCNodePOLYGONTenderly` | Mainnet |
| Thirdweb | `GetRPCNodePOLYGONThirdweb` | Mainnet |
| Amoy PolygonLabs | `GetRPCNodePOLYGONAmoyPolygonLabs` | Testnet |
| Amoy PublicNode | `GetRPCNodePOLYGONAmoyPublicNode` | Testnet |
| Amoy dRPC | `GetRPCNodePOLYGONAmoydRPC` | Testnet |
| Amoy Tenderly | `GetRPCNodePOLYGONAmoyTenderly` | Testnet |

---

### G. Deposit Processor

**Package**: `payram/internal/service`
**File**: `deposit_processor_job.go` (86 lines)
**Struct**: `DepositProcessorJob`

#### Purpose

Processes pending deposits that have been detected by block monitors but require additional handling before they are fully confirmed. This includes verifying transaction receipts, updating confirmation counts, transitioning deposit statuses, and triggering webhook notifications when deposits reach the required confirmation threshold.

#### Constructor

```
NewDepositProcessorJob(
    depositService    DepositService,
    eventEmitter      EventEmitterService,
    blockchainService BlockchainService,
) *DepositProcessorJob
```

Lines 20-31 (11 lines).

#### Key Functions

**StartDepositProcessor** (Lines 31-57, 26 lines)
Starts the deposit processing loop. Sets up a ticker and enters a polling loop that calls `ProcessPendingDeposits` on each tick. Includes the standard deferred recovery pattern (25 lines).

**ProcessPendingDeposits** (Lines 67-81, 14 lines)
Calls `DepositService.ProcessAllPendingDeposits` which handles the actual processing logic. The service spawns additional goroutines (gowrap1, gowrap2) for parallel processing of deposits across different chains.

The underlying `DepositServiceImpl.ProcessAllPendingDeposits` performs:
1. Queries all deposits in "pending" or "confirming" status
2. For each deposit, checks the current block height to calculate confirmations
3. Updates the deposit status based on confirmation count:
   - Below threshold: remains "confirming"
   - At/above threshold: transitions to "confirmed"
4. For newly confirmed deposits, emits `EmitPaymentReceived` events
5. Handles the `ProcessPendingDeposit` repository method which performs the atomic status transition

**Stop** (Lines 83-86, 3 lines)
Closes `stopChan` to terminate the polling loop.

---

### H. Webhook Processor

**Package**: `payram/internal/service`
**File**: `webhook_processor_job.go` (67 lines)
**Struct**: `WebhookProcessorJob`

#### Purpose

Delivers webhook notifications to merchant-configured URLs when payment events occur (deposit received, deposit confirmed, payout created, etc.). Implements a retry strategy with exponential backoff for failed deliveries.

#### Constructor

```
NewWebhookProcessorJob(
    webhookDeliveryService WebhookDeliveryService,
    webhookService         WebhookService,
    eventEmitter           EventEmitterService,
) *WebhookProcessorJob
```

Lines 17-23 (6 lines).

#### Key Functions

**StartWebhookProcessor** (Lines 23-49, 26 lines)
Starts the webhook delivery loop. Polls for pending webhook delivery records and attempts to deliver them. Includes deferred recovery (30 lines).

**Stop** (Lines 64-67, 3 lines)
Closes `stopChan`.

#### Retry Strategy

The `WebhookDeliveryServiceImpl` implements the retry logic:

**getRetryIntervals / GetRetryIntervals**
Returns the configured retry intervals for failed webhook deliveries. The retry schedule uses increasing delays:
- Attempt 1: Immediate
- Attempt 2: 1 minute
- Attempt 3: 5 minutes
- Attempt 4: 30 minutes
- Attempt 5: 1 hour
- Attempt 6: 4 hours
- Attempt 7: 12 hours
- Attempt 8: 24 hours (max)

After all retry attempts are exhausted, the webhook delivery is marked as permanently failed.

**ProcessPayoutWebhook** (102 lines)
Handles payout-specific webhook delivery. Constructs the webhook payload with payout details, signs it with the merchant's webhook secret, and delivers it via HTTP POST. Records the delivery attempt result (success/failure, response status, response body).

**FilterReachableWebhookUrls**
Pre-filters webhook URLs to avoid wasting retries on unreachable endpoints. Performs a lightweight connectivity check (DNS resolution + TCP connect) before attempting the full webhook delivery.

#### Webhook Payload Structure

Webhooks are delivered as HTTP POST requests with:
- `Content-Type: application/json`
- `X-Webhook-Signature`: HMAC-SHA256 signature using the merchant's webhook secret
- `X-Webhook-Timestamp`: Unix timestamp of the event
- Request body: JSON payload containing event type, payment/payout details, and metadata

---

### I. Email Processor

**Package**: `payram/internal/service`
**File**: `email_processor_job.go` (64 lines)
**Struct**: `EmailProcessorJob`

#### Purpose

Processes queued email notifications. Emails are emitted as events by the `EventEmitterService` and this processor picks them up from the queue and delivers them via SMTP.

#### Constructor

```
NewEmailProcessorJob(
    eventEmitter EventEmitterService,
    emailService EmailService,
) *EmailProcessorJob
```

Lines 17-23 (6 lines).

#### Key Functions

**StartEmailProcessor** (Lines 23-49, 26 lines)
Starts the email processing loop with a ticker. On each tick, queries for pending email events and processes them. Includes deferred recovery (27 lines).

**Stop** (Lines 61-64, 3 lines)
Closes `stopChan`.

#### Email Event Types

The `EventEmitterServiceImpl` emits the following email events:

| Event | Function | Lines | Description |
|-------|----------|-------|-------------|
| Hot Wallet Balance Low | `EmitHotWalletBalanceLow` | 54 lines | Alert when hot wallet balance drops below threshold |
| Reset Password | `EmitResetPassword` | 34 lines | Password reset link email |
| Test Email | `EmitEventForTestEmail` | 29 lines | SMTP configuration test email |
| OTP | `EmitEventForOTP` | 62 lines | One-time password for 2FA |
| Payout Created | `EmitPayoutCreated` | 105 lines | Notification when a payout is created |

Each event emitter function:
1. Constructs the email template data
2. Populates sender/reply-to from SMTP configuration (`populateFromAndReplyToFromSMTPConfig`)
3. Queues the event in the events table
4. The Email Processor picks up queued events and sends them via the `EmailService`

---

### J. ERC20 Sweep Approval Processor

**Package**: `payram/internal/service`
**File**: `sweep_approval_processor.go` (926 lines)
**Struct**: `SweepApprovalProcessorJob`

#### Purpose

Monitors all EVM-compatible chains (Ethereum, Base, Polygon) for SmartSweep contract approval events. When a sweep transaction is executed on-chain via the SmartSweep smart contract, this processor detects the approval/execution event and performs the accounting updates: crediting the hot wallet, debiting the deposit addresses, updating sweep transaction statuses, and creating ledger entries.

#### Constructor

```
NewSweepApprovalProcessorJob(
    sweepService            SweepService,
    sweepTxService          SweepTransactionService,
    blockchainService       BlockchainService,
    accountService          AccountService,
    addressService          AddressService,
    configService           ConfigService,
    blockchainProcessors    map[string]*blockchainProcessorImpl,
) *SweepApprovalProcessorJob
```

Lines 54-78 (24 lines). Notable: receives a map of all blockchain processors to access their blockchain clients for querying on-chain state.

#### Key Functions

**StartSweepApprovalListener** (Lines 78-351, 273 lines)
The main processing loop. This is one of the longest single functions in the codebase. It:
1. Sets up a ticker for periodic polling
2. On each tick, queries all chains for sweep transactions in "confirming" state
3. For each confirming sweep transaction:
   a. Checks the transaction receipt to verify it was mined successfully
   b. Verifies the block confirmation count meets the threshold
   c. If confirmed, calls `addressesBalancesAccountingAfterSweep`
   d. Updates the sweep transaction status to "completed"
4. Handles multi-chain iteration (ETH, BASE, POLYGON, TRX)
5. Includes deferred recovery that spans 427 lines due to nested logic

**addressesBalancesAccountingAfterSweep** (Lines 754-923, 169 lines)
Performs the post-sweep accounting for all addresses involved in a sweep transaction:
1. Queries the sweep transaction to get all participating deposit addresses
2. For each address:
   a. Fetches the on-chain balance after the sweep
   b. Calculates the swept amount (balance before - balance after)
   c. Updates the address balance in the database
   d. Creates double-entry ledger entries (debit deposit address, credit hot wallet)
3. Updates the sweep record with final amounts and fees
4. Handles fee calculation (platform fees, network fees)
5. Handles partial sweeps where some addresses may have failed

**Stop** (Lines 923-926, 3 lines)
Closes `stopChan`.

---

### K. SCW Deposit Wallet Broadcaster

**Package**: `payram/internal/service`
**File**: `broadcast_scw_deposit_wallets_processor.go` (417 lines)
**Struct**: `BroadcastSCWDepositWalletProcessorJob`

#### Purpose

Manages the deployment of Smart Contract Wallet (SCW) deposit addresses on EVM chains. When a new deposit address is needed, the system pre-generates the counterfactual address (the address the contract will have when deployed). This worker handles the actual on-chain deployment of these SCW contracts by broadcasting the deployment transactions and tracking their status.

#### Constructor

```
NewBroadcastSCWDepositWalletProcessorJob(
    addressService           AddressService,
    addressDeploymentService AddressDeploymentService,
    blockchainService        BlockchainService,
    configService            ConfigService,
    walletService            WalletService,
    blockchainProcessors     map[string]*blockchainProcessorImpl,
) *BroadcastSCWDepositWalletProcessorJob
```

Lines 35-54 (19 lines).

#### Key Functions

**StartBroadcastSCWDepositListener** (Lines 54-93, 39 lines)
Starts the broadcasting loop with two goroutines:
1. `func1`: Recovery handler
2. `func2`: Main loop with two sub-routines:
   - `func2.1`: Calls `run()` for broadcasting new deployments
   - `func2.2`: Calls `ProcessAddressDeploymentsAccounting` for post-deployment accounting

**run** (Lines 119-126, 7 lines)
Entry point that calls `ValidatePendingAndBroadcastedAddressDeployments` followed by `BroadcastSCWDepositWalletProcessorJob`.

**ValidatePendingAndBroadcastedAddressDeployments** (Lines 138-204, 66 lines)
Checks the status of previously broadcasted deployment transactions:
1. Queries for address deployments in "pending" or "broadcasted" state
2. For each deployment, checks the on-chain transaction status
3. If the deployment transaction was mined successfully, marks the address as "deployed"
4. If the deployment transaction failed or was dropped, marks it for retry
5. Handles the case where a deployment was broadcasted but the transaction was not found on-chain (possible mempool eviction)

**BroadcastSCWDepositWalletProcessorJob** (Lines 204-388, 184 lines)
The core broadcasting function:
1. Queries for eligible SCW address pools that need deployment (`GetEligibleSCWAddressPoolsToBroadcast`)
2. Groups addresses by blockchain and wallet
3. For each batch:
   a. Builds the deployment arguments (`BuildArgumentsForDepositWalletDeployment`)
   b. Calls the factory contract's deployment function on-chain
   c. Records the deployment transaction hash
   d. Marks the address deployment as "broadcasted"
4. Handles batch size limits to avoid exceeding gas limits
5. Handles nonce management for sequential deployments

**BuildArgumentsForDepositWalletDeployment**
Standalone function that constructs the smart contract call arguments for deploying a batch of SCW deposit wallets. Uses the factory contract ABI to encode the constructor parameters including the wallet owner address, the sweep target address, and the salt for deterministic deployment (CREATE2).

**ProcessAddressDeploymentsAccounting** (Lines 126-138, 12 lines)
Post-deployment accounting: updates the address records to reflect their deployed state and creates the necessary database entries linking the on-chain contract address to the deposit address record.

**Stop** (Lines 414-417, 3 lines)
Closes `stopChan`.

---

### L. Seeder Initialize Processor

**Package**: `payram/internal/database/migrations/seeders/data`
**File**: `seeder_initialise_job.go` (217 lines)
**Struct**: `SeederInitialiseJob`

#### Purpose

Seeds the database with initial configuration data required for the system to operate. This is typically run once during initial setup or when upgrading to a new version that introduces new configuration entries. It populates RPC nodes, blockchain configurations, currencies, roles, permissions, payment channels, and JWT secrets.

#### Constructor

```
NewSeederInitialiseJob(
    db          *gorm.DB,
    configPath  string,
) *SeederInitialiseJob
```

Lines 19-26 (7 lines).

#### Key Functions

**StartSeeding** (Lines 26-212, 186 lines)
The main seeding function. Runs sequentially through all seed data categories:

1. **RPC Nodes**: Seeds all mainnet and testnet RPC node configurations for all supported chains:
   - Bitcoin: 4 nodes (mainnet preferred, NOWNodes, PublicNode, testnet preferred)
   - Ethereum: 10 nodes (6 mainnet, 4 testnet Sepolia)
   - Base: 10 nodes (6 mainnet, 4 testnet Sepolia)
   - Polygon: 9 nodes (5 mainnet, 4 testnet Amoy)
   - Tron: 2 nodes (TronGrid mainnet, Nile testnet)

2. **Blockchains**: Seeds blockchain configuration records (BTC, ETH, BASE, POLYGON, TRX) with chain IDs, confirmation requirements, block times, and explorer URLs.

3. **Currencies**: Seeds supported currency configurations (BTC, ETH, USDT, USDC, etc.) with decimal precision, contract addresses per chain, and minimum deposit/withdrawal amounts.

4. **Roles**: Seeds the role hierarchy (super_admin, admin, merchant, viewer).

5. **Permissions**: Seeds granular permissions for each role.

6. **Payment Channels**: Seeds payment channel configurations:
   - `GetPaymentChannelCrypto`: Cryptocurrency payment channel
   - `GetPaymentChannelPaymentsApp`: Card-to-crypto payment channel

7. **JWT Secrets**: Seeds JWT access and refresh token secrets (`GetJWTAccessSecret`, `GetJWTRefreshSecret`).

Uses the `RunSeeders` utility (from `seeders` package) which handles upsert logic via `processFieldChanges` and `updateNonNoUpdateFields` to avoid overwriting user-customized values.

**Stop** (Lines 214-217, 3 lines)
Closes `stopChan`. Since seeding is a one-shot operation, Stop is typically a no-op after seeding completes.

#### Supporting Utilities

| Function | Package | Description |
|----------|---------|-------------|
| `RunSeeders` | `seeders` | Orchestrates the seeding process with upsert logic (85 lines) |
| `GetSeedEntries` | `seeders` | Returns the list of seed data entries |
| `InitSeedCfg` | `seeders/utils` | Initializes seed configuration from file |
| `GenerateHash` | `seeders/utils` | Generates content hash for change detection |
| `DecimalHookFunction` | `seeders/utils` | Custom decode hook for decimal fields |
| `rpcNodesMainnet` | `seeders` | Returns mainnet RPC node seed data |
| `rpcNodesTestnet` | `seeders` | Returns testnet RPC node seed data |

---

### M. Accounting Duplicate Deposits Processor

**Package**: `payram/internal/service`
**File**: `accounting_duplicate_deposits_job.go` (30 lines)
**Struct**: `AccountingDuplicateDepositsJob`

#### Purpose

Detects and reconciles duplicate deposit records that may have been created due to race conditions, block reorganizations, or RPC node inconsistencies. When duplicate deposits are found (same transaction hash + output index + address), this processor determines the canonical record and marks duplicates appropriately to prevent double-crediting.

#### Constructor

```
NewAccountingDuplicateDepositsJob(
    depositService    DepositService,
    accountService    AccountService,
    blockchainService BlockchainService,
) *AccountingDuplicateDepositsJob
```

Lines 13-20 (7 lines).

#### Key Functions

**StartProcessing** (Lines 20-27, 7 lines)
Starts the duplicate detection loop. Runs on a relatively infrequent ticker (compared to other workers) since duplicates are rare edge cases. Queries for deposit records that share the same transaction hash and address, identifies duplicates, and either:
- Marks the duplicate as "duplicate" status (preserving the original)
- Adjusts account balances if a duplicate was erroneously credited

**Stop** (Lines 27-30, 3 lines)
Closes `stopChan`.

---

## 4. Shared Infrastructure

### blockchainProcessorImpl Base

The `blockchainProcessorImpl` struct is the shared base for all blockchain-monitoring workers (Ethereum, Base, Polygon, Tron, Bitcoin). It provides common functionality that is specialized per-chain via the `BlockchainClient` interface and chain-specific configuration.

#### Struct Fields

```go
type blockchainProcessorImpl struct {
    blockchain          *models.Blockchain        // Chain configuration from DB
    client              BlockchainClient           // Chain-specific client (ETH/BTC/TRX)
    rpcPool             *rpcpool.CentralPool       // RPC node pool with failover
    depositService      DepositService
    addressService      AddressService
    eventEmitter        EventEmitterService
    sweepService        SweepService
    sweepTxService      SweepTransactionService
    configService       ConfigService
    blockchainService   BlockchainService
    walletService       WalletService
    lastProcessedBlock  uint64                     // Last block number processed
    blockTimeHistory    []time.Duration            // Recent block time samples
    stopChan            chan struct{}
    systemAddresses     map[string]bool            // Hot wallet, fee addresses
    blacklistedAddrs    map[string]bool            // Addresses to ignore
}
```

#### CreateBlockchainProcessor (Lines 83-142, 59 lines)

Factory function that:
1. Loads the blockchain configuration from the database
2. Creates the appropriate `BlockchainClient` via `NewBlockchainClient`
3. Initializes the RPC pool with seeded nodes
4. Loads system addresses (hot wallet, fee collector)
5. Loads blacklisted addresses
6. Returns the configured processor

#### Common Methods

| Method | Lines | Description |
|--------|-------|-------------|
| `initializeClient` | 54 | Creates and connects the blockchain client |
| `Start` | 45 | Launches polling and audit goroutines |
| `Stop` | 8 | Sends shutdown signal and waits for goroutines |
| `startPolling` | 47 | Initializes and starts the block polling loop |
| `polling` | 135 | Main polling iteration logic |
| `ProcessBlockRange` | 139 | Processes a range of blocks sequentially |
| `ProcessSingleBlock` | 49 | Processes one block's transactions |
| `buildProcessBlockConfig` | 55 | Builds the config for processing a single block |
| `handleExtractedDeposit` | 407 | Processes a detected deposit |
| `handleSweepEvent` | 75 | Processes a detected sweep event |
| `handleFactoryEvent` | 106 | Processes a detected factory deployment event |
| `verifyAndUpsertDepositsForTxHash` | 183 | Verifies and upserts deposit records |
| `buildDepositDataFromExtracted` | 271 | Constructs deposit data from extracted info |
| `processConfirmingDeposits` | 32 | Checks confirming deposits for finality |
| `processConfirmingSweeps` | 161 | Checks confirming sweeps for finality |
| `identifyOurAddresses` | 22 | Checks if addresses belong to our wallets |
| `identifyOurUTXOs` | 22 | Checks if UTXOs belong to our addresses |
| `isOurUTXO` | 21 | Single UTXO ownership check |
| `createUTXOEntry` | 16 | Creates a UTXO record |
| `queueBTCSweepTransaction` | 17 | Queues a BTC sweep tx |
| `handleBTCSweepTransaction` | 41 | Processes a mined BTC sweep tx |
| `loadSystemAddresses` | 51 | Loads hot wallet and system addresses |
| `loadBlacklistedAddresses` | 30 | Loads address blacklist |
| `getSweepContractAddresses` | 48 | Gets SmartSweep contract addresses |
| `getFactoryContractAddresses` | 25 | Gets SCW factory contract addresses |
| `updateBlockTimeHistory` | 23 | Tracks block production rate |
| `calculatePollingInterval` | 63 | Dynamically adjusts polling interval |
| `hasConfirmingDeposits` | 14 | Checks if there are deposits awaiting confirmation |
| `RecoverMissedDeposits` | 70 | Rescans blocks for missed deposits |
| `runPoolMaintenance` | 12 | Triggers RPC pool health checks |
| `startWithdrawAuditProcessor` | 42 | Starts the withdrawal audit sub-processor |
| `withdrawAuditProcess` | 34 | Processes withdrawal audits |
| `auditSingleWithdrawal` | 107 | Audits a single withdrawal transaction |

### RPC Pool (bclient/rpcpool)

The RPC pool provides resilient multi-node connectivity with automatic failover, health scoring, and load balancing across RPC endpoints.

#### Architecture

```
CentralPool
  └── ChainGroup (one per blockchain)
       ├── Node (PublicNode, priority=1)
       ├── Node (LlamaRPC, priority=2)
       ├── Node (dRPC, priority=3)
       └── Node (Tenderly, priority=4)
```

#### Key Components

**CentralPool** (99 functions in package)
- `Acquire()`: Gets a connection from the best available node
- `AcquirePreferred()`: Gets a connection from the preferred (primary) node
- `AddChain()`: Registers a new chain's RPC nodes
- `RunMaintenance()`: Health checks, cooldown management, connection cleanup
- `HighestBlockSeen()`: Returns the highest block number seen across all nodes
- `NodeStates()`: Returns health status of all nodes

**ChainGroup**
- `pickBestNode()`: Selects the best node based on health score
- `acquire()`: Gets a connection with automatic failover
- `acquirePreferred()`: Gets a preferred node connection
- `emergencyRecoverySweep()`: Attempts to recover when all nodes are down (55 lines)
- `CleanupIdleConnections()`: Closes idle connections to conserve resources
- `ApplyIdleDecay()`: Reduces health scores of idle nodes

**Node**
- `EffectiveScore()`: Calculates the node's health score based on latency, failure rate, and staleness
- `HealthScore()`: Raw health score before priority adjustments
- `RecordSuccess()` / `RecordFailure()`: Updates health metrics
- `RecordRateLimit()`: Tracks rate limiting events
- `RecordStaleness()`: Tracks stale block data
- `IsAvailable()`: Checks if node is past cooldown period
- `UpdateBlockSeen()`: Updates the latest block number seen from this node

### BlockchainClient Interface

All chain clients implement the `BlockchainClient` interface (from `bclient` package). The `AbstractBlockchainClient` provides default implementations, with chain-specific overrides in `ETHBlockchainClient`, `BTCBlockchainClient`, and `TRXBlockchainClient`.

#### Key Interface Methods

| Method | Description |
|--------|-------------|
| `GetLatestBlockNumber()` | Current chain tip block number |
| `GetBlockByNumber(n)` | Fetch block data by number |
| `GetBlockByHash(h)` | Fetch block data by hash |
| `ProcessBlock(config)` | Scan block for relevant transactions |
| `ProcessBlockRange(start, end)` | Process multiple blocks |
| `GetBalance(addr)` | Native currency balance |
| `GetTokenBalance(addr, token)` | ERC20/TRC20 token balance |
| `GetTransactionDetails(hash)` | Full transaction data |
| `GetTransactionReceipt(hash)` | Transaction receipt with logs |
| `BroadcastRawTransaction(raw)` | Broadcast signed transaction |
| `SignTransaction(tx)` | Sign a transaction |
| `SignSmartContractTransaction(tx)` | Sign a contract interaction tx |
| `DeriveKeys(path)` | Derive HD wallet keys |
| `ValidateAddress(addr)` | Validate address format |
| `EnsureTxConfirmed(hash)` | Wait for transaction confirmation |
| `TestConnection()` | Test RPC node connectivity |

---

## 5. Block Monitor Flow

The following sequence describes the complete flow from block detection to webhook delivery, applicable to all chain-monitoring workers.

### Step 1: Fetch Latest Block

```
polling()
  ├── client.GetLatestBlockNumber()
  │     └── RPC call to node via rpcPool.Acquire()
  ├── Calculate: blocksToProcess = latestBlock - lastProcessedBlock
  └── If blocksToProcess > 0: call ProcessBlockRange(lastProcessed+1, latestBlock)
```

### Step 2: Scan Transactions

```
ProcessBlockRange(startBlock, endBlock)
  └── For each block in range:
        ProcessSingleBlock(blockNumber)
          ├── client.ProcessBlock(config)
          │     ├── GetNativeTransfersInBlock()   → native currency deposits
          │     ├── GetTokenTransfersInBlock()     → ERC20/TRC20 deposits
          │     ├── processSweepEventsInBlock()    → SmartSweep events
          │     └── processFactoryEventsInBlock()  → SCW deployment events
          └── Returns: []ExtractedDeposit, []SweepEvent, []FactoryEvent
```

### Step 3: Match Addresses

```
For each ExtractedDeposit:
  identifyOurAddresses(deposit.ToAddress)
    └── Query addresses table WHERE blockchain_code = ? AND address = ? AND status = 'active'
    └── If match found: this is a deposit to one of our wallets
```

For Bitcoin (UTXO-based):
```
For each UTXO in block:
  identifyOurUTXOs(utxo)
    └── isOurUTXO(utxo.ScriptPubKeyAddress)
          └── Query addresses table with WHERE clauses for blockchain, address, status
```

### Step 4: Create Deposit Record

```
handleExtractedDeposit(deposit) [407 lines]
  ├── Verify transaction on-chain (re-fetch tx details)
  ├── Check for duplicates (same txHash + address + currency)
  ├── Check blacklisted addresses
  ├── buildDepositDataFromExtracted() [271 lines]
  │     ├── Determine deposit type (native, ERC20, SCW)
  │     ├── Calculate amount with decimal precision
  │     ├── Set required confirmations from blockchain config
  │     └── Build DepositData struct
  ├── Create or update deposit record in database
  └── Set initial status: "pending" or "confirming"
```

### Step 5: Confirm Deposit

```
processConfirmingDeposits() [called each polling cycle]
  ├── Query deposits WHERE status = 'confirming'
  ├── For each deposit:
  │     ├── currentBlock = client.GetLatestBlockNumber()
  │     ├── confirmations = currentBlock - deposit.BlockNumber
  │     └── If confirmations >= requiredConfirmations:
  │           ├── verifyAndUpsertDepositsForTxHash() [183 lines]
  │           │     ├── Re-verify transaction receipt
  │           │     ├── Check tx was not reverted
  │           │     └── Update deposit status to "confirmed"
  │           └── Transition deposit to "confirmed"
  └── Continue monitoring unconfirmed deposits
```

### Step 6: Emit Webhook Event

```
On deposit confirmed:
  eventEmitter.EmitPaymentReceived(deposit)
    └── Creates event record in events table
          ├── event_type: "payment.received"
          ├── payload: {payment_id, amount, currency, tx_hash, ...}
          └── status: "pending"

WebhookProcessor picks up the event:
  ├── Constructs webhook payload
  ├── Signs with merchant webhook secret (HMAC-SHA256)
  ├── HTTP POST to merchant webhook URL
  ├── Records delivery result
  └── If failed: schedules retry per retry strategy
```

### Step 7: Trigger Sweep (asynchronous, via Account Processor)

```
AccountProcessor.processETHAutoSweep() / processBitcoinSweeps() / processERC20Sweeps()
  ├── Query confirmed deposits with sweep-eligible balances
  ├── Check: balance >= sweep threshold
  ├── Create sweep transaction record
  ├── Build and sign the sweep transaction
  ├── Broadcast to blockchain
  └── Set sweep status to "initiated" → "confirming" → "completed"
```

---

## 6. Scheduling Patterns

### Dynamic Polling Intervals

The `calculatePollingInterval` method (63 lines) in `blockchainProcessorImpl` dynamically adjusts the polling frequency based on the observed block production rate of the chain.

#### Algorithm

```
calculatePollingInterval():
  1. Maintain a sliding window of recent block timestamps (blockTimeHistory)
  2. Calculate the average block time from the window
  3. Apply a multiplier:
     - If caught up (no blocks behind): interval = avgBlockTime * 0.8
     - If slightly behind (1-5 blocks): interval = avgBlockTime * 0.5
     - If far behind (>5 blocks): interval = minimum (1-2 seconds)
  4. Clamp interval to [minInterval, maxInterval]:
     - minInterval: 1 second (to avoid excessive RPC calls)
     - maxInterval: 30 seconds (to ensure timely detection)
  5. If confirming deposits exist: reduce interval by 50%
     (to verify confirmations faster)
```

#### Chain-Specific Defaults

| Chain | Avg Block Time | Min Interval | Max Interval |
|-------|---------------|--------------|--------------|
| Ethereum | ~12s | 2s | 30s |
| Base | ~2s | 1s | 10s |
| Polygon | ~2s | 1s | 10s |
| Bitcoin | ~600s | 10s | 60s |
| Tron | ~3s | 1s | 15s |

### Worker Ticker Intervals

Non-blockchain workers use fixed ticker intervals:

| Worker | Typical Interval | Notes |
|--------|-----------------|-------|
| Account Processor | 10-30s per sub-task | Each goroutine has its own ticker |
| Deposit Processor | 5-15s | Checks for pending deposits |
| Webhook Processor | 5-10s | Checks for pending webhooks |
| Email Processor | 10-30s | Checks for pending emails |
| Sweep Approval | 15-30s | Checks for confirming sweeps |
| SCW Broadcaster | 30-60s | Checks for pending deployments |
| Accounting Dup Deposits | 60-300s | Infrequent check for duplicates |

### Block Time History Tracking

```
updateBlockTimeHistory(blockTimestamp):
  1. Calculate: elapsed = blockTimestamp - previousBlockTimestamp
  2. Append elapsed to blockTimeHistory
  3. If len(blockTimeHistory) > windowSize (e.g., 20):
       Remove oldest entry
  4. previousBlockTimestamp = blockTimestamp
```

This sliding window approach ensures the polling interval adapts to temporary changes in block production rate (e.g., network congestion, missed slots).

---

## 7. Error Handling

### Panic Recovery Pattern

Every worker goroutine uses the following recovery pattern:

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Error("Worker panic recovered", "error", r, "stack", debug.Stack())
            // Optionally restart the goroutine after a delay
        }
    }()
    // Worker logic here
}()
```

This ensures that a panic in one goroutine does not crash the entire process. The deferred recovery functions typically span 20-35 lines and include structured logging of the panic details and stack trace.

### Retry Strategies

#### Block Processing Retries

When `ProcessBlockRange` encounters an error processing a specific block:
1. Retry the block up to 3 times with exponential backoff
2. If all retries fail, skip the block and log a critical error
3. Continue processing subsequent blocks
4. The `RecoverMissedDeposits` method (70 lines) can be triggered manually or automatically to rescan skipped blocks

#### RPC Node Failover

When an RPC call fails:
1. The `rpcPool` records the failure via `Node.RecordFailure`
2. Node enters cooldown if consecutive failures exceed threshold (`enterCooldownIfNeeded`)
3. Next `Acquire()` call picks a different node via `pickBestNode`
4. If all nodes are in cooldown, `emergencyRecoverySweep` (55 lines) attempts recovery

#### Stale Transaction Handling

For sweep transactions that are "stuck":

**BTC Stale Sweep Retries** (`retryStaleBTCSweepTransactions`, 26 lines):
1. Query sweep transactions in "initiated" state older than threshold
2. Check if the transaction exists in mempool
3. If not in mempool: re-broadcast the transaction
4. If in mempool but stale: attempt RBF (Replace-By-Fee) with higher fee

**Stale Initiated Sweeps** (`processStaleInitiatedSweeps`, 51 lines):
1. Query all sweep transactions in "initiated" state beyond expected time
2. For EVM chains: check if tx was dropped from mempool
3. If dropped: reset to "pending" for re-processing
4. If still pending: increment retry counter

#### Failed Reward Processing

`processFailedRewards` (29 lines):
1. Query rewards with "failed" status
2. Check retry count against maximum (typically 3-5 retries)
3. If under max retries: reprocess with incremented retry counter
4. If max retries exceeded: mark as "permanently_failed" and alert

### Webhook Delivery Retries

See [Section H. Webhook Processor](#h-webhook-processor) for the exponential backoff retry schedule (8 attempts over 24 hours).

### Duplicate Deposit Detection

Duplicates can arise from:
- Block reorganizations (reorg) where a block is replaced
- RPC node inconsistencies returning stale data
- Race conditions between multiple polling cycles

The Accounting Duplicate Deposits Processor handles these by:
1. Querying for deposits with matching (tx_hash, address, currency) tuples
2. Keeping the record with the earliest `created_at` timestamp as canonical
3. Marking others as "duplicate" and reversing any erroneous balance credits

---

## 8. Graceful Shutdown

### Stop() Pattern

Every worker implements a `Stop()` method that follows the same pattern:

```go
func (j *SomeProcessorJob) Stop() {
    close(j.stopChan)
}
```

This is typically 3 lines. The `stopChan` is a `chan struct{}` that is created during construction and checked in every polling loop iteration.

### Polling Loop Shutdown

Worker polling loops check for the stop signal on each iteration:

```go
for {
    select {
    case <-j.stopChan:
        log.Info("Worker stopping")
        return
    case <-ticker.C:
        // Process work
    }
}
```

This ensures that the worker exits cleanly between processing cycles, typically within one ticker interval of the stop signal being sent.

### Goroutine Cleanup

For workers with multiple goroutines (e.g., Account Processor with 9):
1. `Stop()` closes the single `stopChan`
2. All goroutines share the same `stopChan` reference
3. Each goroutine's select loop picks up the close signal independently
4. The parent `WaitGroup` (if used) ensures all goroutines have exited before the process terminates

### System-Level Shutdown Sequence

When the Payminto process receives a termination signal (SIGTERM/SIGINT):

```
1. Signal handler catches SIGTERM
2. HTTP server enters graceful shutdown (stop accepting new requests)
3. SystemServiceImpl iterates all registered workers
4. Calls Stop() on each worker
5. Waits for in-flight HTTP requests to complete (with timeout)
6. Closes database connections
7. Closes RPC pool connections (CentralPool.Close())
8. Process exits
```

### Blockchain Processor Shutdown

The `blockchainProcessorImpl.Stop()` method (8 lines) performs additional cleanup:
1. Closes `stopChan` to signal the polling goroutine
2. Closes the withdrawal audit processor goroutine
3. Saves the last processed block number to the database (to resume from this point on restart)
4. Closes the blockchain client connection (`client.Close()`)

### RPC Pool Cleanup

`CentralPool.Close()` iterates all chain groups and:
1. Closes all active connections per node
2. Stops the maintenance goroutine
3. Logs final node health statistics for debugging

---

## Appendix: Worker Cross-Reference Table

| Worker | Struct | File | Start Method | Goroutines | Key Services |
|--------|--------|------|-------------|------------|--------------|
| Account Processor | `AccountProcessorJob` | `account_processor_job.go` | `StartAccountProcessor` | 9 | Deposit, Sweep, SweepTx, Blockchain, Reward, Withdrawal |
| Base ERC20 | `blockchainProcessorImpl` | `blockchain_processor_impl.go` | `Start` (via factory) | 2 | Deposit, Address, EventEmitter, Sweep |
| Bitcoin | `blockchainProcessorImpl` | `blockchain_processor_impl.go` | `StartBitcoinListener` | 2 | Deposit, Address, EventEmitter, UTXO, Sweep |
| Ethereum | `blockchainProcessorImpl` | `blockchain_processor_impl.go` | `StartEtherAndERC20Listener` | 2 | Deposit, Address, EventEmitter, Sweep |
| Tron | `blockchainProcessorImpl` | `blockchain_processor_impl.go` | `StartTronAndTRC20Listener` | 2 | Deposit, Address, EventEmitter, Sweep |
| Polygon | `blockchainProcessorImpl` | `blockchain_processor_impl.go` | `Start` (via factory) | 2 | Deposit, Address, EventEmitter, Sweep |
| Deposit Processor | `DepositProcessorJob` | `deposit_processor_job.go` | `StartDepositProcessor` | 1 | Deposit, EventEmitter, Blockchain |
| Webhook Processor | `WebhookProcessorJob` | `webhook_processor_job.go` | `StartWebhookProcessor` | 1 | WebhookDelivery, Webhook, EventEmitter |
| Email Processor | `EmailProcessorJob` | `email_processor_job.go` | `StartEmailProcessor` | 1 | EventEmitter, Email |
| Sweep Approval | `SweepApprovalProcessorJob` | `sweep_approval_processor.go` | `StartSweepApprovalListener` | 1 | Sweep, SweepTx, Blockchain, Account, Address |
| SCW Broadcaster | `BroadcastSCWDepositWalletProcessorJob` | `broadcast_scw_deposit_wallets_processor.go` | `StartBroadcastSCWDepositListener` | 2 | Address, AddressDeployment, Blockchain, Wallet |
| Seeder Initialize | `SeederInitialiseJob` | `seeder_initialise_job.go` | `StartSeeding` | 1 | DB (gorm.DB) |
| Accounting Dup Deposits | `AccountingDuplicateDepositsJob` | `accounting_duplicate_deposits_job.go` | `StartProcessing` | 1 | Deposit, Account, Blockchain |

---

## Appendix: Function Line Count Reference

Key functions sorted by size (lines of code):

| Function | Lines | Worker |
|----------|-------|--------|
| `handleExtractedDeposit` | 407 | All block monitors |
| `StartSweepApprovalListener` | 273 | Sweep Approval |
| `buildDepositDataFromExtracted` | 271 | All block monitors |
| `StartSeeding` | 186 | Seeder Initialize |
| `BroadcastSCWDepositWalletProcessorJob` | 184 | SCW Broadcaster |
| `verifyAndUpsertDepositsForTxHash` | 183 | All block monitors |
| `addressesBalancesAccountingAfterSweep` | 169 | Sweep Approval |
| `processConfirmingSweeps` | 161 | All block monitors |
| `ProcessBlockRange` | 139 | All block monitors |
| `polling` | 135 | All block monitors |
| `auditSingleWithdrawal` | 107 | All block monitors |
| `ProcessPayoutWebhook` | 102 | Webhook Processor |
| `RecoverMissedDeposits` | 70 | All block monitors |
| `ValidatePendingAndBroadcastedAddressDeployments` | 66 | SCW Broadcaster |
| `calculatePollingInterval` | 63 | All block monitors |
| `CreateBlockchainProcessor` | 59 | All block monitors |
| `ParseWorkerName` | 58 | Worker Management |
| `initializeClient` | 54 | All block monitors |
| `createAllPendingUTXOsAndAccountForSweepTransaction` | 54 | Account Processor |
| `processStaleInitiatedSweeps` | 51 | Account Processor |
| `processConfirmingSweepsOnce` | 50 | Account Processor |
| `ProcessSingleBlock` | 49 | All block monitors |
| `startPolling` | 47 | All block monitors |
| `Start` | 45 | All block monitors |
| `handleBTCSweepTransaction` | 41 | Bitcoin monitor |
| `GetAllWorkerStatuses` | 37 | Worker Management |
| `processWithdrawals` | 35 | Account Processor |
| `processERC20Sweeps` | 32 | Account Processor |
| `processRewards` | 32 | Account Processor |
| `StartAccountProcessor` | 32 | Account Processor |
| `processFailedRewards` | 29 | Account Processor |
| `createSweepTransactionPayload` | 28 | Account Processor |
| `processETHAutoSweep` | 26 | Account Processor |
| `processBitcoinSweeps` | 26 | Account Processor |
| `retryStaleBTCSweepTransactions` | 26 | Account Processor |
| `processConfirmingBTCSweeps` | 26 | Account Processor |
| `StartDepositProcessor` | 26 | Deposit Processor |
| `StartWebhookProcessor` | 26 | Webhook Processor |
| `StartEmailProcessor` | 26 | Email Processor |
| `RestartAllWorkers` | 24 | Worker Management |
| `NewAccountProcessorJob` | 24 | Account Processor |
| `NewSweepApprovalProcessorJob` | 24 | Sweep Approval |
