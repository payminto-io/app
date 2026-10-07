"use client";

import { useState, useEffect, useRef } from "react";
import Link from "next/link";
import { QRCodeSVG } from "qrcode.react";
import { Check, ExternalLink, Loader2, Search, UserPlus, X } from "lucide-react";
import { useCreatePayment } from "@/lib/query/hooks/use-payments";
import { useCustomersList, type Customer } from "@/lib/query/hooks/use-customers";
import { isApiError } from "@/lib/query/hooks/use-auth";
import { checkoutURL } from "@/lib/checkout-url";
import { cn } from "@/lib/utils";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { CurrencyDisplay } from "@/components/currency-display";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const CRUMBS = [{ label: "Payments", href: "/dashboard/payments" }, { label: "Create" }];

/**
 * Create a payment link: pick or type a customer, enter a USD amount. The
 * backend creates the customer record when the payment is submitted.
 */
export function CreatePaymentForm() {
  const create = useCreatePayment();

  const [amountInUSD, setAmountInUSD] = useState("");
  const [selectedCustomer, setSelectedCustomer] = useState<Customer | null>(null);
  const [newCustomerEmail, setNewCustomerEmail] = useState<string>("");
  const [error, setError] = useState<string | null>(null);

  const [paymentURL, setPaymentURL] = useState<string | null>(null);
  const [referenceID, setReferenceID] = useState<string | null>(null);
  const [createdAmount, setCreatedAmount] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);

    const email = selectedCustomer?.email || newCustomerEmail;
    if (!email) {
      setError("Choose a customer or enter an email.");
      return;
    }
    if (!amountInUSD || Number(amountInUSD) <= 0) {
      setError("Enter an amount above 0.");
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
      setError(isApiError(err) ? err.message : "The payment link could not be created.");
    }
  }

  function handleReset() {
    setPaymentURL(null);
    setReferenceID(null);
    setCreatedAmount("");
    setAmountInUSD("");
    setSelectedCustomer(null);
    setNewCustomerEmail("");
    setError(null);
  }

  function shareURL(platform: "whatsapp" | "telegram" | "sms") {
    if (!paymentURL) return;
    const text = `Pay ${createdAmount} USD: ${paymentURL}`;
    const encoded = encodeURIComponent(text);
    const urls: Record<string, string> = {
      whatsapp: `https://wa.me/?text=${encoded}`,
      telegram: `https://t.me/share/url?url=${encodeURIComponent(paymentURL)}&text=${encodeURIComponent(`Pay ${createdAmount} USD`)}`,
      sms: `sms:?body=${encoded}`,
    };
    window.open(urls[platform], "_blank");
  }

  if (paymentURL) {
    return (
      <div className="mx-auto w-full max-w-[560px] space-y-6">
        <PageHeader breadcrumbs={CRUMBS} title="Payment link ready" />

        <Card>
          <CardContent className="space-y-5">
            <div className="flex items-center gap-3">
              <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-ok-tint text-ok">
                <Check className="size-4" strokeWidth={2.25} />
              </span>
              <CurrencyDisplay amount={createdAmount} currency="USD" size="lg" />
            </div>

            <div className="flex justify-center rounded-lg border border-line bg-white p-5">
              <QRCodeSVG value={paymentURL} size={184} level="M" />
            </div>

            <CopyField value={paymentURL} />

            <div className="grid grid-cols-3 gap-2">
              <Button variant="outline" size="sm" onClick={() => shareURL("whatsapp")}>
                WhatsApp
              </Button>
              <Button variant="outline" size="sm" onClick={() => shareURL("telegram")}>
                Telegram
              </Button>
              <Button variant="outline" size="sm" onClick={() => shareURL("sms")}>
                SMS
              </Button>
            </div>
          </CardContent>
        </Card>

        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center sm:justify-between">
          <Button variant="ghost" onClick={handleReset}>
            Create another
          </Button>
          <div className="flex flex-col gap-2 sm:flex-row">
            <a
              href={paymentURL}
              target="_blank"
              rel="noopener noreferrer"
              className={buttonVariants({ variant: "outline" })}
            >
              Open checkout
              <ExternalLink />
            </a>
            <Link href={`/dashboard/payments/${referenceID}`} className={buttonVariants()}>
              View payment
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto w-full max-w-[560px] space-y-6">
      <PageHeader breadcrumbs={CRUMBS} title="Create payment" />

      <Card>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4" noValidate>
            <div className="space-y-1.5">
              <Label htmlFor="customer-search">Customer</Label>
              <CustomerCombobox
                selected={selectedCustomer}
                onSelect={setSelectedCustomer}
                newEmail={newCustomerEmail}
                onNewEmail={setNewCustomerEmail}
              />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="amount">Amount</Label>
              <div className="relative">
                <Input
                  id="amount"
                  type="number"
                  inputMode="decimal"
                  step="0.01"
                  min="0.01"
                  required
                  placeholder="0.00"
                  value={amountInUSD}
                  onChange={(e) => setAmountInUSD(e.target.value)}
                  className="num pr-14"
                />
                <span className="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-label font-medium text-ink-soft">
                  USD
                </span>
              </div>
            </div>

            {error ? (
              <p role="alert" className="text-body-sm text-bad">
                {error}
              </p>
            ) : null}

            <div className="flex justify-end pt-2">
              <Button type="submit" disabled={create.isPending} className="max-sm:w-full">
                {create.isPending ? "Creating..." : "Create payment link"}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

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

  const [debouncedQuery, setDebouncedQuery] = useState(query);
  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query), 250);
    return () => clearTimeout(t);
  }, [query]);

  const { data, isFetching } = useCustomersList(
    debouncedQuery ? { search: debouncedQuery, limit: 10 } : { limit: 10 }
  );
  const customers = data?.customers ?? [];

  useEffect(() => {
    function handler(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const isEmail = EMAIL_RE.test(query);
  const existingMatch = customers.find((c) => c.email?.toLowerCase() === query.toLowerCase());
  const canCreateNew = isEmail && !existingMatch && query.length > 0;

  if (selected) {
    return (
      <div className="flex min-h-9 items-center justify-between gap-3 rounded-sm border border-line-strong bg-surface py-1 pr-1 pl-3">
        <div className="min-w-0">
          <div className="truncate text-body text-ink">{selected.name || selected.email}</div>
          {selected.email && selected.name !== selected.email ? (
            <div className="truncate text-caption text-ink-soft">{selected.email}</div>
          ) : null}
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label="Clear customer"
          onClick={() => {
            onSelect(null);
            onNewEmail("");
            setQuery("");
          }}
        >
          <X />
        </Button>
      </div>
    );
  }

  const showHint = !open && newEmail && EMAIL_RE.test(newEmail);

  return (
    <div ref={containerRef} className="relative">
      <div className="relative">
        <Search
          aria-hidden
          className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-ink-faint"
        />
        <Input
          id="customer-search"
          type="text"
          role="combobox"
          aria-expanded={open}
          aria-controls="customer-options"
          placeholder="name@company.com"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            onNewEmail(e.target.value);
            setOpen(true);
          }}
          onFocus={() => setOpen(true)}
          className="pr-9 pl-9"
          autoComplete="off"
        />
        {isFetching ? (
          <Loader2 className="absolute top-1/2 right-3 size-4 -translate-y-1/2 animate-spin text-ink-faint" />
        ) : null}
      </div>

      {open && (customers.length > 0 || canCreateNew || (query && !isFetching)) ? (
        <div
          id="customer-options"
          role="listbox"
          className="absolute z-20 mt-1 w-full overflow-hidden rounded-md border border-line bg-surface-raised shadow-2"
        >
          {customers.length > 0 ? (
            <div className="max-h-60 overflow-y-auto p-1">
              {customers.map((c) => (
                <button
                  key={c.id}
                  type="button"
                  role="option"
                  aria-selected={false}
                  onClick={() => {
                    onSelect(c);
                    onNewEmail("");
                    setOpen(false);
                    setQuery("");
                  }}
                  className="flex w-full flex-col items-start rounded-xs px-2 py-1.5 text-left transition-colors duration-120 hover:bg-surface-sunken focus-visible:bg-surface-sunken"
                >
                  <span className="w-full truncate text-body text-ink">{c.name || c.email}</span>
                  {c.email && c.name !== c.email ? (
                    <span className="w-full truncate text-caption text-ink-soft">{c.email}</span>
                  ) : null}
                </button>
              ))}
            </div>
          ) : query && !isFetching ? (
            <p className="px-3 py-2 text-body-sm text-ink-soft">No customers match.</p>
          ) : null}

          {canCreateNew ? (
            <div className={cn("p-1", customers.length > 0 && "border-t border-line")}>
              <button
                type="button"
                role="option"
                aria-selected={false}
                onClick={() => {
                  // Display-only placeholder; the backend creates the record on submit.
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
                className="flex w-full items-center gap-2 rounded-xs px-2 py-1.5 text-left transition-colors duration-120 hover:bg-surface-sunken focus-visible:bg-surface-sunken"
              >
                <UserPlus className="size-4 shrink-0 text-tide" />
                <span className="min-w-0 truncate text-body text-ink">
                  Add <span className="font-medium">{query}</span>
                </span>
              </button>
            </div>
          ) : null}
        </div>
      ) : null}

      {showHint ? (
        <p className="mt-1.5 text-caption text-ink-soft">A new customer is added when the link is created.</p>
      ) : null}
    </div>
  );
}
