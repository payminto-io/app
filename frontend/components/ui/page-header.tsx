import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * PageHeader — backward-compatible version.
 *
 * Supports both the old `actions` prop (used by existing pages)
 * and the new `children` + `breadcrumbs` + `icon` props from the
 * worktree design. Both work simultaneously.
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
        <nav className="mb-3 flex items-center gap-1.5 text-[12px] text-muted-foreground">
          {breadcrumbs.map((crumb, i) => (
            <span key={i} className="inline-flex items-center gap-1.5">
              {i > 0 && <span className="text-muted-foreground/50">/</span>}
              <span
                className={cn(
                  i === breadcrumbs.length - 1 && "text-primary font-medium"
                )}
              >
                {crumb.label}
              </span>
            </span>
          ))}
        </nav>
      )}
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3 min-w-0">
          {icon && (
            <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
              {icon}
            </div>
          )}
          <div className="min-w-0">
            <h1 className="text-[26px] font-bold leading-tight text-foreground tracking-tight">
              {title}
            </h1>
            {description && (
              <p className="mt-1 text-[14px] text-muted-foreground">
                {description}
              </p>
            )}
          </div>
        </div>
        {(actions || children) && (
          <div className="flex items-center gap-2 shrink-0">
            {actions}
            {children}
          </div>
        )}
      </div>
    </div>
  );
}
