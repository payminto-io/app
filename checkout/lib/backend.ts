import { NextResponse } from "next/server";

const API_BASE = (process.env.PAYMINTO_API_URL ?? "http://localhost:8080/api/v1").replace(/\/$/, "");

export function backendURL(path: string): string {
  if (!path.startsWith("/public/")) throw new Error("Only public checkout routes may be proxied");
  return `${API_BASE}${path}`;
}

export async function proxyJSON(response: Response): Promise<NextResponse> {
  const text = await response.text();
  const headers = { "Cache-Control": "no-store", "Content-Type": "application/json; charset=utf-8" };
  try {
    return NextResponse.json(JSON.parse(text), { status: response.status, headers });
  } catch {
    return NextResponse.json(
      { error: response.ok ? "Invalid response from payment service" : "Payment service unavailable" },
      { status: response.ok ? 502 : response.status, headers },
    );
  }
}

export function validReference(value: string): boolean {
  return value.length >= 8 && value.length <= 128 && /^[A-Za-z0-9_-]+$/.test(value);
}
