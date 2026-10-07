import type { ReactNode } from "react";

/**
 * Public layout for /pay checkout pages.
 * No navbar, no sidebar.
 * Just a minimal wrapper so the checkout view controls the entire viewport.
 */
export default function PublicLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-canvas">
      {children}
    </div>
  );
}
