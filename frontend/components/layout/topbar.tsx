"use client";

import { Menu } from "lucide-react";
import { AccountMenu } from "./account-menu";
import { CommandPaletteTrigger } from "./command-palette";
import { EnvironmentSwitch } from "./environment-switch";

export function Topbar({
  onMenuClick,
  onSearchClick,
}: {
  onMenuClick: () => void;
  onSearchClick: () => void;
}) {
  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center gap-3 border-b border-line bg-surface px-4 sm:px-6">
      <button
        type="button"
        onClick={onMenuClick}
        className="tap -ml-1 inline-flex size-8 items-center justify-center rounded-xs text-ink-soft hover:bg-surface-sunken hover:text-ink lg:hidden"
        aria-label="Open navigation"
      >
        <Menu className="size-5" strokeWidth={1.75} />
      </button>

      <CommandPaletteTrigger onClick={onSearchClick} />

      <div className="flex-1" />

      <EnvironmentSwitch />
      <AccountMenu />
    </header>
  );
}
