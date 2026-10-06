import type { ReactNode } from "react";

/** Checkout-inspired chrome shared by login and account-access screens. */
export default function AuthLayout({ children }: { children: ReactNode }) {
  return (
    <div
      className="min-h-dvh bg-white text-[#15171c]"
      style={{ fontFamily: "var(--font-inter), Inter, ui-sans-serif, system-ui" }}
    >
      <div className="flex min-h-10 items-center justify-center gap-2 bg-[#17132f] px-4 py-2 text-center text-[11px] font-semibold tracking-[0.02em] text-white/75 sm:text-xs">
        <span className="size-1.5 rounded-full bg-emerald-400 shadow-[0_0_12px_rgba(52,211,153,.9)]" />
        Secure local demo environment
        <span className="hidden text-white/35 sm:inline">•</span>
        <span className="hidden text-white/55 sm:inline">Use testnet assets only</span>
      </div>
      {children}
    </div>
  );
}
