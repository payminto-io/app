"use client";

import { useState } from "react";
import { User, Lock, Shield } from "lucide-react";
import { useAuth } from "@/lib/auth/store";
import { useChangePassword, useUpdateProfile } from "@/lib/query/hooks/use-auth";
import { isApiError } from "@/lib/query/hooks/use-auth";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { FormField, TextInput } from "@/components/ui/form-field";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export default function SettingsPage() {
  const member = useAuth((s) => s.member);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-xl font-bold tracking-tight">Settings</h1>
        <p className="text-[13px] text-muted-foreground mt-0.5">
          Manage your account and security preferences
        </p>
      </div>

      <Tabs defaultValue="profile">
        <TabsList>
          <TabsTrigger value="profile" className="gap-1.5">
            <User className="size-3.5" />
            Profile
          </TabsTrigger>
          <TabsTrigger value="security" className="gap-1.5">
            <Lock className="size-3.5" />
            Security
          </TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="mt-4">
          <ProfileCard member={member} />
        </TabsContent>

        <TabsContent value="security" className="mt-4 space-y-4">
          <ChangePasswordForm />

          <Card className="border-border shadow-sm">
            <CardHeader className="pb-3">
              <CardTitle className="text-[15px] font-semibold flex items-center gap-2">
                <Shield className="size-4 text-muted-foreground" />
                Two-Factor Authentication
              </CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-[13px] text-muted-foreground">
                Two-factor authentication adds an extra layer of security to your
                account. Configuration will be available in a future update.
              </p>
              <Button variant="outline" size="sm" className="mt-3" disabled>
                Enable 2FA
              </Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function ProfileCard({
  member,
}: {
  member: { id: number; email: string; name: string; memberType?: string } | null;
}) {
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(member?.name ?? "");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);
  const updateProfile = useUpdateProfile();

  // Sync name when member changes externally
  const displayName = member?.name ?? "\u2014";
  if (!editing && name !== displayName && displayName !== "\u2014") {
    setName(displayName);
  }

  function startEdit() {
    setName(member?.name ?? "");
    setError(null);
    setSuccess(false);
    setEditing(true);
  }

  function cancelEdit() {
    setEditing(false);
    setError(null);
    setSuccess(false);
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
      setError(isApiError(err) ? err.message : "Failed to update profile.");
    }
  }

  return (
    <Card className="border-border shadow-sm">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <CardTitle className="text-[15px] font-semibold flex items-center gap-2">
            <User className="size-4 text-muted-foreground" />
            Profile Information
          </CardTitle>
          {!editing && (
            <Button variant="outline" size="sm" className="h-7 text-[11px]" onClick={startEdit}>
              Edit Profile
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {editing ? (
          <form onSubmit={onSubmit} className="max-w-sm space-y-4">
            <FormField label="Name" htmlFor="prof-name">
              <TextInput
                id="prof-name"
                required
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </FormField>
            <div className="space-y-1.5">
              <span className="pm-label">Email</span>
              <p className="text-[14px] font-medium">{member?.email ?? "\u2014"}</p>
            </div>
            <div className="space-y-1.5">
              <span className="pm-label">Role</span>
              <p className="text-[14px] font-medium capitalize">{member?.memberType ?? "\u2014"}</p>
            </div>
            {error ? <p role="alert" className="text-sm text-destructive">{error}</p> : null}
            <div className="flex gap-2">
              <Button variant="outline" type="button" onClick={cancelEdit}>
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={updateProfile.isPending}
                className="bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]"
              >
                {updateProfile.isPending ? "Saving..." : "Save"}
              </Button>
            </div>
          </form>
        ) : (
          <div className="grid gap-5 sm:grid-cols-2 max-w-lg">
            <div className="space-y-1.5">
              <span className="pm-label">Name</span>
              <p className="text-[14px] font-medium">{member?.name ?? "\u2014"}</p>
            </div>
            <div className="space-y-1.5">
              <span className="pm-label">Email</span>
              <p className="text-[14px] font-medium">{member?.email ?? "\u2014"}</p>
            </div>
            <div className="space-y-1.5">
              <span className="pm-label">Role</span>
              <p className="text-[14px] font-medium capitalize">{member?.memberType ?? "\u2014"}</p>
            </div>
            {success ? <p className="text-sm text-emerald-400 sm:col-span-2">Profile updated successfully.</p> : null}
          </div>
        )}
      </CardContent>
    </Card>
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
      setError(isApiError(err) ? err.message : "Failed to change password.");
    }
  }

  return (
    <Card className="border-border shadow-sm">
      <CardHeader className="pb-3">
        <CardTitle className="text-[15px] font-semibold flex items-center gap-2">
          <Lock className="size-4 text-muted-foreground" />
          Change Password
        </CardTitle>
      </CardHeader>
      <CardContent>
        <form onSubmit={onSubmit} className="max-w-sm space-y-4">
          <FormField label="Current Password" htmlFor="cur-pw">
            <TextInput
              id="cur-pw"
              type="password"
              autoComplete="current-password"
              required
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
            />
          </FormField>
          <FormField label="New Password" htmlFor="new-pw">
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
            <p role="alert" className="text-sm text-destructive">{error}</p>
          ) : null}
          {success ? (
            <p className="text-sm text-emerald-400">Password changed successfully.</p>
          ) : null}
          <Button
            type="submit"
            disabled={changePw.isPending}
            className="bg-[var(--pm-primary)] text-white hover:bg-[var(--pm-primary-deep)]"
          >
            {changePw.isPending ? "Changing..." : "Change Password"}
          </Button>
        </form>
      </CardContent>
    </Card>
  );
}
