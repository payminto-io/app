/**
 * Empty-state renders from docs/brand/brand.yaml (ids and output paths are the
 * same there). Files are written by scripts/brand/generate.mjs --promote.
 */
export const EMPTY_ILLUSTRATIONS = {
  payments: "/brand/generated/empty-payments.webp",
  payouts: "/brand/generated/empty-payouts.webp",
  wallets: "/brand/generated/empty-wallets.webp",
  webhooks: "/brand/generated/empty-webhooks.webp",
  "api-keys": "/brand/generated/empty-api-keys.webp",
  customers: "/brand/generated/empty-customers.webp",
} as const;

export type EmptyIllustration = keyof typeof EMPTY_ILLUSTRATIONS;

/** Delivered at 720x480; shown at 3x for crisp edges on the matte surfaces. */
export const EMPTY_ILLUSTRATION_SIZE = { width: 240, height: 160 } as const;
