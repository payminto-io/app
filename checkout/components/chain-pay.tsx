"use client";

import { QRCodeSVG } from "qrcode.react";
import { Clock3 } from "lucide-react";
import type { ChainLeg, Money } from "@/lib/model";
import { formatAmount } from "@/lib/money";
import { Amount, CopyField, Countdown, cx, useCopy } from "./ui";

/**
 * Address, exact amount and QR for a chain leg. Shared by awaiting and under
 * paid. QR is always on white so scanners get contrast (references.md, Base
 * App receive).
 */
export function ChainPay({ chain, due, onExpire, note }: { chain: ChainLeg; due?: Money; onExpire?: () => void; note?: React.ReactNode }) {
  const [copied, copy] = useCopy();
  const amount = due ?? chain.due;

  return (
    <div className="flex flex-col gap-5">
      {amount ? (
        <CopyField
          label="Amount"
          value={formatAmount(amount.amount, amount.code)}
          display={<Amount money={amount} size="h2" />}
          copied={copied === "amount"}
          onCopy={() => copy(formatAmount(amount.amount, amount.code), "amount")}
          mono={false}
        />
      ) : null}

      <CopyField
        label={`${chain.asset} address on ${chain.networkName}`}
        value={chain.address}
        copied={copied === "address"}
        onCopy={() => copy(chain.address, "address")}
      />

      <div className="flex flex-col items-center gap-3">
        <div className="rounded-lg border border-line bg-white p-3">
          <QRCodeSVG value={chain.address} size={168} level="M" marginSize={0} title={`${chain.asset} address on ${chain.networkName}`} bgColor="#ffffff" fgColor="#15181d" />
        </div>
        <p className="text-center text-caption text-ink-soft">
          Only {chain.asset} on {chain.networkName} is credited to this address.
        </p>
      </div>

      <div className={cx("flex items-center justify-between gap-4 rounded-sm bg-surface-sunken px-3 py-2.5")}>
        <span className="inline-flex items-center gap-2 text-body-sm text-ink-soft">
          <Clock3 size={16} strokeWidth={1.75} />
          {note ?? "Waiting for your transfer"}
        </span>
        <Countdown until={chain.quoteExpiresAt} label="Expires in" onExpire={onExpire} />
      </div>
    </div>
  );
}
