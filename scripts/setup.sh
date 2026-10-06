#!/bin/bash
set -euo pipefail

echo "========================================="
echo "  Payminto - Self-Hosted Payment Gateway"
echo "========================================="

# Check requirements
command -v docker >/dev/null 2>&1 || { echo "Docker required. Install: https://docs.docker.com/get-docker/"; exit 1; }
command -v docker compose >/dev/null 2>&1 || { echo "Docker Compose required."; exit 1; }

# Interactive config
read -p "Domain (e.g. pay.yoursite.com): " DOMAIN
read -p "Email for SSL certificate: " EMAIL
read -p "Cold wallet address (ETH): " COLD_WALLET

# Generate secrets
AES_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -hex 32)
POSTGRES_PASSWORD=$(openssl rand -hex 16)
MCP_API_KEY="pm_$(openssl rand -hex 32)"

# Write .env
cat > .env << EOF
DOMAIN=${DOMAIN}
POSTGRES_DATABASE=payminto
POSTGRES_USERNAME=payminto
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
AES_KEY=${AES_KEY}
JWT_SECRET=${JWT_SECRET}
MCP_API_KEY=${MCP_API_KEY}
COLD_WALLET=${COLD_WALLET}
EOF

echo "Generated .env with secure secrets"

# Build and start
echo "Building and starting services..."
docker compose up -d --build

# Wait for healthy
echo "Waiting for services..."
sleep 15

# Health check
if curl -sf "http://localhost:8080/healthz" > /dev/null 2>&1; then
  echo ""
  echo "========================================="
  echo "  Payminto is running!"
  echo "========================================="
  echo "  Dashboard: https://${DOMAIN}"
  echo "  API:       https://${DOMAIN}/api/v1"
  echo "  MCP:       https://${DOMAIN}/mcp"
  echo "  API Key:   ${MCP_API_KEY}"
  echo "========================================="
else
  echo "Health check failed. Run: docker compose logs"
  exit 1
fi
