/**
 * Browser wallet path for the Tempo rail: one TIP-20 `transferWithMemo` call
 * on Moderato. viem loads only when the payer presses the button.
 */

import type { EIP1193Provider } from "viem";
import { TEMPO_TESTNET, toBaseUnits } from "./tempo";

const TIP20_TRANSFER_WITH_MEMO = [
  {
    type: "function",
    name: "transferWithMemo",
    stateMutability: "nonpayable",
    inputs: [
      { name: "to", type: "address" },
      { name: "amount", type: "uint256" },
      { name: "memo", type: "bytes32" },
    ],
    outputs: [],
  },
] as const;

export function walletProvider(): EIP1193Provider | undefined {
  if (typeof window === "undefined") return undefined;
  return (window as unknown as { ethereum?: EIP1193Provider }).ethereum;
}

export function txUrl(hash: string): string {
  return `${TEMPO_TESTNET.explorerUrl}/tx/${hash}`;
}

/** Sends `amount` pathUSD to `to` with `memo`; resolves to the transaction hash. */
export async function payWithWallet({ to, amount, memo }: { to: `0x${string}`; amount: string; memo: `0x${string}` }): Promise<`0x${string}`> {
  const provider = walletProvider();
  if (!provider) throw new Error("No browser wallet found.");
  const { createWalletClient, custom, defineChain } = await import("viem");
  // A plain EVM chain definition: wallets reject Tempo-specific request fields.
  const chain = defineChain({
    id: TEMPO_TESTNET.chainId,
    name: "Tempo Testnet (Moderato)",
    nativeCurrency: { name: "USD", symbol: "USD", decimals: 18 },
    rpcUrls: { default: { http: [TEMPO_TESTNET.rpcUrl] } },
    blockExplorers: { default: { name: "Tempo Explorer", url: TEMPO_TESTNET.explorerUrl } },
    testnet: true,
  });
  const client = createWalletClient({ chain, transport: custom(provider) });
  const [account] = await client.requestAddresses();
  if (!account) throw new Error("The wallet shared no account.");
  try {
    await client.switchChain({ id: chain.id });
  } catch {
    await client.addChain({ chain });
    await client.switchChain({ id: chain.id });
  }
  return client.writeContract({
    account,
    address: TEMPO_TESTNET.token.address,
    abi: TIP20_TRANSFER_WITH_MEMO,
    functionName: "transferWithMemo",
    args: [to, toBaseUnits(amount, TEMPO_TESTNET.token.decimals), memo],
  });
}

/** A short, payer-facing sentence for a wallet failure. */
export function walletErrorMessage(cause: unknown): string {
  const code = (cause as { code?: number })?.code;
  if (code === 4001) return "The wallet request was declined. Nothing was sent.";
  const short = (cause as { shortMessage?: string })?.shortMessage;
  return short ?? (cause instanceof Error ? cause.message : "The wallet could not send this payment.");
}
