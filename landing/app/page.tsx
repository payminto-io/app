"use client";

import Image from "next/image";
import { useEffect, useRef, useState } from "react";
import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { MotionPathPlugin } from "gsap/MotionPathPlugin";

if (typeof window !== "undefined") {
  gsap.registerPlugin(ScrollTrigger, MotionPathPlugin);
}


// ─── Navbar ───────────────────────────────────────────────────────────────────
function Navbar() {
  return (
    <nav className="fixed top-0 left-0 right-0 z-50 border-b border-border bg-background/85 backdrop-blur-xl">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <a href="#" className="flex items-center gap-2.5">
          <span className="grid h-7 w-7 place-items-center rounded-full bg-brand text-[13px] font-black text-[#3b1d8a]">
            P
          </span>
          <span className="text-[16px] font-bold tracking-tight text-foreground">
            payminto
          </span>
        </a>
        <div className="hidden items-center gap-8 md:flex">
          {[
            ["Product", "#card-to-crypto"],
            ["Agents", "#agents"],
            ["Pricing", "#pricing"],
            ["Docs", "#"],
          ].map(([label, href]) => (
            <a
              key={label}
              href={href}
              className="nav-hover rounded-full px-3 py-1.5 text-[14px] font-semibold text-foreground/80 transition-colors hover:text-foreground"
            >
              {label}
            </a>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <a href="#" className="btn-pill btn-secondary !px-4 !py-2 !text-[14px]">
            GitHub
          </a>
          <a href="#setup" className="btn-pill btn-primary !px-5 !py-2 !text-[14px]">
            Self-host
          </a>
        </div>
      </div>
    </nav>
  );
}

// ─── Hero ─────────────────────────────────────────────────────────────────────
// Wise-style: full-width massive text → subtitle → CTAs → real product
// screenshot below. No fake mockups, no floating pills, no ticker animations.
function Hero() {
  return (
    <section className="relative overflow-hidden pt-36 pb-0">
      <div className="mx-auto max-w-6xl px-6" data-reveal-stagger>
        <h1
          data-stagger-child
          className="font-display mb-8 max-w-5xl text-[52px] font-black text-foreground md:text-[88px] lg:text-[120px]"
          style={{ lineHeight: 0.85, letterSpacing: "-0.02em" }}
        >
          The payment processor
          <br />
          you <span className="brand-underline">actually own.</span>
        </h1>

        <div data-stagger-child className="mb-10 flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
          <p className="max-w-lg text-[18px] font-semibold leading-[1.5] text-foreground-soft">
            Self-hosted crypto and card payments. Zero fees. Zero approvals.
            Yours by deployment, not by license.
          </p>
          <div className="flex flex-shrink-0 gap-3">
            <a href="#setup" className="btn-pill btn-primary">
              Deploy now <span aria-hidden>→</span>
            </a>
            <a href="#card-to-crypto" className="btn-pill btn-secondary">
              How it works
            </a>
          </div>
        </div>

        {/* Real product screenshot — not a fake mockup */}
        <div
          data-stagger-child
          className="card-ring-lg hover-tilt relative mt-6 aspect-[16/9] overflow-hidden"
        >
          <Image
            src="/generated/dashboard-mockup.png"
            alt="Payminto merchant dashboard"
            fill
            priority
            className="object-cover"
          />
        </div>

        {/* Stats — inline under the screenshot, no card wrapper */}
        <div
          data-stagger-child
          className="mt-12 mb-16 flex flex-wrap gap-x-16 gap-y-4"
        >
          {[
            { value: "0%", label: "Processing fee" },
            { value: "10 min", label: "From clone to live" },
            { value: "100%", label: "Funds in your wallet" },
          ].map((stat) => (
            <div key={stat.label}>
              <div
                className="font-display text-[40px] font-black text-foreground md:text-[52px]"
                style={{ lineHeight: 0.85 }}
              >
                {stat.value}
              </div>
              <div className="mt-2 text-[13px] font-bold text-foreground-muted">
                {stat.label}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ─── Trust Strip ──────────────────────────────────────────────────────────────
function TrustStrip() {
  const marks = ["AP NEWS", "COINTELEGRAPH", "TECHBULLION", "BENZINGA", "DECRYPT", "THE BLOCK"];
  return (
    <section className="border-y border-border bg-background py-10">
      <div className="mx-auto max-w-6xl px-6">
        <p className="mb-6 text-center text-[11px] font-bold uppercase tracking-[0.2em] text-foreground-muted">
          As covered in
        </p>
        <div className="flex flex-wrap items-center justify-center gap-x-12 gap-y-4">
          {marks.map((m) => (
            <span
              key={m}
              className="text-[14px] font-bold tracking-[0.12em] text-foreground/50 transition-colors hover:text-foreground"
            >
              {m}
            </span>
          ))}
        </div>
      </div>
    </section>
  );
}

// ─── Card to Crypto ───────────────────────────────────────────────────────────
function CardToCrypto() {
  return (
    <section id="card-to-crypto" className="py-28">
      <div className="mx-auto grid max-w-6xl items-center gap-16 px-6 lg:grid-cols-2">
        <div>
          <div className="mb-6 inline-flex rounded-full bg-surface-mint px-4 py-1.5 text-[11px] font-bold uppercase tracking-[0.15em] text-[#3b1d8a]">
            Live onramp
          </div>
          <h2
            className="font-display mb-6 text-[48px] font-black text-foreground md:text-[72px]"
            style={{ lineHeight: 0.85 }}
            data-reveal
          >
            Cards in.
            <br />
            <span className="brand-underline">Crypto out.</span>
          </h2>
          <p className="mb-8 max-w-md text-[18px] font-semibold leading-[1.44] text-foreground-soft">
            Customers pay with Visa, Mastercard, or Apple Pay. You receive
            stablecoins in your own wallet. The onramp partner clears the card.
            Payminto settles to you. No middleman ever holds your money.
          </p>
          <ul className="space-y-3 text-[16px] font-semibold text-foreground">
            {[
              "Card → USDT, USDC, BTC, or ETH",
              "Direct settlement to your wallet — never to us",
              "Single integration, multi-rail under the hood",
            ].map((line) => (
              <li key={line} className="flex items-start gap-3">
                <span className="mt-2 h-2 w-2 flex-shrink-0 rounded-full bg-brand" />
                {line}
              </li>
            ))}
          </ul>
        </div>
        <div className="card-ring-lg hover-tilt relative aspect-[3/2] overflow-hidden">
          <Image
            src="/generated/checkout-screen.png"
            alt="Payminto hosted checkout screen"
            fill
            className="object-cover"
          />
        </div>
      </div>
    </section>
  );
}

// ─── Custody Explained (educational infographic) ──────────────────────────────
function CustodyExplained() {
  return (
    <section className="border-y border-border bg-background py-28">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mb-12 grid items-end gap-8 lg:grid-cols-[1.2fr_1fr]">
          <div>
            <p className="mb-4 text-[12px] font-bold uppercase tracking-[0.18em] text-foreground-muted">
              First time hearing this?
            </p>
            <h2
              className="font-display text-[44px] font-black text-foreground md:text-[72px]"
              style={{ lineHeight: 0.85 }}
              data-reveal
            >
              Where does your
              <br />
              <span className="brand-underline">money actually sit?</span>
            </h2>
          </div>
          <p className="text-[17px] font-semibold leading-[1.5] text-foreground-soft">
            Most payment processors hold your funds in their account for days
            before passing them on. Payminto removes the middle step entirely —
            payments settle straight into a wallet you control. Here&apos;s the
            difference, side by side.
          </p>
        </div>

        <div className="card-ring-lg relative aspect-[2/1] overflow-hidden bg-surface">
          <Image
            src="/generated/infographic-custody.png"
            alt="Custody comparison: traditional processor vs Payminto direct settlement"
            fill
            className="object-cover"
          />
        </div>

        <div className="mt-10 grid gap-8 sm:grid-cols-3">
          {[
            ["Traditional", "T+2 to T+7 settlement. Processor controls your funds in transit. Chargebacks pull from their balance, not yours."],
            ["Payminto", "Real-time settlement on confirmation. Funds land in a wallet you hold the keys to. No third-party balance, ever."],
            ["Why it matters", "If the processor freezes your account, blocks your industry, or goes under — your money is gone. Non-custodial means you're not exposed."],
          ].map(([title, desc]) => (
            <div key={title}>
              <div className="mb-1 text-[15px] font-bold text-foreground">{title}</div>
              <p className="text-[14px] font-semibold leading-[1.5] text-foreground-soft">{desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ─── Architecture (educational infographic) ───────────────────────────────────
function ArchitectureSection() {
  return (
    <section className="bg-surface-mint border-t border-border py-28">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mb-12 max-w-3xl">
          <p className="mb-4 text-[12px] font-bold uppercase tracking-[0.18em] text-[#3b1d8a]">
            What&apos;s actually deploying
          </p>
          <h2
            className="font-display text-[44px] font-black text-foreground md:text-[72px]"
            style={{ lineHeight: 0.85 }}
            data-reveal
          >
            One server.
            <br />
            <span className="brand-underline">Five components.</span>
          </h2>
          <p className="mt-6 max-w-xl text-[17px] font-semibold leading-[1.5] text-foreground-soft">
            When you run the install command, this is what spins up on your VPS.
            Each component talks only to the others — no calls to a Payminto
            cloud, no telemetry, nothing leaves your box.
          </p>
        </div>

        <div className="card-ring-lg relative aspect-[16/9] overflow-hidden bg-surface">
          <Image
            src="/generated/infographic-architecture.png"
            alt="Payminto architecture: API, dashboard, MCP server, block monitor on a self-hosted VPS connecting to a cold wallet"
            fill
            className="object-cover"
          />
        </div>

        <div className="mt-10 grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          {[
            ["API", "Go service, REST + webhooks. The brain that creates invoices, validates payments, and signs payouts."],
            ["Dashboard", "Next.js merchant UI. Manage payments, wallets, sweeps, webhooks, and API keys."],
            ["MCP server", "Native Model Context Protocol bridge. Lets Claude and other agents call your Payminto instance."],
            ["Block monitor", "Per-chain workers that watch BTC / ETH / Base / Tron and reconcile on-chain confirmations."],
          ].map(([title, desc]) => (
            <div key={title} className="card-ring hover-lift p-5">
              <div className="mb-1 text-[15px] font-bold text-foreground">{title}</div>
              <p className="text-[13px] font-semibold leading-[1.5] text-foreground-soft">{desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ─── Setup ────────────────────────────────────────────────────────────────────
function SetupSection() {
  const [copied, setCopied] = useState(false);
  const cmd = "curl -fsSL https://payminto.dev/install.sh | bash";

  function copy() {
    navigator.clipboard.writeText(cmd).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 1800);
    });
  }

  return (
    <section id="setup" className="bg-surface-mint border-b border-border py-28">
      <div className="mx-auto max-w-3xl px-6 text-center">
        <h2
          className="font-display mb-6 text-[48px] font-black text-foreground md:text-[72px]"
          style={{ lineHeight: 0.85 }}
          data-reveal
        >
          One command.
          <br />
          One server. <span className="brand-underline">Yours.</span>
        </h2>
        <p className="mb-12 text-[18px] font-semibold text-foreground-soft">
          Docker, Nginx, and HTTPS configured automatically. No SaaS dashboard
          to log into. No usage limits.
        </p>

        <div className="card-ring overflow-hidden text-left font-mono text-[13px]">
          <div className="flex items-center justify-between border-b border-border bg-surface-soft px-4 py-3">
            <div className="flex gap-1.5">
              <span className="h-2.5 w-2.5 rounded-full bg-[#ff5f57]" />
              <span className="h-2.5 w-2.5 rounded-full bg-[#ffbd2e]" />
              <span className="h-2.5 w-2.5 rounded-full bg-[#28c840]" />
            </div>
            <span className="text-[11px] font-semibold text-foreground-muted">vps ~ root</span>
            <button
              onClick={copy}
              className="rounded-full bg-surface-mint px-3 py-1 text-[11px] font-bold text-[#3b1d8a] transition-colors hover:bg-brand-hover"
            >
              {copied ? "Copied ✓" : "Copy"}
            </button>
          </div>
          <div className="bg-[#0e0f0c] px-5 py-5 text-[#fafaf7]">
            <div>
              <span className="text-brand">$ </span>
              <span>{cmd}</span>
            </div>
            <div className="mt-3 space-y-1 text-white/60">
              <div className="type-line">▸ Pulling payminto/core:0.1.0...</div>
              <div className="type-line" style={{ animationDelay: "0.4s" }}>
                ▸ Provisioning Postgres + Redis...
              </div>
              <div className="type-line" style={{ animationDelay: "1.0s" }}>
                ▸ Issuing TLS via Let&apos;s Encrypt...
              </div>
              <div className="type-line text-brand" style={{ animationDelay: "1.6s" }}>
                ✓ Ready at https://pay.yourdomain.com
              </div>
            </div>
          </div>
        </div>

        <div className="mt-6 flex flex-wrap justify-center gap-x-5 gap-y-2 text-[13px] font-semibold text-foreground-muted">
          <span>Ubuntu 20.04+</span>
          <span>·</span>
          <span>Auto-installed Docker</span>
          <span>·</span>
          <span>Let&apos;s Encrypt TLS</span>
        </div>
      </div>
    </section>
  );
}

// ─── Features Grid ────────────────────────────────────────────────────────────
const FEATURES = [
  {
    icon: "/generated/icon-custody.png",
    title: "Control & custody",
    desc: "Funds settle directly into wallets you control. Payminto never holds a key, never touches a balance.",
  },
  {
    icon: "/generated/icon-payments.png",
    title: "Payment options",
    desc: "Cards via onramp. BTC, ETH, USDT, USDC, MATIC, TRX natively. One integration covers all rails.",
  },
  {
    icon: "/generated/icon-auth.png",
    title: "Scoped API keys",
    desc: "Per-key permission scopes. Rotate without downtime. Full audit log of every authenticated call.",
  },
  {
    icon: "/generated/icon-funds.png",
    title: "SmartSweep",
    desc: "Auto-consolidates incoming deposits to a cold address you set. Optimized batching, minimal gas.",
  },
  {
    icon: "/generated/icon-compliance.png",
    title: "Audit trail",
    desc: "Append-only ledger, double-entry accounting, CSV export. Built for the books, not just the dashboard.",
  },
  {
    icon: "/generated/icon-automation.png",
    title: "Agent-ready",
    desc: "Native MCP server. Claude and other agents create invoices, check balances, and trigger payouts.",
  },
];

function FeaturesGrid() {
  return (
    <section className="py-28">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mb-16 max-w-3xl">
          <p className="mb-4 text-[12px] font-bold uppercase tracking-[0.18em] text-foreground-muted">
            What you get
          </p>
          <h2
            className="font-display text-[48px] font-black text-foreground md:text-[84px]"
            style={{ lineHeight: 0.85 }}
            data-reveal
          >
            Everything PayPal
            <br />
            won&apos;t let you build.
          </h2>
        </div>

        <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f, i) => (
            <div
              key={f.title}
              className={`card-ring hover-lift p-8 ${
                i === 0 ? "sm:col-span-2 sm:row-span-1 bg-surface-mint" : ""
              }`}
            >
              <div className="icon-pop mb-6 grid h-14 w-14 place-items-center rounded-2xl bg-brand">
                <Image src={f.icon} alt="" width={36} height={36} className="opacity-90" />
              </div>
              <h3 className="mb-2 text-[22px] font-bold text-foreground" style={{ letterSpacing: "-0.01em" }}>
                {f.title}
              </h3>
              <p className="text-[16px] font-semibold leading-[1.44] text-foreground-soft">
                {f.desc}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ─── Payment Flow — pinned scroll-scrubbed coin on SVG path ───────────────────
const FLOW_STEPS = [
  {
    num: "01",
    title: "Capture",
    desc: "Customer pays via card or crypto on your hosted checkout. Address generated, intent locked.",
  },
  {
    num: "02",
    title: "Verify",
    desc: "Block monitors confirm on-chain. Webhooks fire to your app the moment finality is reached.",
  },
  {
    num: "03",
    title: "Sweep",
    desc: "SmartSweep batches funds to your cold storage address. Gas-optimized, fully autonomous.",
  },
];

function FlowDiagram() {
  const sectionRef = useRef<HTMLElement>(null);
  const pathRef = useRef<SVGPathElement>(null);
  const coinRef = useRef<SVGGElement>(null);
  const stepRefs = useRef<(HTMLDivElement | null)[]>([]);
  const nodeRefs = useRef<(SVGCircleElement | null)[]>([]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const path = pathRef.current;
    const coin = coinRef.current;
    const section = sectionRef.current;
    if (!path || !coin || !section) return;

    const length = path.getTotalLength();
    gsap.set(path, { strokeDasharray: length, strokeDashoffset: reduce ? 0 : length });
    gsap.set(coin, { autoAlpha: reduce ? 1 : 0 });
    if (reduce) {
      stepRefs.current.forEach((s) => s && gsap.set(s, { opacity: 1 }));
      nodeRefs.current.forEach((n) => n && gsap.set(n, { scale: 1, transformOrigin: "center" }));
      return;
    }

    const ctx = gsap.context(() => {
      const tl = gsap.timeline({
        scrollTrigger: {
          trigger: section,
          start: "top top",
          end: "+=2400",
          pin: true,
          scrub: 0.8,
          anticipatePin: 1,
        },
      });

      tl.to(path, { strokeDashoffset: 0, ease: "none", duration: 3 }, 0);
      tl.to(coin, { autoAlpha: 1, duration: 0.2 }, 0.1);
      tl.to(
        coin,
        {
          motionPath: { path: path, align: path, alignOrigin: [0.5, 0.5], autoRotate: false },
          ease: "none",
          duration: 3,
        },
        0
      );

      stepRefs.current.forEach((card, i) => {
        if (!card) return;
        const at = 0.4 + i * 0.9;
        tl.to(card, { opacity: 1, y: 0, duration: 0.4 }, at);
        const node = nodeRefs.current[i];
        if (node) {
          tl.fromTo(
            node,
            { scale: 0.6, transformOrigin: "center" },
            { scale: 1.4, duration: 0.3, ease: "back.out(2)" },
            at
          );
          tl.to(node, { scale: 1, duration: 0.4 }, at + 0.3);
        }
      });
    }, section);

    return () => ctx.revert();
  }, []);

  return (
    <section
      ref={sectionRef}
      className="relative border-y border-border bg-background overflow-hidden"
      style={{ minHeight: "100vh" }}
    >
      <div className="relative h-screen w-full">
        <div className="absolute inset-0 grid-bg opacity-50" aria-hidden />
        <div className="relative mx-auto flex h-full max-w-6xl flex-col px-6 pt-28 pb-12">
          <div className="mb-10 max-w-3xl">
            <p className="mb-4 text-[12px] font-bold uppercase tracking-[0.18em] text-foreground-muted">
              How it flows
            </p>
            <h2
              className="font-display text-[48px] font-black text-foreground md:text-[84px]"
              style={{ lineHeight: 0.85 }}
            >
              Checkout to
              <br />
              <span className="brand-underline">cold storage,</span>
              <br />
              one motion.
            </h2>
          </div>

          <div className="relative flex-1">
            <svg
              viewBox="0 0 1200 340"
              fill="none"
              className="absolute inset-x-0 top-1/2 h-auto w-full -translate-y-1/2"
              preserveAspectRatio="xMidYMid meet"
              aria-hidden
            >
              <defs>
                <linearGradient id="flowGrad" x1="0" y1="0" x2="1200" y2="0" gradientUnits="userSpaceOnUse">
                  <stop offset="0%" stopColor="#3b1d8a" />
                  <stop offset="50%" stopColor="#a78bfa" />
                  <stop offset="100%" stopColor="#3b1d8a" />
                </linearGradient>
                <radialGradient id="coinGrad" cx="50%" cy="50%" r="50%">
                  <stop offset="0%" stopColor="#ffffff" />
                  <stop offset="60%" stopColor="#c4b5fd" />
                  <stop offset="100%" stopColor="#a78bfa" />
                </radialGradient>
                <filter id="coinGlow" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="6" result="b" />
                  <feMerge>
                    <feMergeNode in="b" />
                    <feMergeNode in="SourceGraphic" />
                  </feMerge>
                </filter>
              </defs>

              <path
                ref={pathRef}
                d="M 80 240 C 260 240, 360 60, 600 120 S 940 260, 1120 110"
                stroke="url(#flowGrad)"
                strokeWidth="3"
                strokeLinecap="round"
              />

              {[
                { cx: 80, cy: 240, label: "01" },
                { cx: 600, cy: 120, label: "02" },
                { cx: 1120, cy: 110, label: "03" },
              ].map((n, i) => (
                <g key={n.label}>
                  <circle
                    ref={(el) => {
                      nodeRefs.current[i] = el;
                    }}
                    cx={n.cx}
                    cy={n.cy}
                    r="16"
                    fill="#fafaf7"
                    stroke="#3b1d8a"
                    strokeWidth="2.5"
                  />
                  <text
                    x={n.cx}
                    y={n.cy + 4}
                    textAnchor="middle"
                    fontSize="11"
                    fontFamily="ui-monospace, monospace"
                    fontWeight="700"
                    fill="#3b1d8a"
                  >
                    {n.label}
                  </text>
                </g>
              ))}

              <g ref={coinRef} filter="url(#coinGlow)">
                <circle r="20" fill="url(#coinGrad)" stroke="#3b1d8a" strokeWidth="2" />
                <text
                  textAnchor="middle"
                  y="5"
                  fontSize="17"
                  fontWeight="900"
                  fill="#3b1d8a"
                  fontFamily="system-ui"
                >
                  $
                </text>
              </g>
            </svg>
          </div>

          <div className="mt-auto grid gap-5 pt-8 sm:grid-cols-3">
            {FLOW_STEPS.map((step, i) => (
              <div
                key={step.num}
                ref={(el) => {
                  stepRefs.current[i] = el;
                }}
                className="card-ring p-6 opacity-30"
                style={{ transform: "translateY(8px)" }}
              >
                <div className="mb-2 font-mono text-[11px] font-bold text-[#3b1d8a]">
                  {step.num}
                </div>
                <div className="mb-1 text-[18px] font-bold text-foreground">{step.title}</div>
                <p className="text-[14px] font-semibold leading-[1.5] text-foreground-soft">
                  {step.desc}
                </p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

// ─── AI Agents Section ────────────────────────────────────────────────────────
function AgentsSection() {
  return (
    <section id="agents" className="section-alt border-y border-border py-28">
      <div className="mx-auto max-w-5xl px-6">
        <div className="mb-16 text-center">
          <p className="mb-4 text-[12px] font-bold uppercase tracking-[0.18em] text-foreground-muted">
            For AI agents
          </p>
          <h2
            className="font-display mb-6 text-[48px] font-black text-foreground md:text-[84px]"
            style={{ lineHeight: 0.85 }}
            data-reveal
          >
            Built for agents
            <br />
            that <span className="brand-underline">move money.</span>
          </h2>
          <p className="mx-auto max-w-2xl text-[18px] font-semibold text-foreground-soft">
            Payminto ships with a native MCP server. Agents create invoices,
            check balances, and trigger payouts — without a human in the loop.
          </p>
        </div>

        <div className="grid gap-5 md:grid-cols-2">
          <div className="card-ring hover-lift p-8">
            <div className="mb-5 inline-flex rounded-full bg-[#d03238]/10 px-3 py-1 text-[11px] font-bold uppercase tracking-wide text-[#d03238]">
              Hosted gateways
            </div>
            <ul className="space-y-3 text-[15px] font-semibold text-foreground-soft">
              {[
                "Manual KYB before any API call",
                "2.9% + $0.30 per card transaction",
                "Human approval required for payouts",
                "Funds sit in the processor's account",
                "Polling-only — webhooks behind paywall",
              ].map((item) => (
                <li key={item} className="flex items-start gap-3">
                  <span className="mt-1 text-[#d03238]">×</span>
                  {item}
                </li>
              ))}
            </ul>
          </div>

          <div className="card-ring hover-lift p-8 bg-surface-mint">
            <div className="mb-5 inline-flex rounded-full bg-[#3b1d8a] px-3 py-1 text-[11px] font-bold uppercase tracking-wide text-brand">
              Payminto
            </div>
            <ul className="space-y-3 text-[15px] font-semibold text-foreground">
              {[
                "API key in, payments out — no KYB",
                "0% processing fee, gas only",
                "Fully autonomous payouts via MCP",
                "Funds settle to your wallet directly",
                "Push webhooks the moment a tx confirms",
              ].map((item) => (
                <li key={item} className="flex items-start gap-3">
                  <span className="mt-1 text-[#3b1d8a]">✓</span>
                  {item}
                </li>
              ))}
            </ul>
          </div>
        </div>

        <div className="card-ring mt-6 overflow-hidden font-mono text-[13px]">
          <div className="border-b border-border bg-surface-soft px-4 py-2.5 text-[11px] font-bold text-foreground-muted">
            agent.ts — Claude using Payminto MCP
          </div>
          <div className="bg-[#0e0f0c] px-5 py-5 leading-relaxed text-[#fafaf7]">
            <div className="text-white/50">{"// expose payminto tools to Claude"}</div>
            <div>
              <span className="text-brand">const</span>{" "}
              <span className="text-white">invoice</span>{" "}
              <span className="text-white/50">=</span>{" "}
              <span className="text-brand">await</span>{" "}
              <span className="text-brand-hover">payminto</span>
              <span className="text-white/50">.createInvoice({"{"}</span>
            </div>
            <div className="pl-6">
              amount<span className="text-white/50">:</span>{" "}
              <span className="text-[#ffd11a]">49.99</span>
              <span className="text-white/50">,</span>
            </div>
            <div className="pl-6">
              currency<span className="text-white/50">:</span>{" "}
              <span className="text-brand-hover">&quot;USDC&quot;</span>
              <span className="text-white/50">,</span>
            </div>
            <div className="pl-6">
              chain<span className="text-white/50">:</span>{" "}
              <span className="text-brand-hover">&quot;base&quot;</span>
              <span className="text-white/50">,</span>
            </div>
            <div className="pl-6">
              memo<span className="text-white/50">:</span>{" "}
              <span className="text-brand-hover">&quot;Pro plan — April 2026&quot;</span>
              <span className="text-white/50">,</span>
            </div>
            <div className="text-white/50">{"});"}</div>
          </div>
        </div>
      </div>
    </section>
  );
}

// ─── Dashboard Showcase ───────────────────────────────────────────────────────
function DashboardShowcase() {
  return (
    <section className="py-28">
      <div className="mx-auto max-w-6xl px-6">
        <div className="mb-12 max-w-3xl">
          <p className="mb-4 text-[12px] font-bold uppercase tracking-[0.18em] text-foreground-muted">
            The dashboard
          </p>
          <h2
            className="font-display text-[48px] font-black text-foreground md:text-[84px]"
            style={{ lineHeight: 0.85 }}
            data-reveal
          >
            See every dollar.
            <br />
            <span className="brand-underline">Move every coin.</span>
          </h2>
        </div>
        <div className="card-ring-lg hover-tilt relative aspect-[3/2] overflow-hidden">
          <Image
            src="/generated/dashboard-mockup.png"
            alt="Payminto merchant dashboard"
            fill
            className="object-cover"
          />
        </div>
        <div className="mt-12 grid gap-10 sm:grid-cols-2 lg:grid-cols-4">
          {[
            ["Real-time volume", "Live charts of payment volume across every chain."],
            ["Transaction stream", "Every confirmation, every webhook, every retry."],
            ["Wallet balances", "Hot, warm, cold — split by asset and network."],
            ["Withdrawal queue", "Approve and broadcast multi-sig payouts in one click."],
          ].map(([title, desc]) => (
            <div key={title}>
              <div className="mb-1 text-[16px] font-bold text-foreground">{title}</div>
              <p className="text-[14px] font-semibold leading-[1.5] text-foreground-soft">{desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ─── Mobile App ───────────────────────────────────────────────────────────────
function MobileApp() {
  return (
    <section className="section-alt border-y border-border py-28">
      <div className="mx-auto grid max-w-6xl items-center gap-16 px-6 lg:grid-cols-2">
        <div className="relative mx-auto aspect-[3/4] w-full max-w-sm">
          <Image
            src="/generated/mobile-app.png"
            alt="Payminto mobile app"
            fill
            className="object-contain"
          />
        </div>
        <div>
          <p className="mb-4 text-[12px] font-bold uppercase tracking-[0.18em] text-foreground-muted">
            Mobile companion
          </p>
          <h2
            className="font-display mb-6 text-[48px] font-black text-foreground md:text-[72px]"
            style={{ lineHeight: 0.85 }}
            data-reveal
          >
            Approve payouts
            <br />
            from <span className="brand-underline">anywhere.</span>
          </h2>
          <p className="mb-8 max-w-md text-[18px] font-semibold leading-[1.44] text-foreground-soft">
            The Payminto mobile app pairs with your self-hosted server over an
            end-to-end encrypted channel. Approve withdrawals, monitor balances,
            get push alerts on every transaction.
          </p>
          <ul className="space-y-4">
            {[
              ["Push notifications", "Get pinged the moment a payment confirms."],
              ["Multi-sig approval", "Co-sign payouts with biometric unlock."],
              ["Pair via QR", "End-to-end encrypted, no cloud account."],
            ].map(([title, desc]) => (
              <li key={title} className="flex gap-4">
                <span className="mt-2 h-2 w-2 flex-shrink-0 rounded-full bg-brand" />
                <div>
                  <div className="text-[16px] font-bold text-foreground">{title}</div>
                  <div className="text-[14px] font-semibold text-foreground-soft">{desc}</div>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}

// ─── Supported Chains ─────────────────────────────────────────────────────────
const COINS = [
  { symbol: "BTC", name: "Bitcoin", color: "#f7931a" },
  { symbol: "ETH", name: "Ethereum", color: "#627eea" },
  { symbol: "USDT", name: "Tether", color: "#26a17b" },
  { symbol: "USDC", name: "USD Coin", color: "#2775ca" },
  { symbol: "TRX", name: "TRON", color: "#ef0027" },
  { symbol: "BASE", name: "Base", color: "#0052ff" },
];

function SupportedChains() {
  return (
    <section className="py-24">
      <div className="mx-auto max-w-5xl px-6 text-center">
        <p className="mb-4 text-[12px] font-bold uppercase tracking-[0.18em] text-foreground-muted">
          Multi-chain
        </p>
        <h2
          className="font-display mb-12 text-[40px] font-black text-foreground md:text-[60px]"
          style={{ lineHeight: 0.85 }}
          data-reveal
        >
          Six chains today.
          <br />
          <span className="brand-underline">More by config.</span>
        </h2>

        <div className="flex flex-wrap justify-center gap-3">
          {COINS.map((coin) => (
            <div
              key={coin.symbol}
              className="card-ring hover-chip flex items-center gap-3 px-5 py-3"
            >
              <div
                className="grid h-9 w-9 place-items-center rounded-full text-[12px] font-black text-white"
                style={{ backgroundColor: coin.color }}
              >
                {coin.symbol.slice(0, 1)}
              </div>
              <div className="text-left">
                <div className="text-[14px] font-bold text-foreground">{coin.symbol}</div>
                <div className="text-[11px] font-semibold text-foreground-muted">{coin.name}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ─── Testimonial ──────────────────────────────────────────────────────────────
function Testimonial() {
  return (
    <section className="py-28">
      <div className="mx-auto max-w-4xl px-6">
        <div className="card-ring-lg hover-tilt bg-surface-mint p-12 text-center md:p-16">
          <div className="mb-6 font-display text-[80px] font-black leading-none text-[#3b1d8a]">
            &ldquo;
          </div>
          <p
            className="font-display mb-10 text-[28px] font-black leading-[0.95] text-foreground md:text-[42px]"
          >
            We replaced a $9k/month payment stack
            <br />
            with a single VPS running Payminto.
            <br />
            Settlement is faster, the fees are gone,
            <br />
            and the keys never leave our infra.
          </p>
          <div className="flex items-center justify-center gap-4">
            <div className="relative h-14 w-14 overflow-hidden rounded-full ring-4 ring-brand">
              <Image
                src="/generated/testimonial-avatar.png"
                alt="Maya Chen"
                fill
                className="object-cover"
              />
            </div>
            <div className="text-left">
              <div className="text-[16px] font-bold text-foreground">Maya Chen</div>
              <div className="text-[13px] font-semibold text-foreground-soft">
                Head of Payments, Northwind AI
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}

// ─── FAQ ──────────────────────────────────────────────────────────────────────
const FAQS = [
  {
    q: "Is Payminto really 0% fees?",
    a: "Yes. Payminto is free, open-source software you run yourself. You pay only the underlying blockchain gas — which goes to validators, not us. There is no platform fee, no take rate, no processing margin.",
  },
  {
    q: "Do I need KYC or KYB to use it?",
    a: "No. Because Payminto is software you self-host, there is no central party to verify you. You own the server, the keys, and the funds. Compliance with your local jurisdiction is your responsibility.",
  },
  {
    q: "How does SmartSweep work?",
    a: "A background worker monitors every deposit address, batches incoming funds, and sweeps them to a cold storage address you configure. Batching is gas-optimized — you set the threshold, it takes care of the rest.",
  },
  {
    q: "Can AI agents really use this autonomously?",
    a: "Yes. The bundled MCP server exposes payment tools (createInvoice, checkBalance, listPayments, triggerSweep, requestPayout) to Claude and any MCP-compatible agent. No human approval is required at the protocol level.",
  },
  {
    q: "What if my server crashes mid-payment?",
    a: "Funds are non-custodial and live on-chain — they cannot be lost in a server crash. When the server comes back, block monitors re-scan from the last confirmed height and reconcile any in-flight payments automatically.",
  },
];

function FAQ() {
  const [open, setOpen] = useState<number | null>(0);
  return (
    <section className="bg-background border-y border-border py-28">
      <div className="mx-auto max-w-3xl px-6">
        <h2
          className="font-display mb-12 text-center text-[48px] font-black text-foreground md:text-[72px]"
          style={{ lineHeight: 0.85 }}
          data-reveal
        >
          Questions,
          <br />
          <span className="brand-underline">answered.</span>
        </h2>
        <div className="space-y-3">
          {FAQS.map((faq, i) => (
            <div key={i} className="card-ring overflow-hidden">
              <button
                onClick={() => setOpen(open === i ? null : i)}
                className="hover-faq flex w-full items-center justify-between px-6 py-5 text-left text-[17px] font-bold text-foreground"
              >
                {faq.q}
                <span
                  className={`ml-4 grid h-8 w-8 place-items-center rounded-full bg-surface-mint text-[#3b1d8a] transition-transform duration-300 ${
                    open === i ? "rotate-45" : ""
                  }`}
                >
                  +
                </span>
              </button>
              {open === i && (
                <div className="border-t border-border px-6 py-5 text-[15px] font-semibold leading-[1.6] text-foreground-soft">
                  {faq.a}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

// ─── CTA Banner ───────────────────────────────────────────────────────────────
function CTABanner() {
  return (
    <section className="relative overflow-hidden py-28">
      <div className="mesh-bg opacity-80" aria-hidden />
      <div className="absolute inset-0 grid-bg opacity-40" aria-hidden />
      <div className="relative mx-auto max-w-4xl px-6 text-center">
        <h2
          className="font-display mb-8 text-[56px] font-black text-foreground md:text-[112px]"
          style={{ lineHeight: 0.85 }}
        >
          Stop renting your
          <br />
          <span className="brand-underline">payment stack.</span>
        </h2>
        <p className="mx-auto mb-10 max-w-xl text-[18px] font-semibold text-foreground-soft">
          Deploy Payminto in 10 minutes. Own your processor for as long as your
          server runs.
        </p>
        <div className="flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a href="#setup" className="btn-pill btn-primary !text-[18px] !px-7 !py-4">
            Self-host now →
          </a>
          <a href="#" className="btn-pill btn-secondary !text-[18px] !px-7 !py-4">
            Read the docs
          </a>
        </div>
      </div>
    </section>
  );
}

// ─── Footer ───────────────────────────────────────────────────────────────────
function Footer() {
  const cols: Record<string, string[]> = {
    Product: ["Features", "Pricing", "Changelog", "Roadmap"],
    Developers: ["Documentation", "API reference", "MCP guide", "GitHub"],
    Company: ["About", "Blog", "Press", "Contact"],
    Legal: ["Privacy", "Terms", "Security"],
  };

  return (
    <footer className="border-t border-border bg-[#0e0f0c] py-20 text-[#fafaf7]">
      <div className="mx-auto max-w-6xl px-6">
        <div className="grid grid-cols-2 gap-10 md:grid-cols-6">
          <div className="col-span-2">
            <a href="#" className="mb-5 flex items-center gap-2.5">
              <span className="grid h-8 w-8 place-items-center rounded-full bg-brand text-[14px] font-black text-[#3b1d8a]">
                P
              </span>
              <span className="text-[18px] font-bold text-white">payminto</span>
            </a>
            <p className="mb-6 max-w-xs text-[14px] font-semibold leading-[1.5] text-white/60">
              The payment processor you actually own. Self-hosted, non-custodial,
              and built for the agent era.
            </p>
            <div className="flex gap-2">
              {["GitHub", "X", "Discord"].map((s) => (
                <a
                  key={s}
                  href="#"
                  className="btn-pill hover-social bg-white/5 !px-4 !py-2 !text-[13px] text-white/80"
                >
                  {s}
                </a>
              ))}
            </div>
          </div>

          {Object.entries(cols).map(([section, items]) => (
            <div key={section}>
              <h4 className="mb-4 text-[11px] font-bold uppercase tracking-[0.15em] text-white/50">
                {section}
              </h4>
              <ul className="space-y-2.5">
                {items.map((item) => (
                  <li key={item}>
                    <a
                      href="#"
                      className="text-[14px] font-semibold text-white/80 transition-colors hover:text-brand"
                    >
                      {item}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-16 flex flex-col items-center justify-between gap-3 border-t border-white/10 pt-8 sm:flex-row">
          <p className="text-[12px] font-semibold text-white/50">
            © 2026 Payminto. Open source under MIT License.
          </p>
          <p className="text-[12px] font-semibold text-white/50">
            Self-hosted · Non-custodial · No KYC
          </p>
        </div>
      </div>
    </footer>
  );
}

// ─── Page ─────────────────────────────────────────────────────────────────────
//
// Section grouping (color zones — 7 transitions across 16 sections instead of
// alternating every section):
//
//   ZONE A — Light canvas (intro + first promise)
//     1. Hero
//     2. Trust strip
//     3. Card-to-Crypto
//     4. Custody explained ← educational infographic
//
//   ZONE B — Lavender mist (architecture + setup)
//     5. Architecture explained ← educational infographic
//     6. Setup
//
//   ZONE C — Light canvas (the product)
//     7. Features grid
//     8. Flow diagram (pinned scroll moment)
//     9. Dashboard showcase
//
//   ZONE D — Section alt (warm gray) — agents + mobile
//    10. AI Agents
//    11. Mobile app
//
//   ZONE E — Light canvas (proof + conversion)
//    12. Supported chains
//    13. Testimonial
//    14. FAQ
//    15. CTA banner
//
//   ZONE F — Dark
//    16. Footer
//
export default function Home() {
  return (
    <main className="min-h-screen bg-background text-foreground">
      <Navbar />
      <Hero />
      <TrustStrip />
      <CardToCrypto />
      <CustodyExplained />
      <ArchitectureSection />
      <SetupSection />
      <FeaturesGrid />
      <FlowDiagram />
      <DashboardShowcase />
      <AgentsSection />
      <MobileApp />
      <SupportedChains />
      <Testimonial />
      <FAQ />
      <CTABanner />
      <Footer />
    </main>
  );
}
