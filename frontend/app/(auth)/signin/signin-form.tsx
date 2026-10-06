"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { ArrowRight, Eye, EyeOff, LockKeyhole } from "lucide-react";
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
      title="Welcome back."
      description="Sign in to manage payments, wallets, and settlement activity."
      footer={
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Link
            href="#"
            className="font-bold text-[#c91e1e] underline-offset-4 hover:underline"
          >
            Forgot password?
          </Link>
          <span>
            New to Payminto?{" "}
            <Link
              href="/signup"
              className="font-bold text-[#30384a] underline-offset-4 hover:text-[#c91e1e] hover:underline"
            >
              Create account
            </Link>
          </span>
        </div>
      }
    >
      <form onSubmit={onSubmit} className="space-y-4" aria-busy={signin.isPending}>
        <div className="space-y-2">
          <Label htmlFor="email" className="text-[12px] font-bold text-[#30384a]">
            Email address
          </Label>
          <Input
            id="email"
            type="email"
            placeholder="you@company.com"
            autoComplete="email"
            autoFocus
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="h-[52px] rounded-xl border-[#dfe2ea] bg-white px-4 text-[15px] text-[#111a2e] shadow-[0_1px_2px_rgba(20,25,50,.03)] placeholder:text-[#a1a8b8] focus-visible:border-[#e22323] focus-visible:ring-[#e22323]/15"
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="password" className="text-[12px] font-bold text-[#30384a]">
            Password
          </Label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              placeholder="Enter your password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="h-[52px] rounded-xl border-[#dfe2ea] bg-white px-4 pr-12 text-[15px] text-[#111a2e] shadow-[0_1px_2px_rgba(20,25,50,.03)] placeholder:text-[#a1a8b8] focus-visible:border-[#e22323] focus-visible:ring-[#e22323]/15"
            />
            <button
              type="button"
              onClick={() => setShowPassword((value) => !value)}
              className="absolute inset-y-0 right-0 grid w-12 place-items-center text-[#939bad] transition-colors hover:text-[#c91e1e] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#e22323]/30"
              aria-label={showPassword ? "Hide password" : "Show password"}
            >
              {showPassword ? <EyeOff className="size-[18px]" /> : <Eye className="size-[18px]" />}
            </button>
          </div>
        </div>

        {error ? (
          <p
            className="rounded-xl border border-red-200 bg-red-50 px-3.5 py-3 text-[12px] font-semibold text-red-700"
            role="alert"
          >
            {error}
          </p>
        ) : null}

        <Button
          type="submit"
          disabled={signin.isPending}
          className="group h-[52px] w-full rounded-xl bg-[#e22323] text-[13px] font-bold text-white shadow-[0_12px_28px_rgba(226,35,35,.2)] transition-all hover:-translate-y-px hover:bg-[#c91e1e] hover:shadow-[0_16px_34px_rgba(226,35,35,.26)]"
        >
          {signin.isPending ? (
            <span className="flex items-center gap-2">
              <span className="size-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
              Signing in...
            </span>
          ) : (
            <span className="flex items-center gap-2">
              Continue to dashboard
              <ArrowRight className="size-4 transition-transform group-hover:translate-x-0.5" />
            </span>
          )}
        </Button>

        <p className="flex items-center justify-center gap-2 pt-1 text-[10px] font-semibold text-[#9aa1b1]">
          <LockKeyhole className="size-3.5 text-[#e22323]" />
          Your session is encrypted and securely managed
        </p>
      </form>
    </AuthShell>
  );
}
