import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { AttestationStatus } from "@/lib/api/attestations";

const status = vi.fn();
vi.mock("@/lib/query/hooks/use-attestations", () => ({ useAttestationStatus: () => status() }));
vi.mock("../_components/settings-tabs", () => ({ SettingsTabs: () => <nav data-testid="tabs" /> }));

import AttestationsSettingsPage from "./page";

const onStatus: AttestationStatus = {
  enabled: true, provider: "mock", degraded_from: "", missing_keys: [], environment: "test", chain: "ethereum-testnet-sepolia-base-1",
  consumer_address: "", forwarder_address: "", workflow_owner: "0x4c1e9d2a7b3f5e8c0a6d1f2b3c4d5e6f7a8b9c0d", trigger_signer: "keyring://cre-trigger",
  trigger_signer_address: "0x4c1e9d2a7b3f5e8c0a6d1f2b3c4d5e6f7a8b9c0d", gateway_id: "0x12", public_base_url: "http://localhost:8090", public_verify_enabled: true,
  health: { status: "ok", message: "mock provider" },
  workflows: [
    {
      kind: "deposit_finality", workflow_id: "0xcd", workflow_name: "7d1a4f2c9b", interval_seconds: 60, credential_configured: true, state: "never",
      last_run: null,
      last_attestation: { id: "r2", kind: "deposit_finality", subject_type: "deposit", subject_id: "dep-1", status: "mismatch", provider: "mock", simulated: false, chain: "x", tx_hash: null, block_number: 0, workflow_id: "0xcd", workflow_owner: "0x4c", observed_at: "2026-10-07T10:00:00Z", recorded_at: "2026-10-07T10:00:00Z", reason: "token differs", item: {} },
      last_verified: null,
    },
  ],
};

describe("attestations settings page", () => {
  it("renders the off state with the explanation when the route is absent", () => {
    status.mockReturnValue({ data: null, error: null, isPending: false, refetch: vi.fn() });
    render(<AttestationsSettingsPage />);
    expect(screen.getByText("Attestations are off")).toBeInTheDocument();
    expect(screen.getByText(/nothing to attest/)).toBeInTheDocument();
    expect(screen.queryByText("Workflows")).toBeNull();
  });

  it("never shows an unverified record as fresh or attested", () => {
    status.mockReturnValue({ data: onStatus, error: null, isPending: false, refetch: vi.fn() });
    render(<AttestationsSettingsPage />);
    expect(screen.getByText("Mock")).toBeInTheDocument();
    expect(screen.getAllByText("Never attested").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Mismatch").length).toBeGreaterThan(0);
    expect(screen.queryByText("Attested")).toBeNull();
    expect(screen.queryByText("Fresh")).toBeNull();
    // Empty addresses are omitted rather than rendered blank.
    expect(screen.queryByText("Consumer contract")).toBeNull();
  });

  it("shows the real error, not a guess, when the status cannot load", () => {
    status.mockReturnValue({ data: undefined, error: new Error("boom"), isPending: false, refetch: vi.fn() });
    render(<AttestationsSettingsPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("The attestation status could not be loaded.");
  });
});
