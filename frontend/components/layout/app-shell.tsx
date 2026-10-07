"use client";

import * as React from "react";
import { Sheet, SheetContent, SheetDescription, SheetTitle } from "@/components/ui/sheet";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useEnvironment } from "@/lib/environment/store";
import { cn } from "@/lib/utils";
import { CommandPalette, useCommandPalette } from "./command-palette";
import { Sidebar } from "./sidebar";
import { Topbar } from "./topbar";

/**
 * One shell for the dashboard and the admin area: 240px sidebar from lg up,
 * a left drawer below it, the top bar with the environment switch, and a 3px
 * amber rule along the top edge while in test.
 */
export function AppShell({ children }: { children: React.ReactNode }) {
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const palette = useCommandPalette();
  const environment = useEnvironment((s) => s.environment);

  return (
    <TooltipProvider>
      <div className="flex min-h-dvh flex-col bg-canvas">
        <div
          aria-hidden
          className={cn(
            "h-[3px] w-full shrink-0 transition-colors duration-120",
            environment === "test" ? "bg-env-test" : "bg-transparent"
          )}
        />
        <div className="flex flex-1">
          <aside className="sticky top-0 hidden h-[calc(100dvh-3px)] w-60 shrink-0 border-r border-line lg:block">
            <Sidebar />
          </aside>

          <Sheet open={drawerOpen} onOpenChange={setDrawerOpen}>
            <SheetContent side="left" className="w-[280px] gap-0 p-0 sm:max-w-[280px]" showCloseButton={false}>
              <SheetTitle className="sr-only">Navigation</SheetTitle>
              <SheetDescription className="sr-only">Pages in the dashboard</SheetDescription>
              <Sidebar onNavigate={() => setDrawerOpen(false)} />
            </SheetContent>
          </Sheet>

          <div className="flex min-w-0 flex-1 flex-col">
            <Topbar onMenuClick={() => setDrawerOpen(true)} onSearchClick={() => palette.setOpen(true)} />
            <main className="flex-1 px-4 py-6 sm:px-6 lg:py-8">
              <div className="mx-auto w-full max-w-[1280px]">{children}</div>
            </main>
          </div>
        </div>
      </div>
      <CommandPalette open={palette.open} setOpen={palette.setOpen} />
    </TooltipProvider>
  );
}
