import { backendURL, proxyJSON, validReference } from "@/lib/backend";

export async function GET(_request: Request, { params }: { params: Promise<{ referenceId: string }> }) {
  const { referenceId } = await params;
  if (!validReference(referenceId)) return Response.json({ error: "Invalid payment reference" }, { status: 400 });
  const response = await fetch(backendURL(`/public/payment/${encodeURIComponent(referenceId)}`), { cache: "no-store" });
  return proxyJSON(response);
}
