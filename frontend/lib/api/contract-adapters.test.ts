import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";

import {
  normalizeCreatedWithdrawal,
  normalizeWithdrawal,
  toCreateWithdrawalRequest,
} from "./withdrawal-contract";
import { withdrawalsApi } from "./withdrawals";
import {
  normalizeCreatedWebhook,
  normalizeWebhook,
  unwrapWebhook,
} from "./webhook-contract";
import { webhooksApi } from "./webhooks";

const API = "http://localhost:8080/api/v1";
const server = setupServer();

const withdrawalWire = {
  id: 17,
  externalPlatformID: 41,
  memberID: 3,
  referenceID: "wd_17",
  toAddress: "0xabc",
  blockchainCode: "ETH",
  currencyCode: "USDT",
  amount: "12.5",
  state: "pending-approval",
  createdAt: "2026-08-26T00:00:00Z",
  updatedAt: "2026-08-26T00:00:00Z",
};

const webhookWire = {
  id: 9,
  externalPlatformID: 41,
  url: "https://merchant.example/hooks",
  events: ["payment.confirmed", "withdrawal.sent"],
  active: true,
  createdAt: "2026-08-26T00:00:00Z",
  updatedAt: "2026-08-26T00:00:00Z",
};

vi.mock("./client", () => ({
  apiFetch: async (path: string, init: { method?: string; body?: unknown } = {}) => {
    const response = await fetch(`http://localhost:8080/api/v1${path}`, {
      method: init.method,
      headers: init.body === undefined ? undefined : { "Content-Type": "application/json" },
      body: init.body === undefined ? undefined : JSON.stringify(init.body),
    });
    if (!response.ok) {
      const error = new Error(response.statusText) as Error & { status: number };
      error.status = response.status;
      throw error;
    }
    return response.status === 204 ? undefined : response.json();
  },
}));

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe("withdrawals HTTP contract", () => {
  it("sends toAddress and unwraps the created withdrawal", async () => {
    const request = toCreateWithdrawalRequest({
      recipientAddress: "0xabc",
      blockchainCode: "ETH",
      currencyCode: "USDT",
      amount: "12.5",
    });
    expect(request).toEqual({
      toAddress: "0xabc",
      blockchainCode: "ETH",
      currencyCode: "USDT",
      amount: "12.5",
    });

    const result = normalizeCreatedWithdrawal({
      withdrawal: {
        id: 17,
        externalPlatformID: 41,
        memberID: 3,
        referenceID: "wd_17",
        toAddress: "0xabc",
        blockchainCode: "ETH",
        currencyCode: "USDT",
        amount: "12.5",
        state: "pending-approval",
        createdAt: "2026-08-26T00:00:00Z",
        updatedAt: "2026-08-26T00:00:00Z",
      },
      otp: { otpRequired: true },
    });
    expect(result.withdrawal.id).toBe(17);
    expect(result.withdrawal.recipientAddress).toBe("0xabc");
    expect(result.withdrawal.state).toBe("pending-approval");
    expect(result.otp.otpRequired).toBe(true);
  });

  it("rejects numeric money and non-canonical states", () => {
    const base = {
      id: 17,
      externalPlatformID: 41,
      memberID: 3,
      toAddress: "0xabc",
      blockchainCode: "ETH",
      currencyCode: "USDT",
      amount: "12.5",
      state: "pending-approval",
      createdAt: "2026-08-26T00:00:00Z",
      updatedAt: "2026-08-26T00:00:00Z",
    };

    expect(() => normalizeWithdrawal({ ...base, amount: 12.5 })).toThrow(
      "Invalid withdrawal amount"
    );
    expect(() =>
      toCreateWithdrawalRequest({
        recipientAddress: "0xabc",
        blockchainCode: "ETH",
        currencyCode: "USDT",
        amount: 12.5 as unknown as string,
      })
    ).toThrow("Invalid withdrawal amount");
    expect(() => normalizeWithdrawal({ ...base, state: "processing" })).toThrow(
      "Invalid withdrawal state"
    );
  });

  it("does not send unsupported filters or invent a total", async () => {
    let requestURL = "";
    server.use(
      http.get(`${API}/withdrawal/merchant`, ({ request }) => {
        requestURL = request.url;
        return HttpResponse.json({ withdrawals: [] });
      })
    );

    await expect(withdrawalsApi.list()).resolves.toEqual({ withdrawals: [] });
    expect(requestURL).toBe(`${API}/withdrawal/merchant`);
  });

  it("creates a payout using the public payload and preserves a no-OTP prompt", async () => {
    let requestBody: unknown;
    server.use(
      http.post(`${API}/withdrawal/merchant`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json(
          { withdrawal: withdrawalWire, otp: { otpRequired: false } },
          { status: 201 }
        );
      })
    );

    const result = await withdrawalsApi.create({
      recipientAddress: "0xabc",
      blockchainCode: "ETH",
      currencyCode: "USDT",
      amount: "12.5",
      memo: "invoice 42",
    });

    expect(requestBody).toEqual({
      toAddress: "0xabc",
      blockchainCode: "ETH",
      currencyCode: "USDT",
      amount: "12.5",
      memo: "invoice 42",
    });
    expect(result.otp).toEqual({ otpRequired: false });
    expect(result.withdrawal.state).toBe("pending-approval");
  });

  it("round-trips OTP, approval, and cancellation message responses", async () => {
    server.use(
      http.post(`${API}/withdrawal/17/otp/verify`, async ({ request }) => {
        expect(await request.json()).toEqual({ code: "123456" });
        return HttpResponse.json({ message: "OTP verified, withdrawal pending approval" });
      }),
      http.post(`${API}/withdrawal/17/approve`, () =>
        HttpResponse.json({ message: "withdrawal approved" })
      ),
      http.post(`${API}/withdrawal/17/cancel`, () =>
        HttpResponse.json({ message: "withdrawal cancelled" })
      )
    );

    await expect(withdrawalsApi.verifyOTP(17, "123456")).resolves.toEqual({
      message: "OTP verified, withdrawal pending approval",
    });
    await expect(withdrawalsApi.approve(17)).resolves.toEqual({
      message: "withdrawal approved",
    });
    await expect(withdrawalsApi.cancel(17)).resolves.toEqual({
      message: "withdrawal cancelled",
    });
  });
});

describe("webhooks HTTP contract", () => {
  it("unwraps webhook detail envelopes and preserves event arrays", async () => {
    const result = unwrapWebhook({
      webhook: {
        id: 9,
        externalPlatformID: 41,
        url: "https://merchant.example/hooks",
        events: ["payment.confirmed", "withdrawal.sent"],
        active: true,
        createdAt: "2026-08-26T00:00:00Z",
        updatedAt: "2026-08-26T00:00:00Z",
      },
    });

    expect(result.id).toBe(9);
    expect(result.events).toEqual(["payment.confirmed", "withdrawal.sent"]);
  });

  it("rejects non-canonical event arrays", () => {
    const base = {
      id: 9,
      externalPlatformID: 41,
      url: "https://merchant.example/hooks",
      active: true,
      createdAt: "2026-08-26T00:00:00Z",
      updatedAt: "2026-08-26T00:00:00Z",
    };

    expect(() => normalizeWebhook({ ...base, events: [" payment.confirmed"] })).toThrow(
      "Invalid webhook events"
    );
    expect(() => normalizeWebhook({ ...base, events: ["payment.confirmed,payment.failed"] })).toThrow(
      "Invalid webhook events"
    );
    expect(() => normalizeWebhook({ ...base, events: [] })).toThrow(
      "Invalid webhook events"
    );
  });

  it("propagates list and delivery 404s as unhealthy responses", async () => {
    server.use(
      http.get(`${API}/webhooks`, () =>
        HttpResponse.json({ error: "route unavailable" }, { status: 404 })
      ),
      http.get(`${API}/webhooks/9/deliveries`, () =>
        HttpResponse.json({ error: "route unavailable" }, { status: 404 })
      )
    );

    await expect(webhooksApi.list()).rejects.toMatchObject({ status: 404 });
    await expect(webhooksApi.deliveries(9)).rejects.toMatchObject({ status: 404 });
  });

  it("round-trips canonical create payloads and requires the reveal-once secret", async () => {
    let requestBody: unknown;
    server.use(
      http.post(`${API}/webhooks`, async ({ request }) => {
        requestBody = await request.json();
        return HttpResponse.json(
          {
            webhook: {
              ...webhookWire,
              secret: "whsec_reveal_once",
            },
          },
          { status: 201 }
        );
      })
    );

    const created = await webhooksApi.create({
      url: "https://merchant.example/hooks",
      events: ["payment.confirmed", "withdrawal.sent"],
    });
    expect(requestBody).toEqual({
      url: "https://merchant.example/hooks",
      events: ["payment.confirmed", "withdrawal.sent"],
    });
    expect(created.secret).toBe("whsec_reveal_once");
  });

  it("requires a signing secret in the create response", () => {
    expect(() => normalizeCreatedWebhook(webhookWire)).toThrow(
      "Invalid created webhook secret"
    );
  });

  it("round-trips redacted list, get, and update envelopes", async () => {
    let updateBody: unknown;
    server.use(
      http.get(`${API}/webhooks`, () =>
        HttpResponse.json({ webhooks: [webhookWire] })
      ),
      http.get(`${API}/webhooks/9`, () =>
        HttpResponse.json({ webhook: webhookWire })
      ),
      http.put(`${API}/webhooks/9`, async ({ request }) => {
        updateBody = await request.json();
        return HttpResponse.json({
          webhook: { ...webhookWire, active: false },
        });
      })
    );

    await expect(webhooksApi.list()).resolves.toEqual([webhookWire]);
    await expect(webhooksApi.get(9)).resolves.toEqual(webhookWire);
    await expect(webhooksApi.update(9, { active: false })).resolves.toEqual({
      ...webhookWire,
      active: false,
    });
    expect(updateBody).toEqual({ active: false });
  });

  it("fails closed when list, get, or update leaks a secret", async () => {
    const leaked = { ...webhookWire, secret: "must-not-leak" };
    server.use(
      http.get(`${API}/webhooks`, () =>
        HttpResponse.json({ webhooks: [leaked] })
      ),
      http.get(`${API}/webhooks/9`, () =>
        HttpResponse.json({ webhook: leaked })
      ),
      http.put(`${API}/webhooks/9`, () =>
        HttpResponse.json({ webhook: leaked })
      )
    );

    await expect(webhooksApi.list()).rejects.toThrow("Unexpected webhook secret");
    await expect(webhooksApi.get(9)).rejects.toThrow("Unexpected webhook secret");
    await expect(webhooksApi.update(9, { active: false })).rejects.toThrow(
      "Unexpected webhook secret"
    );
  });
});
