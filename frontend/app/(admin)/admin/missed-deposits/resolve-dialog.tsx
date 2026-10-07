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
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Resolve deposit</DialogTitle>
            <DialogDescription>
              <CurrencyDisplay amount={deposit.amount} currency={deposit.currencyCode} size="sm" /> on{" "}
              {chainName(deposit.blockchainCode)}
            </DialogDescription>
          </DialogHeader>

          <FormField label="Payment reference" htmlFor="resolve-ref" hint="Optional. Links the deposit to that payment.">
            <TextInput
              id="resolve-ref"
              value={paymentRef}
              onChange={(e) => setPaymentRef(e.target.value)}
              placeholder="pay_abc123"
              className="font-mono"
            />
          </FormField>

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
