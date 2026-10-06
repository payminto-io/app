# PayRam API Contracts (observed)

Aggregated from every `network.json` produced by the Phase 2 crawl. This
captures only endpoints actually invoked by the dashboard during initial page
loads + 158 safe button interactions — write-mutating endpoints that require
form submissions (create payment link, deploy contract, invite user, etc.) are
documented in `FLOWS/*.md` instead.

- **Unique endpoints:** 62
- **Base URL:** `http://localhost:8080/api/v1/` (or whatever
  `NEXT_PUBLIC_BACKEND_URL` points at)
- **Auth:** `Authorization: Bearer <jwt>` from `localStorage.payram_access_token`.
- **Refresh:** `localStorage.payram_refresh_token` is exchanged via the same
  `/signin`-style flow when a 401 is received (not directly observed).

## Conventions

- Numeric IDs and the literal string `all` (the meta-project) are normalized
  to `{id}` in paths below.
- The pseudo-project `/project/all/...` aggregates every real project; PayRam
  ships with this default and most dashboard pages call the `all` variant.
- Many endpoints respond with an array directly at the root (no envelope) —
  these show as `{ 0, 1, 2, ... }` in the response shape column.
- A `POST /api/v1/websocket-token/create` is fired on every page load —
  PayRam opens a websocket per browser tab for live deposit/sweep updates.

## Endpoints by namespace

### `/api/v1/activity-log`

- **GET /api/v1/activity-log** — status `200` · seen 3× across 1 route(s)
  - query: `limit`, `offset`, `order`, `excludeActionStatusPairs`, `startDate`, `endDate`
  - response: `{ data, total_count }`
  - callers: [settings-activityLog](SCREENS/settings-activityLog.md)
- **GET /api/v1/activity-log/event-categories** — status `200` · seen 1× across 1 route(s)
  - response: `array<string>`
  - callers: [settings-activityLog](SCREENS/settings-activityLog.md)

### `/api/v1/addresses`

- **GET /api/v1/addresses/balance** — status `200` · seen 1× across 1 route(s)
  - response: `array<object>`
  - callers: [sweepIn](SCREENS/sweepIn.md)

### `/api/v1/blockchain`

- **GET /api/v1/blockchain/ETH** — status `200` · seen 1× across 1 route(s)
  - response: `{ id, createdAt, updatedAt, code, name, family, client, height, heightTimestamp, explorerAddress, explorerTransaction, minConfirmations }`
  - callers: [sweepIn](SCREENS/sweepIn.md)

### `/api/v1/blockchain-contract`

- **GET /api/v1/blockchain-contract/blockchain/TRX/contract/factory_contract** — status `200` · seen 3× across 1 route(s)
  - response: `{ id, createdAt, updatedAt, blockchainCode, contractType, abi, bytecode, address, description, status, blockchain_id, currencyStandard }`
  - callers: [manageWallet-sweepContract-deploy](SCREENS/manageWallet-sweepContract-deploy.md)
- **GET /api/v1/blockchain-contract/contract/factory_contract/address-contract/sweep_approval** — status `200` · seen 1× across 1 route(s)
  - response: `array<object>`
  - callers: [manageWallet-sweepContract](SCREENS/manageWallet-sweepContract.md)

### `/api/v1/blockchain-currency`

- **GET /api/v1/blockchain-currency** — status `200` · seen 7× across 5 route(s)
  - response: `array<object>`

### `/api/v1/blockchains`

- **GET /api/v1/blockchains** — status `200` · seen 155× across 70 route(s)
  - response: `array<object>`

### `/api/v1/config`

- **GET /api/v1/config/smtp/** — status `404` · seen 1× across 1 route(s)
  - response: `{ error }`
  - callers: [settings-integrations](SCREENS/settings-integrations.md)

### `/api/v1/configuration`

- **GET /api/v1/configuration/default** — status `200` · seen 7× across 6 route(s)
  - response: `{ server, walletConnectID, tronHTTPProvider, tronAPIKey, sweepBatchSizeETH, sweepApprovalBatchSizeETH, sweepBatchSizeTRX, sweepApprovalBatchSizeTRX }`
- **GET /api/v1/configuration/key/wallet_connect_id** — status `200` · seen 6× across 4 route(s)
  - response: `{ id, createdAt, updatedAt, key, value, overrideValue, description, encrypt }`
  - callers: [manageWallet-deposit-wallet](SCREENS/manageWallet-deposit-wallet.md), [manageWallet-hot-wallet](SCREENS/manageWallet-hot-wallet.md), [manageWallet-hot-wallet-id](SCREENS/manageWallet-hot-wallet-id.md), [settings-integrations](SCREENS/settings-integrations.md)
- **GET /api/v1/configuration/key/withdrawal-payout-min-amount** — status `200` · seen 1× across 1 route(s)
  - response: `{ id, createdAt, updatedAt, key, value, overrideValue, description, encrypt }`
  - callers: [settings-withdrawal](SCREENS/settings-withdrawal.md)

### `/api/v1/contract-address`

- **GET /api/v1/contract-address/blockchain/ETH/contract/sweep_approval** — status `404` · seen 1× across 1 route(s)
  - response: `{ error }`
  - callers: [manageWallet-sweepContract](SCREENS/manageWallet-sweepContract.md)
- **GET /api/v1/contract-address/blockchain/TRX/contract/sweep_approval** — status `404` · seen 3× across 1 route(s)
  - response: `{ error }`
  - callers: [manageWallet-sweepContract-deploy](SCREENS/manageWallet-sweepContract-deploy.md)

### `/api/v1/external-platform`

- **GET /api/v1/external-platform/{id}** — status `200` · seen 82× across 70 route(s)
  - response: `array<object>`
- **GET /api/v1/external-platform/{id}/analytics/groups** — status `200` · seen 14× across 7 route(s)
  - response: `array<object>`
- **GET /api/v1/external-platform/{id}/members** — status `200` · seen 2× across 2 route(s)
  - response: `{ members }`
  - callers: [project-projectId-payments-createPaymentLink](SCREENS/project-projectId-payments-createPaymentLink.md), [settings-paymentsApp](SCREENS/settings-paymentsApp.md)
- **GET /api/v1/external-platform/{id}/referral/campaigns** — status `200` · seen 8× across 3 route(s)
  - response: `array<undefined>`
  - callers: [project-projectId-growth-analytics](SCREENS/project-projectId-growth-analytics.md), [project-projectId-growth-analytics-promoter-promoterId](SCREENS/project-projectId-growth-analytics-promoter-promoterId.md), [project-projectId-growth-campaigns](SCREENS/project-projectId-growth-campaigns.md)
- **GET /api/v1/external-platform/{id}/referral/payouts** — status `200` · seen 1× across 1 route(s)
  - callers: [withdraw-referral-payouts](SCREENS/withdraw-referral-payouts.md)
- **GET /api/v1/external-platform/{id}/withdrawal** — status `200` · seen 1× across 1 route(s)
  - response: `array<undefined>`
  - callers: [withdraw-user-payouts](SCREENS/withdraw-user-payouts.md)
- **GET /api/v1/external-platform/details** — status `200` · seen 81× across 70 route(s)
  - response: `array<object>`
- **POST /api/v1/external-platform/{id}/analytics/campaign_count** — status `200` · seen 3× across 1 route(s)
  - query: `status`, `startDateMax`
  - request: `{ analytics_date_filter }`
  - response: `{ id, type, data, attributes }`
  - callers: [project-projectId-growth-analytics](SCREENS/project-projectId-growth-analytics.md)
- **POST /api/v1/external-platform/{id}/analytics/groups/{id}/graph/{id}/data** — status `200` · seen 172× across 7 route(s)
  - request: `{ analytics_date_filter }`
  - response: `{ id, type, data, attributes }`
- **POST /api/v1/external-platform/{id}/analytics/promoters** — status `200` · seen 3× across 1 route(s)
  - request: `{ analytics_date_filter, status }`
  - response: `{ id, type, data, attributes }`
  - callers: [project-projectId-growth-analytics](SCREENS/project-projectId-growth-analytics.md)
- **POST /api/v1/external-platform/{id}/analytics/promoters_count** — status `200` · seen 3× across 1 route(s)
  - request: `{ analytics_date_filter }`
  - response: `{ id, type, data, attributes }`
  - callers: [project-projectId-growth-analytics](SCREENS/project-projectId-growth-analytics.md)
- **POST /api/v1/external-platform/{id}/analytics/referee_count** — status `200` · seen 3× across 1 route(s)
  - request: `{ analytics_date_filter }`
  - response: `{ id, type, data, attributes }`
  - callers: [project-projectId-growth-analytics](SCREENS/project-projectId-growth-analytics.md)
- **POST /api/v1/external-platform/{id}/analytics/reward_claimed** — status `200` · seen 3× across 1 route(s)
  - request: `{ analytics_date_filter }`
  - response: `{ id, type, data, attributes }`
  - callers: [project-projectId-growth-analytics](SCREENS/project-projectId-growth-analytics.md)
- **POST /api/v1/external-platform/{id}/analytics/reward_value** — status `200` · seen 5× across 2 route(s)
  - request: `{ analytics_date_filter }`
  - response: `{ id, type, data, attributes }`
  - callers: [project-projectId-growth-analytics](SCREENS/project-projectId-growth-analytics.md), [project-projectId-growth-campaigns](SCREENS/project-projectId-growth-campaigns.md)
- **POST /api/v1/external-platform/{id}/analytics/total_revenue** — status `200` · seen 3× across 1 route(s)
  - request: `{ analytics_date_filter }`
  - response: `{ id, type, data, attributes }`
  - callers: [project-projectId-growth-analytics](SCREENS/project-projectId-growth-analytics.md)
- **POST /api/v1/external-platform/{id}/payment/search** — status `200` · seen 8× across 1 route(s)
  - request: `{ Query, sortBy, sortDirection, limit, offset, paymentStatus, webhookStatus, currency, network, createdBy, dateFrom, dateTo }`
  - response: `{ data, totalCount, attributes }`
  - callers: [project-projectId-payments-allPayments](SCREENS/project-projectId-payments-allPayments.md)
- **POST /api/v1/external-platform/{id}/payment/summary** — status `200` · seen 1× across 1 route(s)
  - request: `{  }`
  - response: `{ totalCount, closedCount, openCount, cancelledCount }`
  - callers: [project-projectId-payments-allPayments](SCREENS/project-projectId-payments-allPayments.md)

### `/api/v1/health`

- **GET /api/v1/health** — status `404` · seen 1× across 1 route(s)
  - callers: [settings-updater](SCREENS/settings-updater.md)

### `/api/v1/internalMembers`

- **POST /api/v1/internalMembers** — status `200` · seen 3× across 2 route(s)
  - request: `{ query, sortBy, sortDirection, limit, offset }`
  - response: `{ members }`
  - callers: [settings-activityLog](SCREENS/settings-activityLog.md), [settings-userManagement](SCREENS/settings-userManagement.md)

### `/api/v1/missed-deposit`

- **GET /api/v1/missed-deposit** — status `200` · seen 1× across 1 route(s)
  - query: `startDate`, `endDate`
  - response: `array<undefined>`
  - callers: [project-projectId-payments-missed-payments](SCREENS/project-projectId-payments-missed-payments.md)

### `/api/v1/onramper-payments`

- **GET /api/v1/onramper-payments** — status `200` · seen 1× across 1 route(s)
  - query: `startDate`, `endDate`, `limit`, `offset`, `sortBy`, `order`
  - response: `{ data, totalCount, limit, offset }`
  - callers: [onramp-payments](SCREENS/onramp-payments.md)
- **GET /api/v1/onramper-payments/metrics** — status `200` · seen 1× across 1 route(s)
  - query: `startDate`, `endDate`
  - response: `{ totalOnrampValue, totalFeeExpense, onrampContributionPercent, uniqueOnrampUsers }`
  - callers: [onramp-payments](SCREENS/onramp-payments.md)

### `/api/v1/payment-channels`

- **GET /api/v1/payment-channels** — status `200` · seen 2× across 2 route(s)
  - response: `{ channels, total }`
  - callers: [onramp-payments](SCREENS/onramp-payments.md), [settings-paymentChannels](SCREENS/settings-paymentChannels.md)
- **GET /api/v1/payment-channels/project/{id}** — status `200` · seen 1× across 1 route(s)
  - response: `array<object>`
  - callers: [onramp-payments](SCREENS/onramp-payments.md)

### `/api/v1/payments-app`

- **GET /api/v1/payments-app** — status `200` · seen 2× across 1 route(s)
  - query: `projectIDs`
  - response: `array<object>`
  - callers: [onramp-payments](SCREENS/onramp-payments.md)

### `/api/v1/project`

- **GET /api/v1/project/{id}/blockchain-currency** — status `200` · seen 1× across 1 route(s)
  - response: `array<object>`
  - callers: [onramp-payments](SCREENS/onramp-payments.md)

### `/api/v1/recipients`

- **GET /api/v1/recipients/** — status `200` · seen 3× across 2 route(s)
  - query: `ids`
  - response: `{ recipients }`
  - callers: [withdraw-address-book](SCREENS/withdraw-address-book.md), [withdraw-address-book-id](SCREENS/withdraw-address-book-id.md)

### `/api/v1/referral`

- **GET /api/v1/referral/referrers** — status `401` · seen 1× across 1 route(s)
  - response: `{ error }`
  - callers: [referral-token](SCREENS/referral-token.md)

### `/api/v1/roles`

- **GET /api/v1/roles** — status `200` · seen 1× across 1 route(s)
  - response: `{ roles }`
  - callers: [settings-userManagement](SCREENS/settings-userManagement.md)

### `/api/v1/secrets-vault-activities`

- **GET /api/v1/secrets-vault-activities** — status `200` · seen 2× across 2 route(s)
  - response: `array<object>`
  - callers: [manageWallet-gasFeeWallet-add](SCREENS/manageWallet-gasFeeWallet-add.md), [manageWallet-gasFeeWallet-edit](SCREENS/manageWallet-gasFeeWallet-edit.md)

### `/api/v1/secrets-vaults`

- **GET /api/v1/secrets-vaults** — status `404` · seen 1× across 1 route(s)
  - callers: [manageWallet-gasFeeWallet](SCREENS/manageWallet-gasFeeWallet.md)

### `/api/v1/sweeps`

- **GET /api/v1/sweeps** — status `200` · seen 1× across 1 route(s)
  - query: `count`
  - response: `array<undefined>`
  - callers: [sweepIn](SCREENS/sweepIn.md)

### `/api/v1/system`

- **GET /api/v1/system** — status `200` · seen 4× across 3 route(s)
  - response: `{ code, frontendServer, backendServer, server, blockchainNetwork }`
  - callers: [settings-account](SCREENS/settings-account.md), [settings-account-projectId](SCREENS/settings-account-projectId.md), [settings-mobileApp](SCREENS/settings-mobileApp.md)
- **GET /api/v1/system/updater/history** — status `404` · seen 2× across 1 route(s)
  - query: `limit`
  - callers: [settings-updater](SCREENS/settings-updater.md)
- **GET /api/v1/system/updater/inspect** — status `404` · seen 2× across 1 route(s)
  - callers: [settings-updater](SCREENS/settings-updater.md)
- **GET /api/v1/system/updater/status** — status `404` · seen 1× across 1 route(s)
  - callers: [settings-updater](SCREENS/settings-updater.md)

### `/api/v1/ticker`

- **GET /api/v1/ticker** — status `200` · seen 1× across 1 route(s)
  - response: `array<object>`
  - callers: [payments](SCREENS/payments.md)

### `/api/v1/version`

- **GET /api/v1/version** — status `404` · seen 1× across 1 route(s)
  - callers: [settings-updater](SCREENS/settings-updater.md)

### `/api/v1/wallets`

- **GET /api/v1/wallets** — status `200` · seen 6× across 6 route(s)
  - query: `walletType`
  - response: `array<object>`
- **GET /api/v1/wallets/{id}** — status `200` · seen 2× across 2 route(s)
  - response: `{ id, createdAt, updatedAt, name, family, secretType, walletType, walletSubType, publicKey, memberID, status, default }`
  - callers: [manageWallet-hot-wallet-id](SCREENS/manageWallet-hot-wallet-id.md), [manageWallet-wallets-details-id](SCREENS/manageWallet-wallets-details-id.md)
- **GET /api/v1/wallets/balance** — status `400` · seen 1× across 1 route(s)
  - response: `{ error }`
  - callers: [manageWallet-wallets-details-id](SCREENS/manageWallet-wallets-details-id.md)

### `/api/v1/websocket-token`

- **POST /api/v1/websocket-token/create** — status `200` · seen 70× across 70 route(s)
  - query: `clientID`
  - response: `{ id, createdAt, updatedAt, token, serverUrl, expiry, clientId, memberId, Member }`

### `/null/api/v1`

- **GET /null/api/v1/blockchain-currency/reference/null** — status `404` · seen 1× across 1 route(s)
  - callers: [payments](SCREENS/payments.md)
- **GET /null/api/v1/external-platform/reference/null** — status `404` · seen 1× across 1 route(s)
  - callers: [payments](SCREENS/payments.md)
- **GET /null/api/v1/payment-channels/reference/null** — status `404` · seen 1× across 1 route(s)
  - callers: [payments](SCREENS/payments.md)
- **GET /null/api/v1/payment/reference/null** — status `404` · seen 1× across 1 route(s)
  - callers: [payments](SCREENS/payments.md)
- **GET /null/api/v1/wallet/reference/null** — status `404` · seen 1× across 1 route(s)
  - callers: [payments](SCREENS/payments.md)
- **GET /null/api/v1/wallets/reference/null** — status `404` · seen 1× across 1 route(s)
  - callers: [payments](SCREENS/payments.md)

## Known 404 / wishful endpoints

The frontend code calls these but they 404 against the current build —
either dead code, feature-flagged, or stripped from the OSS build:

- `GET /api/v1/health`, `/api/v1/version`
- `GET /api/v1/system/updater/{status,inspect,history}` — settings/updater UI
  is wired up but the backend route is gone
- `GET /api/v1/secrets-vaults`, `/api/v1/contract-address/blockchain/*/contract/sweep_approval`
- `GET /api/v1/config/smtp/` — `settings/emailConfig` calls this
- `GET /null/api/v1/...` (literal `null` in the URL) — bug in the SPA when
  a project context is missing; harmless but worth fixing in our clone
