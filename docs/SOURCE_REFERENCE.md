# Payminto — Source Code Reference

**Purpose:** Go backend source structure reverse-engineered from PayRam binary
**Date:** April 7, 2026

This document consolidates the reverse-engineered source code intelligence from PayRam's Go backend.

---

## Contents

This reference is split into three data files for practical use:

| File | Description |
|------|-------------|
| `data/PAYRAM_GO_MODELS.txt` | All Go struct definitions (models) extracted from the binary |
| `data/PAYRAM_FUNCTIONS.txt` | Complete function listing by package |
| `data/PAYRAM_SOURCE_PROJECTION.txt` | Source file structure with function signatures and line ranges |

---

## Key Packages (from binary analysis)

### Core Application
| Package | Purpose |
|---------|---------|
| `payram/internal/api` | HTTP handlers (Gin framework) |
| `payram/internal/service` | Business logic layer |
| `payram/internal/repository` | Database access (GORM) |
| `payram/internal/models` | Data models / structs |
| `payram/internal/config` | Configuration management |
| `payram/internal/worker` | Background job processors |

### Blockchain
| Package | Purpose |
|---------|---------|
| `payram/internal/blockchain` | Chain adapter interface |
| `payram/internal/blockchain/ethereum` | ETH/Base/Polygon adapter |
| `payram/internal/blockchain/bitcoin` | BTC adapter |
| `payram/internal/blockchain/tron` | TRX adapter |

### Supporting
| Package | Purpose |
|---------|---------|
| `github.com/PayRam/activity-log` | Activity logging middleware |
| `github.com/PayRam/payram-node-sdk` | Node.js SDK (separate repo) |

### Worker Processes (from log files)
| Worker | Log File | Purpose |
|--------|----------|---------|
| Account Processor | `account_processor.log` | Account balance processing |
| Base Processor | `base_processor.log` | Base chain block monitoring |
| Bitcoin Processor | `bitcoin_processor.log` | BTC block monitoring |
| Deposit Processor | `deposit_processor.log` | Deposit detection & confirmation |
| ETH Processor | `eth_processor.log` | Ethereum block monitoring |
| TRX Processor | `trx_processor.log` | Tron block monitoring |
| Webhook Processor | `webhook_processor.log` | Webhook delivery & retries |
| Email Processor | `email_processor.log` | Email notifications |
| ERC20 Sweep Approval | `erc20_sweep_approval_processor.log` | Token sweep approvals |
| SCW Deposit Wallet | `broadcast_scw_deposit_wallet_processor.log` | Smart contract wallet broadcasts |

---

## How to Use This Reference

When building Payminto's Go backend:
1. Use `PAYRAM_GO_MODELS.txt` to understand the exact data model structures
2. Use `PAYRAM_SOURCE_PROJECTION.txt` to understand the repository/service/API layer organization
3. Use `PAYRAM_FUNCTIONS.txt` to see the full function inventory by package
4. The worker processes above tell you exactly which background processors to implement
