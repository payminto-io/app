# Payminto MCP Server — CLAUDE.md

## What This Is

The Payminto MCP (Model Context Protocol) server enables AI agents — Claude, GPT, and others — to interact with the Payminto payment gateway programmatically. It exposes a set of structured tools that an AI agent can call to create payment requests, look up payment status, check balances, and retrieve analytics. It runs on port 3333. The MCP server is a thin adapter: it receives tool calls from an AI host, translates them into REST API calls to the Payminto Go backend, and returns formatted text responses. It does NOT access the database directly.

## Tech Stack

| Component | Version / Package |
|-----------|------------------|
| Language | TypeScript 5.7 (strict) |
| Module system | ESM only (`"type": "module"` in package.json) |
| MCP SDK | `@modelcontextprotocol/sdk` ^1.12.1 |
| HTTP server | `express` ^4.21.0 |
| Schema validation | `zod` ^3.24.0 |
| Dev runner | `tsx` ^4.19.0 |
| Runtime | Node.js 20+ |

## Directory Layout

```
mcp-server/
├── src/
│   ├── index.ts         # Express HTTP server: starts on port 3333, /healthz endpoint, mounts MCP
│   ├── server.ts        # MCP Server instance: registers all 6 tools via createServer()
│   └── api-client.ts    # Typed API client: calls Payminto Go backend REST API
├── dist/                # Compiled JS output (npx tsc → here)
├── tsconfig.json        # ESM, strict, target ES2022, outDir: ./dist
└── package.json         # name: @payminto/mcp-server, type: module
```

## The 6 MCP Tools

All tools are registered in `src/server.ts`. Each tool:
1. Receives validated input (Zod schema)
2. Calls a function in `src/api-client.ts`
3. Returns `{ content: [{ type: "text" as const, text: "..." }] }`

| Tool Name | Purpose | Backend Endpoint |
|-----------|---------|-----------------|
| `test-connection` | Verifies the MCP server can reach the backend | `GET /healthz` |
| `create-payee` | Creates a new payment request / invoice | `POST /api/v1/payments` |
| `lookup-payment` | Gets full details of one payment by ID | `GET /api/v1/payments/:id` |
| `search-payments` | Lists payments with optional filters | `GET /api/v1/payments` |
| `get-balance` | Gets current balance of a wallet address | `GET /api/v1/wallets/:address/balance` |
| `get-daily-volume` | Gets payment volume stats for a date range | `GET /api/v1/analytics/summary` |

## Common Commands

```bash
# Development (runs src/index.ts directly with tsx, no compile step)
npm run dev

# Build TypeScript to dist/
npm run build
# equivalent: npx tsc

# Run compiled server
npm start
# equivalent: node dist/index.js

# Type-check without emitting
npx tsc --noEmit
```

## Environment Variables

```
PAYMINTO_API_URL=http://localhost:8080/api/v1   # Go backend URL
PAYMINTO_API_KEY=<merchant-api-key>              # API key for authenticating to backend
PORT=3333                                         # MCP server port (default 3333)
```

Both `PAYMINTO_API_URL` and `PAYMINTO_API_KEY` default to the local development values if not set.

## Code Conventions

### ESM-Only TypeScript

The package uses `"type": "module"` in `package.json`, so ALL imports must use full file extensions:

```ts
// CORRECT
import { createServer } from "./server.js"  // .js extension even in TypeScript source

// WRONG
import { createServer } from "./server"      // will fail at runtime in ESM
```

### Tool Return Type — Use `as const`

The MCP SDK requires the `type` field to be a string literal, not just `string`. This means you MUST use `as const`:

```ts
// CORRECT
return {
  content: [{
    type: "text" as const,
    text: `Payment created: ${payment.id}`
  }]
}

// WRONG — TypeScript infers type as string, not "text", SDK rejects it
return {
  content: [{
    type: "text",
    text: `Payment created: ${payment.id}`
  }]
}
```

### Adding a New Tool

1. Add the tool registration in `src/server.ts` inside `createServer()`:

```ts
server.tool(
  "my-new-tool",
  "Description for the AI agent",
  {
    // Zod schema for input parameters
    paymentId: z.string().describe("The payment ID to look up"),
    currency: z.enum(["BTC", "ETH", "USDT"]).optional(),
  },
  async ({ paymentId, currency }) => {
    const result = await myNewApiCall(paymentId, currency)
    return {
      content: [{
        type: "text" as const,
        text: JSON.stringify(result, null, 2)
      }]
    }
  }
)
```

2. Add the corresponding API function in `src/api-client.ts`:

```ts
export async function myNewApiCall(paymentId: string, currency?: string) {
  const url = new URL(`${API_URL}/payments/${paymentId}`)
  if (currency) url.searchParams.set("currency", currency)
  const res = await fetch(url.toString(), {
    headers: { "X-API-Key": API_KEY }
  })
  if (!res.ok) throw new Error(`API error: ${res.status}`)
  return res.json()
}
```

### Error Handling

Tool handlers should catch errors and return them as text (not throw), so the AI agent receives a readable error message:

```ts
try {
  const result = await apiCall(params)
  return { content: [{ type: "text" as const, text: JSON.stringify(result) }] }
} catch (err) {
  return {
    content: [{
      type: "text" as const,
      text: `Error: ${err instanceof Error ? err.message : String(err)}`
    }]
  }
}
```

## Integration Points

| System | Direction | Details |
|--------|-----------|---------|
| AI host (Claude Desktop, GPT plugins, etc.) | inbound | MCP protocol over stdio or HTTP |
| Payminto Go backend | outbound | REST HTTP via `PAYMINTO_API_URL` with `X-API-Key` auth |

The MCP server is intentionally stateless — it holds no data and makes no DB connections. All state lives in the Go backend.

## Connecting to Claude Desktop

To test locally with Claude Desktop, add this to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "payminto": {
      "command": "node",
      "args": ["/path/to/payminto/mcp-server/dist/index.js"],
      "env": {
        "PAYMINTO_API_URL": "http://localhost:8080/api/v1",
        "PAYMINTO_API_KEY": "your-api-key"
      }
    }
  }
}
```

## Key Files to Read First

1. `src/server.ts` — all 6 tool registrations with schemas and handlers
2. `src/api-client.ts` — API client with typed response shapes
3. `src/index.ts` — Express server setup and MCP server mounting
4. `tsconfig.json` — ESM config, important for understanding import requirements
