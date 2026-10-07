"use client";

import { useEffect, useState } from "react";
import { CheckoutView } from "@/components/checkout-view";
import type { MethodKind } from "@/components/methods";
import { Empty, ErrorScreen, Loading, NotFound } from "@/components/screens";
import { FIXTURES, PAID_WITH_MESSAGE, PAID_WITHOUT_LINES } from "@/lib/fixtures";
import type { ViewState } from "@/lib/model";

const EXTRA = ["loading", "notfound", "error", "empty", "paid-message", "paid-plain"] as const;
const STATES = [...Object.keys(FIXTURES), ...EXTRA];

export function Preview({ state, theme }: { state: string; theme?: "light" | "dark" }) {
  const [method, setMethod] = useState<MethodKind | undefined>(state === "card" ? "card" : "stablecoin");
  const [network, setNetwork] = useState<string | undefined>("SOL");

  useEffect(() => {
    const root = document.documentElement;
    if (theme) root.dataset.theme = theme;
    else delete root.dataset.theme;
  }, [theme]);

  let body: React.ReactNode;
  if (state === "loading") body = <Loading />;
  else if (state === "notfound") body = <NotFound />;
  else if (state === "error") body = <ErrorScreen message="Payment service unavailable" onRetry={() => undefined} />;
  else if (state === "empty") body = <Empty />;
  else {
    const fixture =
      state === "paid-message" ? { payment: PAID_WITH_MESSAGE, state: "paid" as ViewState }
      : state === "paid-plain" ? { payment: PAID_WITHOUT_LINES, state: "paid" as ViewState }
      : FIXTURES[state as keyof typeof FIXTURES];
    if (!fixture) body = <NotFound />;
    else {
      const force = fixture.state ?? (state in FIXTURES ? (state as ViewState) : undefined);
      body = (
        <CheckoutView
          payment={fixture.payment}
          method={method}
          network={network}
          handlers={{
            onMethod: setMethod,
            onNetwork: setNetwork,
            onContinue: () => undefined,
            onBack: () => setMethod("stablecoin"),
            onRetry: () => undefined,
          }}
          forceState={force === "choose" && method === "card" ? "card" : force}
        />
      );
    }
  }

  return (
    <>
      <div className="sticky top-0 z-10 flex items-center gap-3 overflow-x-auto border-b border-line bg-surface-raised px-4 py-2 text-caption" data-preview-bar>
        <span className="shrink-0 rounded-xs bg-env-test px-1.5 py-0.5 font-medium text-[#15181d]">Sample data</span>
        <nav className="flex shrink-0 gap-1" aria-label="Preview states">
          {STATES.map((s) => (
            <a key={s} href={`?state=${s}${theme ? `&theme=${theme}` : ""}`} className={s === state ? "rounded-xs bg-ink px-1.5 py-0.5 text-ink-inverse" : "rounded-xs px-1.5 py-0.5 text-ink-soft hover:text-ink"}>
              {s}
            </a>
          ))}
        </nav>
        <span className="ml-auto flex shrink-0 gap-1">
          {(["light", "dark"] as const).map((t) => (
            <a key={t} href={`?state=${state}&theme=${t}`} className={t === theme ? "rounded-xs bg-ink px-1.5 py-0.5 text-ink-inverse" : "rounded-xs px-1.5 py-0.5 text-ink-soft hover:text-ink"}>
              {t}
            </a>
          ))}
        </span>
      </div>
      {body}
    </>
  );
}
