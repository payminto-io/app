import { notFound } from "next/navigation";
import { DesignGallery } from "./gallery";

/** Development-only gallery of the primitives inside the real shell. 404 in production. */
export default function DesignPage() {
  if (process.env.NODE_ENV === "production") notFound();
  return <DesignGallery />;
}
