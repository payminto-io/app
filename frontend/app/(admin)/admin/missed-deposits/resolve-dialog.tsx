"use client";

import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { useResolveMissedDeposit } from "@/lib/query/hooks/use-admin";
import type { MissedDeposit } from "@/lib/query/hooks/use-admin";

export function ResolveDialog({
  deposit,
  onClose,
}: {
  deposit: MissedDeposit;
  onClose: () => void;
}) {
  const resolve = useResolveMissedDeposit();
  const [paymentRef, setPaymentRef] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await resolve.mutateAsync({
      id: deposit.id,
      paymentReferenceID: paymentRef || undefined,
    });
    onClose();
  }

  return (
    <Dialog open onOpenChange={() => onClose()}>
      <DialogContent>
        <form onSubmit={handleSubmit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>Resolve Missed Deposit</DialogTitle>
            <DialogDescription>
              Deposit of{" "}
              <span className="font-medium tabular-nums">{deposit.amount}</span>{" "}
              {deposit.currencyCode} on {deposit.blockchainCode}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="resolve-ref">Payment Reference ID</Label>
            <Input
              id="resolve-ref"
              value={paymentRef}
              onChange={(e) => setPaymentRef(e.target.value)}
              placeholder="pay_abc123"
            />
            <p className="text-[11px] text-muted-foreground">
              Optional. Link this deposit to an existing payment.
            </p>
          </div>

          <DialogFooter>
            <Button variant="outline" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={resolve.isPending}>
              {resolve.isPending ? "Resolving..." : "Resolve"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
