import { backendURL, validReference } from "@/lib/backend";

export const dynamic = "force-dynamic";

export async function GET(request: Request, { params }: { params: Promise<{ referenceId: string }> }) {
  const { referenceId } = await params;
  if (!validReference(referenceId)) return Response.json({ error: "Invalid payment reference" }, { status: 400 });
  const response = await fetch(backendURL(`/public/events/${encodeURIComponent(referenceId)}`), {
    cache: "no-store",
    headers: { Accept: "text/event-stream" },
    signal: request.signal,
  });
  if (!response.ok || !response.body) return Response.json({ error: "Status stream unavailable" }, { status: response.status || 502 });
  return new Response(response.body, {
    status: 200,
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache, no-transform",
      "X-Accel-Buffering": "no",
    },
  });
}
