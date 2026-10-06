"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  CreditCard,
  Wallet,
  ArrowDownUp,
  Webhook,
  BarChart3,
  Settings,
  Key,
  ArrowUpRight,
  BookUser,
  Gift,
  CreditCard as CardIcon,
  Building2,
  Users,
  Shield,
  Sliders,
  Server,
  AlertCircle,
  LogOut,
  Menu,
  Bell,
  X,
  ChevronRight,
  ChevronDown,
  CircleHelp,
} from "lucide-react";
import { useAuth } from "@/lib/auth/store";
import { useSignout } from "@/lib/query/hooks/use-auth";
import { cn } from "@/lib/utils";

// ---------------------------------------------------------------------------
// Navigation definition — flat items grouped by section
// ---------------------------------------------------------------------------

interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
}

interface NavSection {
  title?: string;
  items: NavItem[];
}

const sections: NavSection[] = [
  {
    title: "Workspace",
    items: [
      { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
      { label: "Payments", href: "/dashboard/payments", icon: CreditCard },
      { label: "Withdrawals", href: "/dashboard/withdrawals", icon: ArrowUpRight },
      { label: "Recipients", href: "/dashboard/recipients", icon: BookUser },
      { label: "Wallets", href: "/dashboard/wallets", icon: Wallet },
      { label: "Sweeps", href: "/dashboard/sweeps", icon: ArrowDownUp },
    ],
  },
  {
    title: "Developers",
    items: [
      { label: "Webhooks", href: "/dashboard/webhooks", icon: Webhook },
      { label: "API Keys", href: "/dashboard/settings/api-keys", icon: Key },
    ],
  },
  {
    title: "Insights",
    items: [
      { label: "Analytics", href: "/dashboard/analytics", icon: BarChart3 },
      { label: "Referrals", href: "/dashboard/referrals", icon: Gift },
      { label: "Onramper", href: "/dashboard/onramper", icon: CardIcon },
    ],
  },
  {
    title: "Admin",
    items: [
      { label: "Projects", href: "/admin/external-platforms", icon: Building2 },
      { label: "Members", href: "/admin/members", icon: Users },
      { label: "Roles", href: "/admin/roles", icon: Shield },
      { label: "Config", href: "/admin/configurations", icon: Sliders },
      { label: "System", href: "/admin/system", icon: Server },
      { label: "Missed Deposits", href: "/admin/missed-deposits", icon: AlertCircle },
    ],
  },
  {
    items: [
      { label: "Settings", href: "/dashboard/settings", icon: Settings },
    ],
  },
];

// ---------------------------------------------------------------------------
// Dashboard Layout
// ---------------------------------------------------------------------------

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const [mobileOpen, setMobileOpen] = React.useState(false);

  return (
    <div className="flex min-h-dvh flex-col bg-background">
      <DemoBanner />

      <div className="flex flex-1">
        {/* Desktop sidebar */}
        <aside className="hidden md:flex md:w-[264px] md:flex-col md:shrink-0">
          <Sidebar pathname={pathname} />
        </aside>

        {/* Mobile sidebar */}
        {mobileOpen && (
          <div className="fixed inset-0 z-50 md:hidden">
            <div
              className="absolute inset-0 bg-black/60 backdrop-blur-sm"
              onClick={() => setMobileOpen(false)}
            />
            <aside className="absolute left-0 top-0 flex h-full w-[280px] flex-col shadow-2xl">
              <Sidebar
                pathname={pathname}
                onNavigate={() => setMobileOpen(false)}
              />
            </aside>
          </div>
        )}

        {/* Content */}
        <main className="flex-1 min-w-0 flex flex-col">
          <TopBar onMenuClick={() => setMobileOpen(true)} />
          <div className="flex-1 overflow-auto px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
            <div className="mx-auto w-full max-w-[1540px]">{children}</div>
          </div>
        </main>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Demo banner (top)
// ---------------------------------------------------------------------------

function DemoBanner() {
  return (
    <div className="flex min-h-9 w-full items-center justify-center gap-2 bg-[#17152f] px-4 py-2 text-center text-[11px] font-semibold tracking-[0.01em] text-white/75 sm:text-xs">
      <span className="size-1.5 rounded-full bg-[var(--pm-success)] shadow-[0_0_10px_rgba(34,184,106,.85)]" />
      Test environment
      <span className="text-white/30">•</span>
      Do not send mainnet assets
    </div>
  );
}

// ---------------------------------------------------------------------------
// Top Bar (above content area) — keeps auth integration from current topbar
// ---------------------------------------------------------------------------

function TopBar({ onMenuClick }: { onMenuClick: () => void }) {
  const member = useAuth((s) => s.member);
  const signout = useSignout();
  const initial = member?.name?.charAt(0)?.toUpperCase() ?? "?";

  return (
    <header className="sticky top-0 z-30 flex h-[68px] shrink-0 items-center gap-3 border-b border-border bg-white/95 px-4 backdrop-blur-xl md:px-6">
      <button
        onClick={onMenuClick}
        className="md:hidden rounded-md p-1.5 hover:bg-muted text-foreground"
        aria-label="Open sidebar"
      >
        <Menu className="size-5" />
      </button>

      <button className="hidden items-center gap-3 rounded-xl border border-border bg-white px-3 py-2 text-left shadow-[0_1px_2px_rgba(16,24,40,.03)] transition-colors hover:bg-muted/50 sm:flex">
        <span className="grid size-7 place-items-center rounded-lg bg-[#fceeea] text-[var(--pm-primary)]">
          <Building2 className="size-3.5" />
        </span>
        <span className="min-w-0">
          <span className="block text-[9px] font-bold uppercase tracking-[.09em] text-muted-foreground">
            Workspace
          </span>
          <span className="block max-w-36 truncate text-xs font-semibold text-foreground">
            Payminto Merchant
          </span>
        </span>
        <ChevronDown className="size-3.5 text-muted-foreground" />
      </button>

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
            {member?.memberType ?? "Merchant"}
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

// ---------------------------------------------------------------------------
// Sidebar
// ---------------------------------------------------------------------------

function Sidebar({
  pathname,
  onNavigate,
}: {
  pathname: string;
  onNavigate?: () => void;
}) {
  return (
    <div className="flex h-full w-full flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground">
      <div className="px-5 pb-5 pt-6">
        <div className="flex items-center justify-between">
          <Link href="/dashboard" className="flex items-center gap-2.5">
            <span className="relative grid size-9 place-items-center rounded-xl bg-[#17152f] text-sm font-black text-white shadow-[0_8px_20px_rgba(23,21,47,.15)]">
              P
              <span className="absolute -right-0.5 -top-0.5 size-2.5 rounded-full border-2 border-white bg-[var(--pm-success)]" />
            </span>
            <span>
              <strong className="block text-[18px] font-bold leading-none tracking-[-.04em] text-[#17152f]">
                Payminto
              </strong>
              <small className="mt-1 block text-[8px] font-bold uppercase tracking-[.16em] text-muted-foreground">
                Merchant infrastructure
              </small>
            </span>
          </Link>
          <button
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground md:hidden"
            onClick={onNavigate}
            aria-label="Close sidebar"
          >
            <X className="size-4" />
          </button>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 space-y-5 overflow-y-auto px-3 pb-4 pm-scrollbar">
        {sections.map((section, i) => (
          <div key={i}>
            {section.title ? (
              <div className="px-3 mb-2 text-[9px] font-bold uppercase tracking-[.13em] text-[#98a2b3]">
                {section.title}
              </div>
            ) : null}
            <ul className="space-y-0.5">
              {section.items.map((item) => {
                const isActive =
                  pathname === item.href ||
                  (item.href !== "/dashboard" &&
                    pathname.startsWith(item.href));
                const Icon = item.icon;
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      onClick={onNavigate}
                      aria-current={isActive ? "page" : undefined}
                      className={cn(
                        "group relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-[12px] font-medium text-[#58606d] transition-colors hover:bg-[#f5f6f8] hover:text-[#17152f]",
                        isActive && "pm-sidebar-active"
                      )}
                    >
                      <Icon className="size-4 shrink-0" />
                      <span className="flex-1">{item.label}</span>
                      {isActive && (
                        <ChevronRight className="size-3.5 shrink-0 text-[var(--pm-primary)]/55" />
                      )}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>

      {/* User footer */}
      <SidebarFooter />
    </div>
  );
}

function SidebarFooter() {
  const member = useAuth((s) => s.member);
  const signout = useSignout();
  const initial = member?.name?.charAt(0)?.toUpperCase() ?? "?";

  return (
    <div className="border-t border-sidebar-border p-3">
      <Link
        href="/dashboard/settings"
        className="mb-1 flex items-center gap-3 rounded-lg px-3 py-2 text-[12px] font-medium text-[#69707d] transition-colors hover:bg-muted hover:text-foreground"
      >
        <CircleHelp className="size-4" />
        Help & support
      </Link>
      <button
        onClick={() => signout.mutate()}
        disabled={signout.isPending}
        className="flex w-full items-center gap-2.5 rounded-xl border border-transparent px-2 py-2 text-left transition-colors hover:border-border hover:bg-muted/60"
        aria-label="Log out"
      >
        <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-[#17152f] text-xs font-bold text-white">
          {initial}
        </div>
        <div className="flex-1 min-w-0">
          <div className="truncate text-[12px] font-semibold text-foreground">
            {member?.name ?? "User"}
          </div>
          <div className="truncate text-[10px] text-muted-foreground">
            {member?.email ?? ""}
          </div>
        </div>
        <LogOut className="size-4 text-muted-foreground" />
      </button>
    </div>
  );
}
