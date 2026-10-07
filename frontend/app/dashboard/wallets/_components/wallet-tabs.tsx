"use client";

import { RouteTabs } from "@/components/route-tabs";

const TABS = [
  { href: "/dashboard/wallets", label: "Deposit" },
  { href: "/dashboard/wallets/hot", label: "Hot" },
  { href: "/dashboard/wallets/cold", label: "Cold" },
];

export function WalletTabs() {
  return <RouteTabs tabs={TABS} />;
}
