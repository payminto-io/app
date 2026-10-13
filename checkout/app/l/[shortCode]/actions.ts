"use server";

import { randomUUID } from "node:crypto";
import { redirect } from "next/navigation";
import { linksURL, validShortCode } from "@/lib/links";

/**
 * Starts a payment for one published link and hands the payer to the checkout.
 * The gateway owns every number: the chosen method is all that is sent, and the
 * amount, fee and total come back from it.
 */
export async function startPayment(formData: FormData): Promise<void> {
  const shortCode = String(formData.get("short_code") ?? "");
  if (!validShortCode(shortCode)) redirect("/l/invalid");

  const method = String(formData.get("method") ?? "");
  const chain = String(formData.get("chain") ?? "");
  const asset = String(formData.get("asset") ?? "");

  const body: Record<string, unknown> = { method: { method, chain: chain || null, asset: asset || null } };
  const email = String(formData.get("email") ?? "").trim();
  if (email) body.customer = { email };

  const res = await fetch(linksURL(`/public/links/${encodeURIComponent(shortCode)}/pay`), {
    method: "POST",
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      // Makes a double submit replay the same payment instead of opening a second one.
      "Idempotency-Key": randomUUID(),
    },
    body: JSON.stringify(body),
  });

  if (!res.ok) redirect(`/l/${encodeURIComponent(shortCode)}?error=1`);

  const payment = (await res.json()) as { payment_reference?: string };
  if (!payment.payment_reference) redirect(`/l/${encodeURIComponent(shortCode)}?error=1`);

  redirect(`/pay/${encodeURIComponent(payment.payment_reference)}`);
}
