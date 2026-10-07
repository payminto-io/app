#!/usr/bin/env bash
# Loops `cre workflow simulate` over every workflow under /workspace/cre/workflows against the local gateway.
# Simulation only: no login, no deploy, no --broadcast. Workflows arrive with tickets 23 to 25.
set -euo pipefail

WORKFLOWS_DIR="${CRE_WORKFLOWS_DIR:-/workspace/cre/workflows}"
INTERVAL="${CRE_SIMULATE_INTERVAL:-300}"
TARGET="${CRE_SIMULATE_TARGET:-local-simulation}"

echo "[cre-simulator] gateway=${CRE_PUBLIC_BASE_URL:-unset} target=${TARGET} interval=${INTERVAL}s"
if ! command -v cre >/dev/null 2>&1; then
  echo "[cre-simulator] cre CLI not found on PATH; the image build did not install it" >&2
  exit 1
fi
cre version || true  # pinned release, checksum-verified at image build

while true; do
  if [ ! -d "${WORKFLOWS_DIR}" ] || [ -z "$(ls -A "${WORKFLOWS_DIR}" 2>/dev/null)" ]; then
    echo "[cre-simulator] no workflows under ${WORKFLOWS_DIR}; tickets 23 to 25 add solvency, deposit-finality and conversion-reference. Idle."
    sleep "${INTERVAL}"
    continue
  fi
  if [ -f "${WORKFLOWS_DIR}/../package.json" ] && [ ! -d "${WORKFLOWS_DIR}/../node_modules" ]; then
    (cd "${WORKFLOWS_DIR}/.." && bun install --frozen-lockfile) || echo "[cre-simulator] bun install failed; continuing"
  fi
  for wf in "${WORKFLOWS_DIR}"/*/; do
    name="$(basename "${wf}")"
    echo "[cre-simulator] simulate ${name}"
    (cd "${WORKFLOWS_DIR}/.." && cre workflow simulate "workflows/${name}" --target "${TARGET}" --non-interactive --trigger-index 0) \
      || echo "[cre-simulator] ${name} failed; the gateway records nothing for a failed run"
  done
  sleep "${INTERVAL}"
done
