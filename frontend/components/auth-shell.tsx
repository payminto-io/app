import Link from "next/link";
import type { ReactNode } from "react";
import { LogoMark } from "@/components/logo";

/**
 * Auth layout: one centred column. Logo, a short heading, the fields, the
 * primary button, one secondary link. No product copy (owner decision 2026-10-07).
 */
export function AuthShell({
  title,
  description,
  children,
  footer,
}: {
  title: ReactNode;
  description?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  illustration?: ReactNode;
}) {
  return (
    <main className="flex min-h-dvh flex-col items-center justify-start px-5 pb-10 pt-16 sm:justify-center sm:px-8 sm:py-12">
      <div className="w-full max-w-[360px]">
        <Link href="/" className="tap mx-auto flex w-fit rounded-xs" aria-label="Home">
          <LogoMark size={36} />
        </Link>
        <h1 className="mt-6 text-center text-h2 font-semibold text-ink">{title}</h1>
        {description ? <p className="mt-1.5 text-center text-body-sm text-ink-soft">{description}</p> : null}

        <div className="mt-8">{children}</div>

        {footer ? <p className="mt-6 text-center text-body-sm text-ink-soft">{footer}</p> : null}
      </div>
    </main>
  );
}
