"use client";

import { useState, useSyncExternalStore } from "react";
import { TriangleAlert, Wallet } from "lucide-react";
import type { ChainLeg, Money } from "@/lib/model";
import { TEMPO_TESTNET, tempoToken } from "@/lib/tempo";
import { payWithWallet, txUrl, walletErrorMessage, walletProvider } from "@/lib/tempo-wallet";
import { Amount, CopyField, ExplorerLink, Row, truncateMiddle, useCopy, WaitingBar } from "./ui";

const noSubscribe = () => () => undefined;

type Send = { kind: "idle" } | { kind: "sending" } | { kind: "sent"; hash: string } | { kind: "error"; message: string };

/**
 * A Tempo leg: the merchant's fixed address, the exact token amount and the
 * memo that tells this payment apart. No QR: an address alone would send a
 * transfer without the memo, which the watcher cannot match.
 */
export function TempoPay({ chain, due, onExpire, note }: { chain: ChainLeg; due?: Money | null; onExpire?: () => void; note?: React.ReactNode }) {
  const [copied, copy] = useCopy();
  const [send, setSend] = useState<Send>({ kind: "idle" });
  const hasWallet = useSyncExternalStore(noSubscribe, () => Boolean(walletProvider()), () => false);
  const amount = due === null ? undefined : due ?? chain.due;
  const memo = chain.memo;
  const token = tempoToken(chain.asset);

  if (!token) {
    return <p role="alert" className="rounded-sm bg-bad-tint p-3 text-body-sm text-bad">This Tempo token is not supported by this checkout. Contact the merchant before sending.</p>;
  }

  if (!memo) {
    return (
      <p role="alert" className="flex items-start gap-2 rounded-sm bg-bad-tint p-3 text-body-sm text-bad">
        <TriangleAlert size={16} strokeWidth={1.75} className="mt-0.5 shrink-0" />
        This payment has no Tempo memo, so a transfer could not be matched to it. Choose another method.
      </p>
    );
  }

  const pay = async () => {
    if (!amount) return;
    setSend({ kind: "sending" });
    try {
      const hash = await payWithWallet({ to: chain.address as `0x${string}`, amount: amount.amount, memo: memo.hex, asset: chain.asset });
      setSend({ kind: "sent", hash });
    } catch (cause) {
      setSend({ kind: "error", message: walletErrorMessage(cause) });
    }
  };

  return (
    <div className="flex flex-col gap-5">
      {amount ? (
        <CopyField
          label="Amount"
          value={amount.amount}
          display={<Amount money={amount} size="h2" />}
          copied={copied === "amount"}
          onCopy={() => copy(amount.amount, "amount")}
          mono={false}
        />
      ) : null}

      <div className="flex flex-col divide-y divide-line rounded-md border border-line px-4">
        <Row label="Network">
          {TEMPO_TESTNET.name} <span className="whitespace-nowrap text-ink-soft">(Moderato, chain {TEMPO_TESTNET.chainId})</span>
        </Row>
        <Row label="Token">
          {token.symbol} <span className="whitespace-nowrap text-ink-soft">({token.standard}, {token.decimals} decimals)</span>
        </Row>
      </div>

      <CopyField
        label={`Merchant address on ${TEMPO_TESTNET.name}`}
        value={chain.address}
        copied={copied === "address"}
        onCopy={() => copy(chain.address, "address")}
      />

      <div className="flex flex-col gap-1.5">
        <CopyField
          label="Memo (bytes32)"
          value={memo.hex}
          display={
            <span className="flex flex-col gap-0.5">
              <span className="font-medium">{memo.reference}</span>
              <span className="text-ink-soft">{memo.hex}</span>
            </span>
          }
          copied={copied === "memo"}
          onCopy={() => copy(memo.hex, "memo")}
        />
        <p className="text-caption text-ink-soft">
          Send with <span className="mono">transferWithMemo</span> and this bytes32 memo, or the payment cannot be matched.
        </p>
      </div>

      <p className="text-body-sm text-ink-soft">Fees are paid in stablecoins and can be sponsored, so you do not need a gas token.</p>

      {hasWallet && amount ? (
        <div className="flex flex-col gap-2">
          <button type="button" className="btn btn-outline w-full" onClick={() => void pay()} disabled={send.kind === "sending" || send.kind === "sent"}>
            <Wallet size={16} strokeWidth={1.75} />
            {send.kind === "sending" ? "Confirm in your wallet" : send.kind === "sent" ? "Sent from your wallet" : "Pay with browser wallet"}
          </button>
          {send.kind === "sent" ? (
            <p className="text-body-sm text-ink-soft" aria-live="polite">
              Transaction <ExplorerLink href={txUrl(send.hash)}>{truncateMiddle(send.hash)}</ExplorerLink> sent. This page updates when Tempo confirms it.
            </p>
          ) : null}
          {send.kind === "error" ? (
            <p role="alert" className="flex items-start gap-2 rounded-sm bg-bad-tint p-3 text-body-sm text-bad">
              <TriangleAlert size={16} strokeWidth={1.75} className="mt-0.5 shrink-0" />
              {send.message}
            </p>
          ) : null}
        </div>
      ) : null}

      <WaitingBar until={chain.quoteExpiresAt} onExpire={onExpire} note={note} />

      <ExplorerLink href={TEMPO_TESTNET.explorerUrl}>Tempo explorer</ExplorerLink>
    </div>
  );
}
