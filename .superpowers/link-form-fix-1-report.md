# Ticket 04 fix round 1 report

Branch `link-form`.
Review: `.superpowers/link-form-review.md` (3 high, 11 medium, 10 low).

## Commits

- `5be6cb4` merge `links` (its fix round: fee_bearer locked on live links, `fee_preview` per method on link responses, server QR, environment module); clean, no conflicts
- `55c217d` backend: `POST /api/v2/links/preview`, `GET /api/v2/links/options`, `merchant_name` on link responses, handler tests, API spec and README
- `de34ae4` frontend: preview renders only the server's model, fees matched by method key, live fee bearer locked, review fixes, tests
- `3165862` browser-pass fixes and new screenshots

## Checks

- `cd frontend && npx tsc --noEmit && npm run lint && npm test`: clean, no lint warnings, 47 tests pass (6 files).
- `cd backend && go build ./... && go test -count=1 ./internal/links/... ./internal/api/...`: pass.

## Backend (allowed this round, links module and its routes only)

- `Service.render` is the one builder: `Render` (public) and the new `Preview` both call it. It returns the render model plus the methods it dropped, each with its refusal code and message.
- `POST /api/v2/links/preview?link_id=` (merchant auth): decodes the form over `DefaultInput`, runs the save-shape validation (422 with field errors as on save), builds an unsaved `Link` in the process environment (or over the stored link's status, uses and short code when `link_id` is given) and returns `{model, dropped_methods}`. Nothing is stored.
- `GET /api/v2/links/options`: for each `Offering` from an optional `Catalog` on the payment creator, keeps those with a connector and an active fee rule (through the publish path's own `price`), returns `{environment, currencies, methods: [{method, chain, asset, currencies}]}`. `LinkPaymentCreator` implements `Catalog` from active, deposit-enabled `blockchain_currencies` (crypto in USD, matching its `Connectors`). A creator without a catalog yields empty lists.
- `merchant_name` on every merchant link response (list and detail); a failed lookup leaves it null.
- Handler tests (`internal/api/routes_links_preview_test.go`): customer-entered amount carries no totals; `surcharge_needs_quote`, `fee_exceeds_amount` and `surcharge_forbidden` methods are absent from the model and listed with reasons; line items carry server subtotal/tax/total; shape errors come back as 422 with every field; preview over a published link equals `GET /public/links/:code` byte for byte; options keep only publishable offerings and are empty without a catalog.
- H3: `immutableDiff` already refuses `fee_bearer` after the merge (409 `link_published_immutable`).

## Frontend, by finding

- H1, H2, M1, M2, M3: the browser no longer builds a render model or calls `/fees/preview`. The preview posts the settled form (400 ms debounce of the whole body) and renders the response whole. While a new response is on its way the last complete one stays, dimmed and `aria-busy`, so numbers never mix. With a local error or a 422 the preview shows a message instead of numbers. Step 3 reads the saved link's `fee_preview` keyed by method, only while the form's pricing fields equal what was saved ("Priced after save" otherwise), and refusals from the latest preview by method key. The test that enshrined H2 is replaced.
- H3: `fee_bearer` is in `IMMUTABLE_WHEN_PUBLISHED`; on a live link it renders as read-only text; the lock note says "price, currency, methods and fees". Component test added.
- M4: no currency means "Add a currency" and no amount anywhere in the preview.
- M5: one always-mounted `role="status"` line under the title at every width (Saved, Saving, Not saved, Unsaved changes, Draft saves as you type).
- M6: fields link messages with `aria-describedby`; field messages are not alerts; one alert summary on publish failure; focus moves to the first invalid field after the failing steps open.
- M7: visible labels on every repeating row (line items get a header row at sm and up and per-input labels below, questions get labelled fields, prefills get labels with example placeholders, metadata gets a header row).
- M8: segmented controls 32 px on fine pointers and 44 px on coarse; method rows, check labels, toggles and the colour swatch reach 44 px on coarse; copy and QR are equal 32 px ghost icons with the `tap` hit area.
- M9: `ShortLink` truncates the host and never the short code (list, detail).
- M10: live price, currency, amount mode, line items, methods and fee bearer render as text in ink, not disabled controls.
- M11: blank quantity, price, tax rate, tolerance and quote expiry are local errors; `formToInput` no longer fills in values.
- L1: summaries and ranges go through `formatDecimal`. L2: line-item links show per-line server totals, subtotal, tax and one total; no hero. L3: tabs have `tabpanel`, `aria-controls`, roving tabindex and arrow keys; the device toggle is a native radio group; the preview is a labelled `section`; step 3 lists methods in the server's options order and keeps the form in that order. L4: strings moved to `copy.ts`. L5: `effectiveUseLimit` and `methodKey` live once in `model.ts`; the duplicate `formFromLink` per render is gone. L6: one lock note above the steps; header actions keep their labels at 390. L7: payment limit and "ends after" sit together under Multiple payments with one line on how they combine. L8: no accent shows an empty dashed swatch. L9: screenshots retaken without the dev indicator, errors captured at both widths and schemes. L10: noted; `CheckoutPreview` should become the shared renderer when the hosted link checkout ships.
- Options: currency is a select of enabled currencies (current value kept and marked "not enabled here"); methods are the enabled ones for that currency; a chosen method that is not enabled stays visible so it can be removed.

## Screenshots

`docs/design/screens/links/` (old ones replaced): list, builder with all six steps expanded, mobile preview toggle, live detail, failed publish; each at 390 and 1440, light and dark (the preview toggle exists only below 1024 px). Taken from `/design/preview/links*`, whose in-memory sample stand-in now answers preview and options too.

## Not done or noted

- Still not run against a live backend stack (no disposable Postgres and Redis in this worktree); the new endpoints are covered by handler tests.
- `LinkPaymentCreator.Offerings` is SQL and has no unit test; it mirrors the `Connectors` query and is exercised by the integration environment only.
