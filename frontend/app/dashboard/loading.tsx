import { Skeleton } from "@/components/ui/skeleton";

export default function DashboardLoading() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-48" />
      <div className="grid gap-4 sm:grid-cols-3">
        <Skeleton className="h-[108px] rounded-md" />
        <Skeleton className="h-[108px] rounded-md" />
        <Skeleton className="h-[108px] rounded-md" />
      </div>
      <Skeleton className="h-64 rounded-md" />
    </div>
  );
}
