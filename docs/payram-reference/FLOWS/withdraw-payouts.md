# Withdraw / payouts

PayRam exposes three withdraw flows from a single sub-nav: paying customers
back (`user-payouts`), paying referral commissions (`referral-payouts`), and
issuing refunds against past payments (`refunds`). All three share the same
**address book** lookup at `/withdraw/address-book`.

## Screens

| # | Route | Spec |
|---|---|---|
| 1 | `/withdraw/address-book` | [withdraw-address-book.md](../SCREENS/withdraw-address-book.md) |
| 2 | `/withdraw/address-book/{id}` | [withdraw-address-book-id.md](../SCREENS/withdraw-address-book-id.md) |
| 3 | `/withdraw/user-payouts` | [withdraw-user-payouts.md](../SCREENS/withdraw-user-payouts.md) |
| 4 | `/withdraw/referral-payouts` | [withdraw-referral-payouts.md](../SCREENS/withdraw-referral-payouts.md) |
| 5 | `/withdraw/refunds` | [withdraw-refunds.md](../SCREENS/withdraw-refunds.md) |
| 6 | `/settings/withdrawal` (config) | [settings-withdrawal.md](../SCREENS/settings-withdrawal.md) |

## Network calls observed

- `GET /api/v1/recipients/` — the unified address book (returns `{ recipients: [...] }`)
- `GET /api/v1/external-platform/all/withdrawal` — pending + completed payout list
- `GET /api/v1/configuration/key/withdrawal-payout-min-amount` — sanity bound
- `GET /api/v1/external-platform/all/referral/payouts` — referral-specific list
- `GET /api/v1/external-platform/all/referral/campaigns` — campaign sources

## User payouts sequence

1. Merchant lands on `/withdraw/user-payouts`. The table loads from the
   list endpoint above.
2. Filter chips by status: `Pending Approval`, `Approved`, `Processing`,
   `Completed`, `Failed`. (Same chip pattern as `payments/allPayments`.)
3. Click **"New Payout"** → modal with:
   - Recipient picker (autocomplete against `/recipients/`)
   - Asset selector (per-network grid)
   - Amount input
   - Memo / reference field
   - Approval flag (auto-checked if member is admin)
4. Submit → `POST /api/v1/external-platform/all/withdrawal` (mutating, not
   in our crawl). Returns the new pending payout row.
5. **CSV batch upload:** also supports a CSV path for bulk payouts. Drop
   zone parses the CSV, shows a preview table, and lets you select which
   rows to submit. (Component documented in
   [`COMPONENT_LIBRARY.md`](../COMPONENT_LIBRARY.md) §5.)

## Approval workflow

If the merchant has the `write_withdrawal_approve` permission (we saw it in
the JWT we minted: `"perms":["write_withdrawal_approve"]`), they see an
**Approve / Reject** action menu on each `Pending Approval` row. Without
that permission the row is read-only.

The approval action calls
`POST /api/v1/external-platform/all/withdrawal/{id}/approve` (mutating).
Once approved, the payout transitions to `Processing` and the
`payram start-account-processor` worker actually broadcasts the on-chain tx.

## Refund flow

Refunds are issued **from a payment row**, not standalone. From
`/project/all/payments/allPayments`, click a row → drawer → "Refund" button.
The refund modal pre-fills:

- Original recipient address (from the deposit)
- Original asset and amount (capped at the original; partial refunds allowed)
- Reason (free text)

Submit → `POST /api/v1/external-platform/all/payment/{id}/refund` (mutating).
The refund row appears in `/withdraw/refunds` and the original payment row
gets a `Refunded` badge.

## Address book

`/withdraw/address-book` is a CRUD list of named recipients (label, address,
chain, optional email, tags). Used as the autocomplete source for every
payout/refund modal. Supports CSV import.

## Settings → Withdrawal

`/settings/withdrawal` is where the merchant configures:

- Per-asset minimum payout amount
- Cold-wallet destination per chain (validated on save)
- Auto-approval threshold (payouts under $X auto-approve, above $X require
  manual approval)
- Daily cap

`GET /api/v1/configuration/key/withdrawal-payout-min-amount` is one slice of
this; the full config goes through generic
`/api/v1/configuration/{key}` getters and putters.

## Gaps vs. Payminto clone

- `/withdraw/user-payouts` and `/withdraw/referral-payouts` exist as empty
  shells in Payminto without row actions, batch select, or status filters.
- `/withdraw/address-book` exists as an empty state — no CRUD, no CSV upload.
- `/withdraw/refunds` exists as an empty state.
- The "Refund" affordance from a payment row is missing entirely.
- `/settings/withdrawal` exists as a stub.
