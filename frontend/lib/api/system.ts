/**
 * System (admin) domain API — workers + health.
 *
 * Backend routes (see internal/api/router.go, gated by system.admin):
 *   GET  /admin/system/workers       -> { workers: WorkerStatus[] }
 *   POST /admin/system/workers/stop-all
 *   GET  /admin/system/health        -> { status, workers, dbStatus }
 *
 * The backend does NOT currently expose:
 *   - /admin/system/info
 *   - per-worker start/stop/restart endpoints
 *
 * SystemInfo is synthesised from the health response. Per-worker controls are
 * not offered in the UI until the backend implements them.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface SystemInfo {
  version: string;
  commit: string;
  buildTime: string;
  uptime: string;
  mode: "testnet" | "mainnet" | "unknown";
  dbStatus: string;
  overallStatus: string;
}

export interface WorkerStatus {
  name: string;
  running: boolean;
  lastStartedAt?: string;
  lastStoppedAt?: string;
  lastError?: string;
}

/** Backend /admin/system/health response shape. */
interface BackendHealth {
  status: string;
  dbStatus: string;
  workers: Array<{
    name: string;
    running: boolean;
    startedAt?: string;
    stoppedAt?: string;
    error?: string;
  }>;
  // Optional extras the backend may add later.
  version?: string;
  commit?: string;
  buildTime?: string;
  uptime?: string;
  mode?: "testnet" | "mainnet";
}

export const systemApi = {
  /**
   * Compose SystemInfo from /admin/system/health — there is no dedicated
   * /admin/system/info endpoint on the backend yet.
   */
  info: async (): Promise<SystemInfo> => {
    try {
      const h = await apiFetch<BackendHealth>("/admin/system/health");
      return {
        version: h.version ?? "dev",
        commit: h.commit ?? "unknown",
        buildTime: h.buildTime ?? "-",
        uptime: h.uptime ?? "-",
        mode: h.mode ?? "unknown",
        dbStatus: h.dbStatus ?? "unknown",
        overallStatus: h.status ?? "unknown",
      };
    } catch (err) {
      if (isApiError(err) && err.isNotFound) {
        // Health endpoint itself is missing — return a placeholder so the
        // page still renders instead of hitting the error boundary.
        return {
          version: "dev",
          commit: "unknown",
          buildTime: "-",
          uptime: "-",
          mode: "unknown",
          dbStatus: "unknown",
          overallStatus: "unknown",
        };
      }
      throw err;
    }
  },

  workers: async (): Promise<WorkerStatus[]> => {
    try {
      const res = await apiFetch<{ workers: BackendHealth["workers"] }>(
        "/admin/system/workers"
      );
      return (res.workers ?? []).map((w) => ({
        name: w.name,
        running: w.running,
        lastStartedAt: w.startedAt,
        lastStoppedAt: w.stoppedAt,
        lastError: w.error,
      }));
    } catch (err) {
      if (isApiError(err) && err.isNotFound) return [];
      throw err;
    }
  },

  /** Backend exposes POST /admin/system/workers/stop-all. */
  stopAllWorkers: () =>
    apiFetch<{ message: string }>("/admin/system/workers/stop-all", {
      method: "POST",
    }),
};
