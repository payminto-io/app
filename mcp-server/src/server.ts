import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { apiCall } from "./api-client.js";

export function createServer(): McpServer {
  const server = new McpServer({
    name: "payminto",
    version: "1.0.0",
  });

  server.tool("test-connection", "Verify connectivity to Payminto", {}, async () => {
    try {
      await apiCall("/healthz");
      return { content: [{ type: "text" as const, text: "Connected to Payminto successfully" }] };
    } catch (e) {
      return { content: [{ type: "text" as const, text: `Connection failed: ${e}` }] };
    }
  });

  server.tool(
    "create-payee",
    "Create a payment request for a recipient",
    {
      amountInUSD: z.number().positive().describe("Payment amount in USD"),
      customerEmail: z.string().email().optional().describe("Customer email"),
      customerID: z.string().optional().describe("Customer ID"),
    },
    async ({ amountInUSD, customerEmail, customerID }) => {
      const result = await apiCall("/payment", {
        method: "POST",
        body: JSON.stringify({ amountInUSD, customerEmail, customerID }),
      });
      return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
    }
  );

  server.tool(
    "lookup-payment",
    "Get details of a payment by reference ID",
    { referenceId: z.string().describe("Payment reference ID") },
    async ({ referenceId }) => {
      const result = await apiCall(`/payment/reference/${referenceId}`);
      return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
    }
  );

  server.tool(
    "search-payments",
    "Search and list payments",
    {
      state: z.string().optional().describe("Filter by state: OPEN, FILLED, CANCELLED"),
      limit: z.number().optional().describe("Max results (default 25)"),
    },
    async ({ state, limit }) => {
      const params = new URLSearchParams();
      if (state) params.set("state", state);
      if (limit) params.set("limit", String(limit));
      const result = await apiCall(`/payments?${params}`);
      return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
    }
  );

  server.tool("get-balance", "Query account liquidity and wallet balances", {}, async () => {
    const result = await apiCall("/wallets");
    return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
  });

  server.tool(
    "get-daily-volume",
    "Get aggregate daily payment volume metrics",
    {},
    async () => {
      const result = await apiCall("/analytics/volume");
      return { content: [{ type: "text" as const, text: JSON.stringify(result, null, 2) }] };
    }
  );

  return server;
}
