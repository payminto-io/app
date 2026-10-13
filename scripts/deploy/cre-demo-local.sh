#!/usr/bin/env bash
# Reproduces the CRE solvency attestation end to end on a local Anvil:
# a real payment -> a real ledger liability -> a signed report written on chain
# -> the gateway verifying it from its own RPC -> a public attestation.
#
# Nothing here touches the live stack on :8090. It runs a second gateway on
# :8097 with its own database, in development mode, because SERVER=staging
# refuses the simulator's mock Keystone forwarder by design.
#
# Needs: anvil/forge/cast (foundry), the cre CLI, go, and the local Postgres.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
RPC="${RPC:-http://127.0.0.1:8545}"
PORT="${PORT:-8097}"
DB="${DB:-payminto_cre_test}"
DEPLOYER=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266          # Anvil account 0
ANVIL_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80  # public dev key
CUSTODY=0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC            # Anvil account 2
LIABILITY_MAJOR="${LIABILITY_MAJOR:-1250.00}"
RESERVE_MINOR="${RESERVE_MINOR:-130000}"                      # 1300.00 at 2 decimals

step() { printf '\n== %s\n' "$1"; }

step "1. Anvil on Base Sepolia's chain id (finalized must not lag, hence slots-in-an-epoch 1)"
if ! curl -fsS -m 2 -X POST "$RPC" -H 'Content-Type: application/json' \
     -d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' >/dev/null 2>&1; then
  nohup anvil --chain-id 84532 --block-time 1 --slots-in-an-epoch 1 --host 127.0.0.1 >/tmp/anvil.log 2>&1 &
  until curl -fsS -m 2 -X POST "$RPC" -H 'Content-Type: application/json' \
        -d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' >/dev/null 2>&1; do sleep 1; done
fi
echo "   anvil up at $RPC"

step "2. Forwarder, mock token and GatewayAttestations"
"$ROOT/scripts/cre-local-stack.sh" | tee /tmp/cre-stack.out
CONSUMER="$(sed -n 's/^GatewayAttestations: *\(0x[0-9a-fA-F]*\).*/\1/p' /tmp/cre-stack.out | head -1)"
START_BLOCK="$(cast block-number --rpc-url "$RPC")"
[ -n "$CONSUMER" ] || { echo "no consumer address" >&2; exit 1; }

step "3. A 2-decimal reserve token"
# The gateway's USD liability carries 2 decimals and the workflow requires the
# on-chain decimals to equal the configured ones, so the stack's 6-decimal mock
# token cannot back a USD liability.
TOKEN="$(cd "$ROOT/contracts" && forge create src/cre/mocks/MockERC20.sol:MockERC20 \
  --rpc-url "$RPC" --unlocked --from "$DEPLOYER" --broadcast \
  --constructor-args "USD reserve" "USD" 2 | sed -n 's/Deployed to: *\(0x[0-9a-fA-F]*\)/\1/p')"
cast send "$TOKEN" "mint(address,uint256)" "$CUSTODY" "$RESERVE_MINOR" \
  --rpc-url "$RPC" --unlocked --from "$DEPLOYER" >/dev/null
echo "   reserve token $TOKEN holds $RESERVE_MINOR minor units at $CUSTODY"

step "4. A throwaway database for the demo gateway"
sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='$DB'" | grep -q 1 \
  || sudo -u postgres createdb -O payminto_mig "$DB"

step "5. Gateway env (backend/.env.cre-local)"
echo "   write it from /tmp/cre-stack.out; CRE_START_BLOCK=$START_BLOCK is required —"
echo "   without it the module degrades from chainlink to mock and says so in"
echo "   GET /api/v1/cre/status (degraded_from, missing_keys)."

step "6. Workflow config"
python3 - "$ROOT" "$TOKEN" "$CONSUMER" <<'PY'
import json, sys
root, token, consumer = sys.argv[1], sys.argv[2], sys.argv[3]
p = f"{root}/cre/workflows/solvency/config.anvil.json"
c = json.load(open(p))
c["reserves"]["evm"] = [{"asset": "USD", "chainSelectorName": "ethereum-testnet-sepolia-base-1",
                         "token": token, "custodyAddress": "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
                         "decimals": 2}]
c["attestation"]["consumerAddress"] = consumer
json.dump(c, open(p, "w"), indent=2)
print("   config.anvil.json updated")
PY

step "7. The liability: a real payment through the keyless mock connector"
cat <<MSG
   With the gateway up on :$PORT and a devseeded merchant API key:
     POST http://127.0.0.1:$PORT/api/v2/payments
     {"amount":"$LIABILITY_MAJOR","asset":"USD","capture_method":"automatic","confirm":true,
      "payment_method":{"type":"card","token":"success"},"idempotency_key":"cre-demo-1"}
   Intents price in USD, so asking for USDC here is refused. The payment posts a
   double-entry journal; restart the gateway so it takes a fresh checkpoint
   (GET /api/v1/cre/liabilities must then show max_journal_id > 0).
MSG

step "8. Broadcast the report and read the attestation"
cat <<MSG
   cd cre && cre workflow simulate workflows/solvency --target staging \\
     --config ./config.anvil.json --non-interactive --trigger-index 0 --broadcast

   Without --broadcast the staging handler fails with "writeReport succeeded
   without a transaction hash", which is correct: nothing was sent.

   Then: GET http://127.0.0.1:$PORT/api/v1/public/attestations/<id>
   expect status=attested, provider=chainlink, on_chain=stored, and
   simulated=true (the mock forwarder is not a DON; independently_signed=false).
MSG
