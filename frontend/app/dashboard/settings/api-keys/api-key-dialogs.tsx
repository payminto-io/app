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
      setErr(isApiError(error) ? error.message : "The key could not be created.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Generate API key</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <FormField label="Name" htmlFor="ak-name">
            <TextInput
              id="ak-name"
              required
              placeholder="Production server"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </FormField>
          {err ? (
            <p role="alert" className="text-body-sm text-bad">{err}</p>
          ) : null}
          <DialogFooter>
            <Button type="submit" disabled={create.isPending}>
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
          <DialogTitle>Your new API key</DialogTitle>
        </DialogHeader>
        {rawKey && (
          <ApiKeyReveal
            secret={rawKey}
            label="Secret key"
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
          <DialogTitle>Revoke this key?</DialogTitle>
          <DialogDescription>Requests signed with it stop working at once. This cannot be undone.</DialogDescription>
        </DialogHeader>
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
