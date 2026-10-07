import Image from "next/image";
import { Hero } from "./sections/hero";
import { DemoVideo } from "./sections/demo-video";
import { Flow } from "./sections/flow";
import { Solvency } from "./sections/solvency";
import { Solana } from "./sections/solana";
import { CopyButton } from "./sections/copy-button";
import { LINKS } from "./sections/links";

// Section order: hero, market, flow (pinned), solvency (pinned), solana, screens,
// features, status, self-host, FAQ, CTA, footer. Motion rules: docs/MOTION.md.

function Navbar() {
  return (
    <nav className="fixed inset-x-0 top-0 z-50 border-b border-border bg-background/85 backdrop-blur-xl">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3.5 sm:px-6">
        <a href="#top" className="flex items-center gap-2.5 rounded-full focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-brand-ink">
          <Image src="/brand/mark.svg" alt="" width={28} height={28} />
          <span className="text-[16px] font-bold tracking-tight text-foreground">payminto</span>
        </a>
        <div className="hidden items-center gap-6 md:flex">
          {[
            ["Demo", "#demo"],
            ["How it flows", "#flow"],
            ["Solvency", "#solvency"],
            ["Solana", "#solana"],
            ["Self-host", "#self-host"],
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
          <a href={LINKS.source} className="btn-pill btn-secondary !px-4 !py-2 !text-[14px]">
            GitHub
          </a>
          <a href={LINKS.app} className="btn-pill btn-primary !px-4 !py-2 !text-[14px]">
            Open the app
          </a>
        </div>
      </div>
    </nav>
  );
}

const MARKET = [
  {
    value: "~$390B",
    label: "genuine stablecoin payments in the 12 months to September 2025",
    source: "a16z State of Crypto 2025",
    href: "https://a16zcrypto.com/posts/article/state-of-crypto-report-2025/",
  },
  {
    value: "$226B",
    label: "of it business to business, up 733% year on year",
    source: "a16z State of Crypto 2025",
    href: "https://a16zcrypto.com/posts/article/state-of-crypto-report-2025/",
  },
  {
    value: "32.6%",
    label: "of weekly adjusted stablecoin transfer volume ran on Solana in April 2026",
    source: "bex.co",
    href: "https://bex.co/blog/2026/04/03/solana-650b-stablecoin-volume-record-svm-settlement-layer",
  },
  {
    value: "2",
    label: "crypto connectors in Hyperswitch, the open-source orchestrator at scale; one shut on 2026-03-31",
    source: "Hyperswitch docs",
    href: "https://docs.hyperswitch.io/integrations/connectors-integrations/payment-processor-capabilities/payment-methods-setup/crypto",
  },
];

function Market() {
  return (
    <section className="section-alt border-y border-border py-24 md:py-28">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="mb-14 grid gap-6 lg:grid-cols-[1.1fr_1fr] lg:items-end">
          <div>
            <p className="eyebrow mb-4">Why this exists</p>
            <h2 className="font-display text-[44px] font-black text-foreground md:text-[52px] xl:text-[56px]" data-reveal>
              Stablecoins became a rail.
              <br />
              <span className="brand-underline">Open gateways did not follow.</span>
            </h2>
          </div>
          <p className="text-[17px] font-semibold leading-[1.5] text-foreground-soft">
            Merchants who want stablecoins next to cards choose between closed, custodial
            processors and open-source tools that cover one rail. And a merchant who lets a
            gateway hold money has to trust its balance sheet.
          </p>
        </div>
        <ul data-reveal-stagger className="grid gap-x-8 gap-y-10 sm:grid-cols-2 lg:grid-cols-4">
          {MARKET.map((m) => (
            <li key={m.label} data-stagger-child className="border-t-2 border-foreground pt-5">
              <div className="font-display text-[48px] font-black text-foreground md:text-[56px]">{m.value}</div>
              <p className="mt-3 text-[15px] font-semibold leading-[1.45] text-foreground-soft">{m.label}</p>
              <a href={m.href} className="link-inline mt-3 inline-block text-[13px]">
                Source: {m.source}
              </a>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}

function Screens() {
  return (
    <section data-parallax-scope className="relative overflow-hidden py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="mb-14 grid gap-6 lg:grid-cols-[1.1fr_1fr] lg:items-end">
          <div>
            <p className="eyebrow mb-4">The product</p>
            <h2 className="font-display text-[44px] font-black text-foreground md:text-[52px] xl:text-[56px]" data-reveal>
              Real screens,
              <br />
              <span className="brand-underline">not renders.</span>
            </h2>
          </div>
          <p className="text-[17px] font-semibold leading-[1.5] text-foreground-soft">
            A payment link with its checkout preview built by the server, so the fee shown is
            the fee charged, and the hosted checkout a payer sees once a USDC payment is final.
            Development previews with sample data.
          </p>
        </div>
        <div className="relative pb-16 md:pb-24">
          <figure className="card-ring-lg overflow-hidden md:mr-[18%]" data-reveal>
            <Image
              src="/screens/link-detail.png"
              alt="Payment link detail: item, payment methods with the fee rule version, and a live checkout preview"
              width={1440}
              height={900}
              sizes="(min-width: 1152px) 900px, 100vw"
              className="h-auto w-full"
            />
          </figure>
          <div data-parallax="-140" className="absolute -bottom-2 right-0 w-[40%] max-w-[300px] md:bottom-0">
            <figure className="overflow-hidden rounded-[22px] bg-surface shadow-float ring-1 ring-border">
              <Image
                src="/screens/checkout-paid.png"
                alt="Hosted checkout showing a paid USDC payment on Solana with received and final steps"
                width={780}
                height={1120}
                sizes="300px"
                className="h-auto w-full"
              />
            </figure>
          </div>
        </div>
        <div className="flex flex-wrap gap-3">
          <a href={LINKS.app} className="btn-pill btn-primary">
            app.payminto.io <span aria-hidden>→</span>
          </a>
          <a href={LINKS.checkout} className="btn-pill btn-secondary">
            checkout.payminto.io
          </a>
        </div>
      </div>
    </section>
  );
}

const FEATURES = [
  ["One ledger for every rail", "Card and stablecoin payments land as the same double-entry lines. Balances are derived, never stored, and the application role cannot rewrite history."],
  ["Fees you can audit", "Versioned fee rules, never edited in place. Every payment records the rule version it paid."],
  ["A payment switch", "Intents and attempts, idempotency keys, exclusive claims so a capture is never sent twice, and a reconciler that resolves unknown outcomes from the provider."],
  ["Payment links", "A six-step builder with a checkout preview built by the server, unguessable short links and QR codes."],
  ["Live and test isolation", "One environment per process. Live refuses to boot with development keys, mock providers, test databases or testnets."],
  ["RPC as evidence", "Live needs two distinct RPC providers per chain. A deposit is dropped only when both agree."],
] as const;

function Features() {
  return (
    <section className="section-alt border-y border-border py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <p className="eyebrow mb-4">What you get</p>
        <h2 className="font-display mb-14 max-w-3xl text-[44px] font-black text-foreground md:text-[64px]" data-reveal>
          The parts a gateway
          <br />
          <span className="brand-underline">gets wrong first.</span>
        </h2>
        <ul data-reveal-stagger className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map(([title, desc], i) => (
            <li key={title} data-stagger-child className="card-ring hover-lift p-7">
              <div className="mb-5 font-mono text-[12px] font-bold text-brand-ink">0{i + 1}</div>
              <h3 className="mb-2 text-[21px] font-bold text-foreground">{title}</h3>
              <p className="text-[15px] font-semibold leading-[1.5] text-foreground-soft">{desc}</p>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}

const BUILT = [
  "Double-entry ledger and versioned fee rules",
  "Live and test isolation",
  "Payment switch",
  "Payment links and the six-step builder",
  "Hosted checkout",
  "USDC and USDT on Solana",
  "Chainlink CRE contract, module and solvency workflow",
  "Design system and merchant dashboard",
];
const BRANCHES = [
  "Custody providers: self-custody fence, BitGo next",
  "Kuberpays (card) and Payvang (UPI) connectors",
  "Routing across rails",
  "Payment links on the switch with every method",
  "NOWNodes RPC preset",
  "One-command compose and CI",
];

function Status() {
  return (
    <section className="py-24 md:py-32">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <p className="eyebrow mb-4">Status</p>
        <h2 className="font-display mb-6 text-[44px] font-black text-foreground md:text-[64px]" data-reveal>
          What is built,
          <br />
          <span className="brand-underline">what is not yet.</span>
        </h2>
        <p className="mb-12 max-w-2xl text-[17px] font-semibold leading-[1.5] text-foreground-soft">
          Every merged module went through independent review rounds; the findings and fixes
          are in the commit history.
        </p>
        <div className="grid gap-5 md:grid-cols-2">
          <div className="card-ring-lg bg-surface-mint p-7 md:p-9">
            <h3 className="mb-5 text-[20px] font-bold text-foreground">Built and in main</h3>
            <ul data-reveal-stagger className="space-y-3 text-[15px] font-semibold text-foreground">
              {BUILT.map((b) => (
                <li key={b} data-stagger-child className="flex gap-3">
                  <span className="text-brand-ink" aria-hidden>✓</span>
                  {b}
                </li>
              ))}
            </ul>
          </div>
          <div className="card-ring-lg p-7 md:p-9">
            <h3 className="mb-5 text-[20px] font-bold text-foreground">On branches, not merged</h3>
            <ul data-reveal-stagger className="space-y-3 text-[15px] font-semibold text-foreground-soft">
              {BRANCHES.map((b) => (
                <li key={b} data-stagger-child className="flex gap-3">
                  <span className="text-foreground-muted" aria-hidden>○</span>
                  {b}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </section>
  );
}

const CLONE = "git clone https://github.com/payminto-io/app.git payminto";

function SelfHost() {
  return (
    <section id="self-host" className="section-mint border-y border-border py-24 md:py-32">
      <div className="mx-auto max-w-3xl px-4 sm:px-6">
        <p className="eyebrow mb-4 !text-brand-ink">Self-host</p>
        <h2 className="font-display mb-6 text-[44px] font-black text-foreground md:text-[64px]" data-reveal>
          Your servers.
          <br />
          <span className="brand-underline">Your ledger.</span>
        </h2>
        <p className="mb-10 text-[17px] font-semibold leading-[1.5] text-foreground-soft">
          Install each Node workspace once, then run each service in its own terminal. The
          backend target loads <code className="font-mono text-[15px]">backend/.env</code>.
        </p>
        <div className="card-ring overflow-hidden font-mono text-[13px]" data-reveal>
          <div className="flex items-center justify-between border-b border-border bg-surface-soft px-4 py-2.5">
            <span className="text-[12px] font-semibold text-foreground-muted">terminal</span>
            <CopyButton text={CLONE} />
          </div>
          <pre className="overflow-x-auto bg-[#0e0f0c] px-5 py-5 leading-[1.8] text-[#fafaf7]">
            <span className="text-brand">$ </span>
            {CLONE}
            {"\n"}
            <span className="text-brand">$ </span>cd payminto
            {"\n"}
            <span className="text-brand">$ </span>make backend   <span className="text-white/45"># API on :8090</span>
            {"\n"}
            <span className="text-brand">$ </span>make frontend  <span className="text-white/45"># dashboard on :3003</span>
            {"\n"}
            <span className="text-brand">$ </span>make checkout  <span className="text-white/45"># hosted checkout on :3002</span>
            {"\n"}
            <span className="text-brand">$ </span>make smoke-local
          </pre>
        </div>
        <p className="mt-5 text-[14px] font-semibold text-foreground-muted">
          Tests, the CRE simulation and the full repository map are in the{" "}
          <a href={LINKS.source} className="link-inline">README</a>.
        </p>
      </div>
    </section>
  );
}

const FAQS = [
  {
    q: "Who holds the money?",
    a: "Payminto is software you run. Card and UPI money moves through the payment providers you connect. Stablecoins arrive at deposit accounts derived from keys you configure and are swept to your hot wallet. Custody providers such as BitGo are in progress on branches.",
  },
  {
    q: "What does Payminto cost?",
    a: "The code is MIT licensed. The payment providers and networks you connect charge their own fees. The fees you charge are versioned rules, and every payment records the version it paid.",
  },
  {
    q: "Do I need Chainlink to run it?",
    a: "No. The CRE module is off by default (CRE_ENABLED=false), and with it off the gateway behaves exactly as without it. Running the workflow in production needs Chainlink CRE Early Access; the steps are in cre/README.md.",
  },
  {
    q: "What happens if a worker crashes mid-sweep?",
    a: "The claim, its links and its lock are written in one transaction before anything is signed, and every signature is saved before it is sent. A restarted or second worker resumes from the database and the chain, and every transition is a compare-and-set, so nothing is sent twice.",
  },
  {
    q: "Why two RPC providers?",
    a: "A single node can lag or briefly lose a transaction, which makes a live deposit look missing or dropped. Live refuses to boot with fewer than two distinct endpoints, and a deposit is dropped only when both agree after finalization.",
  },
];

function FAQ() {
  return (
    <section className="py-24 md:py-32">
      <div className="mx-auto max-w-3xl px-4 sm:px-6">
        <h2 className="font-display mb-12 text-center text-[44px] font-black text-foreground md:text-[64px]" data-reveal>
          Questions,
          <br />
          <span className="brand-underline">answered.</span>
        </h2>
        <div className="space-y-3">
          {FAQS.map((f, i) => (
            <details key={f.q} className="faq card-ring overflow-hidden" open={i === 0}>
              <summary className="hover-faq flex cursor-pointer list-none items-center justify-between gap-4 px-6 py-5 text-[17px] font-bold text-foreground">
                {f.q}
                <span className="faq-icon grid h-8 w-8 flex-shrink-0 place-items-center rounded-full bg-surface-mint text-brand-ink" aria-hidden>
                  +
                </span>
              </summary>
              <div className="border-t border-border px-6 py-5 text-[15px] font-semibold leading-[1.6] text-foreground-soft">{f.a}</div>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}

function CTA() {
  return (
    <section className="relative overflow-hidden border-t border-border py-24 md:py-32">
      <div className="hero-glow absolute inset-0" aria-hidden />
      <div className="relative mx-auto max-w-4xl px-4 text-center sm:px-6">
        <h2 className="font-display mb-8 text-[52px] font-black text-foreground md:text-[96px]" data-reveal>
          Run your own
          <br />
          <span className="brand-underline">payment gateway.</span>
        </h2>
        <p className="mx-auto mb-10 max-w-xl text-[18px] font-semibold text-foreground-soft">
          Read the code, then open the app and the hosted checkout.
        </p>
        <div className="flex flex-col items-center justify-center gap-3 sm:flex-row">
          <a href={LINKS.source} className="btn-pill btn-primary !px-7 !py-4 !text-[18px]">
            View the source <span aria-hidden>→</span>
          </a>
          <a href={LINKS.app} className="btn-pill btn-secondary !px-7 !py-4 !text-[18px]">
            Open the app
          </a>
        </div>
      </div>
    </section>
  );
}

function Footer() {
  const cols: Record<string, [string, string][]> = {
    Product: [
      ["App", LINKS.app],
      ["Checkout", LINKS.checkout],
      ["Demo walkthrough", LINKS.demo],
    ],
    Source: [
      ["GitHub", LINKS.source],
      ["Chainlink CRE", LINKS.chainlinkTrack],
      ["Solana", LINKS.solanaTrack],
      ["NOWNodes RPC", LINKS.nownodesTrack],
    ],
  };
  return (
    <footer className="bg-[#0e0f0c] py-16 text-[#fafaf7]">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="grid gap-10 md:grid-cols-[2fr_1fr_1fr]">
          <div>
            <div className="mb-4 flex items-center gap-2.5">
              <Image src="/brand/mark-dark.svg" alt="" width={30} height={30} />
              <span className="text-[18px] font-bold">payminto</span>
            </div>
            <p className="max-w-sm text-[14px] font-semibold leading-[1.5] text-white/65">
              An open-source, self-hostable payment gateway: payment providers and stablecoins
              on one double-entry ledger, with optional Chainlink CRE solvency attestation.
            </p>
          </div>
          {Object.entries(cols).map(([title, items]) => (
            <div key={title}>
              <h4 className="mb-4 text-[12px] font-bold uppercase tracking-[0.15em] text-white/55">{title}</h4>
              <ul className="space-y-2.5">
                {items.map(([label, href]) => (
                  <li key={label}>
                    <a href={href} className="text-[14px] font-semibold text-white/85 transition-colors hover:text-brand focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand">
                      {label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <p className="mt-14 border-t border-white/10 pt-8 text-[12.5px] font-semibold text-white/55">
          © 2026 Payminto. Open source under the MIT License.
        </p>
      </div>
    </footer>
  );
}

export default function Home() {
  return (
    <main id="top" className="min-h-screen bg-background text-foreground">
      <Navbar />
      <Hero />
      <DemoVideo />
      <Market />
      <Flow />
      <Solvency />
      <Solana />
      <Screens />
      <Features />
      <Status />
      <SelfHost />
      <FAQ />
      <CTA />
      <Footer />
    </main>
  );
}
