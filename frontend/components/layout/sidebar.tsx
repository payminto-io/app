"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { X } from "lucide-react";
import { Logo } from "@/components/logo";
import { cn } from "@/lib/utils";
import { NAV_SECTIONS, SETTINGS_ITEM, isNavActive, type NavItem } from "./nav";

function NavLink({ item, pathname, onNavigate }: { item: NavItem; pathname: string; onNavigate?: () => void }) {
  const active = isNavActive(item, pathname);
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      onClick={onNavigate}
      aria-current={active ? "page" : undefined}
      className={cn(
        "group flex h-[30px] items-center gap-2.5 rounded-xs px-2 text-body-sm transition-colors duration-120",
        active
          ? "bg-tide-tint font-medium text-tide-strong"
          : "text-ink-soft hover:bg-surface-sunken hover:text-ink"
      )}
    >
      <Icon className={cn("size-4 shrink-0", active ? "text-tide" : "text-ink-faint group-hover:text-ink-soft")} strokeWidth={1.75} />
      <span className="truncate">{item.label}</span>
    </Link>
  );
}

/** Sidebar contents; used by the desktop rail and the mobile drawer. */
export function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname();
  return (
    <div className="flex h-full w-full flex-col bg-surface">
      <div className="flex h-14 items-center justify-between px-4">
        <Link href="/dashboard" onClick={onNavigate} className="tap inline-flex rounded-xs" aria-label="Home">
          <Logo />
        </Link>
        {onNavigate ? (
          <button
            type="button"
            onClick={onNavigate}
            className="tap -mr-1 inline-flex size-8 items-center justify-center rounded-xs text-ink-soft hover:bg-surface-sunken hover:text-ink"
            aria-label="Close navigation"
          >
            <X className="size-4" strokeWidth={1.75} />
          </button>
        ) : null}
      </div>

      <nav className="scrollbar-thin flex-1 space-y-5 overflow-y-auto px-2 pb-4 pt-1" aria-label="Primary">
        {NAV_SECTIONS.map((section, i) => (
          <div key={section.title ?? i}>
            {section.title ? (
              <div className="mb-1 px-2 text-caption font-medium text-ink-faint">{section.title}</div>
            ) : null}
            <ul className="space-y-px">
              {section.items.map((item) => (
                <li key={item.href}>
                  <NavLink item={item} pathname={pathname} onNavigate={onNavigate} />
                </li>
              ))}
            </ul>
          </div>
        ))}
      </nav>

      <div className="border-t border-line px-2 py-2">
        <NavLink item={SETTINGS_ITEM} pathname={pathname} onNavigate={onNavigate} />
      </div>
    </div>
  );
}
