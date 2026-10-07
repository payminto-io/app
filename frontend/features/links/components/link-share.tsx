"use client";

import { useState } from "react";
import { QRCodeSVG } from "qrcode.react";
import { QrCode } from "lucide-react";
import { CopyButton } from "@/components/copy-button";
import { CopyField } from "@/components/copy-field";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import { LINKS_COPY } from "../copy";

/** Splits a link URL so the short code, its only distinctive part, never truncates; the host does. */
export function splitLinkUrl(url: string): { head: string; code: string } {
  const bare = url.replace(/^https?:\/\//, "");
  const cut = bare.lastIndexOf("/");
  return cut < 0 ? { head: "", code: bare } : { head: bare.slice(0, cut + 1), code: bare.slice(cut + 1) };
}

/**
 * The short link in mono with the host truncated first, then copy and QR as two equal icon buttons
 * (32px on fine pointers, 44px hit area on coarse ones).
 */
export function ShortLink({ url, title, className }: { url: string; title: string; className?: string }) {
  const { head, code } = splitLinkUrl(url);
  return (
    <div className={cn("flex min-w-0 items-center gap-1", className)} onClick={(e) => e.stopPropagation()}>
      <code title={url} className="flex min-w-0 font-mono text-body-sm text-ink">
        <span className="min-w-0 truncate text-ink-soft">{head}</span>
        <span className="shrink-0">{code}</span>
      </code>
      <CopyButton value={url} variant="ghost" size="icon" label="" successMessage={LINKS_COPY.detail.copied} className="tap size-8 shrink-0" />
      <QrButton url={url} title={title} />
    </div>
  );
}

/** QR of the link URL on white (scanners need the contrast), with the URL to copy under it. */
export function QrDialog({ url, title, open, onOpenChange }: { url: string; title: string; open: boolean; onOpenChange: (v: boolean) => void }) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>{title || LINKS_COPY.untitled}</DialogTitle>
          <DialogDescription className="sr-only">{LINKS_COPY.detail.qr}</DialogDescription>
        </DialogHeader>
        <div className="flex justify-center rounded-lg bg-white p-5">
          <QRCodeSVG value={url} size={208} level="M" bgColor="#FFFFFF" fgColor="#15181D" />
        </div>
        <CopyField value={url} />
      </DialogContent>
    </Dialog>
  );
}

export function QrButton({ url, title, labelled = false }: { url: string; title: string; labelled?: boolean }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        type="button"
        variant={labelled ? "outline" : "ghost"}
        size={labelled ? "sm" : "icon"}
        className={cn("tap shrink-0", !labelled && "size-8")}
        aria-label={labelled ? undefined : LINKS_COPY.detail.qr}
        onClick={(e) => {
          e.stopPropagation();
          setOpen(true);
        }}
      >
        <QrCode />
        {labelled ? LINKS_COPY.detail.qr : null}
      </Button>
      <QrDialog url={url} title={title} open={open} onOpenChange={setOpen} />
    </>
  );
}
