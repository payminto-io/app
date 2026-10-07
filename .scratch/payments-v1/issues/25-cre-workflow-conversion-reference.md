# 25 CRE workflow: conversion reference rate stamp

Status: ready-for-agent
Owner: Chainlink engineer (Fable)
Blocked by: 10, 21, 22

## Goal
`cre/workflows/conversion-reference/` per `docs/cre/SPEC.md` section 5.3. Cron every 15 minutes pulling `/api/v1/cre/conversions`; per pair reads the configured Data Feed proxy (`latestRoundData` at finalized, `decimals()` read at runtime, heartbeat staleness check, no `answeredInRound`) or a Data Stream report where configured; computes deviation in basis points with scaled integers; writes one batch report. Never produces a pre-trade quote; the gateway shows the reference only on an executed trade.

## Acceptance
Simulation with fixtures proves deviation maths against known values; stale feed rounds are skipped and logged, not written; the dashboard conversion row shows reference, deviation, feed address and round id, or nothing.
