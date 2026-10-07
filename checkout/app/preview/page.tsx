import { notFound } from "next/navigation";
import { Preview } from "./preview";

/** Development only: renders every checkout state from fixtures. 404 in production. */
export default async function PreviewPage({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  if (process.env.NODE_ENV === "production") notFound();
  const params = await searchParams;
  const state = typeof params.state === "string" ? params.state : "choose";
  const theme = params.theme === "dark" ? "dark" : params.theme === "light" ? "light" : undefined;
  return <Preview state={state} theme={theme} />;
}
