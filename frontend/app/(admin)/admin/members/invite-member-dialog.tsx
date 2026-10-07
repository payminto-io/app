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
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useInviteMember } from "@/lib/query/hooks/use-admin";

const MEMBER_TYPES = [
  { value: "admin", label: "Admin" },
  { value: "operator", label: "Operator" },
  { value: "viewer", label: "Viewer" },
];

export function InviteMemberDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const invite = useInviteMember();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [memberType, setMemberType] = useState("");
  const [error, setError] = useState<string | null>(null);

  function reset() {
    setName("");
    setEmail("");
    setPassword("");
    setMemberType("");
    setError(null);
  }

  function handleOpenChange(next: boolean) {
    if (!next) reset();
    onOpenChange(next);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await invite.mutateAsync({
        name,
        email,
        password,
        memberType: memberType || undefined,
      });
      reset();
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Invite failed");
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Invite member</DialogTitle>
            <DialogDescription>They can sign in as soon as you send this.</DialogDescription>
          </DialogHeader>

          <FormField label="Name" htmlFor="inv-name">
            <TextInput id="inv-name" required placeholder="Jane Doe" value={name} onChange={(e) => setName(e.target.value)} />
          </FormField>

          <FormField label="Email" htmlFor="inv-email">
            <TextInput
              id="inv-email"
              type="email"
              required
              placeholder="jane@company.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </FormField>

          <FormField label="Password" htmlFor="inv-password" hint="At least 8 characters">
            <TextInput
              id="inv-password"
              type="password"
              required
              minLength={8}
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </FormField>

          <FormField label="Role" htmlFor="inv-role">
            <Select value={memberType} onValueChange={(v) => setMemberType(v ?? "")}>
              <SelectTrigger id="inv-role" className="w-full">
                <SelectValue placeholder="Choose a role" />
              </SelectTrigger>
              <SelectContent>
                {MEMBER_TYPES.map((t) => (
                  <SelectItem key={t.value} value={t.value}>
                    {t.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </FormField>

          {error ? (
            <p role="alert" className="text-body-sm text-bad">
              {error}
            </p>
          ) : null}

          <DialogFooter>
            <Button
              variant="outline"
              type="button"
              onClick={() => handleOpenChange(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={invite.isPending}>
              {invite.isPending ? "Inviting..." : "Send invite"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
