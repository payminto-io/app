"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useSignup, isApiError } from "@/lib/query/hooks/use-auth";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { AuthShell } from "@/components/auth-shell";

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
      title={
        <>
          Create your <span className="italic text-primary">Payminto</span>{" "}
          account
        </>
      }
      description="Get started with your self-hosted payment gateway in minutes"
      footer={
        <div className="text-center">
          Already have an account?{" "}
          <Link
            href="/signin"
            className="text-primary underline-offset-4 hover:underline font-medium"
          >
            Sign in
          </Link>
        </div>
      }
    >
      <form onSubmit={onSubmit} className="space-y-5" aria-busy={signupMut.isPending}>
        <div className="space-y-2">
          <Label htmlFor="signup-name" className="text-[14px] font-medium">
            Full name
          </Label>
          <Input
            id="signup-name"
            required
            autoComplete="name"
            autoFocus
            placeholder="Jane Doe"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="h-11 text-[16px] rounded-lg"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="signup-email" className="text-[14px] font-medium">
            Email address
          </Label>
          <Input
            id="signup-email"
            type="email"
            required
            autoComplete="email"
            placeholder="you@company.com"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="h-11 text-[16px] rounded-lg"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="signup-password" className="text-[14px] font-medium">
            Password
          </Label>
          <Input
            id="signup-password"
            type="password"
            required
            minLength={8}
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="h-11 text-[16px] rounded-lg"
          />
        </div>

        {error ? (
          <p className="text-destructive text-sm" role="alert">
            {error}
          </p>
        ) : null}

        <Button
          type="submit"
          disabled={signupMut.isPending}
          className="h-11 w-full rounded-lg bg-primary text-white hover:bg-[var(--pm-primary-deep)] font-medium"
        >
          {signupMut.isPending ? (
            <span className="flex items-center gap-2">
              <span className="size-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
              Creating account...
            </span>
          ) : (
            "Create account"
          )}
        </Button>
      </form>
    </AuthShell>
  );
}
