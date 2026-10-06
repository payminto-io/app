# Payminto — Competitive Intelligence

**Purpose:** Market positioning and competitor analysis to inform build decisions
**Date:** April 7, 2026

---

## Source Documents

This is a consolidated summary. Full competitor research is in `research/`:

| Document | Focus |
|----------|-------|
| `research/PAYRAM_INTELLIGENCE_REPORT.md` | PayRam company overview, SWOT, tech stack |
| `research/COMPETITOR_ANALYSIS.md` | Full competitive landscape |
| `research/competitors/BTCPAY_SERVER.md` | BTCPay Server deep dive (main open-source competitor) |
| `research/competitors/NOWPAYMENTS_AND_COINGATE.md` | NOWPayments & CoinGate analysis |
| `research/competitors/SHKEEPER.md` | SHKeeper analysis |
| `research/competitors/OPEN_SOURCE_DEEP_DIVE.md` | Open source alternatives |
| `research/competitors/INFRASTRUCTURE_AND_PROTOCOLS.md` | Infrastructure protocols |
| `research/DOCS_SITE_AUDIT.md` | PayRam docs site audit |
| `research/USER_FEEDBACK_AND_SENTIMENT.md` | User feedback & sentiment analysis |

---

## Competitive Positioning

### Payminto vs Competitors

| Feature | Payminto (Target) | BTCPay Server | NOWPayments | CoinGate | SHKeeper |
|---------|-------------------|---------------|-------------|----------|----------|
| Self-hosted | Yes | Yes | No (SaaS) | No (SaaS) | Yes |
| Non-custodial | Yes | Yes | No | No | Yes |
| Processing fees | 0% | 0% | 0.5-1% | 1% | 0% |
| Multi-chain | 5+ chains | Bitcoin-focused | 200+ coins | 70+ coins | Limited |
| Card-to-crypto | Yes | No | Yes | Yes | No |
| AI/MCP integration | Yes | No | No | No | No |
| SmartSweep | Yes | No | N/A | N/A | No |
| One-command deploy | Yes | Docker/LunaNode | N/A | N/A | Docker |

### Key Differentiators to Build

1. **Multi-chain from day one** — BTCPay is Bitcoin-first, we support EVM + BTC + Tron
2. **Card-to-crypto bridge** — No open-source competitor has this
3. **MCP/AI-native** — First-mover in agentic commerce
4. **SmartSweep contracts** — Automated cold storage with immutable destinations
5. **Modern dashboard** — Dark-theme, real-time, built with Next.js + shadcn/ui

### Primary Risk: BTCPay Server

BTCPay is the strongest open-source competitor. Key differences:
- BTCPay is C#/.NET; we are Go (lighter, better for blockchain)
- BTCPay is Bitcoin-centric; we are stablecoin/multi-chain native
- BTCPay has no card onramp or AI integration
- BTCPay has a larger community but targets different users
