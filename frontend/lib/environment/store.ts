/**
 * Dashboard environment (test or live). Live and test are isolated at boot on
 * the backend (CLAUDE.md); the dashboard mirrors that as a visible, explicit
 * switch. Live is selectable only when the deployment says it exists.
 */
import { create } from "zustand";

export type Environment = "test" | "live";

export const LIVE_ENABLED = process.env.NEXT_PUBLIC_LIVE_ENABLED === "true";

interface EnvironmentState {
  environment: Environment;
  setEnvironment: (env: Environment) => void;
}

export const useEnvironment = create<EnvironmentState>((set) => ({
  environment: "test",
  setEnvironment: (environment) => {
    if (environment === "live" && !LIVE_ENABLED) return;
    set({ environment });
  },
}));
