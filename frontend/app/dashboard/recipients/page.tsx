"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import {
  useRecipientsList,
  useCreateRecipient,
  useUpdateRecipient,
  useDeleteRecipient,
} from "@/lib/query/hooks/use-recipients";
import { isApiError } from "@/lib/query/hooks/use-auth";
import type { Recipient } from "@/lib/query/hooks/use-recipients";
import { chainName, PAYOUT_CHAINS } from "@/lib/chains";
import { PageHeader } from "@/components/page-header";
import { CopyField } from "@/components/copy-field";
import { DataTable, type DataTableColumn } from "@/components/data-table";
import { RowActions } from "@/components/row-actions";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { DropdownMenuItem } from "@/components/ui/dropdown-menu";
import { FormField, TextInput } from "@/components/ui/form-field";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ErrorState } from "@/components/ui/states";

function truncateAddr(addr: string) {
  if (addr.length <= 16) return addr;
  return `${addr.slice(0, 8)}...${addr.slice(-6)}`;
}

export default function RecipientsPage() {
  const { data, error, refetch } = useRecipientsList();
  const [addOpen, setAddOpen] = useState(false);
  const [editRecipient, setEditRecipient] = useState<Recipient | null>(null);
  const [deleteId, setDeleteId] = useState<number | null>(null);

  const columns: DataTableColumn<Recipient>[] = [
    { key: "name", header: "Name", className: "font-medium", cell: (r) => r.name },
    { key: "email", header: "Email", className: "text-ink-soft", cell: (r) => r.email ?? null },
    {
      key: "asset",
      header: "Asset",
      cell: (r) => (
        <span className="inline-flex items-baseline gap-1.5">
          <span className="text-ink">{r.currencyCode}</span>
          <span className="text-label text-ink-soft">on {chainName(r.blockchainCode)}</span>
        </span>
      ),
    },
    {
      key: "address",
      header: "Address",
      cell: (r) => <CopyField value={r.address} display={truncateAddr(r.address)} boxed={false} />,
    },
    {
      key: "actions",
      header: <span className="sr-only">Actions</span>,
      align: "right",
      className: "w-0",
      cell: (r) => (
        <RowActions label={`Actions for ${r.name}`}>
          <DropdownMenuItem onClick={() => setEditRecipient(r)}>Edit</DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onClick={() => setDeleteId(r.id)}>
            Delete
          </DropdownMenuItem>
        </RowActions>
      ),
    },
  ];

  return (
    <div className="space-y-5">
      <PageHeader title="Recipients">
        <Button onClick={() => setAddOpen(true)}>
          <Plus />
          Add recipient
        </Button>
      </PageHeader>

      {error ? (
        <ErrorState message={error.message} retry={refetch} />
      ) : (
        <DataTable
          columns={columns}
          rows={data ?? []}
          loading={!data}
          getRowId={(r) => r.id}
          emptyTitle="No recipients yet."
          emptyDescription="Saved addresses fill in the payout form."
          emptyAction={
            <Button size="sm" onClick={() => setAddOpen(true)}>
              Add recipient
            </Button>
          }
        />
      )}

      <AddRecipientDialog open={addOpen} onOpenChange={setAddOpen} />
      <EditRecipientDialog recipient={editRecipient} onClose={() => setEditRecipient(null)} />
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
      setErr(isApiError(error) ? error.message : "The recipient could not be added.");
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add recipient</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <FormField label="Name" htmlFor="rc-name">
            <TextInput id="rc-name" required value={name} onChange={(e) => setName(e.target.value)} />
          </FormField>
          <FormField label="Email" htmlFor="rc-email" hint="Optional">
            <TextInput id="rc-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </FormField>
          <FormField label="Chain" htmlFor="rc-chain">
            <Select value={blockchain} onValueChange={(v) => { if (v) setBlockchain(v); }}>
              <SelectTrigger id="rc-chain" className="w-full"><SelectValue /></SelectTrigger>
              <SelectContent>
                {PAYOUT_CHAINS.map((c) => (
                  <SelectItem key={c} value={c}>{chainName(c)}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>
          <FormField label="Asset" htmlFor="rc-cur">
            <TextInput id="rc-cur" required value={currency} onChange={(e) => setCurrency(e.target.value)} className="uppercase" />
          </FormField>
          <FormField label="Address" htmlFor="rc-addr">
            <TextInput id="rc-addr" required value={address} onChange={(e) => setAddress(e.target.value)} className="font-mono" autoComplete="off" spellCheck={false} />
          </FormField>
          <FormField label="Memo" htmlFor="rc-memo" hint="Optional">
            <TextInput id="rc-memo" value={memo} onChange={(e) => setMemo(e.target.value)} />
          </FormField>
          {err ? <p role="alert" className="text-body-sm text-bad">{err}</p> : null}
          <DialogFooter>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? "Adding..." : "Add recipient"}
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
      setErr(isApiError(error) ? error.message : "The recipient could not be saved.");
    }
  }

  return (
    <Dialog open={recipient !== null} onOpenChange={() => onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Edit recipient</DialogTitle>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <FormField label="Name" htmlFor="ed-name">
            <TextInput id="ed-name" required value={name} onChange={(e) => setName(e.target.value)} />
          </FormField>
          <FormField label="Email" htmlFor="ed-email" hint="Optional">
            <TextInput id="ed-email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </FormField>
          <FormField label="Chain" htmlFor="ed-chain">
            <TextInput id="ed-chain" value={chainName(recipient?.blockchainCode)} disabled />
          </FormField>
          <FormField label="Asset" htmlFor="ed-cur">
            <TextInput id="ed-cur" value={recipient?.currencyCode ?? ""} disabled />
          </FormField>
          <FormField label="Address" htmlFor="ed-addr">
            <TextInput id="ed-addr" required value={address} onChange={(e) => setAddress(e.target.value)} className="font-mono" autoComplete="off" spellCheck={false} />
          </FormField>
          <FormField label="Memo" htmlFor="ed-memo" hint="Optional">
            <TextInput id="ed-memo" value={memo} onChange={(e) => setMemo(e.target.value)} />
          </FormField>
          {err ? <p role="alert" className="text-body-sm text-bad">{err}</p> : null}
          <DialogFooter>
            <Button variant="outline" type="button" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={update.isPending}>
              {update.isPending ? "Saving..." : "Save"}
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
          <DialogTitle>Delete this recipient?</DialogTitle>
          <DialogDescription>This cannot be undone.</DialogDescription>
        </DialogHeader>
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
