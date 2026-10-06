import * as React from "react"
import { cn } from "@/lib/utils"

interface PageHeaderProps {
  title: string
  description?: string
  icon?: React.ReactNode
  breadcrumbs?: { label: string; href?: string }[]
  children?: React.ReactNode
  className?: string
}

export function PageHeader({
  title,
  description,
  icon,
  breadcrumbs,
  children,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn("mb-6", className)}>
      {breadcrumbs && breadcrumbs.length > 0 && (
        <nav className="mb-3 flex items-center gap-1.5 text-[12px] text-muted-foreground">
          {breadcrumbs.map((crumb, i) => (
            <React.Fragment key={i}>
              {i > 0 && <span className="text-muted-foreground/50">/</span>}
              <span
                className={cn(
                  i === breadcrumbs.length - 1 &&
                    "text-primary font-medium"
                )}
              >
                {crumb.label}
              </span>
            </React.Fragment>
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
              <p className="mt-1 text-[14px] text-muted-foreground">{description}</p>
            )}
          </div>
        </div>
        {children && <div className="flex items-center gap-2 shrink-0">{children}</div>}
      </div>
    </div>
  )
}
