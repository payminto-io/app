# 30 Payment links on the switch, with every enabled method

Status: ready-for-agent
Owner: Backend engineer with Frontend engineer
Blocked by: 03, 04, 05

## Goal
Owner direction 2026-10-07: link creation shows only two methods. Root cause: links still create payments through the legacy Payminto creator, which offers only crypto priced in USD. Wire links to the switch core instead.

- A switch-backed `PaymentCreator` (idempotent on LinkPaymentID, `FindPayment`, `ErrNotCreated` only when certain) creates a switch intent per link use; the attempt goes through routing when ticket 06 lands, else the switch's default selector.
- `Offerings` lists every enabled connector in the process environment with an active fee rule: card (Keypay), UPI (Payvang), and chain deposits as separate choices per chain and asset (USDC and USDT on Solana, Base, Ethereum, Polygon, Tron as seeded), each with its currencies.
- The builder's payment step becomes a multi-select grouped by method (card, bank/UPI, stablecoin with chain and asset), each row showing the server's fee preview; nothing is hard-coded in the frontend.
- The public render model and checkout list every method the link allows; the legacy creator stays available behind a flag until removed.

## Acceptance
A link with card, UPI and USDC on Solana publishes and its checkout offers all three; each paid use creates exactly one switch intent and one payment journal; the builder lists exactly what the options endpoint returns; tests for each.
