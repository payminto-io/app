"use client";

import { useEffect, useState, type ReactNode } from "react";
import { AppShell } from "@/components/layout/app-shell";
import { authStore } from "@/lib/auth/store";
import { API_BASE_URL, API_V2_BASE_URL } from "@/lib/constants";
import { resolveFixture } from "./fixtures";
import { resolveLinksV1, resolveLinksV2 } from "./links-fixtures";

const PREFIX = "/design/preview";

/**
 * Answers API calls made from /design/preview/* with sample fixtures so the
 * real page components render without a backend or a session. `?state=empty`
 * returns empty lists and `?state=error` returns a 503. Nothing outside the
 * preview prefix is intercepted.
 */
function install() {
  const w = window as Window & { __previewFetch?: boolean };
  if (w.__previewFetch) return;
  w.__previewFetch = true;
  const realFetch = window.fetch.bind(window);
  window.fetch = async (input, init) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    const v2 = url.startsWith(API_V2_BASE_URL);
    if (!location.pathname.startsWith(PREFIX) || !(v2 || url.startsWith(API_BASE_URL))) {
      return realFetch(input, init);
    }
    const mode = new URLSearchParams(location.search).get("state");
    const path = url.slice((v2 ? API_V2_BASE_URL : API_BASE_URL).length);
    const method = (init?.method ?? "GET").toUpperCase();
    const reqBody = typeof init?.body === "string" ? init.body : undefined;
    await new Promise((r) => setTimeout(r, 120));
    if (mode === "error") {
      return new Response(JSON.stringify({ error: "Sample error: the service returned 503." }), { status: 503 });
    }
    const linkReply = v2 ? resolveLinksV2(path, method, reqBody, mode === "empty") : resolveLinksV1(path, method, reqBody);
    if (linkReply) {
      return linkReply.status === 204
        ? new Response(null, { status: 204 })
        : new Response(JSON.stringify(linkReply.body), { status: linkReply.status, headers: { "Content-Type": "application/json" } });
    }
    const body = resolveFixture(path, method, mode === "empty");
    if (body === undefined) {
      return new Response(JSON.stringify({ error: `No preview fixture for ${path}` }), { status: 404 });
    }
    return new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } });
  };
}

const SAMPLE_SESSION = {
  accessToken: "preview-sample",
  member: { id: 1, email: "sample@example.com", name: "Sample merchant", externalPlatformID: 1 },
};

export function PreviewHarness({ children }: { children: ReactNode }) {
  // Before the first child render, so child queries start enabled.
  useState(() => {
    if (typeof window === "undefined") return false;
    install();
    authStore.getState().setSession(SAMPLE_SESSION);
    return true;
  });
  useEffect(() => {
    authStore.getState().setSession(SAMPLE_SESSION);
    return () => authStore.getState().clear();
  }, []);

  return (
    <AppShell>
      <p className="mb-5 rounded-sm border border-dashed border-line-strong bg-surface px-3 py-2 text-label text-ink-soft">
        Sample data. Development preview of the real page; nothing here reads the API.
      </p>
      {children}
    </AppShell>
  );
}
