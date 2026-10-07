import { notFound } from "next/navigation";
import type { ReactNode } from "react";
import { PreviewHarness } from "./harness";

/** Development-only: real dashboard pages over sample fixtures. 404 in production. */
export default function PreviewLayout({ children }: { children: ReactNode }) {
  if (process.env.NODE_ENV === "production") notFound();
  return <PreviewHarness>{children}</PreviewHarness>;
}
