/**
 * API keys domain API — merchant self-service.
 *
 * Backend routes (JWT or API key auth):
 *   GET    /api/v1/api-keys              list keys for caller's platform
 *   POST   /api/v1/api-keys              create new key (returns rawKey once)
 *   POST   /api/v1/api-keys/:id/revoke   revoke existing key
 */
import { apiFetch } from "./client";
import { ApiError, isApiError } from "./errors";

export interface APIKey {
  id: number;
  name: string;
  prefix: string;
  active: boolean;
  status: string;
  externalPlatformID: number;
  memberID?: number;
  description?: string;
  lastUsedAt?: string;
  createdAt: string;
  updatedAt?: string;
  rawKey?: string; // revealed once on create/regenerate
}

export interface CreateAPIKeyInput {
  name: string;
  roleID?: number;
}

function isNotFound(err: unknown): boolean {
  return isApiError(err) && err.isNotFound;
}

export const apiKeysApi = {
  list: async (): Promise<APIKey[]> => {
    try {
      const resp = await apiFetch<{ apiKeys: APIKey[] }>("/api-keys");
      return resp.apiKeys ?? [];
    } catch (err) {
      if (isNotFound(err)) return [];
      throw err;
    }
  },
  create: (input: CreateAPIKeyInput): Promise<APIKey> =>
    apiFetch<APIKey>("/api-keys", { method: "POST", body: input }),
  revoke: (id: number): Promise<APIKey> =>
    apiFetch<APIKey>(`/api-keys/${id}/revoke`, { method: "POST" }),
  regenerate: async (id: number): Promise<APIKey> => {
    // Regenerate = revoke old + create new (backend doesn't have a single endpoint)
    await apiFetch<APIKey>(`/api-keys/${id}/revoke`, { method: "POST" });
    return apiFetch<APIKey>("/api-keys", {
      method: "POST",
      body: { name: "Regenerated" },
    });
  },
};

// Re-export for any callers that still import ApiError from this module
export { ApiError };
