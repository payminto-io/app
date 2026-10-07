import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * PageHeader: breadcrumb, h1, one sentence, actions on the right.
 * Supports both `actions` (older pages) and `children` (newer pages).
 */
export function PageHeader({
  title,
  description,
  actions,
  icon,
  breadcrumbs,
  children,
  className,
}: {
  title: string;
  description?: string;
  /** @deprecated use children instead */
  actions?: ReactNode;
  icon?: ReactNode;
  breadcrumbs?: { label: string; href?: string }[];
  children?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("mb-6", className)}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <nav aria-label="Breadcrumb" className="mb-2 flex items-center gap-1.5 text-label text-ink-soft">
          {breadcrumbs.map((crumb, i) => (
            <span key={i} className="inline-flex items-center gap-1.5">
              {i > 0 && <span className="text-ink-faint">/</span>}
              <span className={cn(i === breadcrumbs.length - 1 && "text-ink")}>
                {crumb.label}
              </span>
            </span>
          ))}
        </nav>
      )}
      <div className="flex flex-wrap items-start justify-between gap-x-6 gap-y-3">
        <div className="flex min-w-0 items-start gap-3">
          {icon && (
            <div className="flex size-9 shrink-0 items-center justify-center rounded-sm border border-line bg-surface text-ink-soft [&_svg]:size-4">
              {icon}
            </div>
          )}
          <div className="min-w-0">
            <h1 className="text-h1 font-semibold text-ink">{title}</h1>
            {description && (
              <p className="mt-1 max-w-[64ch] text-body text-ink-soft">{description}</p>
            )}
          </div>
        </div>
        {(actions || children) && (
          <div className="flex shrink-0 items-center gap-2">
            {actions}
            {children}
          </div>
        )}
      </div>
    </div>
  );
}
