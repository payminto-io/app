export type ChainId = "eip155:1";

export type OpenPaymentCommand = {
  merchantReference: string;
  invoiceAmount: { currency: string; minorUnits: string };
  paymentMethod: { chainId: ChainId; assetId: string };
  expiresAt: string;
};

export type OpenPayment = {
  invoiceId: string;
  merchantReference: string;
  invoiceAmount: { currency: string; minorUnits: string };
  paymentMethod: { chainId: ChainId; assetId: string };
  quote: {
    id: string;
    invoiceCurrency: string;
    invoiceMinorUnits: string;
    chainId: ChainId;
    assetId: string;
    requiredAtomicUnits: string;
    assetDecimals: number;
    rateNumerator: string;
    rateDenominator: string;
    source: string;
    quotedAt: string;
    expiresAt: string;
    rounding: string;
  };
  depositAddress: { assignmentId: string; address: string };
  state: "open";
  revision: number;
  openedAt: string;
  expiresAt: string;
};

export const chainPresentation: Record<ChainId, { name: string }> = {
  "eip155:1": { name: "Ethereum mainnet" },
};

const integerString = /^(0|[1-9][0-9]*)$/;
const currencyCode = /^[A-Z]{3}$/;
const assetSuffix = /^(erc20:0x[0-9a-f]{40}|slip44:[0-9]+)$/;

function record(value: unknown): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw unsafeContract();
  }
  return value as Record<string, unknown>;
}

function stringField(source: Record<string, unknown>, key: string): string {
  const value = source[key];
  if (typeof value !== "string" || value.length === 0) throw unsafeContract();
  return value;
}

function integerField(source: Record<string, unknown>, key: string): string {
  const value = stringField(source, key);
  if (!integerString.test(value)) throw unsafeContract();
  return value;
}

function instantField(source: Record<string, unknown>, key: string): string {
  const value = stringField(source, key);
  if (!Number.isFinite(Date.parse(value))) throw unsafeContract();
  return value;
}

export function assertCommand(command: OpenPaymentCommand): void {
  const raw = record(command);
  if (stringField(raw, "merchantReference").length > 128) throw unsafeContract();
  const invoice = record(raw.invoiceAmount);
  if (!currencyCode.test(stringField(invoice, "currency"))) throw unsafeContract();
  integerField(invoice, "minorUnits");
  const method = record(raw.paymentMethod);
  const chainId = assertKnownChain(stringField(method, "chainId"));
  assertAssetId(chainId, stringField(method, "assetId"));
  instantField(raw, "expiresAt");
}

export function normalizeOpenPayment(payload: unknown, command: OpenPaymentCommand): OpenPayment {
  const envelope = record(payload);
  const payment = record(envelope.payment);
  const invoice = record(payment.invoiceAmount);
  const method = record(payment.paymentMethod);
  const quote = record(payment.quote);
  const depositAddress = record(payment.depositAddress);
  const chainId = assertKnownChain(stringField(method, "chainId"));
  const assetId = stringField(method, "assetId");
  assertAssetId(chainId, assetId);

  const normalized: OpenPayment = {
    invoiceId: stringField(payment, "invoiceId"),
    merchantReference: stringField(payment, "merchantReference"),
    invoiceAmount: {
      currency: stringField(invoice, "currency"),
      minorUnits: integerField(invoice, "minorUnits"),
    },
    paymentMethod: { chainId, assetId },
    quote: {
      id: stringField(quote, "id"),
      invoiceCurrency: stringField(quote, "invoiceCurrency"),
      invoiceMinorUnits: integerField(quote, "invoiceMinorUnits"),
      chainId: assertKnownChain(stringField(quote, "chainId")),
      assetId: stringField(quote, "assetId"),
      requiredAtomicUnits: integerField(quote, "requiredAtomicUnits"),
      assetDecimals: boundedInteger(quote.assetDecimals, 0, 255),
      rateNumerator: integerField(quote, "rateNumerator"),
      rateDenominator: positiveIntegerField(quote, "rateDenominator"),
      source: stringField(quote, "source"),
      quotedAt: instantField(quote, "quotedAt"),
      expiresAt: instantField(quote, "expiresAt"),
      rounding: stringField(quote, "rounding"),
    },
    depositAddress: {
      assignmentId: stringField(depositAddress, "assignmentId"),
      address: stringField(depositAddress, "address"),
    },
    state: assertOpenState(payment.state),
    revision: boundedInteger(payment.revision, 1, Number.MAX_SAFE_INTEGER),
    openedAt: instantField(payment, "openedAt"),
    expiresAt: instantField(payment, "expiresAt"),
  };

  assertAssetId(normalized.quote.chainId, normalized.quote.assetId);
  if (
    normalized.merchantReference !== command.merchantReference ||
    normalized.invoiceAmount.currency !== command.invoiceAmount.currency ||
    normalized.invoiceAmount.minorUnits !== command.invoiceAmount.minorUnits ||
    normalized.paymentMethod.chainId !== command.paymentMethod.chainId ||
    normalized.paymentMethod.assetId !== command.paymentMethod.assetId ||
    normalized.expiresAt !== command.expiresAt ||
    normalized.quote.invoiceCurrency !== normalized.invoiceAmount.currency ||
    normalized.quote.invoiceMinorUnits !== normalized.invoiceAmount.minorUnits ||
    normalized.quote.chainId !== normalized.paymentMethod.chainId ||
    normalized.quote.assetId !== normalized.paymentMethod.assetId
  ) {
    throw unsafeContract();
  }
  return normalized;
}

function assertKnownChain(value: string): ChainId {
  if (Object.prototype.hasOwnProperty.call(chainPresentation, value)) return value as ChainId;
  throw unsafeContract();
}

function assertAssetId(chainId: ChainId, value: string): void {
  const prefix = `${chainId}/`;
  if (!value.startsWith(prefix) || !assetSuffix.test(value.slice(prefix.length))) {
    throw unsafeContract();
  }
}

function assertOpenState(value: unknown): "open" {
  if (value !== "open") throw unsafeContract();
  return value;
}

function positiveIntegerField(source: Record<string, unknown>, key: string): string {
  const value = integerField(source, key);
  if (/^0+$/.test(value)) throw unsafeContract();
  return value;
}

function boundedInteger(value: unknown, min: number, max: number): number {
  if (!Number.isSafeInteger(value) || (value as number) < min || (value as number) > max) {
    throw unsafeContract();
  }
  return value as number;
}

function unsafeContract(): PaymentOpenError {
  return new PaymentOpenError("unsafe_contract");
}

export type PaymentOpenErrorCode =
  | "invalid_request"
  | "unauthenticated"
  | "unsupported_payment_method"
  | "idempotency_conflict"
  | "merchant_reference_conflict"
  | "quote_unavailable"
  | "quote_expired"
  | "quote_mismatched"
  | "deposit_address_unavailable"
  | "storage_unavailable"
  | "unsafe_contract"
  | "transport_unavailable";

export class PaymentOpenError extends Error {
  constructor(public readonly code: PaymentOpenErrorCode) {
    super(code);
    this.name = "PaymentOpenError";
  }
}

export const knownErrorCodes = new Set<PaymentOpenErrorCode>([
  "invalid_request",
  "unauthenticated",
  "unsupported_payment_method",
  "idempotency_conflict",
  "merchant_reference_conflict",
  "quote_unavailable",
  "quote_expired",
  "quote_mismatched",
  "deposit_address_unavailable",
  "storage_unavailable",
]);
