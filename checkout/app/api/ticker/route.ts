import { backendURL, proxyJSON } from "@/lib/backend";

export async function GET(request: Request) {
  const symbols = new URL(request.url).searchParams.get("symbols") ?? "";
  if (!/^[A-Z0-9,]{2,80}$/.test(symbols)) return Response.json({ error: "Invalid symbols" }, { status: 400 });
  const response = await fetch(backendURL(`/public/ticker?symbols=${encodeURIComponent(symbols)}`), { cache: "no-store" });
  return proxyJSON(response);
}
