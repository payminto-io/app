"use client";

import { useState } from "react";
import {
  useCreateApiKey,
  useRevokeApiKey,
} from "@/lib/query/hooks/use-merchant-misc";
import { Button } from "@/components/ui/button";
import { FormField, TextInput } from "@/components/ui/form-field";
import { ApiKeyReveal } from "@/components/api-key-reveal";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { isApiError } from "@/lib/query/hooks/use-auth";

export function CreateKeyDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onCreated: (rawKey: string) => void;
}) {
  const create = useCreateApiKey();
  const [name, setName] = useState("");
  const [err, setErr] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      const result = await create.mutateAsync({ name });
      if (result.rawKey) onCreated(result.rawKey);
    } catch (error) {
      setErr(isApiError(error) ? error.message : "Failed to create key.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Generate API Key</DialogTitle>
          <DialogDescription>
            Create a new API key for programmatic access to the Payminto API.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <FormField label="Key Name" htmlFor="ak-name">
            <TextInput
              id="ak-name"
              required
              placeholder="e.g. Production"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </FormField>
          {err ? (
            <p role="alert" className="text-sm text-destructive">{err}</p>
          ) : null}
          <DialogFooter>
            <Button
              type="submit"
              disabled={create.isPending}
              className="bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]"
            >
              {create.isPending ? "Generating..." : "Generate"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function RevealKeyDialog({
  rawKey,
  onClose,
}: {
  rawKey: string | null;
  onClose: () => void;
}) {
  return (
    <Dialog open={Boolean(rawKey)} onOpenChange={() => onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Your API Key</DialogTitle>
          <DialogDescription>
            Copy this key now. It will not be shown again.
          </DialogDescription>
        </DialogHeader>
        {rawKey && (
          <ApiKeyReveal
            secret={rawKey}
            label="API Key"
            onConfirm={onClose}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

export function RevokeConfirmDialog({
  id,
  onClose,
}: {
  id: number | null;
  onClose: () => void;
}) {
  const revoke = useRevokeApiKey();

  async function onConfirm() {
    if (id === null) return;
    await revoke.mutateAsync(id);
    onClose();
  }

  return (
    <Dialog open={id !== null} onOpenChange={() => onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Revoke API Key</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">
          This key will be permanently deactivated. Any integrations using it
          will stop working.
        </p>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button
            variant="destructive"
            onClick={onConfirm}
            disabled={revoke.isPending}
          >
            {revoke.isPending ? "Revoking..." : "Revoke"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
