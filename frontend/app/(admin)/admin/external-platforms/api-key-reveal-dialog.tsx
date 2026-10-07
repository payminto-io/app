"use client";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ApiKeyReveal } from "@/components/api-key-reveal";

export function ApiKeyRevealDialog({
  apiKey,
  onClose,
}: {
  apiKey: string | null;
  onClose: () => void;
}) {
  return (
    <Dialog open={apiKey !== null} onOpenChange={() => onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Your new API key</DialogTitle>
        </DialogHeader>

        {apiKey ? <ApiKeyReveal secret={apiKey} label="Secret key" onConfirm={onClose} /> : null}
      </DialogContent>
    </Dialog>
  );
}
