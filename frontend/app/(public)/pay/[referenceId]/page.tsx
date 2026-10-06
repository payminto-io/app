import { redirect } from "next/navigation";
import { checkoutURL } from "@/lib/checkout-url";

/** Compatibility redirect for payment links issued before checkout isolation. */
export default async function LegacyPayPage({ params }: { params: Promise<{ referenceId: string }> }) {
  const { referenceId } = await params;
  redirect(checkoutURL(referenceId));
}
