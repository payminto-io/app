import { Sidebar } from "@/components/layout/sidebar";
import { Topbar } from "@/components/layout/topbar";

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-dvh flex-col bg-background">
      <div className="flex min-h-9 w-full items-center justify-center gap-2 bg-[#17152f] px-4 py-2 text-center text-[11px] font-semibold tracking-[0.01em] text-white/75 sm:text-xs">
        <span className="size-1.5 rounded-full bg-[var(--pm-success)] shadow-[0_0_10px_rgba(34,184,106,.85)]" />
        Test environment
        <span className="text-white/30">•</span>
        Do not send mainnet assets
      </div>

      <div className="flex flex-1">
        <Sidebar />
        <div className="flex flex-1 flex-col min-w-0">
          <Topbar />
          <main className="flex-1 overflow-y-auto px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
            <div className="mx-auto w-full max-w-[1540px]">{children}</div>
          </main>
        </div>
      </div>
    </div>
  );
}
