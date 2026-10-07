"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useSignup, isApiError } from "@/lib/query/hooks/use-auth";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { AuthShell } from "@/components/auth-shell";

const MIN_PASSWORD = 8;

export function SignupForm() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const signupMut = useSignup();

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await signupMut.mutateAsync({ name, email, password });
      router.replace("/dashboard");
      router.refresh();
    } catch (err) {
      setError(
        isApiError(err) ? err.message : "Sign up failed. Please try again."
      );
    }
  }

  return (
    <AuthShell
      title="Create your account"
      footer={
        <>
          Already have an account?{" "}
          <Link href="/signin" className="tap font-medium text-tide underline-offset-4 hover:text-tide-strong hover:underline">
            Sign in
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" aria-busy={signupMut.isPending}>
        <div className="space-y-1.5">
          <Label htmlFor="signup-name">Full name</Label>
          <Input
            id="signup-name"
            required
            autoComplete="name"
            autoFocus
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="h-11 text-[16px] md:h-10 md:text-body"
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="signup-email">Email</Label>
          <Input
            id="signup-email"
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="h-11 text-[16px] md:h-10 md:text-body"
          />
        </div>

        <div className="space-y-1.5">
          <div className="flex items-baseline justify-between">
            <Label htmlFor="signup-password">Password</Label>
            <span id="signup-password-hint" className="num text-caption text-ink-faint">
              {MIN_PASSWORD}+ characters
            </span>
          </div>
          <Input
            id="signup-password"
            type="password"
            required
            minLength={MIN_PASSWORD}
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            aria-describedby="signup-password-hint"
            className="h-11 text-[16px] md:h-10 md:text-body"
          />
        </div>

        {error ? (
          <p className="rounded-sm border border-bad/30 bg-bad-tint px-3 py-2 text-body-sm text-bad" role="alert">
            {error}
          </p>
        ) : null}

        <Button type="submit" size="lg" disabled={signupMut.isPending} className="w-full">
          {signupMut.isPending ? "Creating account" : "Create account"}
        </Button>
      </form>
    </AuthShell>
  );
}
