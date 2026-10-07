"use client";

import { useRef } from "react";
import { gsap } from "../motion/gsap";
import { useMotion } from "../motion/use-motion";

const STEPS = [
  ["Payer pays on any rail", "Card, UPI or a stablecoin on Solana: each rail is a connector behind the same switch."],
  ["The switch routes it", "An intent and an attempt, an idempotency key, and an exclusive claim so a capture is never sent twice."],
  ["Ledger lines post", "The payment lands as balanced double-entry lines. Balances are derived from lines, never stored."],
  ["The fee posts", "From a versioned fee rule, never edited in place. The payment records the rule version it paid."],
  ["Settlement", "Settlement is lines too, so the journal still balances and history is append-only."],
] as const;

const RAILS = ["Card", "UPI", "USDC · Solana", "USDT · Solana"] as const;

type Line = { step: number; side: "Dr" | "Cr"; account: string; amount: string; memo?: string };
const LINES: Line[] = [
  { step: 2, side: "Dr", account: "Clearing · Solana", amount: "100.00", memo: "Payment" },
  { step: 2, side: "Cr", account: "Merchant balance", amount: "100.00" },
  { step: 3, side: "Dr", account: "Merchant balance", amount: "1.00", memo: "Fee · rule v3, 1%" },
  { step: 3, side: "Cr", account: "Fee revenue", amount: "1.00" },
  { step: 4, side: "Dr", account: "Merchant balance", amount: "99.00", memo: "Settlement" },
  { step: 4, side: "Cr", account: "Clearing · Solana", amount: "99.00" },
];

export function Flow() {
  const ref = useRef<HTMLElement>(null);

  useMotion(ref, ({ wide }) => {
    const q = gsap.utils.selector(ref);
    const tl = gsap.timeline({
      defaults: { ease: "none", duration: 0.5 },
      scrollTrigger: wide
        ? {
            trigger: q("[data-flow-pin]")[0],
            start: "top top",
            end: () => "+=" + window.innerHeight * 2.6,
            pin: true,
            scrub: 0.6,
            anticipatePin: 1,
            invalidateOnRefresh: true,
          }
        : { trigger: q("[data-flow-stage]")[0], start: "top 75%", end: "bottom 55%", scrub: 0.6 },
    });

    const steps = q("[data-flow-step]");
    const at = (i: number) => i * 1;

    if (wide) {
      tl.fromTo(q("[data-flow-progress]"), { scaleY: 0 }, { scaleY: 1, duration: 5 }, 0);
      steps.forEach((s, i) => {
        tl.fromTo(s, { opacity: 0.32 }, { opacity: 1, duration: 0.3 }, at(i));
        if (i < steps.length - 1) tl.to(s, { opacity: 0.32, duration: 0.3 }, at(i + 1) - 0.05);
      });
    }

    // 0: rails light up, the USDC rail is chosen
    tl.fromTo(q("[data-rail]"), { opacity: 0.35 }, { opacity: 1, stagger: 0.08, duration: 0.3 }, at(0));
    tl.fromTo(q("[data-rail-pick]"), { opacity: 0, scale: 0.9 }, { opacity: 1, scale: 1, duration: 0.3 }, at(0) + 0.5);

    // 1: token drops to the switch, attempt goes processing -> succeeded
    tl.fromTo(q("[data-token]"), { opacity: 0, y: 0 }, { opacity: 1, duration: 0.15 }, at(1));
    tl.to(q("[data-token]"), { y: () => (q("[data-token-track]")[0] as HTMLElement).offsetHeight - 14, duration: 0.6 }, at(1));
    tl.to(q("[data-token]"), { opacity: 0, duration: 0.15 }, at(1) + 0.6);
    tl.fromTo(q("[data-switch]"), { opacity: 0.4 }, { opacity: 1, duration: 0.3 }, at(1) + 0.2);
    tl.fromTo(q("[data-switch-row]"), { opacity: 0, x: -8 }, { opacity: 1, x: 0, stagger: 0.12, duration: 0.25 }, at(1) + 0.4);
    tl.fromTo(q("[data-status='processing']"), { opacity: 1 }, { opacity: 0, duration: 0.15 }, at(1) + 0.8);
    tl.fromTo(q("[data-status='succeeded']"), { opacity: 0 }, { opacity: 1, duration: 0.15 }, at(1) + 0.8);

    // 2..4: ledger lines post, debit then credit
    [2, 3, 4].forEach((step) => {
      tl.fromTo(
        q(`[data-line-step='${step}']`),
        { opacity: 0, x: -12 },
        { opacity: 1, x: 0, stagger: 0.2, duration: 0.3 },
        at(step),
      );
    });
    tl.fromTo(q("[data-balanced]"), { opacity: 0, y: 8 }, { opacity: 1, y: 0, duration: 0.3 }, at(4) + 0.55);
    tl.to({}, { duration: 0.2 }, 5);

    return () => tl.scrollTrigger?.kill();
  });

  return (
    <section ref={ref} id="flow" className="relative border-y border-border bg-background">
      <div data-flow-pin className="relative flex min-h-screen items-center overflow-hidden py-24 lg:pb-10 lg:pt-20">
        <div className="grid-bg absolute inset-0 opacity-40" aria-hidden />
        <div className="relative mx-auto grid w-full max-w-6xl gap-12 px-4 sm:px-6 lg:grid-cols-[5fr_7fr] lg:gap-14">
          <div>
            <p className="eyebrow mb-4">How a payment flows</p>
            <h2 className="font-display mb-8 text-[44px] font-black text-foreground md:text-[60px] lg:text-[52px] xl:text-[60px]">
              One payment,
              <br />
              <span className="brand-underline">every line</span>
              <br />
              accounted for.
            </h2>
            <ol className="relative space-y-4 pl-6">
              <span className="absolute left-0 top-1 h-[calc(100%-8px)] w-[2px] rounded bg-border" aria-hidden />
              <span
                data-flow-progress
                className="absolute left-0 top-1 h-[calc(100%-8px)] w-[2px] origin-top rounded bg-brand-ink"
                aria-hidden
              />
              {STEPS.map(([title, desc], i) => (
                <li key={title} data-flow-step>
                  <div className="text-[16px] font-bold text-foreground">
                    <span className="mr-2 font-mono text-[12px] text-brand-ink">0{i + 1}</span>
                    {title}
                  </div>
                  <p className="mt-1 text-[14px] font-semibold leading-[1.5] text-foreground-soft">{desc}</p>
                </li>
              ))}
            </ol>
          </div>

          <div data-flow-stage className="card-ring-lg relative self-center p-5 sm:p-7">
            <div className="mb-3 flex items-center justify-between">
              <span className="text-[12px] font-bold uppercase tracking-[0.14em] text-foreground-muted">Rails</span>
              <span className="rounded-full bg-surface-soft px-2.5 py-0.5 text-[12px] font-bold text-foreground-muted">
                Illustrative figures
              </span>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              {RAILS.map((r) => (
                <div
                  key={r}
                  data-rail
                  className="relative rounded-xl bg-surface-soft px-3 py-2.5 text-center text-[13px] font-bold text-foreground"
                >
                  {r}
                  {r === "USDC · Solana" && (
                    <span data-rail-pick className="absolute inset-0 rounded-xl ring-2 ring-brand-ink" aria-hidden />
                  )}
                </div>
              ))}
            </div>

            <div data-token-track className="relative mx-auto h-10 w-[2px] bg-border" aria-hidden>
              <span
                data-token
                className="absolute -left-[6px] top-0 h-[14px] w-[14px] rounded-full bg-brand shadow-[0_0_0_4px_rgba(167,139,250,0.25)]"
                style={{ opacity: 0 }}
              />
            </div>

            <div data-switch className="rounded-2xl bg-[#0e0f0c] p-4 font-mono text-[12.5px] text-[#fafaf7] sm:text-[13px]">
              <div className="mb-2 flex items-center justify-between font-sans">
                <span className="text-[13px] font-bold">Payment switch</span>
                <span className="relative inline-grid text-[12px] font-bold">
                  <span data-status="processing" className="col-start-1 row-start-1 rounded-full bg-white/10 px-2 py-0.5 text-white/70 opacity-0">
                    processing
                  </span>
                  <span data-status="succeeded" className="col-start-1 row-start-1 rounded-full bg-[#1f9d55]/25 px-2 py-0.5 text-[#7ee2a8]">
                    succeeded
                  </span>
                </span>
              </div>
              <div data-switch-row className="text-white/70">
                intent <span className="text-brand-hover">pi_8f2c</span> · attempt <span className="text-brand-hover">1</span>
              </div>
              <div data-switch-row className="truncate text-white/70">
                Idempotency-Key <span className="text-brand-hover">ord-1042</span>
              </div>
              <div data-switch-row className="text-white/70">
                claim <span className="text-brand-hover">exclusive</span> · connector <span className="text-brand-hover">solana</span>
              </div>
            </div>

            <div className="mt-4 overflow-hidden rounded-2xl ring-1 ring-border">
              <div className="flex items-center justify-between bg-surface-soft px-4 py-2 text-[12px] font-bold text-foreground-muted">
                <span>Journal · USDC.SOLANA</span>
                <span className="hidden sm:inline">Debit · Credit</span>
              </div>
              <ul className="divide-y divide-border font-mono text-[12.5px] sm:text-[13px]">
                {LINES.map((l, i) => (
                  <li
                    key={i}
                    data-line-step={l.step}
                    className="grid grid-cols-[1fr_auto_auto] items-center gap-3 px-4 py-[7px]"
                  >
                    <span className={l.side === "Cr" ? "pl-4 text-foreground-soft" : "text-foreground"}>
                      {l.side} {l.account}
                      {l.memo && <span className="block font-sans text-[11.5px] font-bold text-foreground-muted sm:ml-2 sm:inline">{l.memo}</span>}
                    </span>
                    <span className="w-14 text-right font-semibold text-foreground">{l.side === "Dr" ? l.amount : ""}</span>
                    <span className="w-14 text-right font-semibold text-foreground">{l.side === "Cr" ? l.amount : ""}</span>
                  </li>
                ))}
              </ul>
              <div
                data-balanced
                className="flex items-center justify-between border-t border-border bg-surface-mint px-4 py-2.5 text-[13px] font-bold text-brand-ink"
              >
                <span>Debits 200.00 = Credits 200.00</span>
                <span>Balanced ✓</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
