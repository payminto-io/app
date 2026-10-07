import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "./errors";

const apiFetch = vi.fn();
vi.mock("./client", () => ({ apiFetch: (...args: unknown[]) => apiFetch(...args) }));

import { attestationsApi, explorerAddressUrl, explorerTxUrl } from "./attestations";

describe("attestationsApi.status", () => {
  beforeEach(() => apiFetch.mockReset());

  it("treats a 404 as the module being off, and nothing else", async () => {
    apiFetch.mockRejectedValueOnce(new ApiError(404, "Not Found"));
    await expect(attestationsApi.status()).resolves.toBeNull();
    expect(apiFetch).toHaveBeenCalledWith("/cre/status");

    apiFetch.mockRejectedValueOnce(new ApiError(503, "down"));
    await expect(attestationsApi.status()).rejects.toBeInstanceOf(ApiError);

    apiFetch.mockResolvedValueOnce({ enabled: true, provider: "mock", workflows: [] });
    await expect(attestationsApi.status()).resolves.toMatchObject({ provider: "mock" });
  });
});

describe("explorer links", () => {
  it("links only chains it knows and never guesses", () => {
    expect(explorerAddressUrl("ethereum-testnet-sepolia-base-1", "0xabc")).toBe("https://sepolia.basescan.org/address/0xabc");
    expect(explorerTxUrl("ethereum-mainnet-base-1", "0x1")).toBe("https://basescan.org/tx/0x1");
    expect(explorerAddressUrl("some-other-chain", "0xabc")).toBeNull();
    expect(explorerTxUrl("ethereum-mainnet-base-1", null)).toBeNull();
    expect(explorerAddressUrl("ethereum-mainnet-base-1", "")).toBeNull();
  });
});
