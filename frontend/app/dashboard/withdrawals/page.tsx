"use client";

import { useState } from "react";
import {
  useWithdrawalsList,
  useCreateWithdrawal,
  useVerifyWithdrawalOTP,
} from "@/lib/query/hooks/use-withdrawals";
import { ErrorState } from "@/components/ui/states";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusBadge } from "@/components/ui/status-badge";
import { BlockchainIcon } from "@/components/blockchain-icon";
import { FormField, TextInput } from "@/components/ui/form-field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { isApiError } from "@/lib/query/hooks/use-auth";
import { presentWithdrawalState } from "@/lib/api/withdrawal-contract";
import type { BlockchainNetwork } from "@/lib/types";
import { Clock3 } from "lucide-react";

const CHAIN_MAP: Record<string, BlockchainNetwork> = {
  ETH: "ethereum", BTC: "bitcoin", BASE: "base", POLYGON: "polygon", TRON: "tron",
};

export default function WithdrawalsPage() {
  const [open, setOpen] = useState(false);
  const { data, isLoading, error, refetch } = useWithdrawalsList();

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">Withdrawals</h1>
        <Button
          onClick={() => setOpen(true)}
          className="h-9 rounded-lg bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]"
        >
          Request Payout
        </Button>
      </div>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full rounded-lg" />
          ))}
        </div>
      ) : data?.withdrawals && data.withdrawals.length > 0 ? (
        <Card className="border-border shadow-sm overflow-hidden">
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="border-border hover:bg-transparent">
                  <TableHead className="pm-label pl-6">ID</TableHead>
                  <TableHead className="pm-label">Recipient</TableHead>
                  <TableHead className="pm-label">Amount</TableHead>
                  <TableHead className="pm-label">Chain</TableHead>
                  <TableHead className="pm-label">Status</TableHead>
                  <TableHead className="pm-label text-right pr-6">Created</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.withdrawals.map((w) => (
                  <TableRow key={w.id} className="border-border/60 hover:bg-muted/30">
                    <TableCell className="pl-6 font-medium text-[13px]">
                      #{w.id}
                    </TableCell>
                    <TableCell className="font-mono text-[12px] text-muted-foreground">
                      {w.recipientAddress.slice(0, 8)}...{w.recipientAddress.slice(-6)}
                    </TableCell>
                    <TableCell className="font-semibold tabular-nums text-[13px]">
                      {w.amount}{" "}
                      <span className="text-muted-foreground font-normal">
                        {w.currencyCode}
                      </span>
                    </TableCell>
                    <TableCell>
                      {CHAIN_MAP[w.blockchainCode?.toUpperCase()] ? (
                        <BlockchainIcon
                          blockchain={CHAIN_MAP[w.blockchainCode.toUpperCase()]}
                          size="sm"
                          showLabel
                        />
                      ) : (
                        <span className="text-xs text-muted-foreground">
                          {w.blockchainCode || "Unknown chain"}
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      <StatusBadge status={presentWithdrawalState(w.state)} />
                    </TableCell>
                    <TableCell className="text-right pr-6 text-[12px] text-muted-foreground">
                      {new Date(w.createdAt).toLocaleDateString()}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-border shadow-sm">
          <CardContent className="flex h-40 items-center justify-center">
            <p className="text-sm text-muted-foreground">No withdrawals yet</p>
          </CardContent>
        </Card>
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
      setErr(isApiError(error) ? error.message : "Failed to create payout.");
    }
  }

  async function onSubmitOTP(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      await verifyOTP.mutateAsync(otpCode);
      setStep("awaiting-approval");
    } catch (error) {
      setErr(isApiError(error) ? error.message : "OTP verification failed.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {step === "form" ? "Request Payout" : step === "otp" ? "Verify OTP" : "Awaiting Approval"}
          </DialogTitle>
        </DialogHeader>

        {step === "form" && (
          <form onSubmit={onSubmitForm} className="space-y-4">
            <FormField label="Recipient Address" htmlFor="wd-addr">
              <TextInput id="wd-addr" required value={address} onChange={(e) => setAddress(e.target.value)} />
            </FormField>
            <div className="grid grid-cols-2 gap-3">
              <FormField label="Blockchain" htmlFor="wd-chain">
                <Select value={blockchain} onValueChange={(v) => { if (v) setBlockchain(v); }}>
                  <SelectTrigger id="wd-chain"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    {["ETH", "BTC", "BASE", "POLYGON", "TRON"].map((c) => (
                      <SelectItem key={c} value={c}>{c}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
              <FormField label="Currency" htmlFor="wd-cur">
                <TextInput id="wd-cur" required value={currency} onChange={(e) => setCurrency(e.target.value)} />
              </FormField>
            </div>
            <FormField label="Amount" htmlFor="wd-amt">
              <TextInput id="wd-amt" required value={amount} onChange={(e) => setAmount(e.target.value)} placeholder="0.00" />
            </FormField>
            <FormField label="Memo" htmlFor="wd-memo" hint="Optional">
              <TextInput id="wd-memo" value={memo} onChange={(e) => setMemo(e.target.value)} />
            </FormField>
            {err ? <p role="alert" className="text-sm text-destructive">{err}</p> : null}
            <DialogFooter>
              <Button type="submit" disabled={create.isPending} className="bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]">
                {create.isPending ? "Submitting..." : "Submit for Review"}
              </Button>
            </DialogFooter>
          </form>
        )}

        {step === "otp" && (
          <form onSubmit={onSubmitOTP} className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {otpPrompt}
            </p>
            <FormField label="OTP Code" htmlFor="otp-code">
              <TextInput
                id="otp-code"
                required
                autoFocus
                autoComplete="one-time-code"
                inputMode="numeric"
                aria-describedby="otp-help"
                value={otpCode}
                onChange={(e) => setOtpCode(e.target.value)}
                placeholder="Enter one-time code"
              />
            </FormField>
            <p id="otp-help" className="sr-only">
              Enter the one-time code. Verification submits the payout for approval; it does not approve or process funds.
            </p>
            {err ? <p role="alert" className="text-sm text-destructive">{err}</p> : null}
            <DialogFooter>
              <Button type="submit" disabled={verifyOTP.isPending} className="bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]">
                {verifyOTP.isPending ? "Verifying..." : "Verify OTP"}
              </Button>
            </DialogFooter>
          </form>
        )}

        {step === "awaiting-approval" && (
          <div
            className="space-y-4 text-center"
            role="status"
            aria-live="polite"
          >
            <div className="mx-auto flex size-12 items-center justify-center rounded-full bg-muted">
              <Clock3 className="size-6 text-muted-foreground" aria-hidden="true" />
            </div>
            <p className="text-sm text-muted-foreground">
              Payout #{withdrawalId} is awaiting approval. No funds have been processed.
            </p>
            <Button variant="outline" onClick={() => handleClose(false)}>
              Close
            </Button>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
