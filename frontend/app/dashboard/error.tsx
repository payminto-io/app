"use client";

import { ErrorState } from "@/components/ui/states";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="mx-auto max-w-[560px] pt-8">
      <ErrorState message={error.message || "The page could not be loaded."} retry={reset} />
    </div>
  );
}
