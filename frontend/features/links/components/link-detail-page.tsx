"use client";

import { useParams } from "next/navigation";
import { useLink } from "@/lib/query/hooks/use-links";
import { ErrorState, LoadingRows } from "@/components/ui/states";
import { LinkBuilder } from "./link-builder";

/** `/links/[id]`: the builder over a stored link, with its lifecycle actions. */
export function LinkDetailPage({ basePath = "/dashboard/links" }: { basePath?: string }) {
  const { id } = useParams<{ id: string }>();
  const { data, error, refetch } = useLink(id);
  if (error) return <ErrorState message={error.message} retry={refetch} />;
  if (!data) return <LoadingRows rows={6} />;
  return <LinkBuilder key={data.id} link={data} basePath={basePath} />;
}

export function NewLinkPage({ basePath = "/dashboard/links" }: { basePath?: string }) {
  return <LinkBuilder link={null} basePath={basePath} />;
}
