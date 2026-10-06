import { backendURL, proxyJSON } from "@/lib/backend";

export async function GET() {
  const response = await fetch(backendURL("/public/blockchain-currencies"), { cache: "no-store" });
  return proxyJSON(response);
}
