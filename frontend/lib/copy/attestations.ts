/**
 * Copy for /dashboard/settings/attestations. Sentences in sentence case; the page
 * never shows a figure the module did not record (docs/cre/SPEC.md section 2).
 */
export const ATTESTATIONS_COPY = {
  title: "Attestations",
  tab: "Attestations",
  explain:
    "An attestation is a second, operator-independent signature on numbers the gateway already holds: ledger liabilities against custody reserves, the finality of a deposit, and the reference rate behind a conversion. It is for deployers that hold balances or settle at size; a merchant paid straight to their own wallet has nothing to attest.",
  off: {
    heading: "Attestations are off",
    body: "This gateway runs without the Chainlink CRE module, which is the default and complete on its own.",
    howTo: "To turn it on, set CRE_ENABLED=true with a provider and restart; docs/cre/OPERATIONS.md walks through mock and Chainlink.",
  },
  provider: {
    label: "Provider",
    none: "Off",
    mock: "Mock",
    chainlink: "Chainlink CRE",
    mockNote: "Mock attestations are signed by a key inside this gateway. They prove the pipeline works, not that anyone independent checked the numbers.",
    degraded: (missing: string[]) =>
      `Chainlink was configured but ${missing.join(", ")} ${missing.length === 1 ? "is" : "are"} missing, so this process fell back to the mock provider. Live would refuse to start.`,
  },
  fields: {
    health: "Health",
    environment: "Environment",
    chain: "Attestation chain",
    consumer: "Consumer contract",
    forwarder: "Forwarder",
    owner: "Workflow owner",
    signer: "Trigger signer",
    signerAddress: "Signer address",
    gatewayId: "Gateway id",
    publicBase: "Public base URL",
    publicVerify: "Public verification",
    publicVerifyOn: "On: anyone can open a record by id",
    publicVerifyOff: "Off",
    explorer: "View on explorer",
  },
  workflows: {
    heading: "Workflows",
    description: "One row per workflow: when it last ran, what it last recorded, and whether the record is still fresh.",
    columns: {
      workflow: "Workflow",
      credential: "Credential",
      lastRun: "Last run",
      lastAttestation: "Last attestation",
      state: "State",
    },
    names: {
      solvency: "Solvency",
      deposit_finality: "Deposit finality",
      conversion_reference: "Conversion reference",
    } as Record<string, string>,
    every: (interval: string) => `Every ${interval}`,
    credentialSet: "Set",
    credentialMissing: "Not set",
    never: "No run yet",
    noRecord: "No record yet",
    state: {
      never: "Never attested",
      fresh: "Fresh",
      stale: "Stale",
    } as Record<string, string>,
  },
  error: "The attestation status could not be loaded.",
} as const;

export function providerLabel(provider: string): string {
  const p = ATTESTATIONS_COPY.provider;
  if (provider === "mock") return p.mock;
  if (provider === "chainlink") return p.chainlink;
  return p.none;
}

export function intervalLabel(seconds: number): string {
  if (seconds % 3600 === 0) return `${seconds / 3600} h`;
  if (seconds % 60 === 0) return `${seconds / 60} min`;
  return `${seconds} s`;
}
