"use client";

import { useTheme } from "next-themes";
import { Check, LogOut, Monitor, Moon, Settings, Sun } from "lucide-react";
import Link from "next/link";
import { useAuth } from "@/lib/auth/store";
import { useSignout } from "@/lib/query/hooks/use-auth";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

const THEMES = [
  { value: "light", label: "Light", icon: Sun },
  { value: "dark", label: "Dark", icon: Moon },
  { value: "system", label: "System", icon: Monitor },
] as const;

export function AccountMenu() {
  const member = useAuth((s) => s.member);
  const signout = useSignout();
  const { theme, setTheme } = useTheme();
  const initial = member?.name?.charAt(0)?.toUpperCase() ?? "?";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label="Account menu"
        className="tap inline-flex size-8 items-center justify-center rounded-full border border-line-strong bg-surface-sunken text-label font-semibold text-ink transition-colors duration-120 hover:border-ink-faint aria-expanded:border-ink"
      >
        {initial}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-60">
        <DropdownMenuLabel className="px-2 py-1.5">
          <div className="truncate text-body font-medium text-ink">{member?.name ?? "Signed in"}</div>
          {member?.email ? <div className="truncate text-caption text-ink-soft">{member.email}</div> : null}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem render={<Link href="/dashboard/settings" />}>
            <Settings />
            Settings
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuLabel className="px-2 py-1 text-caption text-ink-faint">Appearance</DropdownMenuLabel>
        <DropdownMenuGroup>
          {THEMES.map((t) => (
            <DropdownMenuItem key={t.value} onClick={() => setTheme(t.value)}>
              <t.icon />
              {t.label}
              <Check className={cn("ml-auto size-3.5", theme === t.value ? "opacity-100" : "opacity-0")} />
            </DropdownMenuItem>
          ))}
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => signout.mutate()} disabled={signout.isPending}>
          <LogOut />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
