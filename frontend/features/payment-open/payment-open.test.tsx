import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

import {
  PaymentOpenCheckout,
  createPaymentOpenClient,
  type OpenPaymentCommand,
} from "./index";

const API = "http://localhost:8080/api/v1/payments/open";
const server = setupServer();
const command: OpenPaymentCommand = {
  merchantReference: "order-2026-0042",
  invoiceAmount: { currency: "USD", minorUnits: "900719925474099312345" },
  paymentMethod: {
    chainId: "eip155:1",
    assetId: "eip155:1/erc20:0xdac17f958d2ee523a2206206994597c13d831ec7",
  },
  expiresAt: "2026-08-27T15:30:00Z",
};

const paymentWire = {
  invoiceId: "inv_01K4Y3Q9",
  merchantReference: command.merchantReference,
  invoiceAmount: command.invoiceAmount,
  paymentMethod: command.paymentMethod,
  quote: {
    id: "qte_01K4Y3QA",
    invoiceCurrency: "USD",
    invoiceMinorUnits: command.invoiceAmount.minorUnits,
    chainId: command.paymentMethod.chainId,
    assetId: command.paymentMethod.assetId,
    requiredAtomicUnits: "184467440737095516151234",
    assetDecimals: 6,
    rateNumerator: "999999999999999999",
    rateDenominator: "1000000000000000000",
    source: "merchant-oracle-v1",
    quotedAt: "2026-08-27T15:00:00Z",
    expiresAt: "2026-08-27T15:05:00Z",
    rounding: "ceiling",
  },
  depositAddress: {
    assignmentId: "addr_assign_01K4",
    address: "0x1111111111111111111111111111111111111111",
  },
  state: "open",
  revision: 1,
  openedAt: "2026-08-27T15:00:00Z",
  expiresAt: command.expiresAt,
};

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  sessionStorage.clear();
  vi.restoreAllMocks();
});
afterAll(() => server.close());

describe("PaymentLifecycle.Open HTTP adapter seam", () => {
  it("sends exact string money and persists one Idempotency-Key for retries of the same command", async () => {
    const keys: string[] = [];
    const bodies: unknown[] = [];
    server.use(
      http.post(API, async ({ request }) => {
        keys.push(request.headers.get("Idempotency-Key") ?? "");
        bodies.push(await request.json());
        return HttpResponse.json({ payment: paymentWire }, { status: 201 });
      }),
    );
    const client = createPaymentOpenClient({ endpoint: API });

    await expect(client.open(command)).resolves.toMatchObject({
      quote: { requiredAtomicUnits: "184467440737095516151234" },
    });
    await client.open(command);

    expect(bodies).toEqual([command, command]);
    expect(keys[0]).toMatch(/^open_[!-~]{16,}$/);
    expect(keys[1]).toBe(keys[0]);
  });

  it("rejects numeric money, mismatched identities, unknown chains, and unknown states", async () => {
    server.use(
      http.post(API, () =>
        HttpResponse.json(
          {
            payment: {
              ...paymentWire,
              quote: { ...paymentWire.quote, requiredAtomicUnits: 42 },
            },
          },
          { status: 201 },
        ),
      ),
    );
    const client = createPaymentOpenClient({ endpoint: API });
    await expect(client.open(command)).rejects.toMatchObject({ code: "unsafe_contract" });

    await expect(
      client.open({
        ...command,
        invoiceAmount: { currency: "USD", minorUnits: 42 as unknown as string },
      }),
    ).rejects.toMatchObject({ code: "unsafe_contract" });
    await expect(
      client.open({
        ...command,
        paymentMethod: {
          chainId: "eip155:999999" as OpenPaymentCommand["paymentMethod"]["chainId"],
          assetId: "eip155:999999/slip44:60",
        },
      }),
    ).rejects.toMatchObject({ code: "unsafe_contract" });

    server.use(
      http.post(API, () =>
        HttpResponse.json({ payment: { ...paymentWire, state: "confirming" } }, { status: 201 }),
      ),
    );
    await expect(client.open({ ...command, merchantReference: "order-state" })).rejects.toMatchObject({
      code: "unsafe_contract",
    });
  });
});

describe("authoritative checkout presentation seam", () => {
  it("renders immutable server values, neutral pending language, QR, copy, and server expiry", async () => {
    server.use(http.post(API, () => HttpResponse.json({ payment: paymentWire }, { status: 201 })));
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });

    render(<PaymentOpenCheckout command={command} client={createPaymentOpenClient({ endpoint: API })} />);

    expect(screen.getByRole("status")).toHaveTextContent("Preparing secure payment instructions");
    expect(await screen.findByRole("heading", { name: "Awaiting payment" })).toBeInTheDocument();
    expect(screen.getByText("184467440737095516151234")).toBeInTheDocument();
    expect(screen.getByText(command.paymentMethod.assetId)).toBeInTheDocument();
    expect(screen.getByText("Ethereum mainnet")).toBeInTheDocument();
    expect(screen.getByText("27 Aug 2026, 3:05 pm UTC")).toBeInTheDocument();
    expect(screen.queryByText(/successful|confirmed/i)).not.toBeInTheDocument();
    expect(screen.getByTitle("Deposit address QR code")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Copy deposit address" }));
    await waitFor(() => expect(writeText).toHaveBeenCalledWith(paymentWire.depositAddress.address));
  });

  it.each([
    [401, "unauthenticated", "Access required"],
    [422, "unsupported_payment_method", "Payment method not configured"],
    [503, "quote_unavailable", "Payment service degraded"],
    [409, "idempotency_conflict", "We could not prepare this payment"],
  ])("maps %s/%s to one safe state", async (status, code, heading) => {
    server.use(
      http.post(API, () =>
        HttpResponse.json({ error: { code, message: "internal prose must not drive UI" } }, { status }),
      ),
    );
    render(<PaymentOpenCheckout command={command} client={createPaymentOpenClient({ endpoint: API })} />);
    expect(await screen.findByRole("heading", { name: heading })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Awaiting payment" })).not.toBeInTheDocument();
  });
});
