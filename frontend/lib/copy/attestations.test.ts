import { describe, expect, it } from "vitest";

import { ATTESTATIONS_COPY, intervalLabel, providerLabel } from "./attestations";

describe("attestations copy", () => {
  it("explains the module in at most two sentences", () => {
    const sentences = ATTESTATIONS_COPY.explain.split(/(?<=[.!?])\s+/).filter(Boolean);
    expect(sentences.length).toBeLessThanOrEqual(2);
    expect(ATTESTATIONS_COPY.explain).toContain("nothing to attest");
  });

  it("names providers and intervals plainly", () => {
    expect(providerLabel("none")).toBe("Off");
    expect(providerLabel("mock")).toBe("Mock");
    expect(providerLabel("chainlink")).toBe("Chainlink CRE");
    expect(intervalLabel(3600)).toBe("1 h");
    expect(intervalLabel(60)).toBe("1 min");
    expect(intervalLabel(45)).toBe("45 s");
    expect(ATTESTATIONS_COPY.provider.degraded(["CRE_CONSUMER_ADDRESS"])).toContain("CRE_CONSUMER_ADDRESS is missing");
  });
});
