/**
 * Roles + permissions domain API.
 * Routes: /admin/roles, /admin/permissions.
 *
 * Backend wraps list responses as { roles: [...] } and { permissions: [...] };
 * we unwrap to return plain arrays.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface Role {
  id: number;
  name: string;
  description?: string;
  builtIn: boolean;
  permissions: string[];
}

export interface Permission {
  id: number;
  name: string;
  description?: string;
  category?: string;
}

export const rolesApi = {
  list: async (): Promise<Role[]> => {
    try {
      const res = await apiFetch<{ roles: Role[] }>("/admin/roles");
      return res.roles ?? [];
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
  get: (id: number) => apiFetch<Role>(`/admin/roles/${id}`),
  setPermissions: (id: number, permissions: string[]) =>
    apiFetch<Role>(`/admin/roles/${id}/permissions`, {
      method: "PUT",
      body: { permissions },
    }),
};

export const permissionsApi = {
  list: async (): Promise<Permission[]> => {
    try {
      const res = await apiFetch<{ permissions: Permission[] }>(
        "/admin/permissions"
      );
      return res.permissions ?? [];
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
};
