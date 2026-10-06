import express from "express";
import { createServer } from "./server.js";

const app = express();
const PORT = parseInt(process.env.MCP_PORT || "3333");

app.use(express.json());

app.get("/healthz", (_req, res) => {
  res.json({ status: "ok", service: "payminto-mcp" });
});

app.listen(PORT, () => {
  console.log(`Payminto MCP server running on port ${PORT}`);
  console.log(`Health: http://localhost:${PORT}/healthz`);
});

export { createServer };
