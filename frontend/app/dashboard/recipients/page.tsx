"use client";

import { useState } from "react";
import {
  useRecipientsList,
  useCreateRecipient,
  useUpdateRecipient,
  useDeleteRecipient,
} from "@/lib/query/hooks/use-recipients";
import { ErrorState } from "@/components/ui/states";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
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
import type { Recipient } from "@/lib/query/hooks/use-recipients";
import type { BlockchainNetwork } from "@/lib/types";

const CHAIN_MAP: Record<string, BlockchainNetwork> = {
  ETH: "ethereum", BTC: "bitcoin", BASE: "base", POLYGON: "polygon", TRON: "tron",
};

function truncateAddr(addr: string) {
  if (addr.length <= 16) return addr;
  return `${addr.slice(0, 8)}...${addr.slice(-6)}`;
}

export default function RecipientsPage() {
  const { data, isLoading, error, refetch } = useRecipientsList();
  const [addOpen, setAddOpen] = useState(false);
  const [editRecipient, setEditRecipient] = useState<Recipient | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Address Book</h1>
          <p className="text-[13px] text-muted-foreground mt-0.5">
            Manage your saved payout recipients
          </p>
        </div>
        <Button
          onClick={() => setAddOpen(true)}
          className="h-9 rounded-lg bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]"
        >
          Add Recipient
        </Button>
      </div>

      {error ? <ErrorState message={error.message} retry={refetch} /> : null}

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full rounded-lg" />
          ))}
        </div>
      ) : data && data.length > 0 ? (
        <Card className="border-border shadow-sm overflow-hidden">
          <CardContent className="p-0">
            <Table>
              <TableHeader>
                <TableRow className="border-border hover:bg-transparent">
                  <TableHead className="pm-label pl-6">Name</TableHead>
                  <TableHead className="pm-label">Email</TableHead>
                  <TableHead className="pm-label">Chain</TableHead>
                  <TableHead className="pm-label">Currency</TableHead>
                  <TableHead className="pm-label">Address</TableHead>
                  <TableHead className="pm-label text-right pr-6">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.map((r) => (
                  <TableRow key={r.id} className="border-border/60 hover:bg-muted/30">
                    <TableCell className="pl-6 font-medium text-[13px]">
                      {r.name}
                    </TableCell>
                    <TableCell className="text-[13px] text-muted-foreground">
                      {r.email ?? "\u2014"}
                    </TableCell>
                    <TableCell>
                      <BlockchainIcon
                        blockchain={CHAIN_MAP[r.blockchainCode?.toUpperCase()] ?? "ethereum"}
                        size="sm"
                        showLabel
                      />
                    </TableCell>
                    <TableCell className="text-[13px]">{r.currencyCode}</TableCell>
                    <TableCell className="font-mono text-[12px] text-muted-foreground">
                      {truncateAddr(r.address)}
                    </TableCell>
                    <TableCell className="text-right pr-6 space-x-2">
                      <Button
                        variant="outline"
                        size="sm"
                        className="h-7 text-[11px]"
                        onClick={() => setEditRecipient(r)}
                      >
                        Edit
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        className="h-7 text-[11px]"
                        onClick={() => setDeleteId(r.id)}
                      >
                        Delete
                      </Button>
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
            <div className="text-center">
              <p className="text-sm text-muted-foreground">No recipients yet</p>
              <p className="text-xs text-muted-foreground/60 mt-1">
                Add your first payout recipient.
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      <AddRecipientDialog open={addOpen} onOpenChange={setAddOpen} />
      <EditRecipientDialog
        recipient={editRecipient}
        onClose={() => setEditRecipient(null)}
      />
      <DeleteConfirmDialog id={deleteId} onClose={() => setDeleteId(null)} />
    </div>
  );
}

function AddRecipientDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const create = useCreateRecipient();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [blockchain, setBlockchain] = useState("ETH");
  const [currency, setCurrency] = useState("USDT");
  const [address, setAddress] = useState("");
  const [memo, setMemo] = useState("");
  const [err, setErr] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    try {
      await create.mutateAsync({
        name,
        email: email || undefined,
        blockchainCode: blockchain,
        currencyCode: currency,
        address,
        memo: memo || undefined,
      });
      onOpenChange(false);
    } catch (error) {
      setErr(isApiError(error) ? error.message : "Failed to add recipient.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add Recipient</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <FormField label="Name" htmlFor="rc-name">
            <TextInput id="rc-name" required value={name} onChange={(e) => setName(e.target.value)} />
          </FormField>
          <FormField label="Email" htmlFor="rc-email" hint="Optional">
            <TextInput id="rc-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </FormField>
          <FormField label="Blockchain" htmlFor="rc-chain">
            <Select value={blockchain} onValueChange={(v) => { if (v) setBlockchain(v); }}>
              <SelectTrigger id="rc-chain"><SelectValue /></SelectTrigger>
              <SelectContent>
                {["ETH", "BTC", "BASE", "POLYGON", "TRON"].map((c) => (
                  <SelectItem key={c} value={c}>{c}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="Currency" htmlFor="rc-cur">
            <TextInput id="rc-cur" required value={currency} onChange={(e) => setCurrency(e.target.value)} />
          </FormField>
          <FormField label="Address" htmlFor="rc-addr">
            <TextInput id="rc-addr" required value={address} onChange={(e) => setAddress(e.target.value)} />
          </FormField>
          <FormField label="Memo" htmlFor="rc-memo" hint="Optional">
            <TextInput id="rc-memo" value={memo} onChange={(e) => setMemo(e.target.value)} />
          </FormField>
          {err ? <p role="alert" className="text-sm text-destructive">{err}</p> : null}
          <DialogFooter>
            <Button type="submit" disabled={create.isPending} className="bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]">
              {create.isPending ? "Adding..." : "Add Recipient"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function EditRecipientDialog({
  recipient,
  onClose,
}: {
  recipient: Recipient | null;
  onClose: () => void;
}) {
  const update = useUpdateRecipient();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [address, setAddress] = useState("");
  const [memo, setMemo] = useState("");
  const [err, setErr] = useState<string | null>(null);

  // Sync form state when a different recipient is selected
  const [prevId, setPrevId] = useState<number | null>(null);
  if (recipient && recipient.id !== prevId) {
    setPrevId(recipient.id);
    setName(recipient.name);
    setEmail(recipient.email ?? "");
    setAddress(recipient.address);
    setMemo(recipient.memo ?? "");
    setErr(null);
  }
  if (!recipient && prevId !== null) {
    setPrevId(null);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!recipient) return;
    setErr(null);
    try {
      await update.mutateAsync({
        id: recipient.id,
        input: {
          name: name || undefined,
          email: email || undefined,
          address: address || undefined,
          memo: memo || undefined,
        },
      });
      onClose();
    } catch (error) {
      setErr(isApiError(error) ? error.message : "Failed to update recipient.");
    }
  }

  return (
    <Dialog open={recipient !== null} onOpenChange={() => onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit Recipient</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <FormField label="Name" htmlFor="ed-name">
            <TextInput id="ed-name" required value={name} onChange={(e) => setName(e.target.value)} />
          </FormField>
          <FormField label="Email" htmlFor="ed-email" hint="Optional">
            <TextInput id="ed-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </FormField>
          <FormField label="Blockchain" htmlFor="ed-chain">
            <TextInput id="ed-chain" value={recipient?.blockchainCode ?? ""} disabled className="opacity-60" />
          </FormField>
          <FormField label="Currency" htmlFor="ed-cur">
            <TextInput id="ed-cur" value={recipient?.currencyCode ?? ""} disabled className="opacity-60" />
          </FormField>
          <FormField label="Address" htmlFor="ed-addr">
            <TextInput id="ed-addr" required value={address} onChange={(e) => setAddress(e.target.value)} />
          </FormField>
          <FormField label="Memo" htmlFor="ed-memo" hint="Optional">
            <TextInput id="ed-memo" value={memo} onChange={(e) => setMemo(e.target.value)} />
          </FormField>
          {err ? <p role="alert" className="text-sm text-destructive">{err}</p> : null}
          <DialogFooter>
            <Button variant="outline" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={update.isPending} className="bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]">
              {update.isPending ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function DeleteConfirmDialog({
  id,
  onClose,
}: {
  id: number | null;
  onClose: () => void;
}) {
  const remove = useDeleteRecipient();

  async function onConfirm() {
    if (id === null) return;
    await remove.mutateAsync(id);
    onClose();
  }

  return (
    <Dialog open={id !== null} onOpenChange={() => onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete Recipient</DialogTitle>
        </DialogHeader>
        <p className="text-sm text-muted-foreground">
          Are you sure you want to delete this recipient? This cannot be undone.
        </p>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Cancel</Button>
          <Button variant="destructive" onClick={onConfirm} disabled={remove.isPending}>
            {remove.isPending ? "Deleting..." : "Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
