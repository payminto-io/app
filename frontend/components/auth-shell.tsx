import {
  Activity,
  ArrowUpRight,
  Check,
  Globe2,
  LockKeyhole,
  ShieldCheck,
  Sparkles,
} from "lucide-react";
import type { ReactNode } from "react";

export function AuthShell({
  title,
  description,
  children,
  footer,
}: {
  title: ReactNode;
  description: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  illustration?: ReactNode;
}) {
  return (
    <main className="grid min-h-[calc(100dvh-40px)] bg-white lg:grid-cols-[minmax(430px,.92fr)_minmax(520px,1.08fr)]">
      <aside className="relative hidden overflow-hidden bg-[radial-gradient(circle_at_12%_14%,#ff8a64_0,transparent_30%),radial-gradient(circle_at_90%_82%,#e96991_0,transparent_34%),linear-gradient(145deg,#b91722_0%,#e22323_46%,#f06b28_100%)] px-10 py-9 text-white lg:flex lg:flex-col xl:px-16 xl:py-12">
        <div className="pointer-events-none absolute inset-0 opacity-[.13] [background-image:linear-gradient(rgba(255,255,255,.28)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,.28)_1px,transparent_1px)] [background-size:48px_48px] [mask-image:linear-gradient(to_bottom,black,transparent_88%)]" />
        <div className="pointer-events-none absolute -bottom-48 -left-48 size-[430px] rounded-full bg-orange-200/35 blur-sm" />
        <div className="pointer-events-none absolute -right-32 top-1/4 size-72 rounded-full bg-pink-200/30 blur-sm" />

        <div className="relative z-10 flex items-center gap-3">
          <span className="relative grid size-10 place-items-center rounded-xl bg-white text-[19px] font-black text-[#17152f] shadow-[0_10px_30px_rgba(67,9,14,.2)]">
            P
            <span className="absolute -right-0.5 -top-0.5 size-2.5 rounded-full border-2 border-white bg-emerald-500" />
          </span>
          <span>
            <strong
              className="block text-[20px] font-bold tracking-[-.7px]"
              style={{ fontFamily: "var(--font-inter), sans-serif" }}
            >
              Payminto
            </strong>
            <small className="block text-[8px] font-bold uppercase tracking-[.18em] text-white/50">
              Merchant infrastructure
            </small>
          </span>
        </div>

        <div className="relative z-10 my-auto max-w-[520px] py-10">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-white/15 bg-[#10103a]/20 px-3 py-1.5 text-[10px] font-bold uppercase tracking-[.14em] text-white/65 backdrop-blur-xl">
            <Sparkles className="size-3.5 text-orange-100" />
            Payments without borders
          </div>
          <h2
            className="max-w-[500px] text-[42px] font-semibold leading-[1.03] tracking-[-2px] xl:text-[54px]"
            style={{ fontFamily: "var(--font-inter), sans-serif" }}
          >
            The operating layer for internet money.
          </h2>
          <p className="mt-5 max-w-[440px] text-[14px] leading-6 text-white/62 xl:text-[15px]">
            Accept, monitor, and reconcile crypto payments from one secure
            merchant workspace.
          </p>

          <div className="mt-8 rounded-[24px] border border-white/20 bg-[linear-gradient(145deg,rgba(58,7,12,.48),rgba(98,26,18,.28))] p-5 shadow-[0_32px_80px_rgba(71,10,16,.26),inset_0_1px_rgba(255,255,255,.16)] backdrop-blur-2xl xl:p-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <span className="grid size-10 place-items-center rounded-xl bg-white/10">
                  <Activity className="size-5 text-orange-100" />
                </span>
                <span>
                  <small className="block text-[9px] font-bold uppercase tracking-[.16em] text-white/42">
                    Payment activity
                  </small>
                  <strong className="mt-1 block text-sm">Live settlement feed</strong>
                </span>
              </div>
              <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-400/15 px-2.5 py-1 text-[9px] font-bold uppercase tracking-wider text-emerald-200">
                <span className="size-1.5 rounded-full bg-emerald-300" /> Live
              </span>
            </div>
            <div className="my-5 h-px bg-white/10" />
            <div className="grid grid-cols-3 gap-3">
              <Signal label="Networks" value="Multi-chain" icon={<Globe2 />} />
              <Signal label="Protection" value="Encrypted" icon={<ShieldCheck />} />
              <Signal label="Status" value="Operational" icon={<Check />} />
            </div>
          </div>
        </div>

        <div className="relative z-10 flex items-center gap-2 text-[10px] font-semibold text-white/45">
          <LockKeyhole className="size-3.5" />
          Encrypted sessions
          <span className="size-1 rounded-full bg-white/25" />
          Role-based access
        </div>
      </aside>

      <section className="flex min-h-[calc(100dvh-40px)] flex-col bg-[radial-gradient(circle_at_85%_5%,#fff0eb_0,transparent_30%),#fff]">
        <header className="flex items-center justify-between px-5 py-5 sm:px-8 lg:px-10">
          <div className="flex items-center gap-2.5 lg:hidden">
            <span className="relative grid size-9 place-items-center rounded-[11px] bg-[#17152f] text-base font-black text-white shadow-[0_8px_20px_rgba(23,21,47,.18)]">
              P
              <span className="absolute -right-0.5 -top-0.5 size-2.5 rounded-full border-2 border-white bg-emerald-500" />
            </span>
            <strong
              className="text-[19px] tracking-[-.6px]"
              style={{ fontFamily: "var(--font-inter), sans-serif" }}
            >
              Payminto
            </strong>
          </div>
          <div className="ml-auto inline-flex items-center gap-2 rounded-full border border-[#e8e8f2] bg-white/80 px-3 py-1.5 text-[10px] font-bold text-[#798198] shadow-sm backdrop-blur-lg">
            <ShieldCheck className="size-3.5 text-[#e22323]" />
            Secure merchant access
          </div>
        </header>

        <div className="flex flex-1 items-center justify-center px-5 py-8 sm:px-8 lg:px-12">
          <div className="w-full max-w-[455px]">
            <div className="mb-8">
              <p className="mb-3 text-[10px] font-extrabold uppercase tracking-[.18em] text-[#e22323]">
                Merchant control center
              </p>
              <h1
                className="text-[34px] font-semibold leading-[1.08] tracking-[-1.35px] text-[#111a2e] sm:text-[40px]"
                style={{ fontFamily: "var(--font-inter), sans-serif" }}
              >
                {title}
              </h1>
              <p className="mt-3 text-[14px] leading-6 text-[#717b91]">
                {description}
              </p>
            </div>

            {children}

            {footer ? (
              <div className="mt-6 border-t border-[#eceef4] pt-5 text-[12px] text-[#7b8498]">
                {footer}
              </div>
            ) : null}
          </div>
        </div>

        <footer className="flex flex-wrap items-center justify-between gap-3 px-5 py-5 text-[10px] text-[#9aa1b1] sm:px-8 lg:px-10">
          <span>© 2026 Payminto</span>
          <span className="inline-flex items-center gap-1.5">
            Privacy-first infrastructure <ArrowUpRight className="size-3" />
          </span>
        </footer>
      </section>
    </main>
  );
}

function Signal({
  label,
  value,
  icon,
}: {
  label: string;
  value: string;
  icon: ReactNode;
}) {
  return (
    <div className="min-w-0 rounded-xl bg-white/[.06] px-3 py-3">
      <span className="mb-2 block size-4 text-white/50 [&_svg]:size-4">{icon}</span>
      <small className="block truncate text-[8px] font-bold uppercase tracking-wider text-white/35">
        {label}
      </small>
      <strong className="mt-1 block truncate text-[10px] font-semibold text-white/80">
        {value}
      </strong>
    </div>
  );
}
