/**
 * Browser EventSource wrapper for the hosted checkout's real-time payment
 * stream. Transport-only: it knows how to open/parse/close the SSE connection
 * and nothing about React. The `use-payment-events` hook adapts it to the UI.
 */
import { API_BASE_URL, PUBLIC_PATHS } from "@/lib/constants";

export interface PaymentEvent {
  referenceID: string;
  state: string;
  confirmations?: number;
  requiredConfirmations?: number;
  txID?: string;
}

export interface PaymentEventHandlers {
  onEvent: (event: PaymentEvent) => void;
  onError?: (error: Event) => void;
  onOpen?: () => void;
}

/**
 * Opens an SSE connection for a payment and forwards parsed events. Returns a
 * disposer that closes the underlying EventSource. Safe to call only in the
 * browser (guarded for SSR).
 */
export function subscribeToPaymentEvents(
  referenceId: string,
  handlers: PaymentEventHandlers
): () => void {
  if (typeof window === "undefined" || typeof EventSource === "undefined") {
    return () => {};
  }

  const url = `${API_BASE_URL}${PUBLIC_PATHS.paymentEvents(referenceId)}`;
  const source = new EventSource(url);

  source.addEventListener("open", () => handlers.onOpen?.());

  source.addEventListener("payment", (e) => {
    try {
      const data = JSON.parse((e as MessageEvent).data) as PaymentEvent;
      handlers.onEvent(data);
    } catch {
      // Ignore malformed frames; the reconciliation poll will catch up.
    }
  });

  source.addEventListener("error", (e) => handlers.onError?.(e));

  return () => source.close();
}
