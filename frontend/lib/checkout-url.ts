const CHECKOUT_BASE_URL = (
  process.env.NEXT_PUBLIC_CHECKOUT_URL ?? "http://localhost:3002"
).replace(/\/$/, "");

export function checkoutURL(referenceId: string): string {
  return `${CHECKOUT_BASE_URL}/pay/${encodeURIComponent(referenceId)}`;
}
