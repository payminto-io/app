"use client";

import { useState } from "react";
import { useAuth } from "@/lib/auth/store";
import { isApiError, useChangePassword, useUpdateProfile } from "@/lib/query/hooks/use-auth";
import { PageHeader } from "@/components/page-header";
import { DetailItem, DetailList } from "@/components/detail-list";
import { SettingsSection } from "@/components/settings-section";
import { Button } from "@/components/ui/button";
import { FormField, TextInput } from "@/components/ui/form-field";
import { SettingsTabs } from "./_components/settings-tabs";

export default function SettingsPage() {
  const member = useAuth((s) => s.member);

  return (
    <div className="space-y-5">
      <PageHeader title="Settings" />
      <SettingsTabs />
      <div className="space-y-8 pt-3">
        <SettingsSection title="Profile">
          <ProfileForm member={member} />
        </SettingsSection>
        <SettingsSection title="Password">
          <ChangePasswordForm />
        </SettingsSection>
      </div>
    </div>
  );
}

function ProfileForm({
  member,
}: {
  member: { id: number; email: string; name: string; memberType?: string } | null;
}) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(member?.name ?? "");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const updateProfile = useUpdateProfile();

  // Keep the field in step with the session until the user starts editing.
  if (!editing && member?.name && name !== member.name) {
    setName(member.name);
  }

  function startEdit() {
    setName(member?.name ?? "");
    setError(null);
    setSuccess(false);
    setEditing(true);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(false);
    try {
      await updateProfile.mutateAsync({ name });
      setSuccess(true);
      setEditing(false);
    } catch (err) {
      setError(isApiError(err) ? err.message : "The profile could not be saved.");
    }
  }

  if (editing) {
    return (
      <form onSubmit={onSubmit} className="space-y-4">
        <FormField label="Name" htmlFor="prof-name">
          <TextInput id="prof-name" required value={name} onChange={(e) => setName(e.target.value)} />
        </FormField>
        {error ? (
          <p role="alert" className="text-body-sm text-bad">
            {error}
          </p>
        ) : null}
        <div className="flex justify-end gap-2">
          <Button variant="outline" type="button" onClick={() => setEditing(false)}>
            Cancel
          </Button>
          <Button type="submit" disabled={updateProfile.isPending}>
            {updateProfile.isPending ? "Saving..." : "Save"}
          </Button>
        </div>
      </form>
    );
  }

  return (
    <div className="space-y-4">
      <DetailList columns={2}>
        <DetailItem label="Name">{member?.name}</DetailItem>
        <DetailItem label="Email">{member?.email}</DetailItem>
        <DetailItem label="Role">
          {member?.memberType ? <span className="capitalize">{member.memberType}</span> : null}
        </DetailItem>
      </DetailList>
      <div className="flex items-center justify-between gap-3">
        <p role="status" className="text-body-sm text-ok">
          {success ? "Saved." : ""}
        </p>
        <Button variant="outline" size="sm" onClick={startEdit}>
          Edit
        </Button>
      </div>
    </div>
  );
}

function ChangePasswordForm() {
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const changePw = useChangePassword();

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setSuccess(false);
    try {
      await changePw.mutateAsync({ currentPassword, newPassword });
      setSuccess(true);
      setCurrentPassword("");
      setNewPassword("");
    } catch (err) {
      setError(isApiError(err) ? err.message : "The password could not be changed.");
    }
  }

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <FormField label="Current password" htmlFor="cur-pw">
        <TextInput
          id="cur-pw"
          type="password"
          autoComplete="current-password"
          required
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
        />
      </FormField>
      <FormField label="New password" htmlFor="new-pw" hint="At least 8 characters">
        <TextInput
          id="new-pw"
          type="password"
          autoComplete="new-password"
          required
          minLength={8}
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
        />
      </FormField>
      {error ? (
        <p role="alert" className="text-body-sm text-bad">
          {error}
        </p>
      ) : null}
      <div className="flex items-center justify-between gap-3">
        <p role="status" className="text-body-sm text-ok">
          {success ? "Password changed." : ""}
        </p>
        <Button type="submit" disabled={changePw.isPending}>
          {changePw.isPending ? "Changing..." : "Change password"}
        </Button>
      </div>
    </form>
  );
}
