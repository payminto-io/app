# 10 Conversion at receipt

Status: ready-for-agent
Owner: Blockchain engineer (Fable)
Blocked by: 01, 09

## Goal
`backend/internal/conversion/`: on a confirmed receipt in an asset other than the merchant's settlement asset, execute a conversion through a connector (Jupiter on Solana first, mock in tests), record the executed rate and slippage as an explicit ledger trade journal. Hold-in-asset merchants skip conversion and get an exposure line. No oracle quorum in V1.

## Acceptance
Journal shape tested; failed conversion leaves the receipt held and alerts; rate and slippage shown only from the executed trade.
