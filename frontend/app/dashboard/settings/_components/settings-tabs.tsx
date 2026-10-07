"use client";

import { RouteTabs } from "@/components/route-tabs";

const TABS = [
  { href: "/dashboard/settings", label: "Account" },
  { href: "/dashboard/settings/api-keys", label: "API keys" },
  { href: "/dashboard/settings/attestations", label: "Attestations" },
];

export function SettingsTabs() {
  return <RouteTabs tabs={TABS} />;
}
