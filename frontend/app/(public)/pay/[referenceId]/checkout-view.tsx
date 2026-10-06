"use client";

import { useState, useEffect, useCallback, useMemo } from "react";
import { QRCodeSVG } from "qrcode.react";
import {
  Copy,
  Check,
  AlertTriangle,
  Loader2,
  ChevronRight,
  Shield,
  HelpCircle,
  Clock,
  Info,
} from "lucide-react";
import { cn } from "@/lib/utils";
import {
  useBlockchainCurrencies,
  useAssignDepositAddress,
  usePublicTicker,
} from "@/lib/query/hooks/use-public";
import type {
  PublicPayment,
  BlockchainCurrencyOption,
} from "@/lib/query/hooks/use-public";

const TERMINAL = [
  "confirmed",
  "filled",
  "FILLED",
  "expired",
  "cancelled",
  "CANCELLED",
];

const CHAIN_LABELS: Record<string, string> = {
  ETH: "Ethereum",
  BTC: "Bitcoin",
  BASE: "Base",
  POLYGON: "Polygon",
  TRX: "Tron",
};

const CHAIN_ICONS: Record<string, string> = {
  ETH: "\u27e0",
  BTC: "\u20bf",
  BASE: "\ud83d\udfe2",
  POLYGON: "\u2b21",
  TRX: "\u25c8",
};

// Symbols that are always $1.00 — no ticker lookup needed.
const STABLE_COINS = new Set([
  "USDC",
  "USDT",
  "DAI",
  "BUSD",
  "PYUSD",
  "USDP",
  "TUSD",
]);

/**
 * Compute the exact crypto amount the user must send, given a USD amount
 * and the currency code. Returns null when we don't yet have a price.
 *
 * Decimals:
 *   - BTC        → 8
 *   - ETH / POL  → 8 (visually matches PayRam; ETH has 18 on-chain but 8 is
 *                    enough for any realistic invoice and is readable)
 *   - TRX        → 6
 *   - stablecoin → 2
 */
function computeCryptoAmount(
  amountUSD: string | undefined,
  currencyCode: string | undefined,
  prices: Record<string, string> | undefined
): string | null {
  if (!amountUSD || !currencyCode) return null;
  const usd = Number(amountUSD);
  if (!Number.isFinite(usd) || usd <= 0) return null;

  const symbol = currencyCode.toUpperCase();
  let price: number | null = null;

  if (STABLE_COINS.has(symbol)) {
    price = 1;
  } else {
    const raw = prices?.[symbol];
    if (raw) {
      const parsed = Number(raw);
      if (Number.isFinite(parsed) && parsed > 0) price = parsed;
    }
  }

  if (price === null) return null;

  const amount = usd / price;
  const decimals =
    symbol === "BTC" ? 8 : symbol === "TRX" ? 6 : STABLE_COINS.has(symbol) ? 2 : 8;
  return amount.toFixed(decimals);
}

function groupByChain(currencies: BlockchainCurrencyOption[]) {
  const map = new Map<
    string,
    { code: string; name: string; currencies: BlockchainCurrencyOption[] }
  >();
  for (const c of currencies) {
    const code = c.blockchainCode;
    if (!map.has(code))
      map.set(code, {
        code,
        name: c.blockchain?.name ?? CHAIN_LABELS[code] ?? code,
        currencies: [],
      });
    map.get(code)!.currencies.push(c);
  }
  for (const g of map.values())
    g.currencies.sort((a, b) =>
      ["native", "ETH", "BTC", "TRX", "POL"].includes(a.standard)
        ? -1
        : ["native", "ETH", "BTC", "TRX", "POL"].includes(b.standard)
        ? 1
        : a.currencyCode.localeCompare(b.currencyCode)
    );
  return Array.from(map.values());
}

export function CheckoutView({ payment }: { payment: PublicPayment }) {
  const isExpired =
    payment.state === "expired" ||
    payment.state === "EXPIRED" ||
    (payment.expiresAt && new Date(payment.expiresAt) < new Date());
  const isTerminal = TERMINAL.includes(payment.state);
  const hasAddress = Boolean(payment.depositAddress);

  // Pull the price for the payment currency (and USD as a sentinel) so the
  // merchant card can show the exact crypto amount even before the user has
  // clicked a chain.
  const tickerSymbols = useMemo(() => {
    const arr: string[] = [];
    if (payment.currencyCode && payment.currencyCode !== "USD") {
      arr.push(payment.currencyCode.toUpperCase());
    }
    return arr;
  }, [payment.currencyCode]);

  const ticker = usePublicTicker(tickerSymbols);
  const cryptoAmount = computeCryptoAmount(
    payment.amountInUSD,
    payment.currencyCode,
    ticker.data?.prices
  );

  return (
    <div className="min-h-screen grid grid-cols-1 lg:grid-cols-2">
      {/* LEFT HALF: Gradient + Merchant Card */}
      <LeftPanel payment={payment} cryptoAmount={cryptoAmount} />

      {/* RIGHT HALF: Payment Method Panel */}
      <div className="flex flex-col bg-[#0a0a1a] min-h-[60vh] lg:min-h-screen">
        <div className="flex-1 flex flex-col justify-center px-6 sm:px-8 lg:px-12 py-8 max-w-[540px] mx-auto w-full">
          {/* Header */}
          <h2 className="text-[17px] font-semibold text-white mb-5">
            Select a payment method
          </h2>

          {/* Payment method \u2014 crypto only (card onramp is a future milestone) */}
          <div className="mb-6 rounded-xl bg-white/5 p-1">
            <div className="flex items-center justify-center gap-2 rounded-lg bg-white/10 py-2.5 text-[13px] font-medium text-white shadow-sm">
              <span>{"\u25c8"}</span> Pay with Crypto
            </div>
          </div>

          {/* Content based on state */}
          <div className="flex-1 space-y-5">
            {/* Expired */}
            {isExpired && (
              <div className="rounded-xl border border-red-500/30 bg-red-500/5 p-6 text-center space-y-2">
                <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-red-500/15">
                  <Clock className="size-6 text-red-400" />
                </div>
                <div className="text-[17px] font-semibold text-red-400">
                  Payment Expired
                </div>
                <p className="text-[13px] text-red-400/70">
                  This payment link has expired. Please request a new one from
                  the merchant.
                </p>
              </div>
            )}

            {/* Success */}
            {(payment.state === "confirmed" ||
              payment.state === "filled" ||
              payment.state === "FILLED") && (
              <div className="rounded-xl border border-emerald-500/30 bg-emerald-500/5 p-6 text-center space-y-2">
                <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-emerald-500/15">
                  <Check className="size-6 text-emerald-500" />
                </div>
                <div className="text-[18px] font-semibold text-emerald-400">
                  Payment Successful
                </div>
                <p className="text-[13px] text-emerald-400/70">
                  Your transaction has been confirmed on the blockchain.
                </p>
              </div>
            )}

            {/* Chain selector — no address yet */}
            {!isTerminal && !hasAddress && !isExpired && (
              <ChainSelector
                referenceId={payment.referenceID}
                amountUSD={payment.amountInUSD}
              />
            )}

            {/* QR + address — address assigned */}
            {!isTerminal && hasAddress && (
              <>
                <PaymentDetails
                  payment={payment}
                  cryptoAmount={cryptoAmount}
                  priceLoading={ticker.isLoading}
                />
                <StatusProgression state={payment.state} />
                {/* Warnings */}
                <div className="space-y-2">
                  <div className="flex items-start gap-2.5 rounded-lg bg-amber-500/5 border border-amber-500/20 p-3">
                    <AlertTriangle className="size-4 text-amber-500 mt-0.5 shrink-0" />
                    <p className="text-[11px] text-amber-200/80 leading-relaxed">
                      Please ensure you&apos;re only sending{" "}
                      <strong>{payment.currencyCode}</strong> on the{" "}
                      <strong>
                        {CHAIN_LABELS[payment.blockchainCode ?? ""] ??
                          payment.blockchainCode}
                      </strong>{" "}
                      network to avoid loss of funds.
                    </p>
                  </div>
                  <div className="flex items-start gap-2.5 rounded-lg bg-white/[0.03] border border-white/10 p-3">
                    <Shield className="size-4 text-white/40 mt-0.5 shrink-0" />
                    <p className="text-[11px] text-white/50 leading-relaxed">
                      Payments are almost instant and should reflect in 1-15
                      mins.
                    </p>
                  </div>
                </div>
              </>
            )}

            {/* Expiry */}
            {!isTerminal && payment.expiresAt && (
              <ExpiryCountdown expiresAt={payment.expiresAt} />
            )}
          </div>

          {/* Bottom link — re-checks payment status on demand. */}
          <div className="mt-6 pt-4 border-t border-white/10 text-center">
            <button
              type="button"
              onClick={() => window.location.reload()}
              className="inline-flex items-center gap-1.5 text-[12px] text-white/40 hover:text-white/70 transition-colors"
            >
              <HelpCircle className="size-3.5" />
              Sent payment but don&apos;t see it? Refresh status
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

/* -- Left Panel (gradient + merchant card) -- */

function LeftPanel({
  payment,
  cryptoAmount,
}: {
  payment: PublicPayment;
  cryptoAmount: string | null;
}) {
  const showCrypto =
    payment.currencyCode && payment.currencyCode !== "USD";

  return (
    <div className="relative flex flex-col items-center justify-center p-8 lg:p-12 bg-gradient-to-br from-[#7c3aed] via-[#4f46e5] to-[#06b6d4] overflow-hidden min-h-[40vh] lg:min-h-screen">
      {/* Decorative blobs */}
      <div className="absolute top-[-10%] left-[-10%] size-[50%] rounded-full bg-[#a855f7]/30 blur-3xl pointer-events-none" />
      <div className="absolute bottom-[-10%] right-[-10%] size-[50%] rounded-full bg-[#06b6d4]/30 blur-3xl pointer-events-none" />

      {/* Merchant card */}
      <div className="relative z-10 w-full max-w-[360px]">
        <div className="rounded-2xl bg-[#0f0f23]/90 backdrop-blur-sm p-7 shadow-2xl border border-white/10">
          {/* Merchant identity */}
          <div className="flex items-center gap-3 mb-6">
            <div className="flex size-11 items-center justify-center rounded-xl bg-[var(--pm-primary)] text-white text-sm font-bold shadow-lg shadow-[var(--pm-primary)]/30">
              {(payment.merchantName ?? "P").charAt(0).toUpperCase()}
            </div>
            <div>
              <div className="text-[10px] text-white/40 uppercase tracking-widest">
                Payment to
              </div>
              <div className="text-[15px] font-semibold text-white">
                {payment.merchantName ?? "Merchant"}
              </div>
            </div>
          </div>

          {/* Amount in USD row */}
          <div className="flex items-center justify-between mb-4">
            <span className="text-[12px] text-white/50">Amount in USD</span>
            <span className="text-[22px] font-bold text-white tabular-nums">
              ${payment.amountInUSD}
            </span>
          </div>

          <div className="h-px bg-white/10 mb-5" />

          {/* Send amount in crypto */}
          {showCrypto ? (
            <div className="text-center">
              <div className="text-[10px] text-white/40 uppercase tracking-widest mb-2">
                Send
              </div>
              {cryptoAmount ? (
                <div className="text-[28px] font-bold text-white tabular-nums leading-tight break-all">
                  {cryptoAmount}{" "}
                  <span className="text-[20px] text-white/80">
                    {payment.currencyCode}
                  </span>
                </div>
              ) : (
                <div className="text-[16px] text-white/50 italic leading-tight py-1.5">
                  Amount will be calculated...
                </div>
              )}
              {payment.blockchainCode && (
                <div className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-white/10 px-3 py-1 text-[12px] text-white/70">
                  <span>{CHAIN_ICONS[payment.blockchainCode] ?? "\u25cf"}</span>
                  <span>
                    via{" "}
                    {CHAIN_LABELS[payment.blockchainCode] ??
                      payment.blockchainCode}
                  </span>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center">
              <div className="text-[10px] text-white/40 uppercase tracking-widest mb-1">
                Total
              </div>
              <div className="text-[30px] font-bold text-white tabular-nums leading-tight">
                ${payment.amountInUSD}
              </div>
              <div className="mt-1 text-[12px] text-white/40">
                Select a currency to continue
              </div>
            </div>
          )}
        </div>

        {/* Powered by */}
        <div className="mt-5 text-center">
          <span className="text-[11px] text-white/50">
            Powered by{" "}
            <span className="font-extrabold italic text-white/80">PAYMINTO</span>
          </span>
        </div>
      </div>

      {/* Disclaimer at bottom */}
      <div className="absolute bottom-4 left-6 right-6 hidden lg:block">
        <p className="text-[9px] text-white/30 leading-relaxed text-center">
          Payminto is a self-hosted payment platform. All transactions are
          processed directly on-chain. The merchant maintains full custody of
          funds.
        </p>
      </div>
    </div>
  );
}

/* -- Chain Selector -- */

function ChainSelector({
  referenceId,
  amountUSD,
}: {
  referenceId: string;
  amountUSD: string;
}) {
  const { data, isLoading } = useBlockchainCurrencies();
  const assign = useAssignDepositAddress(referenceId);
  const [selected, setSelected] = useState<string | null>(null);
  const chains = data ? groupByChain(data.currencies) : [];

  if (isLoading)
    return (
      <div className="flex items-center justify-center py-10">
        <Loader2 className="size-5 animate-spin text-white/40" />
      </div>
    );

  return (
    <div className="space-y-3">
      <p className="text-[13px] text-white/50">
        Choose how to pay{" "}
        <strong className="text-white">${amountUSD} USD</strong>
      </p>
      <div className="space-y-2">
        {chains.map((chain) =>
          chain.currencies.map((cur) => {
            const key = `${chain.code}-${cur.currencyCode}`;
            const loading = selected === key && assign.isPending;
            return (
              <button
                key={key}
                type="button"
                disabled={assign.isPending}
                onClick={() => {
                  setSelected(key);
                  assign.mutate({
                    blockchainCode: chain.code,
                    currencyCode: cur.currencyCode,
                  });
                }}
                className={cn(
                  "w-full flex items-center gap-3 rounded-xl border px-4 py-3.5 text-left transition-all",
                  "border-white/10 hover:border-[var(--pm-primary)]/50 hover:bg-[var(--pm-primary)]/5",
                  "disabled:opacity-60 disabled:cursor-not-allowed",
                  loading && "border-[var(--pm-primary)] bg-[var(--pm-primary)]/10"
                )}
              >
                <div className="flex size-10 items-center justify-center rounded-full bg-white/10 text-[16px]">
                  {CHAIN_ICONS[chain.code] ?? "\u25cf"}
                </div>
                <div className="flex-1">
                  <div className="text-[14px] font-medium text-white">
                    {cur.currencyCode}
                  </div>
                  <div className="text-[11px] text-white/40">{chain.name}</div>
                </div>
                {loading ? (
                  <Loader2 className="size-4 animate-spin text-[var(--pm-primary)]" />
                ) : (
                  <ChevronRight className="size-4 text-white/30" />
                )}
              </button>
            );
          })
        )}
      </div>
      {assign.isError && (
        <p className="text-[12px] text-red-400">
          Failed to assign address. Try again.
        </p>
      )}
    </div>
  );
}

/* -- Payment Details (QR + address) -- */

function PaymentDetails({
  payment,
  cryptoAmount,
  priceLoading,
}: {
  payment: PublicPayment;
  cryptoAmount: string | null;
  priceLoading: boolean;
}) {
  const [copied, setCopied] = useState(false);
  function handleCopy() {
    if (!payment.depositAddress) return;
    navigator.clipboard.writeText(payment.depositAddress);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  const chainLabel =
    CHAIN_LABELS[payment.blockchainCode ?? ""] ?? payment.blockchainCode;

  // Address truncation for small screens: 0x1234…abcd
  const addr = payment.depositAddress ?? "";
  const shortAddr =
    addr.length > 20 ? `${addr.slice(0, 10)}\u2026${addr.slice(-8)}` : addr;

  return (
    <div className="space-y-5">
      {/* Scan to send label */}
      <div className="text-center">
        <div className="text-[10px] font-semibold text-white/40 uppercase tracking-[0.15em] mb-3">
          Scan QR to send
        </div>

        {/* PRIMARY: crypto amount — the hero number */}
        <div className="flex items-center justify-center gap-2">
          {cryptoAmount ? (
            <>
              <span className="text-[32px] font-bold text-white tabular-nums leading-none break-all">
                {cryptoAmount}
              </span>
              <span className="text-[18px] font-semibold text-white/70 tabular-nums">
                {payment.currencyCode}
              </span>
              <Info className="size-3.5 text-white/30" />
            </>
          ) : priceLoading ? (
            <span className="inline-flex items-center gap-2 text-[20px] font-semibold text-white/50 italic">
              <Loader2 className="size-4 animate-spin" />
              Calculating...
            </span>
          ) : (
            <span className="text-[18px] font-medium text-white/50 italic">
              Amount will be calculated...
            </span>
          )}
        </div>

        {/* USD equivalent */}
        <div className="mt-1.5 text-[12px] text-white/50 tabular-nums">
          {"\u2248"} ${payment.amountInUSD} USD
        </div>

        {/* Chain pill */}
        {payment.blockchainCode && (
          <div className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-white/10 border border-white/10 px-3 py-1 text-[11px] font-medium text-white/80">
            <span>{CHAIN_ICONS[payment.blockchainCode] ?? "\u25cf"}</span>
            {chainLabel}
          </div>
        )}
      </div>

      {/* QR Code */}
      <div className="flex justify-center rounded-xl bg-white p-5 shadow-inner">
        <QRCodeSVG
          value={payment.depositAddress!}
          size={200}
          level="M"
          bgColor="#ffffff"
          fgColor="#000000"
        />
      </div>

      {/* Deposit address */}
      <div className="space-y-2">
        <label className="text-[10px] font-semibold text-white/40 uppercase tracking-[0.15em]">
          Deposit Address
        </label>
        <div className="flex items-center gap-2">
          <code
            className="flex-1 rounded-lg border border-white/10 bg-white/5 px-3 py-3 font-mono text-[12px] text-white/80 tabular-nums truncate"
            title={addr}
          >
            <span className="hidden sm:inline">{addr}</span>
            <span className="sm:hidden">{shortAddr}</span>
          </code>
          <button
            onClick={handleCopy}
            className={cn(
              "shrink-0 flex items-center gap-1.5 rounded-lg px-4 py-3 text-[12px] font-semibold text-white transition-all",
              copied
                ? "bg-emerald-600 hover:bg-emerald-700"
                : "bg-[var(--pm-primary)] hover:bg-[var(--pm-primary-deep)]"
            )}
          >
            {copied ? (
              <Check className="size-3.5" />
            ) : (
              <Copy className="size-3.5" />
            )}
            {copied ? "Copied!" : "COPY"}
          </button>
        </div>
      </div>

      {/* "Send EXACTLY" warning — impossible to miss */}
      <div className="flex items-start gap-2.5 rounded-lg bg-amber-500/10 border border-amber-500/40 p-3">
        <AlertTriangle className="size-4 text-amber-400 mt-0.5 shrink-0" />
        <div className="flex-1">
          <p className="text-[12px] font-semibold text-amber-200 leading-tight">
            Send EXACTLY this amount
          </p>
          <p className="text-[11px] text-amber-200/70 mt-0.5 leading-relaxed">
            Sending a different amount may result in a partial payment or loss
            of funds.
          </p>
        </div>
      </div>

      {/* Animated checking status */}
      <div className="flex items-center justify-center gap-2 rounded-lg bg-white/[0.03] border border-white/10 py-3 text-[12px] text-amber-400">
        <Loader2 className="size-3.5 animate-spin" />
        Checking for new payments...
      </div>
    </div>
  );
}

/* -- Status Progression -- */

function StatusProgression({ state }: { state: string }) {
  const steps = [
    { key: "pending", label: "Checking" },
    { key: "spotted", label: "Spotted" },
    { key: "confirmed", label: "Confirmed" },
  ];
  const map: Record<string, number> = {
    pending: 0,
    OPEN: 0,
    partially_filled: 1,
    PARTIALLY_FILLED: 1,
    filled: 2,
    FILLED: 2,
    confirmed: 2,
    CONFIRMED: 2,
  };
  const current = map[state] ?? 0;

  return (
    <div className="flex items-center justify-between px-2 py-1">
      {steps.map((step, i) => {
        const active = i <= current;
        const isCurrent = i === current;
        return (
          <div
            key={step.key}
            className="flex items-center gap-0 flex-1 last:flex-none"
          >
            <div className="flex flex-col items-center">
              <div className="relative">
                <div
                  className={cn(
                    "size-3.5 rounded-full transition-colors",
                    active
                      ? "bg-[var(--pm-primary)]"
                      : "border-2 border-white/20 bg-transparent"
                  )}
                />
                {isCurrent && !TERMINAL.includes(state) && (
                  <span className="absolute inset-0 animate-ping rounded-full bg-[var(--pm-primary)]/30" />
                )}
              </div>
              <p
                className={cn(
                  "mt-1.5 text-[10px] whitespace-nowrap",
                  active ? "font-medium text-white" : "text-white/40"
                )}
              >
                {step.label}
              </p>
            </div>
            {i < steps.length - 1 && (
              <div
                className={cn(
                  "h-px flex-1 mx-2 mt-[-14px]",
                  i < current ? "bg-[var(--pm-primary)]" : "bg-white/10"
                )}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}

/* -- Expiry Countdown -- */

function ExpiryCountdown({ expiresAt }: { expiresAt: string }) {
  const calc = useCallback(
    () =>
      Math.max(
        0,
        Math.floor((new Date(expiresAt).getTime() - Date.now()) / 1000)
      ),
    [expiresAt]
  );
  const [remaining, setRemaining] = useState(calc);
  useEffect(() => {
    const t = setInterval(() => setRemaining(calc()), 1000);
    return () => clearInterval(t);
  }, [calc]);
  const mins = Math.floor(remaining / 60);
  const secs = remaining % 60;
  const low = remaining < 120;

  if (remaining <= 0) {
    return (
      <div className="text-center py-2">
        <span className="text-[13px] font-medium text-red-400">
          Payment expired
        </span>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center gap-2 py-2">
      <Clock className={cn("size-3.5", low ? "text-red-400" : "text-white/40")} />
      <span
        className={cn(
          "text-[13px] tabular-nums",
          low ? "text-red-400 font-medium" : "text-white/50"
        )}
      >
        Expires in {mins}:{secs.toString().padStart(2, "0")}
      </span>
    </div>
  );
}
