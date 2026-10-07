# Design references

Studied on Mobbin on 2026-10-07 for the dashboard, checkout and auth redesign.
Every entry links the exact Mobbin screen so a junior designer or another agent can open the same thing.
Mobbin does not index Ramp, Brex or Adyen (searches for them return Square, Mercury, Midday and Xero instead), so those are covered by their nearest indexed peers and by public write-ups, and are marked as such.

How to read each entry: what is specifically good, what we take, what we deliberately avoid.
"Take" never means copy; it means the underlying rule, re-expressed in our tokens.

## Auth

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe sign-in | https://mobbin.com/screens/fb0b91ce-234b-44d5-9f8b-18864cc63941 | One card, labels above fields, "Forgot your password?" on the label row, secondary options (passkey, SSO) as quiet links, the anti-phishing note under the card. | Quiet secondary actions as plain links. | The full-bleed colour-wash background. It is Stripe's signature and it is also the number one AI-slop tell. |
| Mercury log in | https://mobbin.com/screens/95e22d1e-0d70-469b-b7c7-58090c5da404 | Split card: form on the left, one concrete offer on the right. Small type, generous whitespace, disabled button until valid. | Small type, generous whitespace. | The split card: the owner ruled out any two-sided auth layout and any product copy on auth. |
| Mercury sign-up | https://mobbin.com/screens/05c284f5-3e47-4e0f-b7e7-15cbaf0272ac | Question as the heading ("What's your email and password?"), inline rule hint ("Minimum 10 characters") that turns into a green check. | Validation hint that lives under the field and changes state, not a toast. | None. |
| Linear login code | https://mobbin.com/screens/fa5eecc4-532f-4cc2-a6e8-56814eefbf89 | Centre column, logo mark, one sentence, one field, one button. Hierarchy from spacing not colour. | The restraint. The auth screen is the quietest page in the product. This is the layout we ship: one centred column, no product copy (owner decision 2026-10-07). | None. |
| Linear onboarding steps | https://mobbin.com/screens/e0f6ce66-0d22-4858-b8a7-2d2aa3431b23 | A checklist card of three truthful sentences about what the integration will and will not do. | The "will not" line. On sign-up we say what we do not hold (card data, keys). | None. |
| Mintlify sign-in | https://mobbin.com/screens/64a2b751-941c-4f21-ba66-9a02c433f1f9 | Social buttons above a divider, labels, forgot link in the label row, terms line under the button, support link last. | Order of elements. | Grey disabled button that reads as broken. Our disabled state keeps the ink colour at 40% opacity. |

## Dashboard shell and navigation

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe home | https://mobbin.com/screens/e8ddbdc5-bc89-454f-9ebb-846905b83df4 | 190px sidebar, grouped nav ("Shortcuts", "Products"), search bar at the top, "Test mode" toggle top-right, "Developers" pinned at the bottom. | Grouped nav with small group labels; developers pinned bottom; search/command entry in the top bar. | The toggle treatment for test mode (a switch is easy to miss); we use a segmented control with a colour band. |
| Stripe test-mode banner | https://mobbin.com/screens/e777f9b4-8b6e-4412-b39d-ce66ecd577ae | Full-width orange band "Test mode - you're using test data" plus the toggle turning orange. Impossible to miss. | Unmistakable environment state: a colour band at the top edge of the frame plus the control itself changes colour. | A band that eats 40px of every page. Ours is a 3px top rule plus the segmented control; the full sentence lives in a tooltip. |
| Mercury home | https://mobbin.com/screens/dbf9eb63-dcfe-4876-965a-e8a6d8702092 | Search with keyboard hint (Cmd K), "Move Money" primary split button, quick-action chips under the greeting, workspace switcher at top of sidebar. | Keyboard hint on the search trigger; workspace switcher at the top of the sidebar; one primary action in the top bar. | The pastel area chart fill. |
| Square dashboard | https://mobbin.com/screens/c312d376-230c-4e7e-ab4b-1340562d88b1 | Dense black-on-white nav, "Complete your setup 80%" progress in the sidebar head, black pill buttons. | Setup progress belongs in the shell, not on the home page. | Pill-shaped buttons everywhere. |
| Linear settings nav | https://mobbin.com/screens/f826bc38-5f5a-4ccb-8075-056c6d915fbf | 13px nav type, 28px rows, groups separated by 20px, "Back to app" at the top. | Nav row height and type size: 13px, 30px rows. | None. |
| Stripe Dashboard iOS tab bar | https://mobbin.com/screens/fba86a00-cfc6-43b3-af79-052679faa58e | Five tabs: Home, Payments, Balances, Customers, Search. | On mobile the drawer carries everything; the four busiest destinations are one tap. | A bottom tab bar in the merchant dashboard for V1; the drawer is enough until usage data says otherwise. |

## Home and overview

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe home "Today" | https://mobbin.com/screens/e56b763c-5f8a-4002-bfa1-05f9b0341f38 | Gross volume with a 1px line chart and a time stamp; balance and payouts in a right rail; numbers set in the body size, labels in 12px grey. | Numbers are not headlines. A metric is label, value, time stamp. Right rail for balance and the next payout. | The "Get started" card with product illustrations. |
| Mercury home | https://mobbin.com/screens/dc52d8c6-76a8-49f7-a3ce-66c4b7b35500 | Balance card with a 30-day sparkline, accounts list with masked numbers, "Money in / Money out" pair with empty-state actions inside the card. | Empty state inside the card, with the action that fills it. Masked identifiers in a mono face. | Decorative gradient fill under the sparkline. |
| Copilot dashboard | https://mobbin.com/screens/c23ab69a-41bf-423a-be8e-a94e24135341 | Four cards, each with one number and a "View all" link, assets/debts pair set in colour. | Signed amounts coloured only when the sign is the point (money in vs money out). | Green "999%" delta chips. Never show a delta we cannot compute from two real periods. |
| Stripe Dashboard iOS home | https://mobbin.com/screens/f972bbd5-699d-42c3-942e-1cf9f3d25eda | "Today" strip with three numbers, then period pills, then one chart per metric with current vs previous period in two line weights. | Period segmented control; two-series line with the comparison series in grey. | Pill tabs in the brand colour. |

## Lists and tables

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe transactions | https://mobbin.com/screens/f8db8469-2830-4c6b-98ca-53bf4c1e3caf | Status count tabs above the table (All 6, Succeeded 1, Failed 3), filter chips with a plus icon, right-aligned amount with a separate currency column, status badge with an icon glyph, "6 results" under the table, row kebab. | Count tabs; amount right-aligned in tabular figures with the currency as a separate quiet token; status badge carries a glyph not just colour; result count below. | Grey text for the whole row. Only secondary cells are grey. |
| Stripe customers | https://mobbin.com/screens/eade1912-e11c-4d70-9556-1527dbd9837c | Sortable headers with an arrow, 36px rows, "3 results", the toast at the bottom with "1 successful, 0 failed". | 36px rows, 12px header labels in sentence case (not all caps), toasts that report counts. | Edit-columns button on every list in V1. |
| Mercury transactions | https://mobbin.com/screens/f77b24a7-e68e-4fc1-b5ac-91856a92f12b | Signed amounts coloured per sign, method column with an icon, net change for the period above the chart. | Signed amount colouring for ledger views (in = green, out = ink). | The pink area fill. |
| Midday transactions | https://mobbin.com/screens/6ead266a-00ac-4779-ab2f-772183755698 | Full-grid table with 1px rules, mono-ish amounts, a status column that reads "No receipt / Ready to export". | Status vocabulary that says the next action. | Vertical rules between every column; we rule rows only. |

## Detail pages

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe payment detail | https://mobbin.com/screens/ebf7e9fe-326a-4459-ae16-f94aea16f56b | Eyebrow "PAYMENT", amount as the title with currency after it, status badge inline with the title, "Charged to <customer>" line, actions top-right, timeline on the left, payment breakdown as a two-column ledger (amount, fees, net), details rail on the right with ID copy. | The whole structure. Amount-as-title, breakdown as ledger lines, ID with copy, timeline with timestamps. | The all-caps eyebrow. Ours is sentence case, 12px, grey. |
| Stripe payment with refund timeline | https://mobbin.com/screens/d4df0ce4-30b8-4770-bfb6-eb8b553bb2e9 | Timeline entries carry a sentence ("Successfully refunded MYR 0.50 due to customer request") plus a timestamp and a "View details" link. | Timeline rows are sentences with timestamps, never bare status words. | None. |
| Stripe payment link detail | https://mobbin.com/screens/e1a7563a-8ca7-4e2d-9cc8-2dd34d50e2aa | Copyable URL field with copy icon, products table, payment-methods row with logos, details as a label/value list, live preview on the right. | Copyable URL as the hero of a link detail page; label/value list in a two-column grid. | None. |
| Phantom received | https://mobbin.com/screens/ad1289e1-718e-4a5d-9ca1-7a8d71ddb8b6 | Signed amount in green, label/value rows (Date, Status, From with copy, Network), "View on Solscan" as the one button. | Chain receipts always end with "View on explorer". Truncated address with a copy button. | None. |

## Payment link builder

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe create link | https://mobbin.com/screens/ffb06d29-b252-42ea-a2b2-6a732a669a8c | Form left (40%), live preview right (60%) with device toggle, "Create link" top-right, "Advanced options" collapsed. | Two-pane builder with device toggle; advanced options collapsed by default; the single primary action lives in the top bar of the builder. | A preview that invents data. Our preview renders exactly the form state and nothing else. |
| Stripe accept payment settings | https://mobbin.com/screens/f3d0a91f-6277-46f2-99b5-f6481f3e5a43 | Per-product accordion with checkboxes, confirmation-page radio with a sentence under each option. | Each option has a one-line consequence under it. | Pink preview button. |
| Square create link | https://mobbin.com/screens/05ed1528-eda0-4d3f-862a-43ebcdcbc5bc | "Any amount / Exact amount" as a segmented control, character counter under description, preview tabs Details / Checkout / Confirmation. | Segmented control for amount mode; preview tabs that follow the customer's steps. | None. |
| Stripe Dashboard iOS create link | https://mobbin.com/screens/fcaa5b1e-cf08-46e0-a083-8d0e8fecbb8a | Full-screen sheet, review step with quantity stepper and toggles with a sentence each, "Create link" pinned at the bottom. | On mobile the builder is a sheet with the primary action pinned above the safe area. | None. |

## Checkout, desktop and mobile

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe Checkout (preview in link builder) | https://mobbin.com/screens/ffb06d29-b252-42ea-a2b2-6a732a669a8c | Left: merchant, amount, description. Right: Apple Pay first, "or pay with card", email, card, name, country, Pay button. Powered-by line last. | The two-pane order: summary left, pay right; wallet button first, then the divider, then fields. | Left pane painted in the merchant's brand colour as a full fill. We tint lightly and keep text in ink. |
| Lemon Squeezy checkout | https://mobbin.com/screens/d28c26fa-be83-4d5b-906f-bc0d4a4aea56 | Discount code field with inline "Apply", Subtotal / Total lines, "You will be charged $1.00 now, then every month" sentence under the button, "Payments are secure and encrypted" strip. | Sentence under the pay button that states exactly what will be charged. | Lavender disabled button. |
| Luma pay sheet (iOS) | https://mobbin.com/screens/2b87d56e-02ec-48c1-b0de-9630222bdbc3 | Bottom sheet with card fields, "Scan card" link, "Pay US$5.00" button carrying a lock glyph, consent sentence above the button. | Amount on the pay button; lock glyph; consent sentence above, not below. | None. |
| CLEAR pay sheet (iOS) | https://mobbin.com/screens/a899b8e8-2924-4a8c-a9ef-ee4378c39d7b | Apple Pay black button, "Or use a card" divider, card fields, "Subscribe for $209.00" with lock. | Black wallet button at full width; divider copy "Or pay with". | None. |
| Klarna confirm (iOS) | https://mobbin.com/screens/129a0cc2-758b-4e61-8246-0f70bc2922f3 | "Confirm how you pay" heading, rows for identity, plan, method, "Due today" in a larger weight. | "Due today" as the loudest number on the sheet. | None. |
| Cash App pay (iOS) | https://mobbin.com/screens/d21ee823-940f-43d8-bb22-4c35de3d2e21 | Giant amount, keypad, one black button. "You sent $1" done state with a green check. | The done state is a sentence, one check mark and two actions (Receipt, Done). | Brand green as a full-screen fill. |

## Stablecoin pay states

Mobbin has almost no hosted crypto checkout screens, so these come from wallets, which set the expectation a payer carries into our page.

| State | Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- | --- |
| Address and QR | Base App receive | https://mobbin.com/screens/5f919212-3b79-4e43-a0a3-6bba4166e9d5 | QR in a white rounded panel on a dark scrim, truncated address underneath with a copy icon, one sentence about which networks the address accepts. | QR on white always (scanners need contrast), full address in mono with a copy button, network sentence under it. | Avatar inside the QR. |
| Address and QR | Coinbase PayNow deposit | https://mobbin.com/screens/e773d9f8-0539-4387-b42f-650491ff6bbb | Numbered steps next to the QR; a "we do not support X" line. | Numbered steps for first-time payers; a sentence listing what will not be credited (wrong network, wrong token). | None. |
| Summary before send | Phantom send summary | https://mobbin.com/screens/d1e9b056-98fc-4a66-a090-60e46c10304a | "1 USDC" huge, "~$0.99" under it, rows To / Network / Network fee, one button. | Amount in asset first, fiat equivalent second and smaller, network named explicitly. | None. |
| Confirming | Phantom sending | https://mobbin.com/screens/76a832bf-9917-446e-a4e4-5fcd275f3910 | Three-dot pulse, "Sending...", one line naming amount and destination, "View transaction" link, Close. | A confirming state names the amount, the destination, and links to the explorer as soon as a signature exists. | A spinner with no sentence. |
| Confirmed | Phantom sent | https://mobbin.com/screens/bf9ffc9e-7f6a-415c-bbfa-98079468f351 | Green circle check, "Sent!", the sentence again, explorer link. | Confirmed is a check, a sentence, an explorer link. | Confetti. |
| Receipt | Phantom received | https://mobbin.com/screens/ad1289e1-718e-4a5d-9ca1-7a8d71ddb8b6 | Label/value rows, status word in green, "View on Solscan" button. | Final receipt as label/value rows with the status word in its status colour. | None. |
| Countdown, under paid, over paid | No indexed reference | - | - | Our own: a countdown is a mono timer next to the amount, never a progress ring; under paid shows "Received X of Y, send the remaining Z" as a sentence with both numbers; over paid shows the overage and the refund path. | Red for under paid. It is a waiting state, not a failure; amber. |

## Settings

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe profile | https://mobbin.com/screens/e6b65816-eb18-400a-a3ba-545f05de645c | Breadcrumb "Settings >", section headings with a one-line description, label/field rows with the label in a fixed 120px column, Cancel/Save per section. | Label-left form rows for settings; save per section. | None. |
| Stripe business tabs | https://mobbin.com/screens/dcebced2-2285-407c-8f28-84dfaedabdd4 | Horizontal tabs under the page title, 2px underline on the active tab. | Underline tabs for settings sub-pages. | None. |
| Stripe branding | https://mobbin.com/screens/d670c548-b885-417a-8122-f65da3dc2803 | Colour fields show the swatch and the hex; preview on the right with a device toggle. | Hex next to swatch. | None. |
| Lemon Squeezy recovery | https://mobbin.com/screens/fd1ff0a3-4c1e-46e5-93b8-dd0ee4e30e66 | Toggle rows: title, two-line description with a Help link, toggle on the right with its label. | Toggle row anatomy. | None. |

## Empty and error states

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Mercury no transactions | https://mobbin.com/screens/dbbf7fd1-24e8-4494-9c0c-27007652c7e5 | Skeleton rows behind the message, one sentence saying what will appear and when, one button. | Empty state renders the table header and two ghost rows so the page keeps its shape; the sentence says what fills it. | None. |
| Base App no activity | https://mobbin.com/screens/b50e557a-2c87-4280-ac04-550692ce4f2f | Dashed border panel, small clock glyph, two short lines. | Dashed panel for "nothing yet" inside a card. | None. |
| Square balances empty | https://mobbin.com/screens/85db0a8a-e655-4cb2-ad43-e93c4b821841 | A grey info strip "Link a bank account to transfer funds" with the action inline. | Inline info strip when the empty state has one obvious cause. | The illustration cards. |
| Stripe payouts paused | https://mobbin.com/screens/d7461782-27a7-41e9-aa99-ac528bcd4fca | Red alert strip at the top of the page with a one-line cause and a button. | Blocking conditions are a strip at the top of the page, not a modal. | None. |
| Stripe sandbox dialog | https://mobbin.com/screens/e7035aac-0ea1-47a0-97ae-2af754e37b9e | Radio cards with a title and a consequence sentence. | Radio cards for choices with consequences (custody provider, settlement asset). | None. |
| Mercury support panel | https://mobbin.com/screens/d8564614-5b4c-4cdc-8088-0891fc9260df | Right-anchored panel, select, textarea with an example placeholder, drop zone. | Example text in placeholders ("E.g. My transaction is still pending"). | None. |

## Not on Mobbin (public write-ups)

- Ramp: dense tables, 13px type, black primary buttons, status as coloured text not pills. Taken: black primary, status-as-text option in dense tables.
- Brex: strong left nav with icons, large balance number with a smaller currency code after it. Taken: currency code after the number, 60% size.
- Adyen Customer Area: a payment's lifecycle shown as a horizontal stepper (Authorised, Captured, Settled). Taken directly as our settlement rail (see DESIGN.md, "The rail").

## Checkout

Added 2026-10-07 for the hosted checkout redesign (`checkout/`). The earlier "Checkout, desktop and mobile" and "Stablecoin pay states" tables above still apply; these are the extra screens opened for this pass.

| Reference | Screen | Good | Take | Avoid |
| --- | --- | --- | --- | --- |
| Stripe accept-payment preview | https://mobbin.com/screens/f3d0a91f-6277-46f2-99b5-f6481f3e5a43 | Left: merchant, amount, line items with subtotal and total due. Right: contact, payment method tiles (Card, GrabPay), fields, one button. | Line items with a single "Total due" line as the only bold one; method choice as tiles above the fields. | The brand-blue left pane. Ours is ink on canvas. |
| Lemon Squeezy checkout preview | https://mobbin.com/screens/e6993f5b-8386-4400-8eff-090087cb802b | Product with description on the left, variant radio cards with prices, "Payment details" with one field and "Continue" on the right. | Radio cards for the choice with a consequence line under each. | Purple primary button. |
| Ro pay sheet (iOS) | https://mobbin.com/screens/53ede06d-f872-4f28-8509-c0df284af099 | Method rows (card with lock, Apple Pay, PayPal), "Pay $35 today" black button, the exact charge sentence under it. | Method rows at 56px with a radio, a mark and a title; the amount on the button. | The TLS badge line. |
| American Airlines review and pay (iOS) | https://mobbin.com/screens/bb5fa997-be4e-47b8-baa1-260802501055 | Collapsible trip summary with a chevron, "Total amount due" card, method radios, total and pay button pinned at the bottom. | The collapsible summary header on mobile and the pinned pay button above the safe area. | Blue link colour for everything. |
| Coinbase select network (iOS) | https://mobbin.com/screens/f185d754-4fcf-4f5e-9e60-9ba6dd632a4b | Network rows with the chain mark, name and an estimated fee column; an info strip explaining what to pick. | Network is a first-class choice with the chain named in words, never an icon alone. | A fee column we do not hold data for. |
| Coinbase request summary (iOS) | https://mobbin.com/screens/e6f998cc-237e-4e04-b94f-de110dab55fb | Large amount, asset under it, label/value rows (From, Note, Amount, Asset), one button. | Label/value rows for the receipt. | Brand-blue amount. |
| Cash App pay with bitcoin sheet (iOS) | https://mobbin.com/screens/51169081-3312-445f-b486-4e5fcf7abeba | Merchant name as the heading, one sentence about the rail (Lightning), two buttons. | Name the rail in a sentence next to the merchant. | Full-bleed map. |
