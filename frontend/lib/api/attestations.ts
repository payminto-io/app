/**
 * Attestations (Chainlink CRE) domain API.
 *
 * Backend routes (internal/api/routes_cre.go), mounted only when the module is on:
 *   GET /api/v1/cre/status   -> AttestationStatus (snake_case)
 * A 404 means the module is off: the route does not exist, by design.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface AttestationRun {
  kind: string;
  provider: string;
  execution_id: string;
  status: "accepted" | "failed" | string;
  detail: string;
  started_at: string | null;
}

export interface AttestationRecord {
  id: string;
  kind: string;
  subject_type: string;
  subject_id: string;
  status: "attested" | "mismatch" | "ignored" | "failed" | "pending" | "stale" | string;
  provider: string;
  simulated: boolean;
  chain: string;
  tx_hash: string | null;
  block_number: number;
  workflow_id: string;
  workflow_owner: string;
  observed_at: string | null;
  recorded_at: string | null;
  reason: string;
  item: Record<string, unknown>;
}

export interface WorkflowStatus {
  kind: string;
  workflow_id: string;
  workflow_name: string;
  interval_seconds: number;
  credential_configured: boolean;
  /** Derived from the latest verified record only; a failed or mismatched row never reads as fresh. */
  state: "never" | "fresh" | "stale" | string;
  last_run: AttestationRun | null;
  /** Newest row of any status. */
  last_attestation: AttestationRecord | null;
  /** Newest attested row. */
  last_verified: AttestationRecord | null;
}

export interface AttestationStatus {
  enabled: boolean;
  provider: "none" | "mock" | "chainlink" | string;
  degraded_from: string;
  missing_keys: string[];
  environment: string;
  chain: string;
  consumer_address: string;
  forwarder_address: string;
  workflow_owner: string;
  trigger_signer: string;
  trigger_signer_address: string;
  gateway_id: string;
  public_base_url: string;
  public_verify_enabled: boolean;
  health: { status: "ok" | "degraded" | "down" | "off" | string; message: string };
  workflows: WorkflowStatus[];
}

/** Explorer address pages for the CRE chain selector names we know; unknown chains get no link. */
const EXPLORERS: Record<string, string> = {
  "ethereum-testnet-sepolia-base-1": "https://sepolia.basescan.org",
  "ethereum-mainnet-base-1": "https://basescan.org",
  "ethereum-testnet-sepolia": "https://sepolia.etherscan.io",
  "ethereum-mainnet": "https://etherscan.io",
};

export function explorerAddressUrl(chain: string, address: string): string | null {
  const base = EXPLORERS[chain];
  return base && address ? `${base}/address/${address}` : null;
}

export function explorerTxUrl(chain: string, tx: string | null): string | null {
  const base = EXPLORERS[chain];
  return base && tx ? `${base}/tx/${tx}` : null;
}

export const attestationsApi = {
  /** Null when the module is off (the route is absent). */
  status: async (): Promise<AttestationStatus | null> => {
    try {
      return await apiFetch<AttestationStatus>("/cre/status");
    } catch (err) {
      if (isApiError(err) && err.isNotFound) return null;
      throw err;
    }
  },
};
