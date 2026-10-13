"use client";

import { QRCodeSVG } from "qrcode.react";
import { isTempo, type ChainLeg, type Money } from "@/lib/model";
import { formatAmount } from "@/lib/money";
import { TempoPay } from "./tempo-pay";
import { Amount, CopyField, useCopy, WaitingBar } from "./ui";

/**
 * Address, exact amount and QR for a chain leg. Shared by awaiting and under
 * paid. QR is always on white so scanners get contrast (references.md, Base
 * App receive). A Tempo leg renders TempoPay instead: fixed merchant address
 * plus the memo that identifies the payment.
 */
export function ChainPay({ chain, due, onExpire, note }: { chain: ChainLeg; due?: Money; onExpire?: () => void; note?: React.ReactNode }) {
  if (isTempo(chain.network)) return <TempoPay chain={chain} due={due} onExpire={onExpire} note={note} />;
  return <AddressPay chain={chain} due={due} onExpire={onExpire} note={note} />;
}

function AddressPay({ chain, due, onExpire, note }: { chain: ChainLeg; due?: Money; onExpire?: () => void; note?: React.ReactNode }) {
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

      <WaitingBar until={chain.quoteExpiresAt} onExpire={onExpire} note={note} />
    </div>
  );
}
