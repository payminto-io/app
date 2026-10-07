// Solvency attestation workflow (docs/cre/SPEC.md section 5.1).
// Reads the gateway's published liabilities checkpoint and the deployer-configured reserves, encodes one
// versioned report and delivers it to GatewayAttestations through the Keystone forwarder. It talks to the
// gateway only over its public API and fails closed on every validation, consensus, report or write error.
import {
  ConfidentialHTTPClient,
  CronCapability,
  EVMClient,
  HTTPCapability,
  HTTPClient,
  LAST_FINALIZED_BLOCK_NUMBER,
  Runner,
  TxStatus,
  bytesToBase64,
  bytesToHex,
  consensusIdenticalAggregation,
  encodeCallMsg,
  getNetwork,
  handler,
  json,
  ok,
  prepareReportRequest,
  type HTTPPayload,
  type HTTPSendRequester,
  type NodeRuntime,
  type Runtime,
} from '@chainlink/cre-sdk'
import {
  type Address,
  type Hex,
  decodeFunctionResult,
  encodeAbiParameters,
  encodeFunctionData,
  keccak256,
  parseAbi,
  parseAbiParameters,
  stringToBytes,
  stringToHex,
  zeroAddress,
} from 'viem'
import { z } from 'zod'

// ---- Wire format (SPEC section 5; contracts/test/cre/ReportEncoder.sol is the executable reference) ----

export const REPORT_VERSION = 1
export const KIND_SOLVENCY = 1
export const UINT256_MAX = (1n << 256n) - 1n
export const UINT64_MAX = (1n << 64n) - 1n
export const SOLVENCY_TOKEN_SECRET = 'CRE_READ_TOKEN_SOLVENCY'
export const CUSTODIAN_SECRET = 'CUSTODIAN_API_KEY'

const SOLVENCY_REPORT_PARAMS = parseAbiParameters(
  'uint8 version, uint8 kind, bytes32 gatewayId, uint64 observedAt, (bytes32 checkpointHash, bytes32 asset, uint256 liabilities, uint256 reserves, uint8 decimals)[] items',
)

export type SolvencyItem = {
  checkpointHash: Hex
  asset: Hex
  liabilities: bigint
  reserves: bigint
  decimals: number
}

export type SolvencyReport = { gatewayId: Hex; observedAt: bigint; items: SolvencyItem[] }

// gatewayId = keccak256(CRE_PUBLIC_BASE_URL), the same bytes the gateway's verifier computes.
export const gatewayIdOf = (publicBaseUrl: string): Hex => keccak256(stringToHex(publicBaseUrl))

// Mirrors backend/internal/cre/codec.go LabelKey: a label of at most 32 bytes is right-padded, a longer one is hashed.
export const assetKey = (label: string): Hex => {
  const bytes = stringToBytes(label)
  if (bytes.length > 32) return keccak256(bytes)
  const padded = new Uint8Array(32)
  padded.set(bytes)
  return bytesToHex(padded)
}

export const encodeSolvencyReport = (report: SolvencyReport): Hex => {
  if (report.items.length === 0) throw new Error('solvency report carries no items')
  if (report.observedAt < 0n || report.observedAt > UINT64_MAX) throw new Error('observedAt is outside uint64')
  for (const item of report.items) {
    assertUint256(item.liabilities, `liabilities for ${item.asset}`)
    assertUint256(item.reserves, `reserves for ${item.asset}`)
    if (!Number.isInteger(item.decimals) || item.decimals < 0 || item.decimals > 255) {
      throw new Error(`decimals for ${item.asset} is outside uint8`)
    }
  }
  return encodeAbiParameters(SOLVENCY_REPORT_PARAMS, [
    REPORT_VERSION,
    KIND_SOLVENCY,
    report.gatewayId,
    report.observedAt,
    report.items,
  ])
}

export const assertUint256 = (value: bigint, what: string): void => {
  if (value < 0n || value > UINT256_MAX) throw new Error(`${what} is outside uint256`)
}

// ---- Strict numeric parsing: integers in minor units, decimal strings scaled to minor units ----

const MINOR_PATTERN = /^(0|[1-9]\d*)$/
const DECIMAL_PATTERN = /^(0|[1-9]\d*)(?:\.(\d+))?$/

// parseMinor accepts a non-negative base-10 integer string within uint256; nothing else.
export const parseMinor = (value: string, what: string): bigint => {
  if (!MINOR_PATTERN.test(value)) throw new Error(`${what} must be a non-negative integer string, got ${JSON.stringify(value)}`)
  const n = BigInt(value)
  assertUint256(n, what)
  return n
}

// scaleDecimal turns a major-unit decimal string into minor units at exactly `decimals`; finer precision is refused.
export const scaleDecimal = (value: string, decimals: number, what: string): bigint => {
  if (!Number.isInteger(decimals) || decimals < 0 || decimals > 18) throw new Error(`${what}: decimals must be 0..18`)
  const match = DECIMAL_PATTERN.exec(value)
  if (!match) throw new Error(`${what} must be a non-negative base-10 decimal string, got ${JSON.stringify(value)}`)
  const whole = match[1] ?? '0'
  const fraction = match[2] ?? ''
  if (fraction.length > decimals) throw new Error(`${what} has ${fraction.length} fractional digits, more than the asset's ${decimals}`)
  const n = BigInt(whole) * 10n ** BigInt(decimals) + BigInt(fraction.padEnd(decimals, '0') || '0')
  assertUint256(n, what)
  return n
}

// ---- Configuration: a closed union, one variant per mode ----

const hexAddress = z.string().regex(/^0x[0-9a-fA-F]{40}$/, 'must be a 0x-prefixed 20-byte hex address')
// QuickJS has no URL constructor, so zod's .url() cannot run in a workflow; a plain http(s) grammar is enough here.
const httpUrl = z.string().regex(/^https?:\/\/[^\s/?#]+(?:[/?#]\S*)?$/, 'must be an http(s) URL')
const assetCode = z.string().min(2).max(32)
const decimalsField = z.number().int().min(0).max(18)
const sixFieldCron = z.string().regex(/^(\S+\s+){5}\S+$/, 'CRE cron schedules have six fields')

const evmReserveSchema = z.object({
  asset: assetCode,
  chainSelectorName: z.string().min(1),
  token: hexAddress,
  custodyAddress: hexAddress,
  decimals: decimalsField,
})

const solanaReservesSchema = z.object({
  rpcUrls: z.array(httpUrl).min(2, 'at least two Solana RPCs are required for a cross-checked read'),
  slotWindow: z.number().int().positive(),
  accounts: z.array(
    z.object({
      asset: assetCode,
      mint: z.string().min(32).max(44),
      owner: z.string().min(32).max(44),
      tokenAccount: z.string().min(32).max(44),
      decimals: decimalsField,
    }),
  ),
})

// A custodian is read through Confidential HTTP; the endpoint answers GET with
// { "balances": [ { "asset": "USDC", "amount": "<major-unit decimal string>", "decimals": 6 } ] }.
const custodianSchema = z.object({
  url: httpUrl,
  secretOwner: hexAddress,
  assets: z.array(z.object({ asset: assetCode, decimals: decimalsField })).min(1),
})

const reservesSchema = z.object({
  evm: z.array(evmReserveSchema),
  solana: solanaReservesSchema.nullable(),
  custodian: custodianSchema.nullable(),
})

const attestationSchema = z.object({
  chainSelectorName: z.string().min(1),
  consumerAddress: hexAddress,
  gasLimit: z.string().regex(/^[1-9]\d*$/),
})

const sharedConfig = {
  schedule: sixFieldCron,
  authorizedKeys: z.array(hexAddress),
  gateway: z.object({ apiBaseUrl: httpUrl, publicBaseUrl: httpUrl }),
  reserves: reservesSchema,
}

export const configSchema = z.discriminatedUnion('mode', [
  z.object({ mode: z.literal('local-simulation'), ...sharedConfig, attestation: z.undefined().optional() }).strict(),
  z.object({ mode: z.literal('staging'), ...sharedConfig, attestation: attestationSchema }).strict(),
  z.object({ mode: z.literal('production'), ...sharedConfig, attestation: attestationSchema }).strict(),
])

export type Config = z.infer<typeof configSchema>
export type EvmReserve = z.infer<typeof evmReserveSchema>
export type SolanaReserves = z.infer<typeof solanaReservesSchema>
export type SolanaAccount = SolanaReserves['accounts'][number]

// ---- Gateway liabilities: one authenticated read, identical aggregation on the whole body ----

const liabilitiesSchema = z.object({
  checkpoint_id: z.string().min(1),
  checkpoint_hash: z.string().regex(/^0x[0-9a-fA-F]{64}$/),
  taken_at: z.string().optional(),
  max_journal_id: z.number().int().nonnegative().optional(),
  assets: z.array(z.object({ asset: assetCode, liabilities_minor: z.string(), decimals: decimalsField })),
})

export type LiabilitiesSnapshot = z.infer<typeof liabilitiesSchema>

export const liabilitiesUrl = (apiBaseUrl: string): string => `${apiBaseUrl.replace(/\/+$/, '')}/api/v1/cre/liabilities`
export const reportsUrl = (apiBaseUrl: string): string => `${apiBaseUrl.replace(/\/+$/, '')}/api/v1/cre/reports`

export const fetchLiabilities = (sender: HTTPSendRequester, url: string, token: string): LiabilitiesSnapshot => {
  const response = sender
    .sendRequest({ url, method: 'GET', headers: { Authorization: `Bearer ${token}` }, cacheSettings: { store: false } })
    .result()
  if (!ok(response)) throw new Error(`gateway liabilities read failed with HTTP ${response.statusCode}`)
  const snapshot = liabilitiesSchema.parse(json(response))
  if (snapshot.assets.length === 0) throw new Error('gateway liabilities snapshot carries no assets; nothing to attest')
  const seen = new Set<string>()
  for (const asset of snapshot.assets) {
    if (seen.has(asset.asset)) throw new Error(`gateway liabilities snapshot lists ${asset.asset} twice`)
    seen.add(asset.asset)
    parseMinor(asset.liabilities_minor, `liabilities for ${asset.asset}`)
  }
  return snapshot
}

const readLiabilities = (runtime: Runtime<Config>, token: string): LiabilitiesSnapshot =>
  new HTTPClient()
    .sendRequest(runtime, fetchLiabilities, consensusIdenticalAggregation<LiabilitiesSnapshot>())(
      liabilitiesUrl(runtime.config.gateway.apiBaseUrl),
      token,
    )
    .result()

// ---- Reserves ----

export type ReserveObservation = { asset: string; source: string; amount: bigint; decimals: number }

const erc20Abi = parseAbi([
  'function balanceOf(address owner) view returns (uint256)',
  'function decimals() view returns (uint8)',
])

export const resolveSelector = (chainSelectorName: string): bigint => {
  const network = getNetwork({ chainFamily: 'evm', chainSelectorName })
  if (!network) throw new Error(`unknown EVM chain selector name ${chainSelectorName}`)
  return network.chainSelector.selector
}

// EVM reserves: balanceOf(custody) at the last finalized block; decimals() is read, compared with the config
// and never assumed.
export const readEvmReserves = (runtime: Runtime<Config>): ReserveObservation[] => {
  const clients = new Map<string, EVMClient>()
  return runtime.config.reserves.evm.map((reserve) => {
    let client = clients.get(reserve.chainSelectorName)
    if (!client) {
      client = new EVMClient(resolveSelector(reserve.chainSelectorName))
      clients.set(reserve.chainSelectorName, client)
    }
    const call = (data: Hex) =>
      client!
        .callContract(runtime, {
          call: encodeCallMsg({ from: zeroAddress, to: reserve.token as Address, data }),
          blockNumber: LAST_FINALIZED_BLOCK_NUMBER,
        })
        .result()
    const decimals = decodeFunctionResult({
      abi: erc20Abi,
      functionName: 'decimals',
      data: bytesToHex(call(encodeFunctionData({ abi: erc20Abi, functionName: 'decimals' })).data),
    })
    if (Number(decimals) !== reserve.decimals) {
      throw new Error(`${reserve.asset} token ${reserve.token} reports ${decimals} decimals, config says ${reserve.decimals}`)
    }
    const amount = decodeFunctionResult({
      abi: erc20Abi,
      functionName: 'balanceOf',
      data: bytesToHex(
        call(encodeFunctionData({ abi: erc20Abi, functionName: 'balanceOf', args: [reserve.custodyAddress as Address] })).data,
      ),
    })
    assertUint256(amount, `${reserve.asset} reserve at ${reserve.custodyAddress}`)
    return { asset: reserve.asset, source: `evm:${reserve.chainSelectorName}:${reserve.custodyAddress}`, amount, decimals: reserve.decimals }
  })
}

// Solana SPL reserves: getAccountInfo(jsonParsed, finalized) against every configured RPC inside node mode;
// every RPC must agree on (mint, owner, amount, slot bucket), then the DON agrees on the whole list.
export type SolanaView = { mint: string; owner: string; amount: string; decimals: number; slot: number }
export type SolanaObservation = { asset: string; mint: string; owner: string; amount: string; decimals: number; slotBucket: string }

const solanaAccountInfoSchema = z.object({
  result: z.object({
    context: z.object({ slot: z.number().int().nonnegative() }),
    value: z
      .object({
        data: z.object({
          program: z.literal('spl-token'),
          parsed: z.object({
            type: z.literal('account'),
            info: z.object({
              mint: z.string(),
              owner: z.string(),
              tokenAmount: z.object({ amount: z.string(), decimals: z.number().int() }),
            }),
          }),
        }),
      })
      .nullable(),
  }),
})

export const parseSolanaAccountInfo = (body: unknown, tokenAccount: string): SolanaView => {
  const parsed = solanaAccountInfoSchema.parse(body)
  if (!parsed.result.value) throw new Error(`Solana token account ${tokenAccount} does not exist at finalized`)
  const info = parsed.result.value.data.parsed.info
  return { mint: info.mint, owner: info.owner, amount: info.tokenAmount.amount, decimals: info.tokenAmount.decimals, slot: parsed.result.context.slot }
}

export const slotBucket = (slot: number, slotWindow: number): string => String(Math.floor(slot / slotWindow) * slotWindow)

// reconcileSolanaViews is the per-node identical check across RPCs; a disagreement fails the run.
export const reconcileSolanaViews = (account: SolanaAccount, views: SolanaView[], slotWindow: number): SolanaObservation => {
  if (views.length < 2) throw new Error(`${account.asset}: fewer than two Solana RPC views`)
  const first = views[0]!
  const bucket = slotBucket(first.slot, slotWindow)
  for (const view of views) {
    if (view.mint !== first.mint || view.owner !== first.owner || view.amount !== first.amount || slotBucket(view.slot, slotWindow) !== bucket) {
      throw new Error(`${account.asset}: Solana RPCs disagree on (mint, owner, amount, slot bucket)`)
    }
  }
  if (first.mint !== account.mint) throw new Error(`${account.asset}: token account ${account.tokenAccount} holds mint ${first.mint}, config says ${account.mint}`)
  if (first.owner !== account.owner) throw new Error(`${account.asset}: token account ${account.tokenAccount} is owned by ${first.owner}, config says ${account.owner}`)
  if (first.decimals !== account.decimals) throw new Error(`${account.asset}: mint has ${first.decimals} decimals, config says ${account.decimals}`)
  parseMinor(first.amount, `${account.asset} Solana reserve`)
  return { asset: account.asset, mint: first.mint, owner: first.owner, amount: first.amount, decimals: first.decimals, slotBucket: bucket }
}

const jsonBody = (value: unknown): string => bytesToBase64(new TextEncoder().encode(JSON.stringify(value)))

export const fetchSolanaReserves = (node: NodeRuntime<Config>): SolanaObservation[] => {
  const solana = node.config.reserves.solana
  if (!solana || solana.accounts.length === 0) return []
  const http = new HTTPClient()
  return solana.accounts.map((account) => {
    const views = solana.rpcUrls.map((url) => {
      const response = http
        .sendRequest(node, {
          url,
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: jsonBody({
            jsonrpc: '2.0',
            id: 1,
            method: 'getAccountInfo',
            params: [account.tokenAccount, { encoding: 'jsonParsed', commitment: 'finalized' }],
          }),
          cacheSettings: { store: false },
        })
        .result()
      if (!ok(response)) throw new Error(`Solana RPC ${url} answered HTTP ${response.statusCode}`)
      return parseSolanaAccountInfo(json(response), account.tokenAccount)
    })
    return reconcileSolanaViews(account, views, solana.slotWindow)
  })
}

const readSolanaReserves = (runtime: Runtime<Config>): ReserveObservation[] => {
  const solana = runtime.config.reserves.solana
  if (!solana || solana.accounts.length === 0) return []
  const observations = runtime
    .runInNodeMode(fetchSolanaReserves, consensusIdenticalAggregation<SolanaObservation[]>())()
    .result()
  return observations.map((o) => ({ asset: o.asset, source: `solana:${o.owner}`, amount: parseMinor(o.amount, `${o.asset} Solana reserve`), decimals: o.decimals }))
}

// Custodian reserves: one Confidential HTTP read per asset. The API key is resolved inside the enclave from the
// Vault DON under the name in cre/secrets.yaml; the DON agrees on the response before the amount is scaled.
const custodianBalancesSchema = z.object({
  balances: z.array(z.object({ asset: assetCode, amount: z.string(), decimals: decimalsField })),
})

export const custodianAmount = (body: unknown, asset: string, decimals: number): bigint => {
  const row = custodianBalancesSchema.parse(body).balances.find((b) => b.asset === asset)
  if (!row) throw new Error(`custodian response has no balance for ${asset}`)
  if (row.decimals !== decimals) throw new Error(`custodian reports ${row.decimals} decimals for ${asset}, config says ${decimals}`)
  return scaleDecimal(row.amount, decimals, `${asset} custodian balance`)
}

const readCustodianReserves = (runtime: Runtime<Config>): ReserveObservation[] => {
  const custodian = runtime.config.reserves.custodian
  if (!custodian) return []
  const client = new ConfidentialHTTPClient()
  return custodian.assets.map(({ asset, decimals }) => {
    const response = client
      .sendRequest(runtime, {
        request: {
          url: custodian.url,
          method: 'GET',
          multiHeaders: { Authorization: { values: [`Bearer {{.${CUSTODIAN_SECRET}}}`] } },
        },
        vaultDonSecrets: [{ key: CUSTODIAN_SECRET, owner: custodian.secretOwner }],
      })
      .result()
    if (!ok(response)) throw new Error(`custodian ${custodian.url} answered HTTP ${response.statusCode}`)
    return { asset, source: `custodian:${custodian.url}`, amount: custodianAmount(json(response), asset, decimals), decimals }
  })
}

// ---- Items: every snapshot asset needs at least one reserve source with the same decimals ----

export const buildItems = (snapshot: LiabilitiesSnapshot, reserves: ReserveObservation[]): SolvencyItem[] => {
  const checkpointHash = snapshot.checkpoint_hash as Hex
  return snapshot.assets.map((asset) => {
    const sources = reserves.filter((r) => r.asset === asset.asset)
    if (sources.length === 0) {
      throw new Error(`no reserve source is configured for ${asset.asset}, which the ledger owes; refusing to attest a reserve of zero`)
    }
    let total = 0n
    for (const source of sources) {
      if (source.decimals !== asset.decimals) {
        throw new Error(`${asset.asset}: reserve source ${source.source} has ${source.decimals} decimals, the ledger snapshot has ${asset.decimals}`)
      }
      total += source.amount
    }
    assertUint256(total, `${asset.asset} total reserves`)
    return {
      checkpointHash,
      asset: assetKey(asset.asset),
      liabilities: parseMinor(asset.liabilities_minor, `liabilities for ${asset.asset}`),
      reserves: total,
      decimals: asset.decimals,
    }
  })
}

// ---- Gateway notification after a write: best effort, never a reason to fail a delivered report ----

const postReportNotice = (node: NodeRuntime<Config>, url: string, token: string, txHash: Hex, simulated: boolean): boolean => {
  const response = new HTTPClient()
    .sendRequest(node, {
      url,
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: jsonBody({ kind: 'solvency', tx_hash: txHash, simulated }),
      cacheSettings: { store: false },
    })
    .result()
  return response.statusCode === 202
}

const notifyGateway = (runtime: Runtime<Config>, token: string, txHash: Hex): boolean => {
  try {
    return runtime
      .runInNodeMode(postReportNotice, consensusIdenticalAggregation<boolean>())(
        reportsUrl(runtime.config.gateway.apiBaseUrl),
        token,
        txHash,
        runtime.config.mode !== 'production',
      )
      .result()
  } catch (error) {
    runtime.log(`gateway notification failed; its own poller will pick the log up: ${String(error)}`)
    return false
  }
}

// ---- The run ----

const describe = (report: SolvencyReport) => ({
  gatewayId: report.gatewayId,
  observedAt: report.observedAt.toString(),
  items: report.items.map((item) => ({
    asset: item.asset,
    checkpointHash: item.checkpointHash,
    liabilities: item.liabilities.toString(),
    reserves: item.reserves.toString(),
    decimals: item.decimals,
  })),
})

export const attest = (runtime: Runtime<Config>, trigger: 'cron' | 'http'): string => {
  const config = runtime.config
  const token = runtime.getSecret({ id: SOLVENCY_TOKEN_SECRET }).result().value
  const snapshot = readLiabilities(runtime, token)
  const reserves = [...readEvmReserves(runtime), ...readSolanaReserves(runtime), ...readCustodianReserves(runtime)]
  const items = buildItems(snapshot, reserves)
  const observedAt = BigInt(Math.floor(runtime.now().getTime() / 1000))
  const report: SolvencyReport = { gatewayId: gatewayIdOf(config.gateway.publicBaseUrl), observedAt, items }
  const encoded = encodeSolvencyReport(report)
  const summary = { trigger, checkpointId: snapshot.checkpoint_id, ...describe(report), report: encoded, reportHash: keccak256(encoded) }

  if (config.mode === 'local-simulation') {
    const result = { mode: 'local-simulation', wouldWrite: true, written: false, ...summary }
    runtime.log(`local-simulation: report built for ${items.length} asset(s); nothing signed, no EVM client constructed, nothing written`)
    return JSON.stringify(result)
  }

  const attestation = config.attestation
  if (attestation.consumerAddress.toLowerCase() === zeroAddress) throw new Error('attestation.consumerAddress is the zero address')
  const signed = runtime.report(prepareReportRequest(encoded)).result()
  const reply = new EVMClient(resolveSelector(attestation.chainSelectorName))
    .writeReport(runtime, { receiver: attestation.consumerAddress, report: signed, gasConfig: { gasLimit: attestation.gasLimit } })
    .result()
  if (reply.txStatus !== TxStatus.SUCCESS) {
    throw new Error(`writeReport did not succeed: status ${reply.txStatus}${reply.errorMessage ? `, ${reply.errorMessage}` : ''}`)
  }
  if (!reply.txHash || reply.txHash.length === 0) throw new Error('writeReport succeeded without a transaction hash')
  const txHash = bytesToHex(reply.txHash)
  const gatewayNotified = notifyGateway(runtime, token, txHash)
  runtime.log(`${config.mode}: solvency report written in ${txHash} (${items.length} asset(s)), gateway notified: ${gatewayNotified}`)
  return JSON.stringify({ mode: config.mode, written: true, txHash, gatewayNotified, ...summary })
}

export const onCronTrigger = (runtime: Runtime<Config>): string => attest(runtime, 'cron')
export const onHttpTrigger = (runtime: Runtime<Config>, _payload: HTTPPayload): string => attest(runtime, 'http')

export const initWorkflow = (config: Config) => [
  handler(new CronCapability().trigger({ schedule: config.schedule }), onCronTrigger),
  handler(
    new HTTPCapability().trigger({
      authorizedKeys: config.authorizedKeys.map((publicKey) => ({ type: 'KEY_TYPE_ECDSA_EVM', publicKey })),
    }),
    onHttpTrigger,
  ),
]

export async function main() {
  const runner = await Runner.newRunner<Config>({ configSchema })
  await runner.run(initWorkflow)
}
