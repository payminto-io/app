"use client";

import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
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
          <DialogTitle>API Key Created</DialogTitle>
          <DialogDescription>
            Save this API key now. You will not be able to see it again.
          </DialogDescription>
        </DialogHeader>

        {apiKey ? <ApiKeyReveal secret={apiKey} onConfirm={onClose} /> : null}

        <DialogFooter>
          <Button onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
