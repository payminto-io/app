"use client";

import { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { QRCodeSVG } from "qrcode.react";
import {
  Link2,
  Copy,
  Check,
  ExternalLink,
  MessageCircle,
  Send,
  ArrowLeft,
  Search,
  UserPlus,
  X,
  Loader2,
} from "lucide-react";
import { useCreatePayment } from "@/lib/query/hooks/use-payments";
import { useCustomersList, type Customer } from "@/lib/query/hooks/use-customers";
import { isApiError } from "@/lib/query/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { checkoutURL } from "@/lib/checkout-url";

/**
 * Create Payment Link — matches PayRam's flow with proper customer handling.
 *
 * 1. Customer Search: merchants can search existing customers by email, or
 *    create a new one inline. The selected customer is pinned to the form.
 * 2. Amount: USD amount input.
 * 3. Generate: creates the payment, auto-creates the customer record (via
 *    backend), and shows the success state with QR + share buttons.
 */
export function CreatePaymentForm() {
  const router = useRouter();
  const create = useCreatePayment();

  // Form state
  const [amountInUSD, setAmountInUSD] = useState("");
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);
  const [newCustomerEmail, setNewCustomerEmail] = useState<string>("");
  const [error, setError] = useState<string | null>(null);

  // Success state
  const [paymentURL, setPaymentURL] = useState<string | null>(null);
  const [referenceID, setReferenceID] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [createdAmount, setCreatedAmount] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    const email = selectedCustomer?.email || newCustomerEmail;
    if (!email) {
      setError("Please select or enter a customer email.");
      return;
    }
    if (!amountInUSD || Number(amountInUSD) <= 0) {
      setError("Please enter a valid amount.");
      return;
    }

    try {
      const result = await create.mutateAsync({
        amountInUSD,
        customerEmail: email,
        customerID: selectedCustomer?.customerID || undefined,
      });
      const ref = result.reference_id || result.referenceID || "";
      setReferenceID(ref);
      setPaymentURL(checkoutURL(ref));
      setCreatedAmount(amountInUSD);
    } catch (err) {
      setError(isApiError(err) ? err.message : "Failed to create payment link.");
    }
  }

  function handleCopy() {
    if (paymentURL) {
      navigator.clipboard.writeText(paymentURL);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  function handleReset() {
    setPaymentURL(null);
    setReferenceID(null);
    setCopied(false);
    setCreatedAmount("");
    setAmountInUSD("");
    setSelectedCustomer(null);
    setNewCustomerEmail("");
    setError(null);
  }

  function shareURL(platform: "whatsapp" | "telegram" | "sms") {
    if (!paymentURL) return;
    const text = `Pay $${createdAmount} USD via this secure link: ${paymentURL}`;
    const encoded = encodeURIComponent(text);
    const urls: Record<string, string> = {
      whatsapp: `https://wa.me/?text=${encoded}`,
      telegram: `https://t.me/share/url?url=${encodeURIComponent(paymentURL)}&text=${encodeURIComponent(`Pay $${createdAmount} USD`)}`,
      sms: `sms:?body=${encoded}`,
    };
    window.open(urls[platform], "_blank");
  }

  /* ── Success state ──────────────────────────────────── */
  if (paymentURL) {
    return (
      <div className="flex justify-center pt-8">
        <div className="w-full max-w-md space-y-6">
          <button
            type="button"
            onClick={handleReset}
            className="flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground transition-colors"
          >
            <ArrowLeft className="size-3.5" />
            Create another link
          </button>

          <div className="text-center space-y-2">
            <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-emerald-500/10">
              <Check className="size-5 text-emerald-500" />
            </div>
            <h2 className="text-[22px] font-bold tracking-tight">
              Payment Link Created
            </h2>
            <p className="text-[14px] text-muted-foreground">
              Share this link with your customer to collect{" "}
              <span className="font-semibold text-foreground">
                ${createdAmount} USD
              </span>
            </p>
          </div>

          {/* QR */}
          <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
            <div className="flex justify-center rounded-xl border border-border bg-white p-5">
              <QRCodeSVG value={paymentURL} size={200} level="M" />
            </div>
          </div>

          {/* URL + copy */}
          <div className="space-y-1.5">
            <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Payment Link
            </div>
            <div className="flex items-center gap-2">
              <code className="flex-1 truncate rounded-lg border border-border bg-muted/50 px-3 py-2.5 font-mono text-[12px]">
                {paymentURL}
              </code>
              <Button
                variant="outline"
                size="icon"
                className="shrink-0 h-10 w-10"
                onClick={handleCopy}
              >
                {copied ? <Check className="size-4 text-emerald-500" /> : <Copy className="size-4" />}
              </Button>
            </div>
          </div>

          {/* Share */}
          <div className="space-y-1.5">
            <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
              Share via
            </div>
            <div className="flex gap-2">
              <Button variant="outline" className="flex-1 gap-2" onClick={() => shareURL("whatsapp")}>
                <MessageCircle className="size-4 text-emerald-500" /> WhatsApp
              </Button>
              <Button variant="outline" className="flex-1 gap-2" onClick={() => shareURL("telegram")}>
                <Send className="size-4 text-blue-500" /> Telegram
              </Button>
              <Button variant="outline" className="flex-1 gap-2" onClick={() => shareURL("sms")}>
                <MessageCircle className="size-4 text-muted-foreground" /> SMS
              </Button>
            </div>
          </div>

          {/* Actions */}
          <div className="flex gap-3">
            <Button
              variant="outline"
              className="flex-1 gap-2"
              onClick={() => window.open(paymentURL, "_blank")}
            >
              <ExternalLink className="size-4" /> Open Link
            </Button>
            <Button
              className="flex-1 bg-foreground text-background hover:bg-foreground/90"
              onClick={() => router.push(`/dashboard/payments/${referenceID}`)}
            >
              View Payment
            </Button>
          </div>
        </div>
      </div>
    );
  }

  /* ── Form state ───────────────────────────────────── */
  return (
    <div className="flex justify-center pt-8">
      <div className="w-full max-w-lg space-y-8">
        {/* Icon + heading */}
        <div className="text-center space-y-2">
          <div className="mx-auto flex size-11 items-center justify-center rounded-full bg-[var(--pm-primary)]/10">
            <Link2 className="size-5 text-[var(--pm-primary)]" />
          </div>
          <h2 className="text-[22px] font-bold tracking-tight">
            Create Payment Link
          </h2>
          <p className="text-[14px] text-muted-foreground leading-relaxed">
            Select a customer and enter an amount to generate a
            <br />
            shareable payment link.
          </p>
        </div>

        {/* Form card */}
        <div className="rounded-2xl border border-border bg-card p-8 shadow-sm">
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* CUSTOMER SEARCH */}
            <div className="space-y-2">
              <label className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                Select Customer
              </label>
              <CustomerCombobox
                selected={selectedCustomer}
                onSelect={setSelectedCustomer}
                newEmail={newCustomerEmail}
                onNewEmail={setNewCustomerEmail}
              />
            </div>

            {/* AMOUNT */}
            <div className="space-y-2">
              <label className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                Amount
              </label>
              <div className="relative">
                <span className="absolute left-4 top-1/2 -translate-y-1/2 text-muted-foreground text-base font-medium">$</span>
                <input
                  type="number"
                  step="0.01"
                  min="0.01"
                  required
                  placeholder="0.00"
                  value={amountInUSD}
                  onChange={(e) => setAmountInUSD(e.target.value)}
                  className="flex h-12 w-full rounded-lg border border-input bg-background pl-8 pr-16 py-2 text-center text-base font-medium ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 tabular-nums"
                />
                <span className="absolute right-4 top-1/2 -translate-y-1/2 text-muted-foreground text-sm font-medium">USD</span>
              </div>
            </div>

            {error ? (
              <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2.5">
                <p className="text-[13px] text-destructive flex items-center gap-2">
                  <span className="text-destructive">⊘</span>
                  {error}
                </p>
              </div>
            ) : null}

            <Button
              type="submit"
              disabled={create.isPending}
              className="w-full h-12 rounded-full bg-foreground text-background font-semibold hover:bg-foreground/90 text-[15px]"
            >
              {create.isPending ? "Generating..." : "Generate Payment Link"}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}

/* ── Customer Combobox ─────────────────────────────── */

function CustomerCombobox({
  selected,
  onSelect,
  newEmail,
  onNewEmail,
}: {
  selected: Customer | null;
  onSelect: (c: Customer | null) => void;
  newEmail: string;
  onNewEmail: (email: string) => void;
}) {
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Debounce search query
  const [debouncedQuery, setDebouncedQuery] = useState(query);
  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 250);
    return () => clearTimeout(t);
  }, [query]);

  const { data, isFetching } = useCustomersList(
    debouncedQuery ? { search: debouncedQuery, limit: 10 } : { limit: 10 }
  );
  const customers = data?.customers ?? [];

  // Close dropdown on outside click
  useEffect(() => {
    function handler(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  // Check if the query is a valid email
  const isEmail = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(query);
  const existingMatch = customers.find(
    (c) => c.email?.toLowerCase() === query.toLowerCase()
  );
  const canCreateNew = isEmail && !existingMatch && query.length > 0;

  // Show selected chip
  if (selected) {
    return (
      <div className="flex items-center justify-between rounded-lg border border-[var(--pm-primary)]/30 bg-[var(--pm-primary)]/5 px-4 py-3">
        <div className="flex items-center gap-3 min-w-0">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-[var(--pm-primary)]/15 text-[var(--pm-primary)] text-[12px] font-bold uppercase">
            {(selected.name || selected.email || "?").charAt(0)}
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-[14px] font-medium truncate">
              {selected.name || selected.email}
            </div>
            {selected.email && selected.name !== selected.email ? (
              <div className="text-[11px] text-muted-foreground truncate">
                {selected.email}
              </div>
            ) : null}
          </div>
        </div>
        <button
          type="button"
          onClick={() => {
            onSelect(null);
            onNewEmail("");
            setQuery("");
          }}
          className="shrink-0 rounded-md p-1 hover:bg-[var(--pm-primary)]/10 transition-colors"
          aria-label="Clear selection"
        >
          <X className="size-4 text-muted-foreground" />
        </button>
      </div>
    );
  }

  return (
    <div ref={containerRef} className="relative">
      {/* Search input */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 size-4 text-muted-foreground pointer-events-none" />
        <input
          type="text"
          placeholder="Search by email, or enter new customer..."
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            onNewEmail(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          className="flex h-11 w-full rounded-lg border border-input bg-background pl-9 pr-10 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
          autoComplete="off"
        />
        {isFetching ? (
          <Loader2 className="absolute right-3 top-1/2 -translate-y-1/2 size-4 animate-spin text-muted-foreground" />
        ) : null}
      </div>

      {/* Dropdown */}
      {open && (customers.length > 0 || canCreateNew || !query) ? (
        <div className="absolute z-20 mt-1 w-full rounded-lg border border-border bg-popover shadow-lg overflow-hidden">
          {/* Existing customers */}
          {customers.length > 0 ? (
            <div className="max-h-[240px] overflow-y-auto py-1">
              <div className="px-3 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                Existing Customers
              </div>
              {customers.map((c) => (
                <button
                  key={c.id}
                  type="button"
                  onClick={() => {
                    onSelect(c);
                    onNewEmail("");
                    setOpen(false);
                    setQuery("");
                  }}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2 text-left hover:bg-muted/60 transition-colors"
                  )}
                >
                  <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-muted text-[11px] font-bold uppercase">
                    {(c.name || c.email || "?").charAt(0)}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="text-[13px] font-medium truncate">
                      {c.name || c.email}
                    </div>
                    {c.email && c.name !== c.email ? (
                      <div className="text-[11px] text-muted-foreground truncate">{c.email}</div>
                    ) : null}
                  </div>
                </button>
              ))}
            </div>
          ) : query && !isFetching ? (
            <div className="px-3 py-2 text-[12px] text-muted-foreground">
              No customers found
            </div>
          ) : null}

          {/* Create new customer option */}
          {canCreateNew ? (
            <div className="border-t border-border">
              <button
                type="button"
                onClick={() => {
                  // Create a temporary "pseudo-customer" for display
                  // Actual record is created by the backend when payment is submitted
                  onSelect({
                    id: 0,
                    name: query,
                    email: query,
                    customerID: "",
                    state: "active",
                    memberType: "customer",
                    createdAt: "",
                    updatedAt: "",
                  });
                  setOpen(false);
                  setQuery("");
                }}
                className="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-[var(--pm-primary)]/5 transition-colors"
              >
                <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-[var(--pm-primary)]/10 text-[var(--pm-primary)]">
                  <UserPlus className="size-4" />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="text-[13px] font-medium text-[var(--pm-primary)]">
                    Create new customer
                  </div>
                  <div className="text-[11px] text-muted-foreground truncate">{query}</div>
                </div>
              </button>
            </div>
          ) : null}
        </div>
      ) : null}

      {/* Helper text */}
      {!open && !selected ? (
        <p className="mt-1.5 text-[11px] text-muted-foreground">
          {newEmail && /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(newEmail)
            ? <>New customer <strong className="text-foreground">{newEmail}</strong> will be created.</>
            : "Start typing an email to search or create a new customer."}
        </p>
      ) : null}
    </div>
  );
}
