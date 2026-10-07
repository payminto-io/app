"use client";

import { useCallback, useEffect, useState } from "react";
import { ApiError, getJSON } from "@/lib/checkout";
import { fromApi, type ApiMethod, type ApiPayment, type CheckoutPayment } from "@/lib/model";
import { CheckoutView } from "./checkout-view";
import type { MethodKind } from "./methods";
import { ErrorScreen, Loading, NotFound } from "./screens";

type Load = { kind: "loading" } | { kind: "notfound" } | { kind: "error"; message: string } | { kind: "ready"; payment: CheckoutPayment };

export function Checkout({ referenceId }: { referenceId: string }) {
  const [load, setLoad] = useState<Load>({ kind: "loading" });
  const [methods, setMethods] = useState<ApiMethod[]>([]);
  const [method, setMethod] = useState<MethodKind | undefined>();
  const [network, setNetwork] = useState<string | undefined>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const refresh = useCallback(async (): Promise<void> => {
    try {
      const [payment, list] = await Promise.all([
        getJSON<ApiPayment>(`/api/payment/${encodeURIComponent(referenceId)}`),
        getJSON<{ currencies: ApiMethod[] }>("/api/methods").then((d) => d.currencies ?? []).catch(() => [] as ApiMethod[]),
      ]);
      setMethods(list);
      setLoad({ kind: "ready", payment: fromApi(payment, list) });
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 404) setLoad({ kind: "notfound" });
      else setLoad({ kind: "error", message: cause instanceof Error ? cause.message : "" });
    }
  }, [referenceId]);

  useEffect(() => {
    const first = window.setTimeout(() => void refresh(), 0);
    const poll = window.setInterval(() => void refresh(), 20_000);
    return () => {
      window.clearTimeout(first);
      window.clearInterval(poll);
    };
  }, [refresh]);

  useEffect(() => {
    const source = new EventSource(`/api/payment/${encodeURIComponent(referenceId)}/events`);
    source.addEventListener("payment", () => void refresh());
    return () => source.close();
  }, [referenceId, refresh]);

  // Defaults derived at render: a lone stablecoin method and a lone network need no choice.
  const stable = load.kind === "ready" ? load.payment.methods.stablecoin : undefined;
  const cardOffered = load.kind === "ready" && load.payment.methods.card;
  const effectiveMethod = method ?? (stable && !cardOffered ? "stablecoin" : undefined);
  const effectiveNetwork = network ?? (stable?.networks.length === 1 ? stable.networks[0].code : undefined);

  const assign = useCallback(async () => {
    const chosen = methods.find((m) => m.blockchainCode.toUpperCase() === effectiveNetwork && m.currencyCode.toUpperCase() === stable?.asset);
    if (!chosen) return;
    setBusy(true);
    setError("");
    try {
      await getJSON(`/api/payment/${encodeURIComponent(referenceId)}/address`, {
        method: "POST",
        body: JSON.stringify({ blockchainCode: chosen.blockchainCode, currencyCode: chosen.currencyCode }),
      });
      await refresh();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not prepare this payment.");
    } finally {
      setBusy(false);
    }
  }, [methods, effectiveNetwork, stable, referenceId, refresh]);

  if (load.kind === "loading") return <Loading />;
  if (load.kind === "notfound") return <NotFound />;
  if (load.kind === "error") return <ErrorScreen message={load.message} onRetry={() => { setLoad({ kind: "loading" }); void refresh(); }} />;

  return (
    <CheckoutView
      payment={load.payment}
      method={effectiveMethod}
      network={effectiveNetwork}
      busy={busy}
      error={error}
      handlers={{
        onMethod: setMethod,
        onNetwork: setNetwork,
        onContinue: () => { if (effectiveMethod === "stablecoin") void assign(); },
        // The address is bound to the payment; changing method means reloading
        // the page state, which the API currently returns as assigned.
        onBack: () => { setMethod(undefined); void refresh(); },
        onExpire: () => void refresh(),
      }}
    />
  );
}
