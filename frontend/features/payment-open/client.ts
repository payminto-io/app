import {
  PaymentOpenError,
  assertCommand,
  knownErrorCodes,
  normalizeOpenPayment,
  type OpenPayment,
  type OpenPaymentCommand,
  type PaymentOpenErrorCode,
} from "./model";

export type PaymentOpenClient = {
  open(command: OpenPaymentCommand): Promise<OpenPayment>;
};

type StringStore = Pick<Storage, "getItem" | "setItem">;

export type PaymentOpenClientOptions = {
  endpoint?: string;
  fetch?: typeof globalThis.fetch;
  idempotencyStore?: StringStore;
  generateIdempotencyKey?: () => string;
};

export function createPaymentOpenClient(options: PaymentOpenClientOptions = {}): PaymentOpenClient {
  const endpoint = options.endpoint ?? "/api/v1/payments/open";
  const transport = options.fetch ?? globalThis.fetch.bind(globalThis);
  const memory = new Map<string, string>();
  const store = options.idempotencyStore ?? browserStore() ?? mapStore(memory);
  const generate = options.generateIdempotencyKey ?? defaultIdempotencyKey;

  return {
    async open(command) {
      assertCommand(command);
      const storageKey = `payment-open:v1:${encodeURIComponent(canonicalCommand(command))}`;
      let idempotencyKey = store.getItem(storageKey);
      if (idempotencyKey === null) {
        idempotencyKey = generate();
        assertIdempotencyKey(idempotencyKey);
        store.setItem(storageKey, idempotencyKey);
      } else {
        assertIdempotencyKey(idempotencyKey);
      }

      let response: Response;
      try {
        response = await transport(endpoint, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "Idempotency-Key": idempotencyKey,
          },
          body: JSON.stringify(command),
        });
      } catch {
        throw new PaymentOpenError("transport_unavailable");
      }

      const body = await safeJSON(response);
      if (response.status !== 201) throw normalizeError(body);
      return normalizeOpenPayment(body, command);
    },
  };
}

function canonicalCommand(command: OpenPaymentCommand): string {
  return JSON.stringify({
    merchantReference: command.merchantReference,
    invoiceAmount: {
      currency: command.invoiceAmount.currency,
      minorUnits: command.invoiceAmount.minorUnits,
    },
    paymentMethod: {
      chainId: command.paymentMethod.chainId,
      assetId: command.paymentMethod.assetId,
    },
    expiresAt: command.expiresAt,
  });
}

function browserStore(): StringStore | undefined {
  try {
    return typeof sessionStorage === "undefined" ? undefined : sessionStorage;
  } catch {
    return undefined;
  }
}

function mapStore(values: Map<string, string>): StringStore {
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => void values.set(key, value),
  };
}

function defaultIdempotencyKey(): string {
  if (typeof crypto?.randomUUID !== "function") {
    throw new PaymentOpenError("transport_unavailable");
  }
  return `open_${crypto.randomUUID()}`;
}

function assertIdempotencyKey(key: string): void {
  if (key.length < 16 || key.length > 128 || !/^[!-~]+$/.test(key)) {
    throw new PaymentOpenError("unsafe_contract");
  }
}

async function safeJSON(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    throw new PaymentOpenError("unsafe_contract");
  }
}

function normalizeError(payload: unknown): PaymentOpenError {
  if (typeof payload !== "object" || payload === null || Array.isArray(payload)) {
    return new PaymentOpenError("unsafe_contract");
  }
  const error = (payload as Record<string, unknown>).error;
  if (typeof error !== "object" || error === null || Array.isArray(error)) {
    return new PaymentOpenError("unsafe_contract");
  }
  const code = (error as Record<string, unknown>).code;
  if (typeof code !== "string" || !knownErrorCodes.has(code as PaymentOpenErrorCode)) {
    return new PaymentOpenError("unsafe_contract");
  }
  return new PaymentOpenError(code as PaymentOpenErrorCode);
}
