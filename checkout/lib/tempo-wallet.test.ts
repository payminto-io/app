import { afterEach, describe, expect, it, vi } from "vitest";
import { decodeFunctionData, parseAbi, type EIP1193Provider } from "viem";
import { payWithWallet, walletErrorMessage } from "./tempo-wallet";
import { encodeMemo, TEMPO_TESTNET } from "./tempo";

const to = "0x7a3f9c21b84e5d06c1f2a9e8d47b3c5f60e1d2a4" as const;
const hash = `0x${"12".repeat(32)}` as const;
const payment = { to, amount: "1250.000001", memo: encodeMemo("PM-1042") };

function installWallet(switchError?: number) {
  let switches = 0;
  const request = vi.fn(async ({ method }: { method: string; params?: unknown }) => {
    if (method === "eth_requestAccounts") return [to];
    if (method === "wallet_switchEthereumChain") {
      switches++;
      if (switchError && (switchError !== 4902 || switches === 1)) throw { code: switchError, message: "Wallet error" };
      return null;
    }
    if (method === "wallet_addEthereumChain") return null;
    if (method === "eth_chainId") return `0x${TEMPO_TESTNET.chainId.toString(16)}`;
    if (method === "eth_sendTransaction") return hash;
    throw new Error(`Unexpected method ${method}`);
  });
  vi.stubGlobal("window", { ethereum: { request } as unknown as EIP1193Provider });
  return request;
}

afterEach(() => vi.unstubAllGlobals());

describe("Tempo wallet payment", () => {
  it("stops after a declined switch instead of prompting to add a network", async () => {
    const request = installWallet(4001);
    await expect(payWithWallet(payment)).rejects.toBeDefined();
    expect(request.mock.calls.map(([arg]) => arg.method)).toEqual(["eth_requestAccounts", "wallet_switchEthereumChain"]);
  });

  it("adds the network only when the wallet reports an unknown chain", async () => {
    const request = installWallet(4902);
    await expect(payWithWallet(payment)).resolves.toBe(hash);
    expect(request.mock.calls.map(([arg]) => arg.method)).toEqual([
      "eth_requestAccounts", "wallet_switchEthereumChain", "wallet_addEthereumChain", "wallet_switchEthereumChain", "eth_chainId", "eth_sendTransaction",
    ]);
  });

  it.each([
    ["PATHUSD", "0x20c0000000000000000000000000000000000000"],
    ["ALPHAUSD", "0x20c0000000000000000000000000000000000001"],
  ])("sends the issued %s token with the exact amount and memo", async (asset, token) => {
    const request = installWallet();
    await payWithWallet({ ...payment, asset });
    const call = request.mock.calls.find(([arg]) => arg.method === "eth_sendTransaction")?.[0];
    const [tx] = call?.params as [{ to: string; data: `0x${string}` }];
    expect(tx.to.toLowerCase()).toBe(token);
    const decoded = decodeFunctionData({ abi: parseAbi(["function transferWithMemo(address to, uint256 amount, bytes32 memo)"]), data: tx.data });
    expect(decoded.functionName).toBe("transferWithMemo");
    expect(decoded.args).toEqual([expect.stringMatching(new RegExp(`^${to}$`, "i")), BigInt("1250000001"), payment.memo]);
  });

  it.each(["UNKNOWN", "ETH"])("refuses an unsupported %s token before opening the wallet", async (asset) => {
    const request = installWallet();
    await expect(payWithWallet({ ...payment, asset })).rejects.toThrow("not supported");
    expect(request).not.toHaveBeenCalled();
  });

  it.each(["0", "-1", "0.0000001"])("refuses an unsendable amount %s before opening the wallet", async (amount) => {
    const request = installWallet();
    await expect(payWithWallet({ ...payment, amount })).rejects.toBeDefined();
    expect(request).not.toHaveBeenCalled();
  });

  it("recognizes a rejection wrapped by a contract error", () => {
    expect(walletErrorMessage({ shortMessage: "Contract execution failed", cause: { code: -1, cause: { code: 4001 } } })).toBe("The wallet request was declined. Nothing was sent.");
  });
});
