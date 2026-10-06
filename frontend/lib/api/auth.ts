/**
 * Auth domain API — talks to the Next.js route handlers under /api/auth/*,
 * which in turn proxy to the Go backend at /api/v1/auth/*. The route
 * handlers exist so that the long-lived refresh token can live in an
 * httpOnly cookie that React never touches (see ADR-0008, standards §6).
 *
 * DTO source of truth: payminto/backend/internal/api/handler/auth_handler.go
 */
import type { AuthMember } from "@/lib/auth/store";
import { ApiError } from "./errors";

export interface SigninInput {
  email: string;
  password: string;
}

export interface SigninResponse {
  accessToken: string;
  member: AuthMember;
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const parsed = (await res.json().catch(() => ({}))) as {
      error?: string;
    };
    throw new ApiError(
      res.status,
      parsed.error ?? res.statusText,
      parsed
    );
  }
  return (await res.json()) as T;
}

export interface SignupInput {
  name: string;
  email: string;
  password: string;
}

export async function signin(input: SigninInput): Promise<SigninResponse> {
  return postJSON<SigninResponse>("/api/auth/signin", input);
}

export async function signup(input: SignupInput): Promise<SigninResponse> {
  return postJSON<SigninResponse>("/api/auth/signup", input);
}

export async function signout(): Promise<void> {
  await fetch("/api/auth/signout", {
    method: "POST",
    credentials: "include",
  });
}

export async function me(): Promise<AuthMember> {
  const res = await fetch("/api/auth/me", {
    method: "GET",
    credentials: "include",
  });
  if (!res.ok) {
    throw new ApiError(res.status, res.statusText);
  }
  return (await res.json()) as AuthMember;
}
