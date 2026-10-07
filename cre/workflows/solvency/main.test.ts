import { describe, expect } from 'bun:test'
import { EvmMock, HttpActionsMock, ConsensusMock, addContractMock, newTestRuntime, test } from '@chainlink/cre-sdk/test'
import { bytesToBase64, bytesToHex, hexToBytes } from '@chainlink/cre-sdk'
import { keccak256, parseAbi, type Hex } from 'viem'
import fixture from './fixtures/solvency-report.json'
import {
  type Config,
  type LiabilitiesSnapshot,
  type SolanaAccount,
  UINT256_MAX,
  assetKey,
  attest,
  buildItems,
  configSchema,
  custodianAmount,
  encodeSolvencyReport,
  gatewayIdOf,
  initWorkflow,
  parseMinor,
  reconcileSolanaViews,
  resolveSelector,
  scaleDecimal,
  slotBucket,
} from './main'

const TOKEN = '0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512'
const CUSTODY = '0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC'
const CONSUMER = '0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9'
const CHAIN = 'ethereum-testnet-sepolia-base-1'
const erc20Abi = parseAbi([
  'function balanceOf(address owner) view returns (uint256)',
  'function decimals() view returns (uint8)',
])
const consumerAbi = parseAbi(['function forwarder() view returns (address)'])

const snapshot = (assets: LiabilitiesSnapshot['assets']): LiabilitiesSnapshot => ({
  checkpoint_id: 'ckpt-1',
  checkpoint_hash: fixture.checkpointHash,
  assets,
})

const baseConfig = (mode: Config['mode']): Config =>
  configSchema.parse({
    mode,
    schedule: '0 0 * * * *',
    authorizedKeys: [],
    gateway: { apiBaseUrl: 'http://localhost:8097', publicBaseUrl: fixture.publicBaseUrl },
    reserves: {
      evm: [{ asset: 'USDC', chainSelectorName: CHAIN, token: TOKEN, custodyAddress: CUSTODY, decimals: 6 }],
      solana: null,
      custodian: null,
    },
    ...(mode === 'local-simulation' ? {} : { attestation: { chainSelectorName: CHAIN, consumerAddress: CONSUMER, gasLimit: '2500000' } }),
  })

const secrets = () => new Map([['main', new Map([['CRE_READ_TOKEN_SOLVENCY', 'dev-solvency-read-token']])]])

const jsonResponse = (statusCode: number, body: unknown) => ({
  statusCode,
  body: bytesToBase64(new TextEncoder().encode(JSON.stringify(body))),
})

describe('wire format', () => {
  test('assetKey right-pads short labels like the gateway LabelKey and hashes long ones', () => {
    expect(assetKey('USDC')).toBe('0x5553444300000000000000000000000000000000000000000000000000000000')
    const long = 'A'.repeat(33)
    expect(assetKey(long)).toBe(keccak256(new TextEncoder().encode(long)))
  })

  test('gatewayId is keccak256 of the public base URL', () => {
    expect(gatewayIdOf(fixture.publicBaseUrl)).toBe(fixture.gatewayId as Hex)
  })

  test('encodeSolvencyReport matches ReportEncoder.sol byte for byte', () => {
    const encoded = encodeSolvencyReport({
      gatewayId: fixture.gatewayId as Hex,
      observedAt: BigInt(fixture.observedAt),
      items: fixture.items.map((item) => ({
        checkpointHash: fixture.checkpointHash as Hex,
        asset: assetKey(item.asset),
        liabilities: BigInt(item.liabilities),
        reserves: BigInt(item.reserves),
        decimals: item.decimals,
      })),
    })
    expect(encoded).toBe(fixture.report as Hex)
    expect(keccak256(encoded)).toBe(fixture.reportHash as Hex)
  })

  test('encodeSolvencyReport refuses empty batches and out-of-range integers', () => {
    const item = { checkpointHash: fixture.checkpointHash as Hex, asset: assetKey('USDC'), liabilities: 1n, reserves: 1n, decimals: 6 }
    expect(() => encodeSolvencyReport({ gatewayId: fixture.gatewayId as Hex, observedAt: 1n, items: [] })).toThrow('no items')
    expect(() => encodeSolvencyReport({ gatewayId: fixture.gatewayId as Hex, observedAt: 1n << 64n, items: [item] })).toThrow('uint64')
    expect(() => encodeSolvencyReport({ gatewayId: fixture.gatewayId as Hex, observedAt: 1n, items: [{ ...item, reserves: UINT256_MAX + 1n }] })).toThrow('uint256')
    expect(() => encodeSolvencyReport({ gatewayId: fixture.gatewayId as Hex, observedAt: 1n, items: [{ ...item, liabilities: -1n }] })).toThrow('uint256')
    expect(() => encodeSolvencyReport({ gatewayId: fixture.gatewayId as Hex, observedAt: 1n, items: [{ ...item, decimals: 256 }] })).toThrow('uint8')
  })
})

describe('numeric parsing', () => {
  test('parseMinor accepts integers within uint256 and nothing else', () => {
    expect(parseMinor('0', 'x')).toBe(0n)
    expect(parseMinor('1250000000', 'x')).toBe(1250000000n)
    expect(parseMinor(UINT256_MAX.toString(), 'x')).toBe(UINT256_MAX)
    for (const bad of ['', '01', '-1', '1.5', '1e6', ' 1', '0x10', (UINT256_MAX + 1n).toString()]) {
      expect(() => parseMinor(bad, 'x')).toThrow()
    }
  })

  test('scaleDecimal scales major units to minor units and refuses finer precision', () => {
    expect(scaleDecimal('1234.56', 6, 'x')).toBe(1234560000n)
    expect(scaleDecimal('1234', 6, 'x')).toBe(1234000000n)
    expect(scaleDecimal('0.000001', 6, 'x')).toBe(1n)
    expect(scaleDecimal('3500.12345678', 8, 'x')).toBe(350012345678n)
    expect(scaleDecimal('7', 0, 'x')).toBe(7n)
    expect(() => scaleDecimal('3500.123456789', 8, 'x')).toThrow('fractional digits')
    expect(() => scaleDecimal('1.1234567', 6, 'x')).toThrow('fractional digits')
    for (const bad of ['-1', '1e3', '01.5', '.5', '1.', 'abc', '']) {
      expect(() => scaleDecimal(bad, 6, 'x')).toThrow()
    }
    expect(() => scaleDecimal('1', 19, 'x')).toThrow('0..18')
  })
})

describe('items and mismatch handling', () => {
  test('sums every reserve source of an asset', () => {
    const items = buildItems(snapshot([{ asset: 'USDC', liabilities_minor: '1250000000', decimals: 6 }]), [
      { asset: 'USDC', source: 'evm', amount: 1000000000n, decimals: 6 },
      { asset: 'USDC', source: 'solana', amount: 300000000n, decimals: 6 },
      { asset: 'ETH', source: 'evm', amount: 5n, decimals: 18 },
    ])
    expect(items).toHaveLength(1)
    expect(items[0]?.reserves).toBe(1300000000n)
    expect(items[0]?.liabilities).toBe(1250000000n)
    expect(items[0]?.asset).toBe(assetKey('USDC'))
  })

  test('fails closed when the ledger owes an asset with no configured reserve source', () => {
    expect(() => buildItems(snapshot([{ asset: 'USDC', liabilities_minor: '1', decimals: 6 }]), [])).toThrow('no reserve source')
  })

  test('fails closed on a decimals mismatch between a reserve source and the snapshot', () => {
    expect(() =>
      buildItems(snapshot([{ asset: 'USDC', liabilities_minor: '1', decimals: 6 }]), [{ asset: 'USDC', source: 'evm', amount: 1n, decimals: 18 }]),
    ).toThrow('decimals')
  })

  test('fails closed on liabilities that are not a minor-unit integer', () => {
    expect(() =>
      buildItems(snapshot([{ asset: 'USDC', liabilities_minor: '1.5', decimals: 6 }]), [{ asset: 'USDC', source: 'evm', amount: 1n, decimals: 6 }]),
    ).toThrow('integer')
  })

  test('fails closed when summed reserves leave uint256', () => {
    expect(() =>
      buildItems(snapshot([{ asset: 'USDC', liabilities_minor: '1', decimals: 6 }]), [
        { asset: 'USDC', source: 'a', amount: UINT256_MAX, decimals: 6 },
        { asset: 'USDC', source: 'b', amount: 1n, decimals: 6 },
      ]),
    ).toThrow('uint256')
  })
})

describe('solana cross-check', () => {
  const account: SolanaAccount = { asset: 'USDC', mint: 'M'.repeat(32), owner: 'O'.repeat(32), tokenAccount: 'T'.repeat(32), decimals: 6 }
  const view = (overrides = {}) => ({ mint: account.mint, owner: account.owner, amount: '500000000', decimals: 6, slot: 300_000_123, ...overrides })

  test('slot buckets round down to the configured window', () => {
    expect(slotBucket(300_000_123, 1000)).toBe('300000000')
    expect(slotBucket(300_000_999, 1000)).toBe('300000000')
    expect(slotBucket(300_001_000, 1000)).toBe('300001000')
  })

  test('agreeing RPCs produce one observation', () => {
    const o = reconcileSolanaViews(account, [view(), view({ slot: 300_000_900 })], 1000)
    expect(o).toEqual({ asset: 'USDC', mint: account.mint, owner: account.owner, amount: '500000000', decimals: 6, slotBucket: '300000000' })
  })

  test('RPCs that disagree on amount, mint, owner or slot bucket fail the run', () => {
    expect(() => reconcileSolanaViews(account, [view(), view({ amount: '500000001' })], 1000)).toThrow('disagree')
    expect(() => reconcileSolanaViews(account, [view(), view({ mint: 'X'.repeat(32) })], 1000)).toThrow('disagree')
    expect(() => reconcileSolanaViews(account, [view(), view({ owner: 'X'.repeat(32) })], 1000)).toThrow('disagree')
    expect(() => reconcileSolanaViews(account, [view(), view({ slot: 300_001_000 })], 1000)).toThrow('disagree')
  })

  test('a token account that is not the configured (mint, owner, decimals) fails the run', () => {
    expect(() => reconcileSolanaViews({ ...account, mint: 'Z'.repeat(32) }, [view(), view()], 1000)).toThrow('holds mint')
    expect(() => reconcileSolanaViews({ ...account, owner: 'Z'.repeat(32) }, [view(), view()], 1000)).toThrow('owned by')
    expect(() => reconcileSolanaViews({ ...account, decimals: 9 }, [view(), view()], 1000)).toThrow('decimals')
  })

  test('fewer than two views is not a cross-check', () => {
    expect(() => reconcileSolanaViews(account, [view()], 1000)).toThrow('fewer than two')
  })
})

describe('custodian response', () => {
  test('scales the configured asset and refuses a decimals mismatch or a missing asset', () => {
    const body = { balances: [{ asset: 'USDC', amount: '1234.5', decimals: 6 }] }
    expect(custodianAmount(body, 'USDC', 6)).toBe(1234500000n)
    expect(() => custodianAmount(body, 'USDC', 8)).toThrow('decimals')
    expect(() => custodianAmount(body, 'ETH', 18)).toThrow('no balance')
  })
})

describe('config', () => {
  test('rejects unknown modes, a receiver in local-simulation, a missing receiver elsewhere, and five-field cron', () => {
    const base = baseConfig('local-simulation')
    expect(() => configSchema.parse({ ...base, mode: 'dev' })).toThrow()
    expect(() => configSchema.parse({ ...base, attestation: { chainSelectorName: CHAIN, consumerAddress: CONSUMER, gasLimit: '1' } })).toThrow()
    expect(() => configSchema.parse({ ...base, mode: 'production' })).toThrow()
    expect(() => configSchema.parse({ ...base, schedule: '0 * * * *' })).toThrow()
    expect(() => configSchema.parse({ ...base, reserves: { ...base.reserves, solana: { rpcUrls: ['https://one'], slotWindow: 1000, accounts: [] } } })).toThrow()
  })

  test('registers a cron handler and an HTTP handler', () => {
    const handlers = initWorkflow(baseConfig('local-simulation'))
    expect(handlers).toHaveLength(2)
    expect((handlers[0]?.trigger as unknown as { config: { schedule: string } }).config.schedule).toBe('0 0 * * * *')
  })

  test('resolveSelector refuses unknown chain names', () => {
    expect(resolveSelector(CHAIN)).toBe(10344971235874465080n)
    expect(() => resolveSelector('not-a-chain')).toThrow('unknown')
  })
})

describe('runs', () => {
  const liabilitiesBody = { checkpoint_id: 'ckpt-1', checkpoint_hash: fixture.checkpointHash, assets: [{ asset: 'USDC', liabilities_minor: '1250000000', decimals: 6 }] }

  const wireMocks = (opts: { writeReport?: (rawReport: Uint8Array) => unknown; liabilities?: unknown; status?: number }) => {
    const http = HttpActionsMock.testInstance()
    http.sendRequest = (request) => {
      if (request.url.endsWith('/api/v1/cre/liabilities')) {
        expect(request.headers?.Authorization).toBe('Bearer dev-solvency-read-token')
        return jsonResponse(opts.status ?? 200, opts.liabilities ?? liabilitiesBody)
      }
      if (request.url.endsWith('/api/v1/cre/reports')) return jsonResponse(202, { recorded: 1 })
      throw new Error(`unexpected request to ${request.url}`)
    }
    const evm = EvmMock.testInstance(resolveSelector(CHAIN))
    const token = addContractMock(evm, { address: TOKEN, abi: erc20Abi })
    token.decimals = () => 6n
    token.balanceOf = () => 1300000000n
    const consumer = addContractMock(evm, { address: CONSUMER, abi: consumerAbi })
    consumer.writeReport = (input) => {
      const result = opts.writeReport?.(input.report.rawReport)
      return (result as never) ?? { txStatus: 'TX_STATUS_SUCCESS', txHash: bytesToBase64(hexToBytes('0x' + 'ab'.repeat(32))) }
    }
    const consensus = ConsensusMock.testInstance()
    // One node in a unit test: identical consensus is the node's own observation, and a node error stays an error.
    consensus.simple = (input) => {
      if (input.observation.case === 'error') throw new Error(input.observation.value)
      if (input.observation.case !== 'value') throw new Error('no observation')
      return input.observation.value
    }
    consensus.report = (input) => {
      const header = new Uint8Array(109)
      const payload = input.encodedPayload
      const raw = new Uint8Array(header.length + payload.length)
      raw.set(header)
      raw.set(payload, header.length)
      return { rawReport: bytesToBase64(raw), sigs: [], configDigest: bytesToBase64(new Uint8Array(32)), seqNr: '1', reportContext: bytesToBase64(new Uint8Array(96)) }
    }
  }

  test('local-simulation returns a labelled result and never reaches writeReport', () => {
    let writes = 0
    wireMocks({ writeReport: () => { writes++ } })
    const runtime = newTestRuntime<Config>(secrets(), { timeProvider: () => 1_800_000_000_000 }, baseConfig('local-simulation'))
    const result = JSON.parse(attest(runtime, 'cron'))
    expect(result.mode).toBe('local-simulation')
    expect(result.written).toBe(false)
    expect(result.observedAt).toBe('1800000000')
    expect(result.items).toEqual([{ asset: assetKey('USDC'), checkpointHash: fixture.checkpointHash, liabilities: '1250000000', reserves: '1300000000', decimals: 6 }])
    expect(result.reportHash).toBe(keccak256(result.report))
    expect(writes).toBe(0)
  })

  test('staging signs the exact encoded report, writes it and returns the transaction hash', () => {
    let delivered: Uint8Array | undefined
    wireMocks({ writeReport: (raw) => { delivered = raw } })
    const runtime = newTestRuntime<Config>(secrets(), { timeProvider: () => 1_800_000_000_000 }, baseConfig('staging'))
    const result = JSON.parse(attest(runtime, 'http'))
    expect(result.mode).toBe('staging')
    expect(result.written).toBe(true)
    expect(result.txHash).toBe('0x' + 'ab'.repeat(32))
    expect(delivered).toBeDefined()
    const body = delivered!.slice(109)
    expect(bytesToHex(body)).toBe(result.report)
    expect(keccak256(body)).toBe(result.reportHash)
  })

  test('a reverted or hashless write fails closed', () => {
    wireMocks({ writeReport: () => ({ txStatus: 'TX_STATUS_REVERTED', errorMessage: 'UnexpectedWorkflow' }) })
    const runtime = newTestRuntime<Config>(secrets(), { timeProvider: () => 1_800_000_000_000 }, baseConfig('staging'))
    expect(() => attest(runtime, 'cron')).toThrow('did not succeed')
    wireMocks({ writeReport: () => ({ txStatus: 'TX_STATUS_SUCCESS' }) })
    const runtime2 = newTestRuntime<Config>(secrets(), { timeProvider: () => 1_800_000_000_000 }, baseConfig('staging'))
    expect(() => attest(runtime2, 'cron')).toThrow('without a transaction hash')
  })

  test('a gateway error or an empty snapshot fails closed before any write', () => {
    wireMocks({ status: 401, writeReport: () => { throw new Error('must not write') } })
    expect(() => attest(newTestRuntime<Config>(secrets(), {}, baseConfig('staging')), 'cron')).toThrow('HTTP 401')
    wireMocks({ liabilities: { ...liabilitiesBody, assets: [] }, writeReport: () => { throw new Error('must not write') } })
    expect(() => attest(newTestRuntime<Config>(secrets(), {}, baseConfig('staging')), 'cron')).toThrow('no assets')
  })

  test('an on-chain decimals() that differs from the config fails closed', () => {
    wireMocks({ writeReport: () => { throw new Error('must not write') } })
    const evm = EvmMock.testInstance(resolveSelector(CHAIN))
    const token = addContractMock(evm, { address: TOKEN, abi: erc20Abi })
    token.decimals = () => 18n
    token.balanceOf = () => 1n
    expect(() => attest(newTestRuntime<Config>(secrets(), {}, baseConfig('staging')), 'cron')).toThrow('decimals')
  })
})
