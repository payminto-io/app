#!/usr/bin/env bash
# Local Anvil stack for the CRE solvency workflow (cre/README.md "Without Chainlink access").
# Deploys MockKeystoneForwarder (also pinned at the chain's published mock-forwarder address, which is where
# `cre workflow simulate --broadcast` delivers), a reserve token minted to a custody address, and
# GatewayAttestations bound to the simulator's fixed workflow identity. Prints the gateway variables.
# Refuses anything that is not a local Anvil: the deploy script checks anvil_nodeInfo and nothing here broadcasts
# to a public network.
set -euo pipefail

RPC="${CRE_LOCAL_EVM_RPC_URL:-http://127.0.0.1:8545}"
DEPLOYER="${CRE_LOCAL_DEPLOYER:-0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266}"   # Anvil account 0 (unlocked)
PENDING_OWNER="${CRE_LOCAL_CONSUMER_OWNER:-0x70997970C51812dc3A010C7d01b50e0d17dc79C8}"  # Anvil account 1
CUSTODY="${CRE_LOCAL_CUSTODY_ADDRESS:-0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC}"   # Anvil account 2
RESERVE="${CRE_LOCAL_RESERVE_MINOR:-1300000000}"
# The simulator signs every report with this fixed identity; the consumer must be bound to it (cre/README.md).
SIM_WORKFLOW_ID="0x1111111111111111111111111111111111111111111111111111111111111111"
SIM_WORKFLOW_OWNER="0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
# Published MockKeystoneForwarder address for the chain the staging target names (Base Sepolia).
PINNED_FORWARDER="${CRE_LOCAL_FORWARDER_AT:-0x82300bd7c3958625581cc2f77bc6464dcecdf3e5}"

cd "$(dirname "$0")/../contracts"
case "$RPC" in
  http://127.0.0.1:*|http://localhost:*) ;;
  *) echo "refusing: $RPC is not a loopback RPC" >&2; exit 1 ;;
esac

CRE_LOCAL_CUSTODY_ADDRESS="$CUSTODY" CRE_LOCAL_RESERVE_MINOR="$RESERVE" CRE_LOCAL_FORWARDER_AT="$PINNED_FORWARDER" \
  forge script script/DeployCRELocalStack.s.sol --rpc-url "$RPC" --unlocked --sender "$DEPLOYER" --broadcast 2>&1 \
  | grep -E 'MockKeystoneForwarder:|MockERC20|forwarder code' || { echo "local stack deploy failed" >&2; exit 1; }

out="$(CRE_FORWARDER_ADDRESS="$PINNED_FORWARDER" CRE_CONSUMER_OWNER="$PENDING_OWNER" CRE_WORKFLOW_OWNER="$SIM_WORKFLOW_OWNER" \
  CRE_WORKFLOW_ID_SOLVENCY="$SIM_WORKFLOW_ID" CRE_WORKFLOW_ID_DEPOSIT_FINALITY="$SIM_WORKFLOW_ID" CRE_WORKFLOW_ID_CONVERSION_REFERENCE="$SIM_WORKFLOW_ID" \
  CRE_WORKFLOW_NAME_SOLVENCY=solvency-staging CRE_WORKFLOW_NAME_DEPOSIT_FINALITY=deposit-finality-staging CRE_WORKFLOW_NAME_CONVERSION_REFERENCE=conversion-reference-staging \
  forge script script/DeployGatewayAttestations.s.sol --rpc-url "$RPC" --unlocked --sender "$DEPLOYER" --broadcast 2>&1)"
consumer="$(echo "$out" | sed -n 's/.*GatewayAttestations: *\(0x[0-9a-fA-F]*\).*/\1/p' | head -1)"
[ -n "$consumer" ] || { echo "consumer deploy failed:" >&2; echo "$out" >&2; exit 1; }
[ "$(cast call "$consumer" "forwarder()(address)" --rpc-url "$RPC" | tr 'A-F' 'a-f')" = "$PINNED_FORWARDER" ] || { echo "consumer does not trust the pinned forwarder" >&2; exit 1; }

cat <<MSG

GatewayAttestations: $consumer (owner: $DEPLOYER until $PENDING_OWNER accepts)
Forwarder the consumer trusts: $PINNED_FORWARDER (MockKeystoneForwarder code)
Reserve custody: $CUSTODY holds $RESERVE minor units of the mock USDC

Gateway variables (backend/.env), with your own port and read token:
  CRE_ENABLED=true
  CRE_PROVIDER=chainlink
  CRE_CHAIN=ethereum-testnet-sepolia-base-1
  CRE_CHAIN_RPC_URL=$RPC
  CRE_CONSUMER_ADDRESS=$consumer
  CRE_FORWARDER_ADDRESS=$PINNED_FORWARDER
  CRE_WORKFLOW_OWNER=$SIM_WORKFLOW_OWNER
  CRE_WORKFLOW_ID_SOLVENCY=$SIM_WORKFLOW_ID
  CRE_WORKFLOW_ID_DEPOSIT_FINALITY=$SIM_WORKFLOW_ID
  CRE_WORKFLOW_ID_CONVERSION_REFERENCE=$SIM_WORKFLOW_ID
  CRE_TRIGGER_SIGNER=keyring://cre-trigger
  CRE_PUBLIC_BASE_URL=http://localhost:8090
  CRE_READ_TOKEN_SOLVENCY=<same value as SOLVENCY_READ_TOKEN in cre/.env>
  CRE_POLL_INTERVAL=5s
Workflow config (cre/workflows/solvency/config.anvil.json): attestation.consumerAddress=$consumer
MSG
