# Junior brief: restyling a page

Follow this top to bottom for every page you touch.
Do not skip the verification steps; a page is not done until the measurements pass.

## Before you start

1. Read `DESIGN.md` sections 1, 2, 7, 8 and 13. Keep 13 open.
2. Open the matching references in `references.md` for the surface you are on (auth, shell, home, list, detail, builder, checkout, pay states, settings, empty states). Open the Mobbin links; do not work from memory.
3. Open the page at 1440px and 390px in light and dark before changing anything, and screenshot both. You will compare against these.
4. Read the page's data hooks. Know which numbers are real. If a value is computed in the component instead of coming from the API or ledger, flag it; do not style a lie.

## Tokens you may use

- Colour: only the CSS variables in `DESIGN.md` section 2 through Tailwind (`bg-surface`, `text-ink-soft`, `border-line`, `text-tide`, `bg-ok-tint`, and so on). No hex in components. No Tailwind palette colours (`emerald-500`, `zinc-100`) anywhere.
- Type: `text-display`, `text-h1`, `text-h2`, `text-h3`, `text-lead`, `text-body`, `text-body-sm`, `text-label`, `text-caption`. Add `num` on every amount, count, date and time.
- Radius: `rounded-xs` badges, `rounded-sm` controls, `rounded-md` cards, `rounded-lg` dialogs. Nothing else.
- Spacing: 4px grid. Card padding `p-5` (`p-4` below `sm`). Table cells `px-3 py-2.5`.
- Elevation: `shadow-1`, `shadow-2`, `shadow-3`. Cards get none.

## Components you must use instead of writing markup

| Need | Use |
| --- | --- |
| Page title with actions | `PageHeader` |
| A number with a label | `MetricCard` |
| An amount | `CurrencyDisplay` (never format money inline) |
| A status | `StatusBadge` (add missing keys to the map, do not inline a badge) |
| Where a payment is in its life | `Rail` |
| A table | `DataTable` or `Table` primitives |
| Nothing to show | `EmptyState` with a sentence that says what will fill it |
| An error | `ErrorState` with the real message and a retry |
| Buttons, inputs, selects, labels | the `ui/` primitives |

If a primitive cannot do what the page needs, extend the primitive and keep its API; do not fork it into the page.

## Steps

1. Remove every gradient, blur, decorative blob, illustration, uppercase eyebrow, middle dot, trailing arrow and emoji. Remove every hard-coded hex.
2. Replace the page's hero (if it has one) with `PageHeader`. One title, one sentence at most, actions on the right.
3. Lay the page out in this order: blocking strip (if any), page header, metrics row (if any), primary content, secondary content. Max width 1280px.
4. Convert every number to `CurrencyDisplay` or a `num` span. Check decimals against section 8.
5. Convert every status to `StatusBadge`. Check the label reads as a state, not a code.
6. Convert tables to 36px rows, sentence-case 12px headers, right-aligned amounts, a kebab for row actions.
7. Convert empty states: keep the table header, render two ghost rows, one sentence, one action.
8. Write the copy as sentences in sentence case. No marketing adjectives.
9. Check dark mode. Every surface, border and text must come from a token, so dark should just work; if something is invisible, you used a hard-coded colour.
10. Check keyboard: tab through the page; every control shows the tide focus ring.

## Verify at 390 and 1440, light and dark

Run the dev server, open Chrome DevTools, and for each of the four combinations:

1. **Overrun.** `document.documentElement.scrollWidth <= window.innerWidth`. Must be true.
2. **Clipped content.** For every element with `overflow: hidden`, `scrollHeight <= clientHeight + 1` unless it is a marquee, a truncated single line or a table container.
3. **Touch targets** (coarse pointer only): every `a, button, [role=button], input, select` has `getBoundingClientRect()` width and height >= 44 (use the `tap` utility if it does not).
4. **Text size.** No rendered text below 11px: iterate text nodes and check `getComputedStyle(parent).fontSize`.
5. **Contrast.** Spot check body text on its surface and label text on tinted surfaces with the DevTools colour picker: 4.5:1 body, 3:1 large and borders.
6. **Screenshots.** Save all four to `docs/design/screens/<page>-<width>-<scheme>.png` and compare with the before shots. Anything that got bigger, louder or more colourful needs a reason.

Snippet for steps 1 to 4 (paste in the console):

```js
(() => {
  const coarse = matchMedia('(pointer: coarse)').matches;
  const out = { overrun: document.documentElement.scrollWidth > innerWidth, clipped: [], small: [], tinyText: [] };
  for (const el of document.querySelectorAll('*')) {
    const cs = getComputedStyle(el);
    if (cs.overflow.includes('hidden') && el.scrollHeight > el.clientHeight + 1 && !el.closest('[data-slot=table-container]')) out.clipped.push(el);
    if (coarse && el.matches('a,button,[role=button],input,select')) { const r = el.getBoundingClientRect(); if (r.width && (r.width < 44 || r.height < 44)) out.small.push(el); }
    if (el.childNodes.length && [...el.childNodes].some(n => n.nodeType === 3 && n.textContent.trim()) && parseFloat(cs.fontSize) < 11) out.tinyText.push(el);
  }
  return out;
})();
```

## Done means

- All four screenshots saved and linked from the page's ticket.
- The console snippet returns `overrun: false` and empty arrays (minus the documented false positives).
- `npm run lint`, `npx tsc --noEmit` and `npm test` pass in `frontend/`.
- No item from the avoid list is on the page.
- Every number on the page can be traced to a hook that reads the API.
