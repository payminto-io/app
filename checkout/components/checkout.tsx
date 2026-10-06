"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import {
  AlertTriangle,
  ArrowLeft,
  Check,
  CheckCircle2,
  ChevronRight,
  Clock3,
  Copy,
  ExternalLink,
  LoaderCircle,
  LockKeyhole,
  Radio,
  RefreshCw,
  ShieldCheck,
  Smartphone,
  WalletCards,
  Wifi,
  WifiOff,
  XCircle,
} from "lucide-react";
import {
  estimatedAmount,
  getJSON,
  paymentURI,
  phaseFor,
  type CheckoutPhase,
  type Payment,
  type PaymentMethod,
} from "@/lib/checkout";

const chainMeta: Record<string, { label: string; mark: string; tint: string }> = {
  BTC: { label: "Bitcoin", mark: "₿", tint: "#f59e0b" },
  ETH: { label: "Ethereum", mark: "Ξ", tint: "#8b8cf8" },
  BASE: { label: "Base", mark: "B", tint: "#4f8cff" },
  POLYGON: { label: "Polygon", mark: "⬡", tint: "#a77bff" },
  TRX: { label: "Tron", mark: "◈", tint: "#ff5964" },
};

export function Checkout({ referenceId }: { referenceId: string }) {
  const [payment, setPayment] = useState<Payment | null>(null);
  const [methods, setMethods] = useState<PaymentMethod[]>([]);
  const [prices, setPrices] = useState<Record<string, string>>({});
  const [phase, setPhase] = useState<CheckoutPhase>("loading");
  const [error, setError] = useState("");
  const [live, setLive] = useState(false);
  const [assigning, setAssigning] = useState("");

  const loadPayment = useCallback(async () => {
    try {
      const next = await getJSON<Payment>(`/api/payment/${encodeURIComponent(referenceId)}`);
      setPayment(next);
      setPhase(phaseFor(next));
      setError("");
      return next;
    } catch (cause) {
      setPhase("error");
      setError(cause instanceof Error ? cause.message : "This payment link is unavailable.");
      return null;
    }
  }, [referenceId]);

  useEffect(() => {
    const initial = window.setTimeout(() => {
      void Promise.all([
        loadPayment(),
        getJSON<{ currencies: PaymentMethod[] }>("/api/methods")
          .then((data) => setMethods(data.currencies ?? []))
          .catch(() => setMethods([])),
      ]);
    }, 0);
    const poll = window.setInterval(() => void loadPayment(), 20_000);
    return () => { window.clearTimeout(initial); window.clearInterval(poll); };
  }, [loadPayment]);

  useEffect(() => {
    const source = new EventSource(`/api/payment/${encodeURIComponent(referenceId)}/events`);
    source.onopen = () => setLive(true);
    source.onerror = () => setLive(false);
    source.addEventListener("payment", (event) => {
      try {
        const update = JSON.parse((event as MessageEvent<string>).data) as { state?: string };
        if (update.state) {
          setPayment((current) => {
            const next = current ? { ...current, state: update.state as string } : current;
            if (next) setPhase(phaseFor(next));
            return next;
          });
          void loadPayment();
        }
      } catch {
        // The reconciliation poll remains authoritative if a frame is malformed.
      }
    });
    return () => source.close();
  }, [referenceId, loadPayment]);

  useEffect(() => {
    const symbol = payment?.currencyCode?.toUpperCase();
    if (!symbol || ["USDC", "USDT", "DAI", "PYUSD", "USDP", "TUSD"].includes(symbol)) return;
    let active = true;
    const load = () => getJSON<{ prices: Record<string, string> }>(`/api/ticker?symbols=${encodeURIComponent(symbol)}`)
      .then((result) => { if (active) setPrices(result.prices ?? {}); })
      .catch(() => undefined);
    void load();
    const timer = window.setInterval(load, 60_000);
    return () => { active = false; window.clearInterval(timer); };
  }, [payment?.currencyCode]);

  useEffect(() => {
    if (!payment?.expiresAt || ["paid", "cancelled", "expired"].includes(phase)) return;
    const delay = Date.parse(payment.expiresAt) - Date.now();
    const timer = window.setTimeout(() => setPhase("expired"), Math.max(0, Math.min(delay + 50, 2_147_000_000)));
    return () => window.clearTimeout(timer);
  }, [payment?.expiresAt, phase]);

  async function choose(method: PaymentMethod) {
    const key = `${method.blockchainCode}:${method.currencyCode}`;
    setAssigning(key);
    setError("");
    try {
      await getJSON(`/api/payment/${encodeURIComponent(referenceId)}/address`, {
        method: "POST",
        body: JSON.stringify({ blockchainCode: method.blockchainCode, currencyCode: method.currencyCode }),
      });
      await loadPayment();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not prepare this payment method.");
    } finally {
      setAssigning("");
    }
  }

  if (phase === "loading") return <LoadingScreen />;
  if (phase === "error" || !payment) return <ErrorScreen message={error} retry={loadPayment} />;

  const amount = estimatedAmount(payment.amountInUSD, payment.currencyCode, prices);

  return (
    <main className="checkout-shell">
      <SummaryPanel payment={payment} phase={phase} live={live} />
      <section className="action-panel" aria-live="polite">
        <div className="action-inner">
          <header className="mobile-brand">
            <Brand />
            <LivePill live={live} />
          </header>

          {phase === "choose" && (
            <MethodChooser
              methods={methods}
              amountUSD={payment.amountInUSD}
              assigning={assigning}
              error={error}
              choose={choose}
            />
          )}
          {(phase === "awaiting" || phase === "detected") && (
            <PaymentInstructions payment={payment} amount={amount} phase={phase} />
          )}
          {phase === "paid" && <TerminalState kind="paid" merchant={payment.merchantName} />}
          {phase === "expired" && <TerminalState kind="expired" merchant={payment.merchantName} />}
          {phase === "cancelled" && <TerminalState kind="cancelled" merchant={payment.merchantName} />}

          <footer className="trust-footer">
            <LockKeyhole aria-hidden="true" />
            <span>Encrypted checkout</span>
            <span className="dot" />
            <span>No account required</span>
          </footer>
        </div>
      </section>
    </main>
  );
}

function SummaryPanel({ payment, phase, live }: { payment: Payment; phase: CheckoutPhase; live: boolean }) {
  return (
    <aside className="summary-panel">
      <div className="aurora aurora-one" />
      <div className="aurora aurora-two" />
      <div className="summary-content">
        <div className="desktop-brand-row"><Brand /><LivePill live={live} /></div>
        <div className="invoice-card">
          <div className="merchant-row">
            <span className="merchant-avatar">{(payment.merchantName ?? "P").slice(0, 1).toUpperCase()}</span>
            <span><small>PAYMENT TO</small><strong>{payment.merchantName ?? "Payminto merchant"}</strong></span>
            <ShieldCheck className="verified" aria-label="Verified checkout" />
          </div>
          <div className="invoice-total">
            <small>AMOUNT DUE</small>
            <strong><span>$</span>{money(payment.amountInUSD)}</strong>
            <p>USD</p>
          </div>
          <div className="invoice-divider" />
          <div className="invoice-meta">
            <span>REFERENCE</span>
            <code title={payment.referenceID}>{shortReference(payment.referenceID)}</code>
          </div>
          <div className="invoice-meta">
            <span>STATUS</span>
            <b className={`status-${phase}`}>{statusLabel(phase)}</b>
          </div>
        </div>
        <p className="custody-note"><ShieldCheck aria-hidden="true" /> Funds go directly to the merchant&apos;s on-chain address.</p>
      </div>
    </aside>
  );
}

function MethodChooser({ methods, amountUSD, assigning, error, choose }: {
  methods: PaymentMethod[];
  amountUSD: string;
  assigning: string;
  error: string;
  choose: (method: PaymentMethod) => Promise<void>;
}) {
  const [filter, setFilter] = useState("all");
  const chains = useMemo(() => Array.from(new Set(methods.map((item) => item.blockchainCode))), [methods]);
  const visible = filter === "all" ? methods : methods.filter((item) => item.blockchainCode === filter);

  return (
    <div className="step-content">
      <div className="step-heading">
        <span className="step-number">1</span>
        <div><p className="eyebrow">PAY WITH CRYPTO</p><h1>Choose a payment method</h1></div>
      </div>
      <p className="lead">Select the asset and network you want to use for your <strong>${money(amountUSD)} USD</strong> payment.</p>
      {chains.length > 1 && (
        <div className="chain-filters" aria-label="Filter payment networks">
          <button className={filter === "all" ? "active" : ""} onClick={() => setFilter("all")}>All</button>
          {chains.map((chain) => <button key={chain} className={filter === chain ? "active" : ""} onClick={() => setFilter(chain)}>{chainMeta[chain]?.label ?? chain}</button>)}
        </div>
      )}
      <div className="method-list">
        {visible.map((method) => {
          const meta = chainMeta[method.blockchainCode] ?? { label: method.blockchain?.name ?? method.blockchainCode, mark: "●", tint: "#7c8cff" };
          const key = `${method.blockchainCode}:${method.currencyCode}`;
          return (
            <button className="method-card" key={`${method.id}:${key}`} disabled={Boolean(assigning)} onClick={() => void choose(method)}>
              <span className="coin-mark" style={{ "--coin-color": meta.tint } as React.CSSProperties}>{meta.mark}</span>
              <span className="method-copy"><strong>{method.currencyCode}</strong><small>{method.currency?.name ?? method.currencyCode} on {meta.label}</small></span>
              <span className="network-badge">{meta.label}</span>
              {assigning === key ? <LoaderCircle className="spin" /> : <ChevronRight />}
            </button>
          );
        })}
        {visible.length === 0 && <div className="empty-methods"><WalletCards /><p>No payment methods are currently available.</p></div>}
      </div>
      {error && <div className="inline-error"><AlertTriangle />{error}</div>}
      <div className="safety-note"><ShieldCheck /><p><strong>Network matters.</strong> The next screen will repeat the exact network. Sending on another network can permanently lose funds.</p></div>
    </div>
  );
}

function PaymentInstructions({ payment, amount, phase }: { payment: Payment; amount: string | null; phase: CheckoutPhase }) {
  const [copied, setCopied] = useState("");
  const [remaining, setRemaining] = useState(() => secondsRemaining(payment.expiresAt));
  const uri = paymentURI(payment, amount);
  const nativeDeepLink = uri !== payment.depositAddress;
  const meta = chainMeta[payment.blockchainCode ?? ""];

  useEffect(() => {
    const timer = window.setInterval(() => setRemaining(secondsRemaining(payment.expiresAt)), 1000);
    return () => window.clearInterval(timer);
  }, [payment.expiresAt]);

  async function copy(value: string, kind: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(kind);
      window.setTimeout(() => setCopied(""), 1800);
    } catch { setCopied(""); }
  }

  return (
    <div className="step-content instructions">
      <div className="instructions-topline">
        <button type="button" className="back-link" onClick={() => window.location.reload()}><ArrowLeft /> Payment details</button>
        <span className={remaining < 120 ? "countdown danger" : "countdown"}><Clock3 /> {formatRemaining(remaining)}</span>
      </div>
      <div className="step-heading compact">
        <span className="step-number">2</span>
        <div><p className="eyebrow">SEND FROM YOUR WALLET</p><h1>{phase === "detected" ? "Payment detected" : "Complete your payment"}</h1></div>
      </div>

      {phase === "detected" && <div className="detected-banner"><Radio /><div><strong>Transaction detected</strong><p>Keep this page open while the network confirms it.</p></div><LoaderCircle className="spin" /></div>}

      <div className="amount-block">
        <span>ESTIMATED AMOUNT</span>
        <div>{amount ?? "—"} <b>{payment.currencyCode}</b></div>
        <button disabled={!amount} onClick={() => amount && copy(amount, "amount")}><Copy /> {copied === "amount" ? "Copied" : "Copy amount"}</button>
        <p>The conversion is an estimate from the latest ticker. Verify the final amount your merchant expects before sending.</p>
      </div>

      <div className="qr-wrap">
        <div className="qr-card">
          <QRCodeSVG value={uri} size={208} level="M" marginSize={1} title="Wallet payment QR code" />
          <span className="qr-coin" style={{ "--coin-color": meta?.tint ?? "#7c8cff" } as React.CSSProperties}>{meta?.mark ?? "●"}</span>
        </div>
        <p>Scan with a compatible wallet</p>
      </div>

      {nativeDeepLink && <a className="wallet-button" href={uri}><Smartphone /> Open in wallet <ExternalLink /></a>}

      <div className="address-block">
        <div className="address-label"><span>DEPOSIT ADDRESS</span><b>{meta?.label ?? payment.blockchainCode} network</b></div>
        <button className="address-copy" onClick={() => copy(payment.depositAddress ?? "", "address")}>
          <code>{payment.depositAddress}</code><span>{copied === "address" ? <Check /> : <Copy />}{copied === "address" ? "Copied" : "Copy"}</span>
        </button>
      </div>

      <Progress phase={phase} />
      <div className="warning-note"><AlertTriangle /><p>Send only <strong>{payment.currencyCode}</strong> on <strong>{meta?.label ?? payment.blockchainCode}</strong>. Check the full address in your wallet before approving.</p></div>
    </div>
  );
}

function Progress({ phase }: { phase: CheckoutPhase }) {
  const current = phase === "detected" ? 1 : phase === "paid" ? 2 : 0;
  return (
    <div className="progress-row" aria-label="Payment progress">
      {["Waiting", "Detected", "Confirmed"].map((label, index) => (
        <div className={index <= current ? "progress-step active" : "progress-step"} key={label}>
          <span>{index < current ? <Check /> : index + 1}</span><small>{label}</small>
          {index < 2 && <i />}
        </div>
      ))}
    </div>
  );
}

function TerminalState({ kind, merchant }: { kind: "paid" | "expired" | "cancelled"; merchant?: string }) {
  const content = kind === "paid"
    ? { icon: <CheckCircle2 />, title: "Payment confirmed", body: `Your payment to ${merchant ?? "the merchant"} is confirmed on-chain.`, className: "success" }
    : kind === "expired"
      ? { icon: <Clock3 />, title: "Payment link expired", body: "Do not send funds to this address. Ask the merchant for a new payment link.", className: "expired" }
      : { icon: <XCircle />, title: "Payment cancelled", body: "This invoice is no longer accepting payment. Contact the merchant if this seems wrong.", className: "cancelled" };
  return (
    <div className={`terminal-state ${content.className}`}>
      <div className="terminal-icon">{content.icon}</div><p className="eyebrow">PAYMENT STATUS</p><h1>{content.title}</h1><p>{content.body}</p>
      {kind === "paid" && <div className="receipt-note"><ShieldCheck /> You can safely close this page.</div>}
    </div>
  );
}

function Brand() {
  return <div className="brand"><span className="brand-symbol">P</span><strong>payminto</strong><small>CHECKOUT</small></div>;
}

function LivePill({ live }: { live: boolean }) {
  return <span className={live ? "live-pill connected" : "live-pill"}>{live ? <Wifi /> : <WifiOff />}{live ? "Live status" : "Reconnecting"}</span>;
}

function LoadingScreen() {
  return <main className="loading-screen"><Brand /><div className="loading-orbit"><ShieldCheck /><span /></div><h1>Preparing secure checkout</h1><p>Retrieving payment details from the merchant…</p></main>;
}

function ErrorScreen({ message, retry }: { message: string; retry: () => Promise<Payment | null> }) {
  return <main className="error-screen"><div className="error-icon"><AlertTriangle /></div><p className="eyebrow">PAYMENT UNAVAILABLE</p><h1>We couldn&apos;t open this checkout</h1><p>{message || "Check the payment link or ask the merchant for a new one."}</p><button onClick={() => void retry()}><RefreshCw /> Try again</button><small>Never send funds using details from an error screen.</small></main>;
}

function money(value: string): string {
  const amount = Number(value);
  return Number.isFinite(amount) ? amount.toLocaleString("en-US", { minimumFractionDigits: 2, maximumFractionDigits: 2 }) : value;
}

function shortReference(value: string): string {
  return value.length > 22 ? `${value.slice(0, 10)}…${value.slice(-8)}` : value;
}

function statusLabel(phase: CheckoutPhase): string {
  return ({ choose: "Method required", awaiting: "Awaiting payment", detected: "Detected", paid: "Confirmed", expired: "Expired", cancelled: "Cancelled", loading: "Loading", error: "Unavailable" })[phase];
}

function secondsRemaining(expiresAt?: string): number {
  if (!expiresAt) return 0;
  return Math.max(0, Math.floor((Date.parse(expiresAt) - Date.now()) / 1000));
}

function formatRemaining(seconds: number): string {
  if (seconds <= 0) return "Expired";
  const minutes = Math.floor(seconds / 60);
  return `${minutes}:${String(seconds % 60).padStart(2, "0")} left`;
}
