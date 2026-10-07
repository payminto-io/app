"use client";

import { useState, type ReactNode } from "react";
import { QRCodeSVG } from "qrcode.react";
import { QrCode } from "lucide-react";
import { CopyField } from "@/components/copy-field";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { LINKS_COPY } from "../copy";

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

export function QrButton({ url, title, size = "icon-sm", children }: { url: string; title: string; size?: "icon-sm" | "sm"; children?: ReactNode }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button
        type="button"
        variant="outline"
        size={size}
        aria-label={children ? undefined : LINKS_COPY.detail.qr}
        onClick={(e) => {
          e.stopPropagation();
          setOpen(true);
        }}
      >
        <QrCode />
        {children}
      </Button>
      <QrDialog url={url} title={title} open={open} onOpenChange={setOpen} />
    </>
  );
}
