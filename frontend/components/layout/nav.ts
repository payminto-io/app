import {
  AlertCircle,
  ArrowDownUp,
  ArrowUpRight,
  BarChart3,
  BookUser,
  Building2,
  CreditCard,
  Gift,
  Key,
  LayoutDashboard,
  Link2,
  Server,
  Settings,
  Shield,
  Sliders,
  Users,
  Wallet,
  Webhook,
} from "lucide-react";

export interface NavItem {
  label: string;
  href: string;
  icon: typeof LayoutDashboard;
  /** Matches by prefix unless exact. */
  exact?: boolean;
  keywords?: string[];
}

export interface NavSection {
  title?: string;
  items: NavItem[];
}

/** One nav definition for the dashboard, the admin area, the drawer and the command palette. */
export const NAV_SECTIONS: NavSection[] = [
  {
    items: [
      { label: "Home", href: "/dashboard", icon: LayoutDashboard, exact: true },
      { label: "Payments", href: "/dashboard/payments", icon: CreditCard, keywords: ["create"] },
      { label: "Links", href: "/dashboard/links", icon: Link2, keywords: ["payment links", "qr", "checkout"] },
      { label: "Customers", href: "/dashboard/customers", icon: BookUser },
    ],
  },
  {
    title: "Money",
    items: [
      { label: "Wallets", href: "/dashboard/wallets", icon: Wallet, keywords: ["hot", "cold", "addresses"] },
      { label: "Sweeps", href: "/dashboard/sweeps", icon: ArrowDownUp },
      { label: "Withdrawals", href: "/dashboard/withdrawals", icon: ArrowUpRight, keywords: ["payouts"] },
      { label: "Recipients", href: "/dashboard/recipients", icon: BookUser },
      { label: "Referrals", href: "/dashboard/referrals", icon: Gift },
      { label: "Onramp", href: "/dashboard/onramper", icon: CreditCard },
    ],
  },
  {
    title: "Developers",
    items: [
      { label: "Webhooks", href: "/dashboard/webhooks", icon: Webhook },
      { label: "API keys", href: "/dashboard/settings/api-keys", icon: Key },
    ],
  },
  {
    title: "Insights",
    items: [{ label: "Analytics", href: "/dashboard/analytics", icon: BarChart3 }],
  },
  {
    title: "Admin",
    items: [
      { label: "Projects", href: "/admin/external-platforms", icon: Building2 },
      { label: "Members", href: "/admin/members", icon: Users },
      { label: "Roles", href: "/admin/roles", icon: Shield },
      { label: "Config", href: "/admin/configurations", icon: Sliders },
      { label: "System", href: "/admin/system", icon: Server },
      { label: "Missed deposits", href: "/admin/missed-deposits", icon: AlertCircle },
    ],
  },
];

export const SETTINGS_ITEM: NavItem = { label: "Settings", href: "/dashboard/settings", icon: Settings, exact: true };

export function isNavActive(item: NavItem, pathname: string): boolean {
  if (item.exact) return pathname === item.href;
  return pathname === item.href || pathname.startsWith(item.href + "/");
}
