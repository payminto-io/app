"use client";

import { useState } from "react";
import { Clock3, ExternalLink } from "lucide-react";
import {
  useWithdrawalsList,
  useCreateWithdrawal,
  useVerifyWithdrawalOTP,
} from "@/lib/query/hooks/use-withdrawals";
import { isApiError } from "@/lib/query/hooks/use-auth";
import { presentWithdrawalState, type Withdrawal } from "@/lib/api/withdrawal-contract";
import { getExplorerUrl } from "@/lib/formatters";
import { chainName, PAYOUT_CHAINS } from "@/lib/chains";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { CurrencyDisplay } from "@/components/currency-display";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { DateTime } from "@/components/date-time";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormField, TextInput } from "@/components/ui/form-field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ErrorState } from "@/components/ui/states";
import { StatusBadge } from "@/components/ui/status-badge";

const EXPLORER_CHAIN: Record<string, string> = {
  ETH: "ethereum",
  BTC: "bitcoin",
  BASE: "base",
  POLYGON: "polygon",
  TRON: "tron",
};

function truncateMiddle(value: string): string {
  return value.length > 18 ? `${value.slice(0, 8)}...${value.slice(-6)}` : value;
}

const COLUMNS: DataTableColumn<Withdrawal>[] = [
  {
    key: "amount",
    header: "Amount",
    align: "right",
    className: "w-0",
    cell: (w) => <CurrencyDisplay amount={w.amount} currency={w.currencyCode} size="sm" />,
  },
  { key: "status", header: "Status", cell: (w) => <StatusBadge status={presentWithdrawalState(w.state)} /> },
  {
    key: "recipient",
    header: "Recipient",
    cell: (w) => <CopyField value={w.recipientAddress} display={truncateMiddle(w.recipientAddress)} boxed={false} />,
  },
  {
    key: "chain",
    header: "Chain",
    className: "text-ink-soft",
    cell: (w) => chainName(w.blockchainCode),
  },
  {
    key: "tx",
    header: "Transaction",
    cell: (w) => {
      const chain = EXPLORER_CHAIN[w.blockchainCode?.toUpperCase()];
      if (!w.transactionHash || !chain) return null;
      return (
        <a
          href={getExplorerUrl(chain, w.transactionHash)}
          target="_blank"
          rel="noopener noreferrer"
          className="tap inline-flex items-center gap-1 rounded-xs font-mono text-label text-tide hover:text-tide-strong"
        >
          {truncateMiddle(w.transactionHash)}
          <ExternalLink className="size-3" />
        </a>
      );
    },
  },
  {
    key: "created",
    header: "Created",
    align: "right",
    className: "text-ink-soft",
    cell: (w) => <DateTime value={w.createdAt} />,
  },
];

export default function WithdrawalsPage() {
  const [open, setOpen] = useState(false);
  const { data, error, refetch } = useWithdrawalsList();

  return (
    <div className="space-y-5">
      <PageHeader title="Withdrawals">
        <Button onClick={() => setOpen(true)}>Request payout</Button>
      </PageHeader>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={COLUMNS}
          rows={data?.withdrawals ?? []}
          loading={!data}
          getRowId={(w) => w.id}
          emptyTitle="No withdrawals yet."
          emptyDescription="Payouts you request appear here with their approval state."
          emptyAction={
            <Button size="sm" onClick={() => setOpen(true)}>
              Request payout
            </Button>
          }
        />
      )}

      <CreatePayoutDialog open={open} onOpenChange={setOpen} />
    </div>
  );
}

type DialogStep = "form" | "otp" | "awaiting-approval";

function CreatePayoutDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const create = useCreateWithdrawal();
  const [step, setStep] = useState<DialogStep>("form");
  const [withdrawalId, setWithdrawalId] = useState<number>(0);
  const [address, setAddress] = useState("");
  const [blockchain, setBlockchain] = useState("ETH");
  const [currency, setCurrency] = useState("USDT");
  const [amount, setAmount] = useState("");
  const [memo, setMemo] = useState("");
  const [otpCode, setOtpCode] = useState("");
  const [otpPrompt, setOtpPrompt] = useState("");
  const [err, setErr] = useState<string | null>(null);

  const verifyOTP = useVerifyWithdrawalOTP(withdrawalId);

  function reset() {
    setStep("form");
    setWithdrawalId(0);
    setAddress("");
    setBlockchain("ETH");
    setCurrency("USDT");
    setAmount("");
    setMemo("");
    setOtpCode("");
    setOtpPrompt("");
    setErr(null);
  }

  function handleClose(v: boolean) {
    if (!v) reset();
    onOpenChange(v);
  }

  async function onSubmitForm(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      const result = await create.mutateAsync({
        recipientAddress: address,
        blockchainCode: blockchain,
        currencyCode: currency,
        amount,
        memo: memo || undefined,
      });
      setWithdrawalId(result.withdrawal.id);
      if (result.otp.otpRequired) {
        setOtpPrompt(result.otp.message || "Enter the one-time code supplied by the server.");
        setStep("otp");
      } else {
        setStep("awaiting-approval");
      }
    } catch (error) {
      setErr(isApiError(error) ? error.message : "The payout could not be created.");
    }
  }

  async function onSubmitOTP(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      await verifyOTP.mutateAsync(otpCode);
      setStep("awaiting-approval");
    } catch (error) {
      setErr(isApiError(error) ? error.message : "That code was not accepted.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {step === "form" ? "Request payout" : step === "otp" ? "Enter the code" : "Awaiting approval"}
          </DialogTitle>
          {step === "form" ? (
            <DialogDescription>Nothing moves until the payout is approved.</DialogDescription>
          ) : null}
        </DialogHeader>

        {step === "form" && (
          <form onSubmit={onSubmitForm} className="space-y-4">
            <FormField label="Recipient address" htmlFor="wd-addr">
              <TextInput id="wd-addr" required value={address} onChange={(e) => setAddress(e.target.value)} className="font-mono" autoComplete="off" spellCheck={false} />
            </FormField>
            <div className="grid grid-cols-2 gap-3">
              <FormField label="Chain" htmlFor="wd-chain">
                <Select value={blockchain} onValueChange={(v) => { if (v) setBlockchain(v); }}>
                  <SelectTrigger id="wd-chain" className="w-full"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {PAYOUT_CHAINS.map((c) => (
                      <SelectItem key={c} value={c}>{chainName(c)}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField label="Asset" htmlFor="wd-cur">
                <TextInput id="wd-cur" required value={currency} onChange={(e) => setCurrency(e.target.value)} className="uppercase" />
              </FormField>
            </div>
            <FormField label="Amount" htmlFor="wd-amt">
              <TextInput id="wd-amt" required inputMode="decimal" value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0.00" className="num" />
            </FormField>
            <FormField label="Memo" htmlFor="wd-memo" hint="Optional">
              <TextInput id="wd-memo" value={memo} onChange={(e) => setMemo(e.target.value)} />
            </FormField>
            {err ? <p role="alert" className="text-body-sm text-bad">{err}</p> : null}
            <DialogFooter>
              <Button type="submit" disabled={create.isPending}>
                {create.isPending ? "Submitting..." : "Submit for approval"}
              </Button>
            </DialogFooter>
          </form>
        )}

        {step === "otp" && (
          <form onSubmit={onSubmitOTP} className="space-y-4">
            <p className="text-body-sm text-ink-soft">{otpPrompt}</p>
            <FormField label="Code" htmlFor="otp-code">
              <TextInput
                id="otp-code"
                required
                autoFocus
                autoComplete="one-time-code"
                inputMode="numeric"
                aria-describedby="otp-help"
                value={otpCode}
                onChange={(e) => setOtpCode(e.target.value)}
                className="num font-mono tracking-widest"
              />
            </FormField>
            <p id="otp-help" className="sr-only">
              Enter the one-time code. Verification submits the payout for approval; it does not approve or process funds.
            </p>
            {err ? <p role="alert" className="text-body-sm text-bad">{err}</p> : null}
            <DialogFooter>
              <Button type="submit" disabled={verifyOTP.isPending}>
                {verifyOTP.isPending ? "Verifying..." : "Verify"}
              </Button>
            </DialogFooter>
          </form>
        )}

        {step === "awaiting-approval" && (
          <div className="space-y-4" role="status" aria-live="polite">
            <div className="flex items-start gap-3">
              <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-wait-tint text-wait">
                <Clock3 className="size-4" aria-hidden="true" />
              </span>
              <p className="pt-1 text-body text-ink">
                Payout <span className="num font-medium">#{withdrawalId}</span> is waiting for approval. No funds have moved.
              </p>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => handleClose(false)}>
                Close
              </Button>
            </DialogFooter>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
