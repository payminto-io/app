"use client";

/**
 * Root QueryClientProvider + auth bootstrap.
 *
 * On mount, fires a GET /api/auth/me via the Next.js route handler to
 * detect a pre-existing session (httpOnly refresh cookie). On success the
 * returned member is written into the Zustand store so subsequent
 * navigations know the user is signed in. Failure is silent — the user
 * stays "anonymous" and proxy.ts will redirect them on any protected nav.
 */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { authStore } from "@/lib/auth/store";

interface MeResponse {
  accessToken: string;
  member: {
    id: number;
    email: string;
    name: string;
    memberType?: string;
    externalPlatformID?: number;
  };
}

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [client] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            refetchOnWindowFocus: false,
            retry: (failureCount, err) => {
              if (failureCount >= 2) return false;
              const status =
                typeof err === "object" &&
                err !== null &&
                "status" in err &&
                typeof (err as { status?: unknown }).status === "number"
                  ? (err as { status: number }).status
                  : 0;
              return status !== 401 && status !== 403 && status !== 404;
            },
          },
        },
      })
  );

  useEffect(() => {
    let cancelled = false;
    fetch("/api/auth/me", { credentials: "include", cache: "no-store" })
      .then(async (res) => {
        if (!res.ok) return;
        const data = (await res.json()) as MeResponse;
        if (cancelled) return;
        if (!data.accessToken || !data.member) return;
        authStore.getState().setSession({
          accessToken: data.accessToken,
          member: data.member,
        });
      })
      .catch(() => undefined);
    return () => {
      cancelled = true;
    };
  }, []);

  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}
