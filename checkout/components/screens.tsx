"use client";

import { RefreshCw } from "lucide-react";
import { PoweredBy } from "./ui";

function Frame({ children }: { children: React.ReactNode }) {
  return (
    <main className="flex min-h-dvh flex-col items-center justify-center px-4 py-12">
      <div className="flex w-full max-w-[400px] flex-col gap-4 rounded-md border border-line bg-surface p-6 wide:p-8">{children}</div>
      <div className="mt-6"><PoweredBy /></div>
    </main>
  );
}

export function Loading() {
  return (
    <Frame>
      <div className="flex items-center gap-3">
        <span className="skeleton h-9 w-9" />
        <span className="skeleton h-4 w-32" />
      </div>
      <span className="skeleton h-10 w-40" />
      <span className="skeleton h-14 w-full" />
      <span className="skeleton h-14 w-full" />
      <span className="skeleton h-12 w-full" />
      <span className="sr-only" role="status">Loading payment</span>
    </Frame>
  );
}

export function NotFound() {
  return (
    <Frame>
      <h1 className="text-h2 font-semibold text-ink">No payment here</h1>
      <p className="text-body text-ink-soft">No payment matches this link. Check the address you were given or ask the merchant for a new one.</p>
    </Frame>
  );
}

export function ErrorScreen({ message, onRetry }: { message?: string; onRetry: () => void }) {
  return (
    <Frame>
      <h1 className="text-h2 font-semibold text-ink">Could not load this payment</h1>
      <p className="text-body text-ink-soft" role="alert">{message || "The payment service did not answer. Nothing was charged."}</p>
      <button type="button" className="btn btn-outline w-full" onClick={onRetry}>
        <RefreshCw size={16} strokeWidth={1.75} />
        Try again
      </button>
    </Frame>
  );
}

export function Empty() {
  return (
    <Frame>
      <h1 className="text-h2 font-semibold text-ink">Open a payment link to continue</h1>
      <p className="text-body text-ink-soft">This page accepts only a payment reference issued by a merchant.</p>
    </Frame>
  );
}
