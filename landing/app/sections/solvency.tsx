"use client";

import { useRef } from "react";
import { gsap } from "../motion/gsap";
import { useMotion } from "../motion/use-motion";

type Node = { id: string; x: number; y: number; w: number; h: number; title: string; value?: string; sub: string; tone?: "positive" };
type Edge = { id: string; d: string; label?: string; lx?: number; ly?: number; anchor?: "start" | "middle" | "end" };
type Layout = { w: number; h: number; nodes: Node[]; edges: Edge[] };

const COPY = {
  a: { title: "Ledger liabilities", value: "1,250 USDC", sub: "checkpoint + hash" },
  b: { title: "On-chain reserves", value: "1,300 USDC", sub: "balanceOf at finalized" },
  c: { title: "Chainlink CRE workflow", sub: "DON consensus" },
  d: { title: "Keystone forwarder", sub: "delivers the signed report" },
  e: { title: "GatewayAttestations", sub: "replay-checked, stored" },
  f: { title: "Gateway verifier", sub: "re-reads it from its own RPC" },
  g: { title: "Attested", sub: "stored and shown", tone: "positive" as const },
};

const WIDE_LAYOUT: Layout = {
  w: 1100,
  h: 440,
  nodes: [
    { id: "a", x: 20, y: 30, w: 240, h: 96, ...COPY.a },
    { id: "b", x: 20, y: 314, w: 240, h: 96, ...COPY.b },
    { id: "c", x: 360, y: 180, w: 220, h: 80, ...COPY.c },
    { id: "d", x: 660, y: 30, w: 200, h: 80, ...COPY.d },
    { id: "e", x: 890, y: 30, w: 200, h: 80, ...COPY.e },
    { id: "f", x: 890, y: 330, w: 200, h: 80, ...COPY.f },
    { id: "g", x: 660, y: 330, w: 200, h: 80, ...COPY.g },
  ],
  edges: [
    { id: "ac", d: "M260 78 C 320 78, 300 205, 360 205", label: "liabilities", lx: 272, ly: 66, anchor: "start" },
    { id: "bc", d: "M260 362 C 320 362, 300 235, 360 235", label: "reserves", lx: 272, ly: 386, anchor: "start" },
    { id: "cd", d: "M580 220 C 625 220, 615 70, 660 70", label: "signed report", lx: 588, ly: 246, anchor: "start" },
    { id: "de", d: "M860 70 L890 70" },
    { id: "ef", d: "M990 110 L990 330", label: "SolvencyAttested", lx: 980, ly: 225, anchor: "end" },
    { id: "fg", d: "M890 370 L860 370" },
  ],
};

const NARROW_LAYOUT: Layout = {
  w: 360,
  h: 790,
  nodes: [
    { id: "a", x: 4, y: 10, w: 168, h: 96, ...COPY.a, sub: "checkpoint + hash" },
    { id: "b", x: 188, y: 10, w: 168, h: 96, ...COPY.b, sub: "at finalized" },
    { id: "c", x: 70, y: 170, w: 220, h: 72, ...COPY.c },
    { id: "d", x: 70, y: 302, w: 220, h: 72, ...COPY.d },
    { id: "e", x: 70, y: 434, w: 220, h: 72, ...COPY.e },
    { id: "f", x: 70, y: 566, w: 220, h: 72, ...COPY.f },
    { id: "g", x: 70, y: 698, w: 220, h: 72, ...COPY.g },
  ],
  edges: [
    { id: "ac", d: "M88 106 C 88 140, 150 136, 150 170" },
    { id: "bc", d: "M272 106 C 272 140, 210 136, 210 170" },
    { id: "cd", d: "M180 242 L180 302", label: "signed report", lx: 192, ly: 277, anchor: "start" },
    { id: "de", d: "M180 374 L180 434", label: "onReport", lx: 192, ly: 409, anchor: "start" },
    { id: "ef", d: "M180 506 L180 566", label: "SolvencyAttested", lx: 192, ly: 541, anchor: "start" },
    { id: "fg", d: "M180 638 L180 698", label: "re-verified", lx: 192, ly: 673, anchor: "start" },
  ],
};

function Diagram({ layout, className, label }: { layout: Layout; className: string; label: string }) {
  return (
    <svg viewBox={`0 0 ${layout.w} ${layout.h}`} className={className} role="img" aria-label={label}>
      {layout.edges.map((e) => (
        <g key={e.id}>
          <path d={e.d} stroke="rgba(14,15,12,0.12)" strokeWidth="2" fill="none" />
          <path
            data-edge={e.id}
            d={e.d}
            pathLength={1}
            strokeDasharray="1"
            strokeDashoffset="0"
            stroke="#3b1d8a"
            strokeWidth="2.5"
            strokeLinecap="round"
            fill="none"
          />
          {e.label && (
            <text
              data-edge-label={e.id}
              x={e.lx}
              y={e.ly}
              textAnchor={e.anchor}
              className="fill-foreground-muted font-mono"
              fontSize="13"
              fontWeight="600"
            >
              {e.label}
            </text>
          )}
        </g>
      ))}
      {layout.nodes.map((n) => {
        const positive = n.tone === "positive";
        return (
          <g key={n.id} data-node={n.id} style={{ transformBox: "fill-box", transformOrigin: "center" }}>
            <rect
              x={n.x}
              y={n.y}
              width={n.w}
              height={n.h}
              rx="16"
              fill={positive ? "#e6f4ec" : "#ffffff"}
              stroke={positive ? "#1f7a4a" : "rgba(14,15,12,0.14)"}
              strokeWidth={positive ? 2 : 1}
            />
            <text x={n.x + 16} y={n.y + 27} fontSize="14" fontWeight="700" className={positive ? "fill-positive" : "fill-foreground"}>
              {positive ? "✓ " : ""}
              {n.title}
            </text>
            {n.value && (
              <text x={n.x + 16} y={n.y + 56} fontSize="22" fontWeight="900" className="fill-foreground">
                {n.value}
              </text>
            )}
            <text x={n.x + 16} y={n.y + n.h - 16} fontSize="12.5" fontWeight="600" className="fill-foreground-muted">
              {n.sub}
            </text>
          </g>
        );
      })}
    </svg>
  );
}

const TRUST = [
  "Reserve addresses come from the workflow's configuration, never from the gateway.",
  "A figure that does not match what the gateway served is stored as a mismatch, never shown as attested.",
  "Simulated reports are labelled as simulated and refused in live.",
  "Off by default. With CRE off, the gateway behaves exactly as it does without the module.",
];

export function Solvency() {
  const ref = useRef<HTMLElement>(null);

  useMotion(ref, ({ wide }) => {
    const q = gsap.utils.selector(ref);
    const tl = gsap.timeline({
      defaults: { ease: "none", duration: 0.5 },
      scrollTrigger: wide
        ? {
            trigger: q("[data-solvency-pin]")[0],
            start: "top top",
            end: () => "+=" + window.innerHeight * 2,
            pin: true,
            scrub: 0.6,
            anticipatePin: 1,
            invalidateOnRefresh: true,
          }
        : { trigger: q("[data-solvency-stage]")[0], start: "top 80%", end: "bottom 60%", scrub: 0.6 },
    });

    const node = (id: string) => q(`[data-node='${id}']`);
    const edge = (id: string) => q(`[data-edge='${id}']`);
    const label = (id: string) => q(`[data-edge-label='${id}']`);
    const nodeIn = { opacity: 0, y: 10 };
    const nodeTo = { opacity: 1, y: 0, duration: 0.4 };
    const draw = (id: string, at: number) => {
      tl.fromTo(edge(id), { strokeDashoffset: 1 }, { strokeDashoffset: 0, duration: 0.5 }, at);
      tl.fromTo(label(id), { opacity: 0 }, { opacity: 1, duration: 0.3 }, at + 0.2);
    };

    tl.fromTo([...node("a"), ...node("b")], nodeIn, { ...nodeTo, stagger: 0.15 }, 0);
    draw("ac", 0.5);
    draw("bc", 0.6);
    tl.fromTo(node("c"), nodeIn, nodeTo, 1.1);
    draw("cd", 1.6);
    tl.fromTo(node("d"), nodeIn, nodeTo, 2.0);
    draw("de", 2.4);
    tl.fromTo(node("e"), nodeIn, nodeTo, 2.7);
    draw("ef", 3.1);
    tl.fromTo(node("f"), nodeIn, nodeTo, 3.5);
    draw("fg", 3.9);
    tl.fromTo(node("g"), { opacity: 0, scale: 0.85 }, { opacity: 1, scale: 1, duration: 0.4, ease: "back.out(2)" }, 4.3);
    tl.to({}, { duration: 0.3 }, 4.7);

    return () => tl.scrollTrigger?.kill();
  });

  return (
    <section ref={ref} id="solvency" className="section-mint border-y border-border">
      <div data-solvency-pin className="flex min-h-screen items-center py-24 lg:pb-10 lg:pt-20">
        <div className="mx-auto w-full max-w-6xl px-4 sm:px-6">
          <div className="mb-10 grid gap-6 lg:grid-cols-[1.1fr_1fr] lg:items-end">
            <div>
              <p className="eyebrow mb-4 !text-brand-ink">Chainlink CRE · optional</p>
              <h2 className="font-display text-[44px] font-black text-foreground md:text-[52px] xl:text-[56px]">
                Proof the gateway
                <br />
                can <span className="brand-underline">pay what it owes.</span>
              </h2>
            </div>
            <p className="text-[17px] font-semibold leading-[1.5] text-foreground-soft">
              Proof of Reserve shows what is held, not what is owed. A CRE workflow compares
              the ledger&apos;s liabilities with reserves on chain, writes a signed report
              through Chainlink&apos;s forwarder, and the gateway re-verifies it from its own RPC
              before showing it.
            </p>
          </div>

          <div data-solvency-stage>
            <Diagram
              layout={WIDE_LAYOUT}
              className="hidden h-auto w-full md:block"
              label="Ledger liabilities and on-chain reserves flow into the Chainlink CRE workflow, which sends a signed report through the Keystone forwarder to the GatewayAttestations contract; the gateway verifier re-reads it from its own RPC and stores it as attested."
            />
            <Diagram
              layout={NARROW_LAYOUT}
              className="mx-auto block h-auto w-full max-w-[400px] md:hidden"
              label="Ledger liabilities and on-chain reserves flow into the Chainlink CRE workflow, then the Keystone forwarder, the GatewayAttestations contract, the gateway verifier, and an attested record."
            />
            <p className="mt-4 text-[13px] font-semibold text-foreground-muted">
              Figures are from the local end-to-end run (Anvil chain, Chainlink&apos;s Keystone mock
              forwarder).
            </p>
          </div>
        </div>
      </div>

      <div className="mx-auto max-w-6xl px-4 pb-24 sm:px-6">
        <ul data-reveal-stagger className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
          {TRUST.map((t) => (
            <li key={t} data-stagger-child className="card-ring p-5 text-[14px] font-semibold leading-[1.5] text-foreground-soft">
              {t}
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
