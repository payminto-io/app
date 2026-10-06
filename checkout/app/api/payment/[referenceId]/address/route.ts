import { backendURL, proxyJSON, validReference } from "@/lib/backend";

const codePattern = /^[A-Z0-9_-]{2,20}$/;

export async function POST(request: Request, { params }: { params: Promise<{ referenceId: string }> }) {
  const { referenceId } = await params;
  if (!validReference(referenceId)) return Response.json({ error: "Invalid payment reference" }, { status: 400 });
  const length = Number(request.headers.get("content-length") ?? "0");
  if (length > 2048) return Response.json({ error: "Request too large" }, { status: 413 });
  let body: unknown;
  try { body = await request.json(); } catch { return Response.json({ error: "Invalid request" }, { status: 400 }); }
  if (typeof body !== "object" || body === null) return Response.json({ error: "Invalid request" }, { status: 400 });
  const { blockchainCode, currencyCode } = body as Record<string, unknown>;
  if (typeof blockchainCode !== "string" || !codePattern.test(blockchainCode) || typeof currencyCode !== "string" || !codePattern.test(currencyCode)) {
    return Response.json({ error: "Invalid payment method" }, { status: 400 });
  }
  const response = await fetch(backendURL(`/public/deposit-address/reference/${encodeURIComponent(referenceId)}`), {
    method: "POST",
    cache: "no-store",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ blockchainCode, currencyCode }),
  });
  return proxyJSON(response);
}
