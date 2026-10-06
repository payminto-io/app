"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  CreditCard,
  Wallet,
  ArrowDownUp,
  Settings,
  ArrowUpRight,
  BookUser,
  CreditCard as CardIcon,
  Building2,
  Users,
  Shield,
  Sliders,
  Server,
  AlertCircle,
  LogOut,
  ChevronRight,
  Code,
} from "lucide-react";
import { useAuth } from "@/lib/auth/store";
import { useSignout } from "@/lib/query/hooks/use-auth";
import { cn } from "@/lib/utils";

/* ── Types ─────────────────────────────────────────────── */

interface NavChild {
  label: string;
  href: string;
}

interface NavItem {
  label: string;
  href: string;
  icon: typeof LayoutDashboard;
  children?: NavChild[];
}

interface NavSection {
  title?: string;
  items: NavItem[];
}

/* ── Navigation structure (matches PayRam sidebar) ────── */

const sections: NavSection[] = [
  {
    title: "Workspace",
    items: [
      { label: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
      {
        label: "Payments",
        href: "/dashboard/payments",
        icon: CreditCard,
        children: [
          { label: "All Payments", href: "/dashboard/payments" },
          { label: "Create Payment Link", href: "/dashboard/payments/create" },
        ],
      },
      { label: "Customers", href: "/dashboard/customers", icon: BookUser },
      { label: "Onramp", href: "/dashboard/onramper", icon: CardIcon },
    ],
  },
  {
    title: "Assets",
    items: [
      { label: "Sweeps", href: "/dashboard/sweeps", icon: ArrowDownUp },
      {
        label: "Wallet Management",
        href: "/dashboard/wallets",
        icon: Wallet,
        children: [
          { label: "Overview", href: "/dashboard/wallets" },
          { label: "Cold Wallets", href: "/dashboard/wallets/cold" },
          { label: "Hot Wallets", href: "/dashboard/wallets/hot" },
          { label: "Recipients", href: "/dashboard/recipients" },
        ],
      },
      {
        label: "Withdraw",
        href: "/dashboard/withdrawals",
        icon: ArrowUpRight,
        children: [
          { label: "Payouts", href: "/dashboard/withdrawals" },
          { label: "Referral Payouts", href: "/dashboard/referrals" },
        ],
      },
    ],
  },
  {
    title: "General",
    items: [
      {
        label: "Developers",
        href: "/dashboard/webhooks",
        icon: Code,
        children: [
          { label: "Webhooks", href: "/dashboard/webhooks" },
          { label: "API Keys", href: "/dashboard/settings/api-keys" },
        ],
      },
      {
        label: "Settings",
        href: "/dashboard/settings",
        icon: Settings,
        children: [
          { label: "Account", href: "/dashboard/settings" },
          { label: "Analytics", href: "/dashboard/analytics" },
        ],
      },
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
];

/* ── Sidebar component ────────────────────────────────── */

export function Sidebar() {
  const pathname = usePathname();
  const member = useAuth((s) => s.member);
  const signout = useSignout();
  const initial = member?.name?.charAt(0)?.toUpperCase() ?? "?";

  return (
    <aside className="hidden md:flex md:w-[264px] md:flex-col">
      <div className="flex h-full w-full flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground">
        {/* Logo + testnet */}
        <div className="px-5 pb-5 pt-6">
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
        </div>

        {/* Nav */}
        <nav className="flex-1 space-y-5 overflow-y-auto px-3 pb-4 pm-scrollbar">
          {sections.map((section, i) => (
            <div key={i}>
              {section.title ? (
                <div className="mb-2 px-3 text-[9px] font-bold uppercase tracking-[.13em] text-[#98a2b3]">
                  {section.title}
                </div>
              ) : null}
              <div className="space-y-0.5">
                {section.items.map((item) => (
                  <SidebarItem
                    key={item.href}
                    item={item}
                    pathname={pathname}
                  />
                ))}
              </div>
            </div>
          ))}
        </nav>

        {/* User footer */}
        <div className="border-t border-sidebar-border p-3">
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
      </div>
    </aside>
  );
}

/* ── Individual nav item (supports collapsible children) ── */

function SidebarItem({
  item,
  pathname,
}: {
  item: NavItem;
  pathname: string;
}) {
  const hasChildren = item.children && item.children.length > 0;

  // Is any child active?
  const isChildActive = hasChildren
    ? item.children!.some(
        (c) =>
          pathname === c.href ||
          (c.href !== "/dashboard" && pathname.startsWith(c.href + "/"))
      )
    : false;

  // Is this item itself active?
  const isSelfActive =
    pathname === item.href ||
    (item.href !== "/dashboard" && pathname.startsWith(item.href + "/"));

  const isActive = isSelfActive || isChildActive;

  // Auto-expand if a child is active, otherwise collapsed by default
  const [expanded, setExpanded] = useState(isActive);

  if (!hasChildren) {
    // Simple link — no children
    return (
      <Link
        href={item.href}
        aria-current={isActive ? "page" : undefined}
        className={cn(
          "group relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-[12px] font-medium text-[#58606d] transition-colors hover:bg-[#f5f6f8] hover:text-[#17152f]",
          isActive && "pm-sidebar-active"
        )}
      >
        <item.icon className="size-4 shrink-0" />
        <span className="flex-1">{item.label}</span>
      </Link>
    );
  }

  // Collapsible parent with children
  return (
    <div>
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className={cn(
          "group relative flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-[12px] font-medium text-[#58606d] transition-colors hover:bg-[#f5f6f8] hover:text-[#17152f]",
          isActive && "pm-sidebar-active"
        )}
      >
        <item.icon className="size-4 shrink-0" />
        <span className="flex-1 text-left">{item.label}</span>
        <ChevronRight
          className={cn(
            "size-3.5 shrink-0 text-muted-foreground transition-transform duration-200",
            expanded && "rotate-90"
          )}
        />
      </button>

      {/* Children */}
      {expanded ? (
        <div className="ml-7 mt-0.5 space-y-0.5 border-l border-border pl-3">
          {item.children!.map((child) => {
            const isChildCurrent =
              pathname === child.href ||
              (child.href !== "/dashboard" &&
                pathname.startsWith(child.href + "/"));
            return (
              <Link
                key={child.href}
                href={child.href}
                aria-current={isChildCurrent ? "page" : undefined}
                className={cn(
                  "block rounded-md px-3 py-1.5 text-[11px] font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
                  isChildCurrent &&
                    "bg-[#fceeea] text-[#b91c1c]"
                )}
              >
                {child.label}
              </Link>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
