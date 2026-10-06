"use client";

import { useAuth } from "@/lib/auth/store";
import { useSignout } from "@/lib/query/hooks/use-auth";
import { LogOut, Bell } from "lucide-react";

export function Topbar() {
  const member = useAuth((s) => s.member);
  const signout = useSignout();

  const initial = member?.name?.charAt(0)?.toUpperCase() ?? "?";

  return (
    <header className="sticky top-0 z-30 flex h-[68px] shrink-0 items-center gap-3 border-b border-border bg-white/95 px-4 backdrop-blur-xl md:px-6">
      <div className="hidden sm:block">
        <p className="text-[9px] font-bold uppercase tracking-[.1em] text-muted-foreground">
          Workspace
        </p>
        <p className="mt-0.5 text-xs font-semibold text-foreground">
          Payminto Administration
        </p>
      </div>
      <div className="flex-1" />

      <div className="hidden items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1.5 text-[10px] font-bold uppercase tracking-[.07em] text-emerald-700 sm:flex">
        <span className="size-1.5 rounded-full bg-emerald-500" />
        Testnet
      </div>

      <button
        className="rounded-md p-2 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors relative"
        aria-label="Notifications"
      >
        <Bell className="size-4" />
        <span className="absolute top-1.5 right-1.5 size-1.5 rounded-full bg-[var(--pm-danger)]" />
      </button>

      <div className="h-6 w-px bg-border" />

      <div className="flex items-center gap-2">
        <div className="flex size-9 items-center justify-center rounded-full bg-[#17152f] text-xs font-semibold text-white ring-2 ring-white shadow-sm">
          {initial}
        </div>
        <div className="hidden sm:block text-sm">
          <div className="font-semibold leading-none">
            {member?.name ?? "User"}
          </div>
          <div className="mt-1 text-[10px] font-semibold uppercase tracking-[.06em] text-muted-foreground">
            {member?.memberType ?? "Admin"}
          </div>
        </div>
      </div>

      <button
        onClick={() => signout.mutate()}
        disabled={signout.isPending}
        className="rounded-md p-2 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
        aria-label="Log out"
      >
        <LogOut className="size-4" />
      </button>
    </header>
  );
}
