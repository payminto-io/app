/**
 * Server pricing, matched to the form by method identity. The browser computes no fee and no total:
 * per-method pricing is the saved link's `fee_preview`, the checkout view is `POST /links/preview`.
 */
import type { DroppedMethod, MethodPreview, PaymentLink } from "@/lib/api/links";
import { formFromLink, methodKey, pricingFingerprint, type LinkForm } from "./model";

/**
 * The saved link's per-method pricing keyed by method, or null while the form's pricing fields differ
 * from what was saved (the numbers would describe another link).
 */
export function savedPricing(link: PaymentLink | null, form: LinkForm): Map<string, MethodPreview> | null {
  if (!link?.fee_preview) return null;
  if (pricingFingerprint(formFromLink(link)) !== pricingFingerprint(form)) return null;
  return new Map(link.fee_preview.map((p) => [methodKey(p), p]));
}

/** Methods the checkout would leave out, keyed by method. */
export function droppedByKey(dropped: DroppedMethod[] | undefined): Map<string, DroppedMethod> {
  return new Map((dropped ?? []).map((d) => [methodKey(d), d]));
}
