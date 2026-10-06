/**
 * In-memory auth store.
 *
 * Holds the access token and the authenticated member profile for the
 * current browser tab. This store is intentionally **memory-only** —
 * never localStorage, never sessionStorage. The long-lived refresh token
 * lives in an httpOnly cookie set by the Next.js route handler at
 * `app/api/auth/signin/route.ts` and is never readable from JS.
 *
 * See ADR-0008 and FRONTEND_CODING_STANDARDS §6.
 */
import { create } from "zustand";

export interface AuthMember {
  id: number;
  email: string;
  name: string;
  memberType?: string;
  externalPlatformID?: number;
}

interface AuthState {
  accessToken: string | null;
  member: AuthMember | null;
  setAccessToken: (token: string) => void;
  setSession: (input: { accessToken: string; member: AuthMember }) => void;
  clear: () => void;
}

export const authStore = create<AuthState>((set) => ({
  accessToken: null,
  member: null,
  setAccessToken: (token) => set({ accessToken: token }),
  setSession: ({ accessToken, member }) => set({ accessToken, member }),
  clear: () => set({ accessToken: null, member: null }),
}));

/** Convenience React hook — re-exported for component ergonomics. */
export const useAuth = authStore;
