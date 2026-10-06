/**
 * Configurations (admin runtime config) domain API.
 * Routes: /admin/configurations (list), /admin/configurations/:key (get/put).
 *
 * Backend wraps list responses as { configurations: [...] }; we unwrap.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface Configuration {
  key: string;
  value: string;
  description?: string;
  category?: string;
  updatedAt: string;
}

export const configurationsApi = {
  list: async (): Promise<Configuration[]> => {
    try {
      const res = await apiFetch<{ configurations: Configuration[] }>(
        "/admin/configurations"
      );
      return res.configurations ?? [];
    } catch (err) {
      if (
        isApiError(err) &&
        (err.isNotFound || err.isForbidden || err.isUnauthorized)
      ) {
        return [];
      }
      throw err;
    }
  },
  get: (key: string) =>
    apiFetch<Configuration>(
      `/admin/configurations/${encodeURIComponent(key)}`
    ),
  set: (key: string, value: string) =>
    apiFetch<Configuration>(
      `/admin/configurations/${encodeURIComponent(key)}`,
      { method: "PUT", body: { value } }
    ),
};
