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
import { FormField, TextInput } from "@/components/ui/form-field";
import { CurrencyDisplay } from "@/components/currency-display";
import { chainName } from "@/lib/chains";
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
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Dismiss deposit</DialogTitle>
            <DialogDescription>
              <CurrencyDisplay amount={deposit.amount} currency={deposit.currencyCode} size="sm" /> on{" "}
              {chainName(deposit.blockchainCode)}
            </DialogDescription>
          </DialogHeader>

          <FormField label="Reason" htmlFor="dismiss-reason" hint="Optional">
            <TextInput
              id="dismiss-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Duplicate test transfer"
            />
          </FormField>

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
