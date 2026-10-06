/**
 * Members domain API.
 *   - Self-service: /members/me, /members/me/password (PR0)
 *   - Admin: /admin/members (list + get + invite + deactivate)
 *
 * The admin list endpoint (`GET /admin/members`) requires the members.read
 * permission; callers without that grant receive 403, so we downgrade such
 * responses to an empty list so the page shows an empty state instead of
 * an error banner.
 */
import { apiFetch } from "./client";
import { isApiError } from "./errors";

export interface Member {
  id: number;
  email: string;
  name: string;
  memberType?: string;
  externalPlatformID?: number;
  active: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface InviteMemberInput {
  name: string;
  email: string;
  password: string;
  memberType?: string;
}

export interface UpdateProfileInput {
  name?: string;
}

export interface ChangePasswordInput {
  currentPassword: string;
  newPassword: string;
}

export const membersApi = {
  me: () => apiFetch<Member>("/members/me"),
  updateProfile: (input: UpdateProfileInput) =>
    apiFetch<Member>("/members/me", {
      method: "PUT",
      body: input,
    }),
  changeMyPassword: (input: ChangePasswordInput) =>
    apiFetch<{ message: string }>("/members/me/password", {
      method: "PUT",
      body: input,
    }),

  adminList: async (): Promise<Member[]> => {
    try {
      return await apiFetch<Member[]>("/admin/members");
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
  adminGet: (id: number) => apiFetch<Member>(`/admin/members/${id}`),
  adminInvite: (input: InviteMemberInput) =>
    apiFetch<Member>("/admin/members", { method: "POST", body: input }),
  adminDeactivate: (id: number) =>
    apiFetch<{ message: string }>(`/admin/members/${id}`, { method: "DELETE" }),
};
