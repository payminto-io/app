# Triple-A Visual Language & Product UI Research (2026)

**Research date:** 2026-09-09  
**Scope:** First-party Triple-A marketing, merchant login, dashboard, payments, statements, invoicing, and hosted-checkout references. This is a visual reference for Payminto, not permission to reproduce Triple-A's logo, illustrations, or screens pixel-for-pixel.

## Executive direction

Use Triple-A as a reference for a **white, restrained financial-product shell**: Inter for operational UI, black primary type, quiet gray borders, compact controls, outline icons, and a red active/action color. Reserve the red → orange → pink gradient for a few high-salience brand moments such as a page-title strip or large marketing headline. Do not make the whole sidebar colorful.

For Payminto, the useful synthesis is:

- a flat white sidebar with section labels, dividers, outline icons, and a clearly reinforced active item;
- white or ghost-white content surfaces with hairline borders and limited shadow;
- compact 8/12/16/24/32 px spacing for dashboard density;
- a dense labeled filter row above transaction tables;
- a simple two-column desktop login that collapses to one card on small screens;
- a mobile-first hosted checkout that reveals one decision per step.

Everything above is a **design inference** from the first-party evidence catalogued below, not a claim that Triple-A publishes these as brand guidelines.

## Source hierarchy and limits

The authenticated dashboard is not publicly inspectable without an account. This note therefore prioritizes the live login and compiled dashboard styles, plus first-party Help Center screenshots. The newest dashboard screenshots found are dated March 2026; older July 2024 invoicing screenshots are useful for workflow only and should not override the current visual treatment.

Primary sources:

- [Triple-A home page](https://www.triple-a.io/)
- [Digital currency payments product page](https://www.triple-a.io/digital-currency-payments)
- [E-commerce product page](https://www.triple-a.io/ecommerce-stores)
- [Current merchant dashboard login](https://dashboard.triple-a.io/login?redirectTo=%2F)
- [Current dashboard compiled stylesheet](https://dashboard.triple-a.io/_app/immutable/assets/app.B-j-cPTr.css)
- [Current marketing compiled stylesheet](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/css/triple-a-staging.webflow.shared.f96a7032b.min.css)
- [Dashboard functions](https://support.triple-a.io/knowledge/what-are-the-functions-of-tripleas-dashboard)
- [Finding a transaction](https://support.triple-a.io/knowledge/how-can-i-find-a-transaction), including the [full sidebar image](https://support.triple-a.io/hs-fs/hubfs/Payment-1.png) and [March 2026 Payment History filters](https://support.triple-a.io/hs-fs/hubfs/Screenshot%202026-03-11%20at%2011.20.18%20AM.png)
- [Requesting an account statement](https://support.triple-a.io/knowledge/how-do-i-request-an-account-statement), including the [March 2026 statement form](https://support.triple-a.io/hs-fs/hubfs/Screenshot%202026-03-11%20at%2011.33.57%20AM.png)
- [Using the invoicing tool](https://support.triple-a.io/knowledge/how-do-i-receive-payment-from-invoices)
- Current hosted-checkout assets: [order/payment method](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/6a2fe9c095a87620aa2e4873_1%20%283%29.png), [currency/provider choice](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/6a4e174f46ec06ffff3e1d1e_2%20%281%29.png), [wallet/QR payment](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/6a2fea0eb7664e62067a9f8f_3%20%282%29.png), and [success](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/6a2fea364d8f45a0cb1d19e6_4%20%281%29.png)

## Direct observations

### Typography

- The marketing stylesheet declares **Neue Montreal Medium 500** for headings and **Inter 400/500** for supporting text. Its primary display token is 4 rem with 1.1 line-height and weight 500; the next level is 3 rem/1.15/500. ([marketing stylesheet](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/css/triple-a-staging.webflow.shared.f96a7032b.min.css))
- The live dashboard login loads the Inter variable family and the dashboard stylesheet applies `Inter, sans-serif` to the body. The product UI uses high-weight headings, medium labels, and mostly 14–16 px operational copy rather than marketing-scale display type. ([dashboard login](https://dashboard.triple-a.io/login?redirectTo=%2F), [dashboard stylesheet](https://dashboard.triple-a.io/_app/immutable/assets/app.B-j-cPTr.css))

### Palette and surfaces

The current dashboard stylesheet exposes the following tokens directly: ([dashboard stylesheet](https://dashboard.triple-a.io/_app/immutable/assets/app.B-j-cPTr.css))

| Role | Observed token |
|---|---:|
| Canvas / ghost white | `#F7F9FB` |
| Surface | `#FFFFFF` |
| Alternate pale surface | `#ECECFF` |
| Borders | `#E0E0E0`, `#E8E7E6`, `#DFDFDF` |
| Primary text | `#000000` / `#010F0B` |
| Strong secondary text | `#5F5E5E` |
| Primary action / active | `#E22323` |
| Primary hover / active / disabled | `#C91E1E` / `#B01A1A` / `#F5A5A5` |
| Orange accent / pale orange | `#E95E33` / `#FCEEEA` |
| Marketing gradient companions | `#F06B28`, `#E96991` |
| Link green | `#048649` |
| Success | `#06A561` / `#D1F2E3` / `#01311D` |
| Information | `#384CCB` / `#EBEDF9` / `#10163C` |
| Warning | `#FFC107` / `#FFF3CD` / `#664D03` |
| Error | `#DB3232` / `#FFE4E4` / `#44040D` |

Observed surface treatment is overwhelmingly white with black type, pale-gray hairline borders, 8–16 px rounding, and low-opacity shadow. The current login card uses a 1 px light border, 16 px radius, and a soft `0 10px 15px -3px` / `0 4px 6px -4px` shadow; its large input and button controls are 48 px high in the stylesheet. ([dashboard login](https://dashboard.triple-a.io/login?redirectTo=%2F), [dashboard stylesheet](https://dashboard.triple-a.io/_app/immutable/assets/app.B-j-cPTr.css))

The marketing system uses a white rounded navigation shell, small outlined eyebrow pills, black pill buttons, and a red → orange → pink gradient for branded headings and high-salience CTAs. ([digital currency payments page](https://www.triple-a.io/digital-currency-payments), [marketing stylesheet](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/css/triple-a-staging.webflow.shared.f96a7032b.min.css))

### Sidebar and navigation

The newest first-party sidebar image is white and flat. It uses a black logo, black outline icons, uppercase section labels, thin horizontal dividers, and an outlined circular collapse control. The visible hierarchy is Home; **Products** (Request Payment, Stablecoin Payout, Local Payout); **History** (Payments, Refunds, Stablecoin Payouts, Local Payouts, Deposits, Withdrawals, Statements); and **Resources** (FAQ, Contact Us, Integration Docs, API Credentials). The active Payments row changes its icon and label to coral-red without a large filled background. ([transaction article](https://support.triple-a.io/knowledge/how-can-i-find-a-transaction), [full sidebar image](https://support.triple-a.io/hs-fs/hubfs/Payment-1.png))

### Tables, filters, forms, and density

- Payment History uses a full-width rounded red → orange → pink title band with large white title text. Beneath it is one compact row of persistent labels and outlined fields for payment reference, order ID, payer email, and a date range. Actions are a black Search button, an outlined More Filters button, and an icon-only reset/export control. ([Payment History image](https://support.triple-a.io/hs-fs/hubfs/Screenshot%202026-03-11%20at%2011.20.18%20AM.png))
- The current stylesheet uses a 4 px base spacing token and tables default to 14 px text, ellipsized cells, rounded header corners, and row-hover feedback. ([dashboard stylesheet](https://dashboard.triple-a.io/_app/immutable/assets/app.B-j-cPTr.css))
- The Statements screen repeats the gradient title band, uses simple tabs with a black underline for the active tab, and keeps one labeled date input, one black action button, and a gray toggle on a sparse white canvas. ([statement article](https://support.triple-a.io/knowledge/how-do-i-request-an-account-statement), [statement form image](https://support.triple-a.io/hs-fs/hubfs/Screenshot%202026-03-11%20at%2011.33.57%20AM.png))
- Triple-A says the dashboard supports transaction search/details, CSV export, invoices, payment-form branding, payment-option configuration, payouts, and users with different access functions. Those functions justify a task-oriented navigation system rather than a decoration-led dashboard. ([dashboard functions](https://support.triple-a.io/knowledge/what-are-the-functions-of-tripleas-dashboard))

### Login

The live desktop login is a two-column white layout. A top-left logo anchors the page; the left column has a large welcome heading and three benefit statements with small red circular check icons; the right column is a narrow bordered card with a strong Login heading, muted subtitle, visible field labels, leading input icons, password visibility action, recovery link, full-width primary button, and signup/support links. The left marketing column is hidden at smaller breakpoints. ([dashboard login](https://dashboard.triple-a.io/login?redirectTo=%2F))

The page source also exposes accessibility-oriented mechanics: logo alternative text, explicit field labels, a named “Show password” button, focus-visible rings on buttons, and assertive live regions for form errors. ([dashboard login](https://dashboard.triple-a.io/login?redirectTo=%2F))

### Hosted checkout and delivery/payment flow

Triple-A's current product page shows a narrow mobile-first white checkout in four progressive states: order/payment method, currency/provider choice, wallet/QR payment, and completion. The merchant logo is centered and cards use black/gray type, hairline borders, generous vertical whitespace, and black full-width pill actions. ([digital currency payments page](https://www.triple-a.io/digital-currency-payments), [e-commerce page](https://www.triple-a.io/ecommerce-stores))

The currency step uses tall outlined rows with leading asset/provider icons and trailing chevrons. The wallet step promotes amount due, gives the network its own mint-colored header, puts the address in a dashed panel with copy affordances, centers the QR, and retains timing/rate information. Completion removes most chrome and centers a large green confirmation mark with short supporting text. ([currency/provider image](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/6a4e174f46ec06ffff3e1d1e_2%20%281%29.png), [wallet/QR image](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/6a2fea0eb7664e62067a9f8f_3%20%282%29.png), [success image](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/6a2fea364d8f45a0cb1d19e6_4%20%281%29.png))

## Payminto design inferences

These are recommended adaptations, not directly published Triple-A rules.

1. **Use Inter throughout the product.** If Payminto can legally self-host a complementary display face, reserve it for public marketing or the largest page titles; do not mix display typography into tables and forms.
2. **Make white the dominant product color.** Use `#F7F9FB` for the app canvas, white for navigation/cards, `#E0E0E0`-class borders, black headings, and `#5F5E5E`-class secondary copy.
3. **Keep the sidebar quiet.** Use 16–20 px outline icons, 14 px navigation labels, 11–12 px uppercase group labels, compact dividers, and a 44 px row target. Reinforce the active item with coral text/icon plus a pale tint, left rail, or weight change.
4. **Use the brand gradient as punctuation.** A page-title band or a large display phrase can carry the red/orange/pink gradient. Buttons and dense controls should be solid black or coral for predictable contrast and state behavior.
5. **Prefer bordered surfaces over heavy cards.** Use 8–12 px radii in the dashboard, 16 px only for auth/hero cards, and shadows only on overlays, auth, or genuinely elevated content.
6. **Increase operational density.** Use the 4 px spacing basis and common 8/12/16/24/32 px gaps. Keep related filters on one row on wide screens and move secondary controls behind “More filters.”
7. **Treat checkout as a sequence.** Keep order summary, asset/provider choice, send/QR instructions, and result as distinct states. Persist amount, network, expiry/rate, address-copy, and wallet-opening actions where relevant.
8. **Preserve Payminto identity.** Reuse Payminto's wordmark, data model, terminology, icons, and original illustrations; borrow hierarchy and tokens, not Triple-A trade dress.

## Accessibility guardrails

“Triple-A colors” and WCAG “AAA” are different concepts. Contrast calculations from the observed source tokens show `#E22323` against white at about **4.67:1**, while `#F06B28` and `#E96991` are only about **3.07:1** and **3.05:1**. Therefore:

- solid coral may be used for normal-weight action text only after checking the actual size/weight and background;
- orange/pink gradient stops should be limited to large bold display text, fills, or decoration, not small body text;
- prefer `#5F5E5E` (about **6.46:1** on white) or black for secondary small copy;
- never use the gradient, coral, green, warning, or error color as the only state signal;
- pair active navigation with `aria-current` plus a rail/tint/weight change;
- preserve visible labels, focus rings, keyboard order, 44 px interactive targets, accessible names for icon-only controls, and live-region announcements for payment status.

The contrast figures are calculated design QA from the first-party color values in the [dashboard stylesheet](https://dashboard.triple-a.io/_app/immutable/assets/app.B-j-cPTr.css) and [marketing stylesheet](https://cdn.prod.website-files.com/68df9a9a9725a1dbb0ff032e/css/triple-a-staging.webflow.shared.f96a7032b.min.css); the recommendations are Payminto inferences.

## Proposed implementation sequence

1. Define Payminto tokens for typography, canvas/surface/border/text, coral action states, semantic states, radii, shadows, and the 4 px spacing scale.
2. Update the shared application shell and sidebar, including responsive collapse, active-state reinforcement, keyboard focus, and section hierarchy.
3. Rebuild login on the same tokens and validate desktop, tablet, mobile, error, loading, and password-visibility states.
4. Standardize page titles, filter bars, tables, empty/error states, forms, tabs, badges, and dialogs across merchant and admin areas.
5. Apply the same system to hosted checkout/delivery without importing dashboard chrome.
6. Run visual regression and accessibility checks across all major routes before removing legacy styles.

