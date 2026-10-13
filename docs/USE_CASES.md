# Payminto use cases and payment walkthroughs

A merchant creates a payment request, shares its checkout, and checks the resulting payment against the deposit and ledger evidence.
The customer pays the network and token issued for that request.
An address, wallet submission, or attractive success screen alone does not prove that the merchant received money.

This guide describes the runtime source integrated at `1d63fc8` and the separately recorded local demonstrations from October 13, 2026.
It complements the [project disclosure](WORLDS_FAIR.md), [recording outline](../DEMO.md), [setup guide](../CONTRIBUTING.md), and [product URLs](../website.md).
The latest integrated public Tempo deployment remains unverified.
Region, community affiliation, presentation ownership, traction, and submission-media URLs require human confirmation.
The founder reports sharing the project with hackers in Hong Kong and Singapore for developer feedback; exact events, official acceptance, residence, and regional eligibility remain unconfirmed.

## Choose the appropriate flow

| Use case | Current path | Boundary |
| --- | --- | --- |
| Send one customer an invoice | Dashboard Create payment or authenticated v1 payment API, then `/pay/<reference>`. | The working Solana/Tempo showcase path. |
| Accept Solana stablecoins | Issue a Solana token account, observe finalized payment, post deposit journal, optionally sweep. | Local-validator test tokens and public USDC availability are distinct evidence. |
| Accept Tempo stablecoins | Issue the shared merchant recipient and invoice memo, then send the issued TIP-20 token. | Moderato testnet only; no Tempo refund, payout, or sweep. |
| Manage reusable payment links | Dashboard link builder and v2 link lifecycle/render/pay APIs. | The standalone checkout has no `/l/<short_code>` page; link-to-switch migration remains incomplete. |
| Price and reconcile a switch payment | v2 payment intent/attempt API with configured connectors and versioned fee snapshots. | Mock and chain-deposit connectors are wired; real card/UPI and cross-rail routing are unfinished. |
| Integrate a merchant application | Authenticated API creation and merchant lookup; public checkout receives the issued reference. | Keep merchant credentials on the merchant's server. |
| Let an AI agent help a merchant | Inherited MCP tool definitions for payment creation, lookup, and reporting. | The entrypoint has no attached MCP transport; no demonstrated autonomous wallet payment. |
| Embed a payment button | Inherited widget source and build. | It opens a legacy amount URL without creating an invoice; current checkout integration is unfinished. |
| Review balances and operations | Merchant payment/wallet/analytics APIs and operator journal reconciliation. | Payment status and wallet lists are not a complete public ledger or settlement API. |

## 1. Merchant creates a single payment

The dashboard page `/dashboard/payments/create` asks for a customer email and positive USD amount, then shows the generated checkout URL and reference.
The form calls `POST /api/v1/payment` using the merchant's authenticated session.
A server integration can call the same endpoint with its environment-bound API key.
The API also accepts an invoice id and optional `blockchainCode`/`currencyCode` for assignment during creation.

Example request body:

```json
{
  "amountInUSD": "1.50",
  "customerEmail": "payer@example.com"
}
```

Creation writes a `payment_requests` row with a generated UUID reference, merchant/platform ownership, state `OPEN`, and a default 30-minute expiry.
The response carries `reference_id`, `url`, and `host`; an immediately assigned request may also carry its deposit address.
The dashboard builds `/pay/<reference>` using its configured checkout host.
Use the issued reference rather than an invented preview reference.
If requested address assignment fails during creation, the service attempts to cancel the new invoice and returns an error.

The merchant reads `GET /api/v1/payment/reference/<reference>` or the scoped `GET /api/v1/payments` list.
That response calls the state field `paymentState`.
The public checkout response calls it `state`; clients must preserve this API distinction.

Sources: [dashboard form](../frontend/app/dashboard/payments/create/create-payment-form.tsx), [payment handler and DTO](../backend/internal/api/handler/payment_handler.go), [payment service](../backend/internal/service/payment_service.go).

## 2. Customer chooses the issued payment rail

The current standalone checkout lives at `/pay/<reference>`.
Its same-origin routes proxy only the backend's public checkout routes.

| Customer operation | Backend API |
| --- | --- |
| Read invoice, expiry, state, and assigned payment details | `GET /api/v1/public/payment/<reference>` |
| Read offered deposit methods | `GET /api/v1/public/blockchain-currencies` |
| Assign the selected rail | `POST /api/v1/public/deposit-address/reference/<reference>` |
| Listen for status events | `GET /api/v1/public/events/<reference>` |

The assignment body is `{"blockchainCode":"TEMPO","currencyCode":"PATHUSD"}` for pathUSD, or `{"blockchainCode":"SOLANA","currencyCode":"USDC"}` for Solana USDC.
Use the actual codes advertised by the configured backend.
An offered currency can still be unavailable for a merchant without the required wallet or configuration; assignment must succeed before sending money.
After assignment, repeating the same public request returns the issued address, while requesting another method returns HTTP 409 because the invoice's method is locked.
The checkout therefore removes its ineffective Change action once an address is issued.

The reference acts as the capability for this narrowed public view.
Do not expose a merchant API key, private key, or internal customer record in the checkout.
The public projection currently omits received totals and detailed transaction receipts.
The UI shows an amount as received only when that amount is supplied, and labels an invoice amount as Amount due otherwise.
It listens for payment events and also refreshes on a 20-second fallback interval.
A wallet transaction submission leaves the invoice awaiting confirmation until the backend reports the result.

Sources: [public API](../backend/internal/api/handler/public_api_handler.go), [checkout proxy](../checkout/lib/backend.ts), [live checkout container](../checkout/components/checkout.tsx), [view model](../checkout/lib/model.ts).

## 3. Customer pays USDC or USDT on Solana

The merchant needs the configured Solana deposit wallet, supported mint, watcher, and test environment described in the [Solana track](../tracks/solana/README.md).
USDC and USDT remain separate assets, `USDC.SOLANA` and `USDT.SOLANA`.
A local-validator mint named USDC is a local test asset, not Circle's public USDC mint.

1. Create the invoice and assign the Solana asset.
2. The backend derives a deposit owner and its associated token account for that asset's configured mint and token program.
   The public `depositAddress` is the token account; `depositOwnerAddress`, when returned, identifies its owner.
3. Confirm the network, mint, destination token account/owner, and exact token amount before signing a transfer.
   Send an SPL token transfer, not native SOL to the displayed token-account address.
4. The watcher reads signatures and parsed transaction evidence, checks the expected mint and receiving account, and waits for finalized evidence.
   A missing transaction response keeps the cursor retryable rather than treating the signature as processed.
5. It records the observed deposit amount and confirms it through the ledger path.
   Invoice state follows accumulated confirmed and already swept deposits, rather than an assumed amount from the payment screen.
6. Refresh the merchant payment detail and compare the deposit/signature with its journal.
   The checkout's network/token labels and final state must agree with the actual records.

The watcher handles transfers to the tracked token account and supported owner-account payment paths.
Wrong mints, native SOL, unsupported Token-2022 transfers, and unexplained account changes are anomaly paths, not successful stablecoin invoices.
The standalone checkout displays the returned Solana wallet owner in its copy field and QR, falling back to the token account only when an older API omits the owner.
Other chains retain their issued deposit address even if an unexpected owner field is supplied.
It shows the exact amount but does not implement a Solana browser-wallet send button.
A payer uses an appropriate external Solana wallet or a controlled local-validator transfer in the demonstration.

Sources: [token-account assignment](../backend/internal/service/solana_deposit_address.go), [deposit watcher](../backend/internal/service/solana_deposit_service.go), [local-validator test](../backend/internal/service/solana_integration_test.go).

## 4. Merchant consolidates a confirmed Solana deposit

Payment acceptance and sweeping are separate operations.
When custody and the hot wallet/fee payer are configured, the sweep worker claims confirmed deposits and transfers their token balances to the hot associated token account.
It persists the sweep's locks and signed attempt before broadcast, and marks deposits `swept` only after a finalized attempt is booked.
A transport failure does not prove the send failed; the recorded signature remains tracked across restart.
Competing workers cannot independently claim the same account.

Token movement is a transfer journal, not new merchant income.
Network fees are booked separately in `SOL.SOLANA`, and account-close rent recovery has its own accounting entries.
A sweep does not itself demonstrate a bank payout, currency conversion, or withdrawal to a customer.
For the precise recovery states and multi-RPC live requirements, see [Solana sweep rules](../backend/internal/service/SOLANA_SWEEPS.md).
The later local-validator recording below contains an actual transfer and landed sweep on that local chain.

## 5. Customer pays pathUSD or AlphaUSD on Tempo

Tempo is enabled only with a non-zero shared merchant recipient, seeded supported assets, test environment, testnet network mode, and an RPC reporting Moderato chain 42431.
An optional positive `TEMPO_MEMBER_ID` restricts both assignment and credit to that merchant.
See the [Tempo configuration and original proof](../tracks/tempo/README.md).

1. Create an ordinary UUID invoice and successfully assign its Tempo leg.
2. Read the issued token, merchant recipient, exact amount, and invoice memo from that payment.
   All Tempo invoices can share the recipient; the memo distinguishes them.
3. For a UUID, remove dashes to obtain its 32 hexadecimal characters, encode those characters as UTF-8 bytes, and send that bytes32 value.
   This is the ASCII encoding of the compact reference, not the UUID's 16 binary bytes.
   Short references use their UTF-8 text and zero padding; unsupported references longer than 32 bytes are refused.
4. Call `transferWithMemo(recipient, integer_amount, memo)` on the issued token contract.
   On Moderato, pathUSD uses `0x20c0000000000000000000000000000000000000`, and AlphaUSD uses the corresponding address ending in `0001`.
   Both seeded assets use six decimals; 1.5 tokens equals 1,500,000 base units.
5. The browser-wallet path requests the account and correct chain, adds the chain only after an unknown-chain error, and stops after a declined request.
   It preserves the issued token contract and exact decimal amount; an address-only transfer without the memo cannot match the invoice.
6. The watcher reads logs and receipts without decoding full Tempo blocks, verifies receipt success, token, recipient, positive amount, and matching issued rail/token/merchant ownership, then records the deposit.
7. Deposit confirmation, journal posting, and invoice finalization commit together.
   The shared event broker publishes the committed state; an invoice-write failure leaves the deposit pending for retry and does not publish a successful confirmation.
8. Compare the transaction's TransferWithMemo recipient, token, amount, memo, and log index with the local deposit and journal, then read the merchant payment result.

Moderato is a public test network; a local backend consuming a Moderato payment is separate from a public Payminto deployment.
The current rail compares stablecoin units to USD invoices at 1:1 without a conversion trade.
Tempo refunds, withdrawals, payouts, and sweeps are not implemented.

Sources: [memo/token helpers](../backend/internal/blockchain/tempo/tempo.go), [issued-leg checks and finalization](../backend/internal/service/tempo_memo_service.go), [wallet send](../checkout/lib/tempo-wallet.ts), [checkout token display](../checkout/components/tempo-pay.tsx).

## 6. Handle partial payments, extras, replay, and interruptions

| Situation | Implemented response | What the merchant or payer should check |
| --- | --- | --- |
| Less arrives than the invoice requires | The deposited amount remains intact and the invoice becomes `PARTIALLY_FILLED`. | Obtain an authoritative received/remaining amount before a top-up. |
| The received total reaches the invoice | Invoice becomes `FILLED`. | Confirm the actual receipt and the deposited asset. |
| More arrives, including another distinct Tempo log after a full fill | Invoice becomes `OVER_FILLED`; actual money is retained in the deposit/journal records. | Extras are not an automatic refund. |
| A previously credited Tempo log is replayed | `(tx_hash, log_index)` deduplication and journal idempotency prevent another credit. | Separate log indices in one transaction remain separate transfers. |
| Tempo journal/invoice finalization fails | The confirmation transaction rolls back and the pending deposit is retried. | A pending audit row is not proof of a committed journal. |
| RPC or cursor storage fails | Poll reports the failure and preserves retryable work. | Read the worker result; do not call a failed or skipped check successful. |
| Solana sweep broadcast response is uncertain or the process crashes | Persisted signature, locks, and attempt state allow reconciliation. | Do not broadcast an unrelated replacement on the assumption that nothing landed. |
| Link reservation outcome is uncertain | A pending use is fenced/resolved before it can be released safely. | Retry the same idempotency key and respect `Retry-After`. |
| Invoice expires or is cancelled | Public assignment refuses an invoice that is no longer open; UI displays its closed state. | Request a new invoice rather than sending against an expired quote. |

The public invoice projection currently lacks received totals.
For an underpaid invoice, checkout suppresses the amount and wallet send action until a correct remainder is available; it never substitutes the original full amount as the top-up.
Regression coverage includes [Tempo PostgreSQL replay/rollback/partial/extra tests](../backend/internal/service/tempo_memo_integration_test.go), [Tempo regressions](../backend/internal/service/tempo_memo_regression_test.go), and [checkout API-contract tests](../checkout/components/tempo-pay.test.tsx).
These are tests of the behaviors, not additional real-network payments.

## 7. Merchant manages reusable payment links

The dashboard's `/dashboard/links/new` builder creates a different object from the single-invoice Create payment form.
It supports fixed amounts, customer-entered bounded amounts, or line items; customer fields/questions; payment options; fee bearer; availability limits; and checkout presentation settings.
Saving a draft does not publish it or take money.

The current authenticated APIs are under `/api/v2/links`: create/list/get/update, preview, options, publish, pause, archive, and duplicate.
The server preview is `POST /api/v2/links/preview` and contains the rendered form rather than invented totals.
Publication validates enabled methods, active fee rules, amounts, customer fields, and any settlement override references.
Published monetary fields are immutable; duplicate a link to change them.

Public clients can read `GET /api/v2/public/links/<short_code>` and its `/qr.svg`, then call `POST /api/v2/public/links/<short_code>/pay` with `Idempotency-Key` and the enabled method/customer answers.
The pay result contains `payment_reference`, `customer_total`, optional fee/tax preview, and the created invoice's `checkout_url`.
Same-key/same-body replay returns the stored result; same-key/different-body requests are refused.
Paused, archived, expired, exhausted, and excessive-open-payment links are refused before another reservation succeeds.

The integrated registry still uses the legacy `LinkPaymentCreator` for crypto/USD links, rather than the switch-backed creator.
Its generated reference is `pl_<UUID>`, which does not fit Tempo's 32-byte memo rule, so a reusable Tempo-link payment is not a supported end-to-end claim on this snapshot.
The link URL uses `/l/<short_code>`, but the standalone checkout has no corresponding page in this source.
A client can consume the implemented public link API and then open the returned single-invoice checkout URL for a supported rail such as configured Solana USDC.
Do not present the generated `/l` URL as a complete shipped hosted flow until that integration is finished.

Sources: [link routes](../backend/internal/api/routes_links.go), [link lifecycle/validation](../backend/internal/links/README.md), [integrated creator](../backend/internal/service/link_payment_creator.go), [registry wiring](../backend/internal/service/registry.go).

## 8. Explain fees and the payment switch accurately

An operator with the configured platform and session permissions can manage immutable fee-rule versions through `/api/v1/admin/fee-rules`.
A merchant can request `POST /api/v1/fees/preview` for a supported method, currency, amount, and bearer.
The response identifies its rule/version and fee, tax, customer total, and merchant net when a quote can be calculated.
A preview is not a charged fee or a ledger posting.

The v2 switch exposes payment creation/read/confirm/capture/cancel/refund APIs under `/api/v2/payments`.
It owns intents, attempts, idempotency, connector status mapping, unknown-outcome reconciliation, and fee snapshots per attempt.
The fee posting joins the captured-money transaction and has its own idempotency key.
The current wiring offers configured mock and chain-deposit connectors with a first-enabled selector; a mock card success is not a real card charge.

The dashboard's ordinary v1 invoice path does not create a switch attempt or automatically post the switch's fee journal.
The legacy payment-link creator records the selected rule id/version on its invoice, and the link use holds preview information, but that is not the switch's charged-fee snapshot and posting flow.
Do not deduct a hypothetical platform fee from a Solana/Tempo deposit demonstration or claim that a bare invoice exercised routing, conversion, or card processing.

Sources: [fee module](../backend/internal/fees/README.md), [switch routes](../backend/internal/api/routes_paymentswitch.go), [connector wiring](../backend/internal/modules/paymentswitch.go), [switch fee posting](../backend/internal/paymentswitch/apply.go).

## 9. Merchant reads the result; operator reconciles it

The merchant returns to `/dashboard/payments/<reference>` and the payments list, or reads the scoped v1 invoice API.
Match the reference, invoice amount, state, network, and customer record to the intended request.
`GET /api/v1/wallets` reports owned wallets/address capacity; hot-wallet balance endpoints use their configured chain adapter.
Analytics summaries and a wallet balance are not replacements for the payment's actual receipt or its journal.

An operator can reconcile direct invoice deposits using read-only queries on the environment's database, with a reference obtained from the merchant and appropriate access controls.
The current HTTP router does not expose a general merchant journal-export endpoint.
The following examples are for the direct v1 invoice path, not switch-owned attempts whose money is booked by the switch.
Replace the reference placeholder; never reset or seed the database to inspect it.

```sql
SELECT p.reference_id, p.state, d.id AS deposit_id, d.status,
       d.amount, bc.currency_code, bc.blockchain_code, d.tx_id, d.to_address
FROM payment_requests p
JOIN deposits d ON d.payment_request_id = p.id
JOIN blockchain_currencies bc ON bc.id = d.blockchain_currency_id
WHERE p.reference_id = '<issued-reference>'
  AND p.deleted_at IS NULL AND d.deleted_at IS NULL;
```

```sql
SELECT j.id AS journal_id, j.kind, a.owner_id, a.kind AS account_kind,
       a.asset, l.amount
FROM payment_requests p
JOIN deposits d ON d.payment_request_id = p.id
JOIN ledger_journals j ON j.reference_type = 'deposit'
                     AND j.reference_id = d.id::text
JOIN ledger_lines l ON l.journal_id = j.id
JOIN ledger_accounts a ON a.id = l.account_id
WHERE p.reference_id = '<issued-reference>'
ORDER BY j.id, l.id;
```

Compare each deposit amount with the chain receipt and recipient before checking its journal.
Direct invoice journals use the legacy platform accounts `crypto_assets` and `merchant_balance`, qualified by asset; they are not a complete merchant settlement statement.
A receipt for 1.5 pathUSD is booked as `+1.5` asset and `-1.5` liability under the ledger's signed-entry convention.
Check that each journal sums to zero per asset, and keep `PATHUSD.TEMPO`, `USDC.SOLANA`, `USDT.SOLANA`, and `SOL.SOLANA` separate.
After a Solana sweep, the confirmed deposit still counts toward invoice payment even though its deposit status is now `swept`.
The [ledger reconciliation implementation](../backend/internal/ledger/reconcile.go) and [legacy-balance comparison](../backend/internal/service/ledger_service.go) provide internal reconciliation capabilities; they do not imply a newly shipped dashboard export flow.

## 10. Other implemented surfaces and inherited limits

| Surface | Available behavior | End-to-end limit |
| --- | --- | --- |
| Merchant API integration | Create an invoice, fetch it under platform ownership, share public checkout, and poll merchant status. | The v1 create endpoint is not the v2 switch's idempotent intent API. |
| API-key management | Merchant `/api/v1/api-keys` list/create/revoke routes and environment-bound authentication. | A key belongs in a server integration, not an embedded browser widget. |
| AI/MCP | Tool factory defines `create-payee`, `lookup-payment`, `search-payments`, `get-balance`, and `get-daily-volume` over authenticated backend calls. | `mcp-server/src/index.ts` starts a health server but does not connect the factory to an MCP transport; `get-balance` returns wallet listing, not a reconciled balance. |
| Widget | A script injects Pay with Payminto, asks for an amount, and opens a configured legacy URL. | It does not create an issued reference or integrate the current `/pay/<reference>` flow. |
| Wallets/recipients/withdrawals | Inherited merchant wallet/recipient and withdrawal creation, OTP, approval, lookup, and cancellation APIs exist. | Existence of these routes is not evidence of Tempo payouts or a tested end-to-end withdrawal on every rail. |
| Webhooks | Inherited endpoint management, secret rotation, delivery logs, and processing infrastructure exist; switch connector webhooks are verified by their connector. | Do not infer that every direct Tempo deposit exercised a merchant webhook from a Paid screen or SSE event. |
| Optional attestations | Retained CRE modules, workflows, and contract code can be enabled separately. | Outside the current Solana/Tempo demonstration; no new oracle-service launch is claimed. |
| Bitcoin/EVM/Tron | Inherited adapters, deposit and wallet infrastructure remain in the repository. | Prior work, with no new chain payment proof supplied by this walkthrough. |
| Cards/UPI/custody/routing/conversion | Interfaces, specifications, and some development work exist. | Real connectors, extra custody providers, cross-rail routing, and conversion/deployment completion remain unfinished or unmerged. |

Sources: [actual router](../backend/internal/api/router.go), [MCP factory](../mcp-server/src/server.ts), [MCP entrypoint](../mcp-server/src/index.ts), [widget](../widget/src/index.ts).

### Additional chain selections and source status

Submission chain selections declare integrated product scope; a selection alone does not establish a working deployment or payment receipt.
The source audit for the additional Ethereum L1, Arbitrum, and Robinhood Chain selections found the following:

| Chain | Source/configuration found | Verification boundary |
| --- | --- | --- |
| Ethereum L1 | Inherited Ethereum adapter and `ETH` mainnet/Sepolia blockchain, token, and RPC seed rows. | No fresh Ethereum payment or deployment verification in the current recording bundle. |
| Arbitrum | Recognized EVM chain family and chain id 42161 in the inherited adapter, plus checkout display names. | No Arbitrum blockchain/token/RPC seed rows in the audited current seed sets, and no configured or demonstrated payment evidence. |
| Robinhood Chain | No chain-specific implementation, seed/configuration, or payment evidence found in the audited source. | A portal selection is not an implemented integration claim. |

Generic EVM support needs correct chain, RPC, token, custody, and environment configuration before any network-specific payment can be demonstrated.
Sources: [chain-family classification](../backend/internal/blockchain/adapter.go), [Ethereum client](../backend/internal/blockchain/ethereum/client.go), [testnet seeds](../backend/migrations/seeds/testnet/), [mainnet seeds](../backend/migrations/seeds/mainnet/), [checkout network names](../checkout/lib/model.ts).

## 11. Test and live environments

Environment isolation is a process/database boundary, not just an amber label in the UI.
Boot checks the database environment stamp, provider choices, network, keystore, credentials, and deployment settings.
API keys use `sk_test_`/`sk_live_`, and a credential or JWT from the other environment is refused.
Moderato Tempo configuration cannot start in a live money process.
Solana live-mode RPC and signer restrictions are separate from local-validator test configuration.

Use [CONTRIBUTING.md](../CONTRIBUTING.md) and [operations guidance](OPERATIONS.md) for configuration and scoped checks.
Read the setup/cleanup harness before any database-touching test, seed, or script; use owned disposable targets and leave shared data and Docker resources alone.
No preview, skipped test, or successful mock connector proves mainnet settlement or a production deployment.

## 12. Recorded demonstrations and evidence levels

The following are separate runs, with different environments and payment references.
The recording bundle's `local-demo/EVIDENCE.md`, step screenshots, owner logs, and subsequent independent read-only review supplied the later values; the original Tempo run is also recorded in the versioned [Tempo track](../tracks/tempo/README.md#recorded-real-network-evidence).
The video files are local recording assets, not public hosted submission links.

| Run | Payment and chain evidence | Accounting evidence and boundary |
| --- | --- | --- |
| Original Tempo proof | Invoice `84933e7d-d0fc-46dd-b616-b602be747f9e`; 1.5 pathUSD; Moderato [transaction](https://explore.testnet.tempo.xyz/tx/0x0bd786d78b7205537319369b9dc30a8d09a79854eae0ac809223aece94a4a00d), block 39362768, log index 26, receipt status 1; recipient `0x82c70b91731914e7b678a232b366336Af1436455`. | Recorded local `FILLED`, deposit 3 confirmed, memo credit `credited`, journal 3 `+1.5 crypto_assets / -1.5 merchant_balance`; not a public Payminto rollout. |
| Later Solana local-validator recording | Invoice `b7eda30a-ac7d-4e41-95aa-5c4f7b01506b`, independently read as `FILLED`; 25 units of a local six-decimal USDC test mint `8m1h97DwD6evmidTNTDQHWhSN9NiujC5j3xr6VsEiqSp`; recorded transfer signature `Lr12Vaai5NoD3P97qfi8ekAD7NfEhKvLH7atep5nStkQSXZ9fhpuv94uXNuzU7mmNcNhare8uw56zYRZLvdVpqK`. | Manifest journal 1 `+25/-25 USDC.SOLANA`; recorded landed sweep signature `3adsrvFhx1ZWE3jncGHQHrajheNRRJKWp9S8WSfd88kdNBbwvZodGbcDaNcTmABycgicNrTzJEbBeD8MAaeKzUZD`; later independent hot-wallet balance 25 and closed deposit token account; historical receipts unavailable at review time. |
| Later Tempo checkout recording | Invoice `3da2ecad-f409-4d3b-9356-966030f52073`, independently read as `FILLED`; 12.5 pathUSD on Moderato [transaction](https://explore.testnet.tempo.xyz/tx/0x406a5a9f529666fc04b8c609f187d09b3c419d9a3caa04e114100ce701d1cf8f); independent RPC receipt status 1, log index 5, 12,500,000 base units, expected recipient and memo. | Manifest journal 4 `+12.5/-12.5 PATHUSD.TEMPO`; local backend/dashboard/checkout recording, not publicly deployed Tempo. |
| Controlled checkout verification | PathUSD/AlphaUSD, memo, exact clipboard amount, wallet rejection, underpayment suppression, and status events at 390px/1440px in light/dark. | Fake wallet and API/SSE fixtures verify interaction, not chain transfer; separate from the three runs above. |

The later recording filenames are `solana-localnet-flow.mp4` and `tempo-moderato-flow.mp4`.
Independent media inspection found 78 seconds and 56.033 seconds respectively, both H.264 at 1280x720 and 30 fps.
The later captures used actual browser recording and local gateway responses; they are separate from the fake-wallet/API fixtures used for checkout regression verification.
The real Tempo recordings submitted their transfers through an external script while the browser awaited backend confirmation.
They demonstrate actual chain payment and live checkout status, while wallet-button signing was verified separately with controlled test wallets.
The Solana deposit owner was `5V9fNFkbp388PpANzYLMx3NPbNqEJPcqYMnKF1tw8gkK`, and the issued token account was `6YCE2sL1gZZfXQxeJ1eSwqneEwVDztyNsmmtB2oR2Asx`.
The Solana hot wallet recorded after sweeping was `CFVGdawg1ymbDPHAPHL2beVVUHWCWoscm6eDwzzZ544A`.
Historical transaction/status queries returned no record for the local transfer/sweep signatures at independent review time, so the reviewer could verify current mint/account state but could not reconstruct the old local receipts.
The later Tempo transfer carried compact memo `3da2ecadf4094d3b9356966030f52073` to recipient `0x82c70b91731914e7b678a232b366336Af1436455`.
Its recording manifest includes a 0.000010 SOL sweep fee and separate 0.00203928 SOL rent-related entries.
Those entries must not be represented as a platform fee on the 25-token invoice.
Public deployment, public video hosting, portal submission, and any human presentation/region/community claims require their own confirmation.

## 13. Current demo deliverables

A narrated 2-minute-36-second product demo has been rendered from the later actual Solana and Tempo recordings.
It excludes login entry and includes visible local-validator/testnet labels, with captions and a timed transcript.
The film shows invoice creation, issued checkout, backend-confirmed payment state, merchant records, and the Tempo explorer receipt.
The ledger and sweep evidence remains in the recording manifest and this guide; the film does not invent a merchant journal screen.
A six-slide editable demo-day presentation with speaker notes has also been prepared.
These are local deliverables; public hosting, portal media acceptance, and final project submission remain separate steps.
