"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { Eye, EyeOff } from "lucide-react";
import { useSignin, isApiError } from "@/lib/query/hooks/use-auth";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { AuthShell } from "@/components/auth-shell";

export function SigninForm() {
  const router = useRouter();
  const params = useSearchParams();
  const raw = params.get("redirect") ?? "/dashboard";
  const redirectTo =
    raw.startsWith("/") && !raw.startsWith("//") ? raw : "/dashboard";

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const signin = useSignin();

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await signin.mutateAsync({ email, password });
      router.replace(redirectTo);
      router.refresh();
    } catch (err) {
      setError(
        isApiError(err) ? err.message : "Sign in failed. Please try again."
      );
    }
  }

  return (
    <AuthShell
      title="Sign in"
      footer={
        <>
          New here?{" "}
          <Link href="/signup" className="tap font-medium text-tide underline-offset-4 hover:text-tide-strong hover:underline">
            Create an account
          </Link>
        </>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" aria-busy={signin.isPending}>
        <div className="space-y-1.5">
          <Label htmlFor="email">Email</Label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            autoFocus
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="h-11 text-[16px] md:h-10 md:text-body"
          />
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="password">Password</Label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="h-11 pr-11 text-[16px] md:h-10 md:text-body"
            />
            <button
              type="button"
              onClick={() => setShowPassword((value) => !value)}
              className="absolute inset-y-0 right-0 inline-flex w-11 items-center justify-center rounded-r-sm text-ink-faint transition-colors duration-120 hover:text-ink"
              aria-label={showPassword ? "Hide password" : "Show password"}
              aria-pressed={showPassword}
            >
              {showPassword ? <EyeOff className="size-4" strokeWidth={1.75} /> : <Eye className="size-4" strokeWidth={1.75} />}
            </button>
          </div>
        </div>

        {error ? (
          <p className="rounded-sm border border-bad/30 bg-bad-tint px-3 py-2 text-body-sm text-bad" role="alert">
            {error}
          </p>
        ) : null}

        <Button type="submit" size="lg" disabled={signin.isPending} className="w-full">
          {signin.isPending ? "Signing in" : "Sign in"}
        </Button>
      </form>
    </AuthShell>
  );
}
