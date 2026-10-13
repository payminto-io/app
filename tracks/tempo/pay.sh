#!/usr/bin/env bash
# Pay a Payminto Tempo payment with a TIP-20 transferWithMemo.
# Usage: TEMPO_PAYER_KEY_FILE=/path/to/key tracks/tempo/pay.sh <api base> <payment reference>
#   api base: e.g. http://localhost:8095/api/v1
# Env: TEMPO_RPC_URL (default Moderato), TEMPO_PAYER_KEY_FILE (private key file, never in the repo).
set -euo pipefail
API=${1:?api base}; REF=${2:?payment reference}
RPC=${TEMPO_RPC_URL:-https://rpc.moderato.tempo.xyz}
KEY=$(cat "${TEMPO_PAYER_KEY_FILE:?set TEMPO_PAYER_KEY_FILE}")

# Attach the Tempo leg (idempotent) and read address, token, amount and memo from the public API.
curl -sf -X POST "$API/public/deposit-address/reference/$REF" -H 'content-type: application/json' \
  -d '{"blockchainCode":"TEMPO","currencyCode":"PATHUSD"}' >/dev/null || true
P=$(curl -sf "$API/public/payment/$REF")
TO=$(jq -r .depositAddress <<<"$P"); MEMO=$(jq -r '.memo // .referenceID' <<<"$P"); AMT=$(jq -r .amountInUSD <<<"$P")
TOKEN=$(curl -sf "$API/public/blockchain-currencies" | jq -r --arg c "$(jq -r .currencyCode <<<"$P")" \
  '.currencies[] | select(.blockchainCode=="TEMPO" and .currencyCode==$c) | .address')
RAW=$(python3 -c "from decimal import Decimal;import sys;print(int(Decimal(sys.argv[1])*10**6))" "$AMT")
MEMO32=$(cast format-bytes32-string "$MEMO")
echo "pay $AMT ($RAW raw) of $TOKEN to $TO memo '$MEMO' ($MEMO32)"
cast send "$TOKEN" "transferWithMemo(address,uint256,bytes32)" "$TO" "$RAW" "$MEMO32" --private-key "$KEY" --rpc-url "$RPC" --json \
  | jq -r '"tx " + .transactionHash + " status " + .status'
