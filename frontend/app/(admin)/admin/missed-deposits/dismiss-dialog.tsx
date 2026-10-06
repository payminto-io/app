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
import { useDismissMissedDeposit } from "@/lib/query/hooks/use-admin";
import type { MissedDeposit } from "@/lib/query/hooks/use-admin";

export function DismissDialog({
  deposit,
  onClose,
}: {
  deposit: MissedDeposit;
  onClose: () => void;
}) {
  const dismiss = useDismissMissedDeposit();
  const [reason, setReason] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    await dismiss.mutateAsync({
      id: deposit.id,
      reason: reason || undefined,
    });
    onClose();
  }

  return (
    <Dialog open onOpenChange={() => onClose()}>
      <DialogContent>
        <form onSubmit={handleSubmit} className="space-y-5">
          <DialogHeader>
            <DialogTitle>Dismiss Missed Deposit</DialogTitle>
            <DialogDescription>
              Deposit of{" "}
              <span className="font-medium tabular-nums">{deposit.amount}</span>{" "}
              {deposit.currencyCode} on {deposit.blockchainCode}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <Label htmlFor="dismiss-reason">Reason</Label>
            <Input
              id="dismiss-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="e.g. duplicate, test transaction"
            />
            <p className="text-[11px] text-muted-foreground">
              Optional. Provide a reason for dismissing.
            </p>
          </div>

          <DialogFooter>
            <Button variant="outline" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button
              type="submit"
              variant="destructive"
              disabled={dismiss.isPending}
            >
              {dismiss.isPending ? "Dismissing..." : "Dismiss"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
