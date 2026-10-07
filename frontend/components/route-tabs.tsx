"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"

import { cn } from "@/lib/utils"

/**
 * Underline tabs that are links between sibling pages (wallets, settings).
 * Same look as `TabsList variant="line"`. Active when the path ends with the
 * href's part after /dashboard, so the dev preview under /design/preview matches too.
 */
export function RouteTabs({
  tabs,
  className,
}: {
  tabs: { href: string; label: string; match?: (pathname: string) => boolean }[]
  className?: string
}) {
  const pathname = usePathname()
  return (
    <div className={cn("-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0", className)}>
      <nav className="flex h-9 w-fit min-w-full items-stretch gap-5 border-b border-line">
        {tabs.map((t) => {
          const active = t.match ? t.match(pathname) : pathname.endsWith(t.href.replace(/^\/dashboard/, ""))
          return (
            <Link
              key={t.href}
              href={t.href}
              aria-current={active ? "page" : undefined}
              className={cn(
                "tap relative -mb-px inline-flex items-center border-b-2 text-body-sm font-medium whitespace-nowrap transition-colors duration-120",
                active ? "border-tide text-ink" : "border-transparent text-ink-soft hover:text-ink"
              )}
            >
              {t.label}
            </Link>
          )
        })}
      </nav>
    </div>
  )
}
