/**
 * Loading, empty, and error state primitives. Every page uses these —
 * see FRONTEND_CODING_STANDARDS.
 *
 * Upgraded with Payminto design system styling while keeping the
 * exact same export signatures used by all existing pages.
 */
import type { ReactNode } from "react";
import { Skeleton } from "./skeleton";
import { Button } from "./button";
import { cn } from "@/lib/utils";

export function LoadingRows({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} className="h-12 w-full" />
      ))}
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
  icon,
  className,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  icon?: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center rounded-xl border border-dashed border-border bg-card p-10",
        className
      )}
    >
      {icon ? (
        <div className="mb-4 flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
          {icon}
        </div>
      ) : null}
      <h3 className="text-[18px] font-semibold text-foreground">{title}</h3>
      {description ? (
        <p className="mt-1.5 max-w-md text-[14px] text-muted-foreground">
          {description}
        </p>
      ) : null}
      {action ? <div className="mt-5">{action}</div> : null}
    </div>
  );
}

export function ErrorState({
  title = "Something went wrong",
  message,
  retry,
}: {
  title?: string;
  message?: string;
  retry?: () => void;
}) {
  return (
    <div className="rounded-xl border border-destructive/50 bg-destructive/5 p-6">
      <h3 className="text-sm font-semibold text-destructive">{title}</h3>
      {message ? (
        <p className="mt-1 text-sm text-destructive/90">{message}</p>
      ) : null}
      {retry ? (
        <Button
          variant="outline"
          size="sm"
          onClick={retry}
          className="mt-3 border-destructive/30 text-destructive hover:bg-destructive/10"
        >
          Try again
        </Button>
      ) : null}
    </div>
  );
}

export function AccessDeniedState() {
  return (
    <EmptyState
      title="Access denied"
      description="You don't have permission to view this page."
    />
  );
}
