"use client";

import { useRef } from "react";
import { gsap } from "../motion/gsap";
import { useMotion } from "../motion/use-motion";

type Box = { id: string; x: number; y: number; w: number; h: number; title: string; sub: string; tone?: "ok" | "bad" };

const DEPOSIT: Box[] = [
  { id: "payer", x: 6, y: 16, w: 150, h: 60, title: "Payer wallet", sub: "SPL transfer" },
  { id: "account", x: 204, y: 16, w: 150, h: 60, title: "Deposit account", sub: "one per payment" },
  { id: "watcher", x: 204, y: 196, w: 150, h: 60, title: "Watcher", sub: "RPC A + RPC B" },
  { id: "ledger", x: 6, y: 196, w: 150, h: 60, title: "Ledger", sub: "USDC.SOLANA" },
];
const DEPOSIT_PATH = "M81 46 L279 46 L279 226 L81 226";

const SWEEP: Box[] = [
  { id: "processing", x: 6, y: 16, w: 150, h: 56, title: "processing", sub: "claim in one tx" },
  { id: "pending", x: 204, y: 16, w: 150, h: 56, title: "pending", sub: "signature saved, sent" },
  { id: "completed", x: 204, y: 196, w: 150, h: 56, title: "completed", sub: "booked once", tone: "ok" },
  { id: "failed", x: 6, y: 196, w: 150, h: 56, title: "failed", sub: "proven dead", tone: "bad" },
];
const SWEEP_PATH = "M81 44 L279 44 L279 224";

function Boxes({ boxes, scope }: { boxes: Box[]; scope: string }) {
  return (
    <>
      {boxes.map((b) => {
        const fill = b.tone === "ok" ? "#e6f4ec" : b.tone === "bad" ? "#fdecec" : "#ffffff";
        const stroke = b.tone === "ok" ? "#1f7a4a" : b.tone === "bad" ? "#d03238" : "rgba(14,15,12,0.14)";
        return (
          <g key={b.id}>
            <rect
              x={b.x}
              y={b.y}
              width={b.w}
              height={b.h}
              rx="14"
              fill={fill}
              stroke={stroke}
              strokeDasharray={b.tone === "bad" ? "5 4" : undefined}
            />
            <rect
              data-lit={`${scope}-${b.id}`}
              x={b.x - 3}
              y={b.y - 3}
              width={b.w + 6}
              height={b.h + 6}
              rx="16"
              fill="none"
              stroke="#a78bfa"
              strokeWidth="3"
              opacity="0"
            />
            <text x={b.x + 14} y={b.y + 25} fontSize="14" fontWeight="700" className="fill-foreground">
              {b.title}
            </text>
            <text x={b.x + 14} y={b.y + b.h - 13} fontSize="12" fontWeight="600" className="fill-foreground-muted">
              {b.sub}
            </text>
          </g>
        );
      })}
    </>
  );
}

const SWEEP_RULES = [
  "Every signature is saved with its blockhash before it is sent.",
  "One sweep in flight per token account, enforced by a unique lock row.",
  "Every transition is a compare-and-set, so a stale worker changes nothing.",
  "Deposits become swept only when a finalized attempt is booked.",
];

export function Solana() {
  const ref = useRef<HTMLElement>(null);

  useMotion(ref, () => {
    const q = gsap.utils.selector(ref);
    const timelines: gsap.core.Timeline[] = [];

    const ride = (scope: "dep" | "sweep", stops: string[], extra?: (tl: gsap.core.Timeline) => void) => {
      const panel = q(`[data-panel='${scope}']`)[0] as HTMLElement;
      const path = panel.querySelector<SVGPathElement>("[data-ride-path]")!;
      const token = panel.querySelector<SVGGElement>("[data-ride-token]")!;
      const tl = gsap.timeline({
        defaults: { ease: "none" },
        scrollTrigger: { trigger: panel, start: "top 80%", end: "bottom 45%", scrub: 0.6 },
      });
      tl.fromTo(path, { strokeDashoffset: 1 }, { strokeDashoffset: 0, duration: 3 }, 0);
      tl.fromTo(token, { opacity: 0 }, { opacity: 1, duration: 0.2 }, 0);
      tl.to(token, { motionPath: { path, align: path, alignOrigin: [0.5, 0.5] }, duration: 3 }, 0);
      const step = 3 / (stops.length - 1);
      stops.forEach((id, i) => {
        tl.fromTo(q(`[data-lit='${scope}-${id}']`), { opacity: 0 }, { opacity: 1, duration: 0.15 }, Math.max(0, i * step - 0.1));
        if (i < stops.length - 1) tl.to(q(`[data-lit='${scope}-${id}']`), { opacity: 0, duration: 0.15 }, (i + 1) * step - 0.1);
      });
      extra?.(tl);
      timelines.push(tl);
    };

    ride("dep", ["payer", "account", "watcher", "ledger"], (tl) => {
      tl.fromTo(q("[data-dep-note]"), { opacity: 0, y: 6 }, { opacity: 1, y: 0, stagger: 0.4, duration: 0.3 }, 1.4);
    });
    ride("sweep", ["processing", "pending", "completed"], (tl) => {
      tl.fromTo(q("[data-sweep-rule]"), { opacity: 0, x: -8 }, { opacity: 1, x: 0, stagger: 0.5, duration: 0.3 }, 0.4);
    });

    return () => timelines.forEach((t) => t.scrollTrigger?.kill());
  });

  return (
    <section ref={ref} id="solana" className="py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="mb-14 grid gap-6 lg:grid-cols-[1.1fr_1fr] lg:items-end">
          <div>
            <p className="eyebrow mb-4">Stablecoins on Solana</p>
            <h2 className="font-display text-[44px] font-black text-foreground md:text-[52px] xl:text-[56px]" data-reveal>
              USDC and USDT,
              <br />
              <span className="brand-underline">booked like any rail.</span>
            </h2>
          </div>
          <p className="text-[17px] font-semibold leading-[1.5] text-foreground-soft">
            Each payment gets its own deposit account. The credit is the observed balance
            change, never the instruction amount. Wrong tokens are recorded as anomalies,
            never credited. Sweeps to your hot wallet survive crashes, lagging nodes and
            racing workers.
          </p>
        </div>

        <div className="grid gap-6 lg:grid-cols-2">
          <div data-panel="dep" className="card-ring-lg p-5 sm:p-8">
            <h3 className="mb-1 text-[22px] font-bold text-foreground">The deposit</h3>
            <p className="mb-6 text-[14px] font-semibold text-foreground-muted">Confirmed means seen. Finalized means credited.</p>
            <svg viewBox="0 0 360 270" className="mx-auto block h-auto w-full max-w-[460px]" role="img" aria-label="A payer wallet sends an SPL transfer to a deposit account made for the payment; the watcher reads it from two RPC providers and the ledger books it as USDC.SOLANA.">
              <path d={DEPOSIT_PATH} stroke="rgba(14,15,12,0.12)" strokeWidth="2" fill="none" />
              <path data-ride-path d={DEPOSIT_PATH} pathLength={1} strokeDasharray="1" strokeDashoffset="0" stroke="#3b1d8a" strokeWidth="2.5" fill="none" strokeLinejoin="round" />
                            <g data-ride-token style={{ opacity: 0 }}>
                <circle r="11" fill="#2775ca" stroke="#fff" strokeWidth="2.5" />
                <text y="4" textAnchor="middle" fontSize="10" fontWeight="800" fill="#fff">$</text>
              </g>
              <Boxes boxes={DEPOSIT} scope="dep" />
            </svg>
            <ul className="mt-6 grid gap-2 text-[14px] font-semibold text-foreground-soft">
              <li data-dep-note>A null answer from a lagging node never moves the cursor.</li>
              <li data-dep-note>A deposit is dropped only when two distinct endpoints agree.</li>
              <li data-dep-note>Live refuses to boot with fewer than two RPC providers.</li>
            </ul>
          </div>

          <div data-panel="sweep" className="card-ring-lg p-5 sm:p-8">
            <h3 className="mb-1 text-[22px] font-bold text-foreground">The sweep</h3>
            <p className="mb-6 text-[14px] font-semibold text-foreground-muted">A state machine that can be rebuilt from the database and the chain alone.</p>
            <svg viewBox="0 0 360 270" className="mx-auto block h-auto w-full max-w-[460px]" role="img" aria-label="Sweep states: processing, then pending once the attempt is saved and sent, then completed when one attempt is finalized and booked once; failed only when every attempt is proven dead.">
              <path d="M81 72 L81 196" stroke="#d03238" strokeOpacity="0.5" strokeWidth="2" strokeDasharray="5 4" fill="none" />
              <path d="M240 72 L140 196" stroke="#d03238" strokeOpacity="0.5" strokeWidth="2" strokeDasharray="5 4" fill="none" />
              <path d={SWEEP_PATH} stroke="rgba(14,15,12,0.12)" strokeWidth="2" fill="none" />
              <path data-ride-path d={SWEEP_PATH} pathLength={1} strokeDasharray="1" strokeDashoffset="0" stroke="#3b1d8a" strokeWidth="2.5" fill="none" strokeLinejoin="round" />
                            <g data-ride-token style={{ opacity: 0 }}>
                <circle r="9" fill="#a78bfa" stroke="#3b1d8a" strokeWidth="2" />
              </g>
              <Boxes boxes={SWEEP} scope="sweep" />
            </svg>
            <ul className="mt-6 grid gap-2 text-[14px] font-semibold text-foreground-soft">
              {SWEEP_RULES.map((r) => (
                <li key={r} data-sweep-rule>
                  {r}
                </li>
              ))}
            </ul>
          </div>
        </div>

        <p className="mt-6 text-[14px] font-semibold text-foreground-muted">
          Full rules and tests ship with the Solana track write-up. RPC is treated as
          evidence: a deposit is credited only when two distinct providers agree.
        </p>
      </div>
    </section>
  );
}
