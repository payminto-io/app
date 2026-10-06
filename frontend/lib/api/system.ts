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
 * Rather than crash the admin system page on "Not Found", we synthesise a
 * SystemInfo object from the health response and treat the per-worker
 * controls as a gracefully-degraded feature: mutations throw a typed
 * "not available" error instead of bubbling 404 Not Found into the UI
 * error boundary.
 */
import { apiFetch } from "./client";
import { ApiError, isApiError } from "./errors";

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

// featureNotAvailable builds a typed 501 error naming the worker the caller
// tried to control, so the UI can show a specific toast.
const featureNotAvailable = (name: string) =>
  new ApiError(
    501,
    `Per-worker control for "${name}" is not available in this backend build. Use the stop-all endpoint or restart the server process.`
  );

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

  // The per-worker control endpoints are not yet implemented on the backend.
  // Surfacing a typed "not available" error lets the UI render a toast
  // instead of a "Not Found" error page.
  startWorker: async (name: string): Promise<WorkerStatus> => {
    throw featureNotAvailable(name);
  },
  stopWorker: async (name: string): Promise<WorkerStatus> => {
    throw featureNotAvailable(name);
  },
  restartWorker: async (name: string): Promise<WorkerStatus> => {
    throw featureNotAvailable(name);
  },
};
