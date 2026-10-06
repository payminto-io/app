# PayRam Route Inventory

Canonical list of every route exposed by PayRam's Next.js dashboard build,
extracted from `_raw/manifests/app-path-routes-manifest.json` and verified by
crawling each one against the live container at `http://localhost:8880`.

- **Total routes:** 75
- **Crawl summary:** 75/75 succeeded, 158 interactions captured
- **Crawled at:** 2026-04-07T09:05:37.654Z

`auth` = whether a logged-in JWT is required to render the page meaningfully.
Auth/public routes redirect logged-in users elsewhere; dashboard routes redirect
unauthenticated users to `/login`.

`api` = number of distinct `/api/v1/*` endpoints called on first paint
(useful gap-analysis signal — high-API routes are data-heavy).

`int` = number of safe interactions Playwright successfully clicked through
during the crawl (modals/tabs/dropdowns; capped at 10 per route).

| Route | Group | Auth | Params | API | Int | Artifacts |
|---|---|---|---|---|---|---|
| `/about` | misc | yes | no | 4 | 1 | [shot](captures/about/screenshot.png) · [dom](captures/about/dom.html) · [spec](SCREENS/about.md) |
| `/addresses` | public | no | no | 0 | 1 | [shot](captures/addresses/screenshot.png) · [dom](captures/addresses/dom.html) · [spec](SCREENS/addresses.md) |
| `/createPassword` | auth | no | no | 6 | 1 | [shot](captures/createPassword/screenshot.png) · [dom](captures/createPassword/dom.html) · [spec](SCREENS/createPassword.md) |
| `/dashboard` | dashboard | yes | no | 4 | 2 | [shot](captures/dashboard/screenshot.png) · [dom](captures/dashboard/dom.html) · [spec](SCREENS/dashboard.md) |
| `/developers/apiKeys` | developers | yes | no | 4 | 1 | [shot](captures/developers-apiKeys/screenshot.png) · [dom](captures/developers-apiKeys/dom.html) · [spec](SCREENS/developers-apiKeys.md) |
| `/developers/documentation` | developers | yes | no | 4 | 1 | [shot](captures/developers-documentation/screenshot.png) · [dom](captures/developers-documentation/dom.html) · [spec](SCREENS/developers-documentation.md) |
| `/developers/webhook` | developers | yes | no | 4 | 1 | [shot](captures/developers-webhook/screenshot.png) · [dom](captures/developers-webhook/dom.html) · [spec](SCREENS/developers-webhook.md) |
| `/forgotPassword` | auth | no | no | 6 | 1 | [shot](captures/forgotPassword/screenshot.png) · [dom](captures/forgotPassword/dom.html) · [spec](SCREENS/forgotPassword.md) |
| `/login` | auth | no | no | 6 | 1 | [shot](captures/login/screenshot.png) · [dom](captures/login/dom.html) · [spec](SCREENS/login.md) |
| `/manageWallet` | wallets | yes | no | 4 | 1 | [shot](captures/manageWallet/screenshot.png) · [dom](captures/manageWallet/dom.html) · [spec](SCREENS/manageWallet.md) |
| `/manageWallet/deposit-wallet` | wallets | yes | no | 7 | 5 | [shot](captures/manageWallet-deposit-wallet/screenshot.png) · [dom](captures/manageWallet-deposit-wallet/dom.html) · [spec](SCREENS/manageWallet-deposit-wallet.md) |
| `/manageWallet/gasFeeWallet` | wallets | yes | no | 5 | 2 | [shot](captures/manageWallet-gasFeeWallet/screenshot.png) · [dom](captures/manageWallet-gasFeeWallet/dom.html) · [spec](SCREENS/manageWallet-gasFeeWallet.md) |
| `/manageWallet/gasFeeWallet/add` | wallets | yes | no | 5 | 2 | [shot](captures/manageWallet-gasFeeWallet-add/screenshot.png) · [dom](captures/manageWallet-gasFeeWallet-add/dom.html) · [spec](SCREENS/manageWallet-gasFeeWallet-add.md) |
| `/manageWallet/gasFeeWallet/edit` | wallets | yes | no | 5 | 2 | [shot](captures/manageWallet-gasFeeWallet-edit/screenshot.png) · [dom](captures/manageWallet-gasFeeWallet-edit/dom.html) · [spec](SCREENS/manageWallet-gasFeeWallet-edit.md) |
| `/manageWallet/hot-wallet` | wallets | yes | no | 7 | 3 | [shot](captures/manageWallet-hot-wallet/screenshot.png) · [dom](captures/manageWallet-hot-wallet/dom.html) · [spec](SCREENS/manageWallet-hot-wallet.md) |
| `/manageWallet/hot-wallet/1` | wallets | yes | yes | 8 | 2 | [shot](captures/manageWallet-hot-wallet-id/screenshot.png) · [dom](captures/manageWallet-hot-wallet-id/dom.html) · [spec](SCREENS/manageWallet-hot-wallet-id.md) |
| `/manageWallet/sweepContract` | wallets | yes | no | 6 | 3 | [shot](captures/manageWallet-sweepContract/screenshot.png) · [dom](captures/manageWallet-sweepContract/dom.html) · [spec](SCREENS/manageWallet-sweepContract.md) |
| `/manageWallet/sweepContract/deploy` | wallets | yes | no | 7 | 2 | [shot](captures/manageWallet-sweepContract-deploy/screenshot.png) · [dom](captures/manageWallet-sweepContract-deploy/dom.html) · [spec](SCREENS/manageWallet-sweepContract-deploy.md) |
| `/manageWallet/wallets` | wallets | yes | no | 5 | 2 | [shot](captures/manageWallet-wallets/screenshot.png) · [dom](captures/manageWallet-wallets/dom.html) · [spec](SCREENS/manageWallet-wallets.md) |
| `/manageWallet/wallets/add` | wallets | yes | no | 4 | 2 | [shot](captures/manageWallet-wallets-add/screenshot.png) · [dom](captures/manageWallet-wallets-add/dom.html) · [spec](SCREENS/manageWallet-wallets-add.md) |
| `/manageWallet/wallets/cold` | wallets | yes | no | 5 | 1 | [shot](captures/manageWallet-wallets-cold/screenshot.png) · [dom](captures/manageWallet-wallets-cold/dom.html) · [spec](SCREENS/manageWallet-wallets-cold.md) |
| `/manageWallet/wallets/details/1` | wallets | yes | yes | 6 | 2 | [shot](captures/manageWallet-wallets-details-id/screenshot.png) · [dom](captures/manageWallet-wallets-details-id/dom.html) · [spec](SCREENS/manageWallet-wallets-details-id.md) |
| `/merchantMock` | misc | yes | no | 4 | 1 | [shot](captures/merchantMock/screenshot.png) · [dom](captures/merchantMock/dom.html) · [spec](SCREENS/merchantMock.md) |
| `/onramp-payments` | onramp | yes | no | 12 | 4 | [shot](captures/onramp-payments/screenshot.png) · [dom](captures/onramp-payments/dom.html) · [spec](SCREENS/onramp-payments.md) |
| `/pay` | public | no | no | 0 | 0 | [shot](captures/pay/screenshot.png) · [dom](captures/pay/dom.html) · [spec](SCREENS/pay.md) |
| `/payments` | public | no | no | 7 | 1 | [shot](captures/payments/screenshot.png) · [dom](captures/payments/dom.html) · [spec](SCREENS/payments.md) |
| `/project/all/accounts/revenue` | accounts | yes | yes | 4 | 2 | [shot](captures/project-projectId-accounts-revenue/screenshot.png) · [dom](captures/project-projectId-accounts-revenue/dom.html) · [spec](SCREENS/project-projectId-accounts-revenue.md) |
| `/project/all/accounts/userBalance` | accounts | yes | yes | 4 | 2 | [shot](captures/project-projectId-accounts-userBalance/screenshot.png) · [dom](captures/project-projectId-accounts-userBalance/dom.html) · [spec](SCREENS/project-projectId-accounts-userBalance.md) |
| `/project/all/customers` | customers | yes | yes | 4 | 2 | [shot](captures/project-projectId-customers/screenshot.png) · [dom](captures/project-projectId-customers/dom.html) · [spec](SCREENS/project-projectId-customers.md) |
| `/project/all/dashboard` | dashboard | yes | yes | 6 | 7 | [shot](captures/project-projectId-dashboard/screenshot.png) · [dom](captures/project-projectId-dashboard/dom.html) · [spec](SCREENS/project-projectId-dashboard.md) |
| `/project/all/growth/analytics` | growth | yes | yes | 12 | 4 | [shot](captures/project-projectId-growth-analytics/screenshot.png) · [dom](captures/project-projectId-growth-analytics/dom.html) · [spec](SCREENS/project-projectId-growth-analytics.md) |
| `/project/all/growth/analytics/promoter/1` | growth | yes | yes | 5 | 2 | [shot](captures/project-projectId-growth-analytics-promoter-promoterId/screenshot.png) · [dom](captures/project-projectId-growth-analytics-promoter-promoterId/dom.html) · [spec](SCREENS/project-projectId-growth-analytics-promoter-promoterId.md) |
| `/project/all/growth/campaigns` | growth | yes | yes | 6 | 3 | [shot](captures/project-projectId-growth-campaigns/screenshot.png) · [dom](captures/project-projectId-growth-campaigns/dom.html) · [spec](SCREENS/project-projectId-growth-campaigns.md) |
| `/project/all/payments/allPayments` | payments | yes | yes | 6 | 9 | [shot](captures/project-projectId-payments-allPayments/screenshot.png) · [dom](captures/project-projectId-payments-allPayments/dom.html) · [spec](SCREENS/project-projectId-payments-allPayments.md) |
| `/project/all/payments/createPaymentLink` | payments | yes | yes | 5 | 3 | [shot](captures/project-projectId-payments-createPaymentLink/screenshot.png) · [dom](captures/project-projectId-payments-createPaymentLink/dom.html) · [spec](SCREENS/project-projectId-payments-createPaymentLink.md) |
| `/project/all/payments/invoice` | payments | yes | yes | 4 | 2 | [shot](captures/project-projectId-payments-invoice/screenshot.png) · [dom](captures/project-projectId-payments-invoice/dom.html) · [spec](SCREENS/project-projectId-payments-invoice.md) |
| `/project/all/payments/missed-payments` | payments | yes | yes | 5 | 5 | [shot](captures/project-projectId-payments-missed-payments/screenshot.png) · [dom](captures/project-projectId-payments-missed-payments/dom.html) · [spec](SCREENS/project-projectId-payments-missed-payments.md) |
| `/referral/demo` | public | no | no | 1 | 1 | [shot](captures/referral-token/screenshot.png) · [dom](captures/referral-token/dom.html) · [spec](SCREENS/referral-token.md) |
| `/` | misc | yes | no | 6 | 7 | [shot](captures/root/screenshot.png) · [dom](captures/root/dom.html) · [spec](SCREENS/root.md) |
| `/setPassword` | auth | no | no | 6 | 1 | [shot](captures/setPassword/screenshot.png) · [dom](captures/setPassword/dom.html) · [spec](SCREENS/setPassword.md) |
| `/settings` | settings | yes | no | 4 | 1 | [shot](captures/settings/screenshot.png) · [dom](captures/settings/dom.html) · [spec](SCREENS/settings.md) |
| `/settings/account` | settings | yes | no | 5 | 2 | [shot](captures/settings-account/screenshot.png) · [dom](captures/settings-account/dom.html) · [spec](SCREENS/settings-account.md) |
| `/settings/account/all` | settings | yes | yes | 5 | 1 | [shot](captures/settings-account-projectId/screenshot.png) · [dom](captures/settings-account-projectId/dom.html) · [spec](SCREENS/settings-account-projectId.md) |
| `/settings/activityLog` | settings | yes | no | 7 | 8 | [shot](captures/settings-activityLog/screenshot.png) · [dom](captures/settings-activityLog/dom.html) · [spec](SCREENS/settings-activityLog.md) |
| `/settings/adminControl` | settings | yes | no | 4 | 1 | [shot](captures/settings-adminControl/screenshot.png) · [dom](captures/settings-adminControl/dom.html) · [spec](SCREENS/settings-adminControl.md) |
| `/settings/api` | settings | yes | no | 4 | 1 | [shot](captures/settings-api/screenshot.png) · [dom](captures/settings-api/dom.html) · [spec](SCREENS/settings-api.md) |
| `/settings/branding` | settings | yes | no | 4 | 2 | [shot](captures/settings-branding/screenshot.png) · [dom](captures/settings-branding/dom.html) · [spec](SCREENS/settings-branding.md) |
| `/settings/checkoutAppearance` | settings | yes | no | 4 | 1 | [shot](captures/settings-checkoutAppearance/screenshot.png) · [dom](captures/settings-checkoutAppearance/dom.html) · [spec](SCREENS/settings-checkoutAppearance.md) |
| `/settings/emailConfig` | settings | yes | no | 4 | 1 | [shot](captures/settings-emailConfig/screenshot.png) · [dom](captures/settings-emailConfig/dom.html) · [spec](SCREENS/settings-emailConfig.md) |
| `/settings/integrations` | settings | yes | no | 6 | 4 | [shot](captures/settings-integrations/screenshot.png) · [dom](captures/settings-integrations/dom.html) · [spec](SCREENS/settings-integrations.md) |
| `/settings/mobileApp` | settings | yes | no | 5 | 2 | [shot](captures/settings-mobileApp/screenshot.png) · [dom](captures/settings-mobileApp/dom.html) · [spec](SCREENS/settings-mobileApp.md) |
| `/settings/paymentChannels` | settings | yes | no | 6 | 1 | [shot](captures/settings-paymentChannels/screenshot.png) · [dom](captures/settings-paymentChannels/dom.html) · [spec](SCREENS/settings-paymentChannels.md) |
| `/settings/paymentMethods` | settings | yes | no | 4 | 1 | [shot](captures/settings-paymentMethods/screenshot.png) · [dom](captures/settings-paymentMethods/dom.html) · [spec](SCREENS/settings-paymentMethods.md) |
| `/settings/paymentsApp` | settings | yes | no | 6 | 3 | [shot](captures/settings-paymentsApp/screenshot.png) · [dom](captures/settings-paymentsApp/dom.html) · [spec](SCREENS/settings-paymentsApp.md) |
| `/settings/policyManagement` | settings | yes | no | 4 | 1 | [shot](captures/settings-policyManagement/screenshot.png) · [dom](captures/settings-policyManagement/dom.html) · [spec](SCREENS/settings-policyManagement.md) |
| `/settings/roleManagement` | settings | yes | no | 4 | 1 | [shot](captures/settings-roleManagement/screenshot.png) · [dom](captures/settings-roleManagement/dom.html) · [spec](SCREENS/settings-roleManagement.md) |
| `/settings/roleManagement/role/1` | settings | yes | yes | 4 | 2 | [shot](captures/settings-roleManagement-role-roleId/screenshot.png) · [dom](captures/settings-roleManagement-role-roleId/dom.html) · [spec](SCREENS/settings-roleManagement-role-roleId.md) |
| `/settings/testTokens` | settings | yes | no | 4 | 1 | [shot](captures/settings-testTokens/screenshot.png) · [dom](captures/settings-testTokens/dom.html) · [spec](SCREENS/settings-testTokens.md) |
| `/settings/updater` | settings | yes | no | 9 | 3 | [shot](captures/settings-updater/screenshot.png) · [dom](captures/settings-updater/dom.html) · [spec](SCREENS/settings-updater.md) |
| `/settings/userManagement` | settings | yes | no | 6 | 4 | [shot](captures/settings-userManagement/screenshot.png) · [dom](captures/settings-userManagement/dom.html) · [spec](SCREENS/settings-userManagement.md) |
| `/settings/walletManagement` | settings | yes | no | 4 | 1 | [shot](captures/settings-walletManagement/screenshot.png) · [dom](captures/settings-walletManagement/dom.html) · [spec](SCREENS/settings-walletManagement.md) |
| `/settings/webhook` | settings | yes | no | 4 | 1 | [shot](captures/settings-webhook/screenshot.png) · [dom](captures/settings-webhook/dom.html) · [spec](SCREENS/settings-webhook.md) |
| `/settings/withdrawal` | settings | yes | no | 5 | 1 | [shot](captures/settings-withdrawal/screenshot.png) · [dom](captures/settings-withdrawal/dom.html) · [spec](SCREENS/settings-withdrawal.md) |
| `/signup` | auth | no | no | 6 | 1 | [shot](captures/signup/screenshot.png) · [dom](captures/signup/dom.html) · [spec](SCREENS/signup.md) |
| `/success` | public | no | no | 0 | 0 | [shot](captures/success/screenshot.png) · [dom](captures/success/dom.html) · [spec](SCREENS/success.md) |
| `/sweepIn` | sweep | yes | no | 8 | 4 | [shot](captures/sweepIn/screenshot.png) · [dom](captures/sweepIn/dom.html) · [spec](SCREENS/sweepIn.md) |
| `/sweepIn/btc` | sweep | yes | no | 4 | 1 | [shot](captures/sweepIn-btc/screenshot.png) · [dom](captures/sweepIn-btc/dom.html) · [spec](SCREENS/sweepIn-btc.md) |
| `/sweepIn/eth` | sweep | yes | no | 4 | 1 | [shot](captures/sweepIn-eth/screenshot.png) · [dom](captures/sweepIn-eth/dom.html) · [spec](SCREENS/sweepIn-eth.md) |
| `/sweepIn/usdc` | sweep | yes | no | 4 | 1 | [shot](captures/sweepIn-usdc/screenshot.png) · [dom](captures/sweepIn-usdc/dom.html) · [spec](SCREENS/sweepIn-usdc.md) |
| `/sweepIn/usdt` | sweep | yes | no | 4 | 1 | [shot](captures/sweepIn-usdt/screenshot.png) · [dom](captures/sweepIn-usdt/dom.html) · [spec](SCREENS/sweepIn-usdt.md) |
| `/withdraw/address-book` | withdraw | yes | no | 6 | 2 | [shot](captures/withdraw-address-book/screenshot.png) · [dom](captures/withdraw-address-book/dom.html) · [spec](SCREENS/withdraw-address-book.md) |
| `/withdraw/address-book/1` | withdraw | yes | yes | 6 | 1 | [shot](captures/withdraw-address-book-id/screenshot.png) · [dom](captures/withdraw-address-book-id/dom.html) · [spec](SCREENS/withdraw-address-book-id.md) |
| `/withdraw/referral-payouts` | withdraw | yes | no | 5 | 2 | [shot](captures/withdraw-referral-payouts/screenshot.png) · [dom](captures/withdraw-referral-payouts/dom.html) · [spec](SCREENS/withdraw-referral-payouts.md) |
| `/withdraw/refunds` | withdraw | yes | no | 4 | 1 | [shot](captures/withdraw-refunds/screenshot.png) · [dom](captures/withdraw-refunds/dom.html) · [spec](SCREENS/withdraw-refunds.md) |
| `/withdraw/user-payouts` | withdraw | yes | no | 6 | 4 | [shot](captures/withdraw-user-payouts/screenshot.png) · [dom](captures/withdraw-user-payouts/dom.html) · [spec](SCREENS/withdraw-user-payouts.md) |

## Routes intentionally skipped

- `/_not-found`, `/_global-error` — Next.js internal error pages.
- `/favicon.ico` — asset, not a page.
- `/logout` — would destroy our crawler's auth session for every subsequent
  route. Captured by hand instead (it just clears localStorage and redirects
  to `/login`).
- `/(auth)/page` (`/`) — captured as the `root` slug; in practice it
  immediately redirects to `/login` or `/project/all/dashboard` depending
  on auth state.

## Routes that don't exist in the Payminto clone yet

Compared to `payminto/frontend/src/app/`:

- `/manageWallet/gasFeeWallet` (+ `/add`, `/edit`)
- `/manageWallet/sweepContract` (+ `/deploy`)
- `/manageWallet/wallets/cold`, `/manageWallet/wallets/details/[id]`
- `/developers/documentation`
- `/onramp-payments` and `/project/all/accounts/revenue` (Card Onramp)
- `/sweepIn/{btc,eth,usdc,usdt}` (per-asset sweep pages)
- `/settings/{adminControl,api,checkoutAppearance,mobileApp,paymentsApp,paymentMethods,policyManagement,testTokens,updater,walletManagement,webhook}`
- `/project/[id]/payments/{invoice,missed-payments}`

These are tracked in `GAP_ANALYSIS.md` (Phase 4).
