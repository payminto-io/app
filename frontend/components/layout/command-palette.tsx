"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Search } from "lucide-react";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { NAV_SECTIONS, SETTINGS_ITEM } from "./nav";

export function useCommandPalette() {
  const [open, setOpen] = React.useState(false);
  React.useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setOpen((v) => !v);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  return { open, setOpen };
}

export function CommandPaletteTrigger({ onClick }: { onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="tap inline-flex h-8 items-center gap-2 rounded-sm border border-line-strong bg-surface px-2.5 text-body-sm text-ink-faint transition-colors duration-120 hover:border-ink-faint hover:text-ink-soft sm:w-64"
      aria-label="Search or jump to a page"
    >
      <Search className="size-4" strokeWidth={1.75} />
      <span className="hidden flex-1 text-left sm:inline">Search or jump to</span>
      <kbd className="hidden rounded-xs border border-line bg-surface-sunken px-1.5 py-px font-mono text-caption text-ink-soft sm:inline">
        ⌘K
      </kbd>
    </button>
  );
}

export function CommandPalette({ open, setOpen }: { open: boolean; setOpen: (v: boolean) => void }) {
  const router = useRouter();
  const go = (href: string) => {
    setOpen(false);
    router.push(href);
  };
  return (
    <CommandDialog open={open} onOpenChange={setOpen} title="Jump to" description="Pages and actions">
      <CommandInput placeholder="Jump to a page" />
      <CommandList>
        <CommandEmpty>No page matches.</CommandEmpty>
        {NAV_SECTIONS.map((section, i) => (
          <CommandGroup key={section.title ?? i} heading={section.title ?? "Workspace"}>
            {section.items.map((item) => (
              <CommandItem
                key={item.href}
                value={[item.label, ...(item.keywords ?? [])].join(" ")}
                onSelect={() => go(item.href)}
              >
                <item.icon />
                {item.label}
              </CommandItem>
            ))}
          </CommandGroup>
        ))}
        <CommandGroup heading="Account">
          <CommandItem value="settings" onSelect={() => go(SETTINGS_ITEM.href)}>
            <SETTINGS_ITEM.icon />
            Settings
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}
