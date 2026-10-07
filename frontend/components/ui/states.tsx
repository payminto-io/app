/**
 * Loading, empty and error state primitives. Every page uses these.
 * Rules: docs/design/DESIGN.md sections 13 and 14; references.md "Empty and error states".
 */
import type { ReactNode } from "react";
import { Skeleton } from "./skeleton";
import { Button } from "./button";
import { cn } from "@/lib/utils";

export function LoadingRows({ rows = 5 }: { rows?: number }) {
  return (
    <div className="overflow-hidden rounded-md border border-line bg-surface">
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 border-b border-line px-3 last:border-0" style={{ height: 36 }}>
          <Skeleton className="h-3 w-[22%]" />
          <Skeleton className="h-3 w-[38%]" />
          <Skeleton className="ml-auto h-3 w-[14%]" />
        </div>
      ))}
    </div>
  );
}

export type EmptyStateProps = {
  title: string;
  description?: string;
  action?: ReactNode;
  icon?: ReactNode;
  className?: string;
};

export function EmptyState({
  title,
  description,
  action,
  icon,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center rounded-md border border-dashed border-line-strong bg-surface px-6 py-12 text-center",
        className
      )}
    >
      {icon ? (
        <div className="mb-3 flex size-9 items-center justify-center rounded-sm border border-line bg-surface-sunken text-ink-soft [&_svg]:size-4">
          {icon}
        </div>
      ) : null}
      <h3 className="text-body font-medium text-ink">{title}</h3>
      {description ? (
        <p className="mt-1 max-w-[44ch] text-body-sm text-ink-soft">{description}</p>
      ) : null}
      {action ? <div className="mt-4">{action}</div> : null}
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
    <div role="alert" className="rounded-md border border-bad/30 bg-bad-tint px-4 py-3">
      <h3 className="text-body font-medium text-bad">{title}</h3>
      {message ? <p className="mt-0.5 text-body-sm text-ink">{message}</p> : null}
      {retry ? (
        <Button variant="outline" size="sm" onClick={retry} className="mt-3">
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
      description="Your role does not include this page. Ask an owner to change it."
    />
  );
}
