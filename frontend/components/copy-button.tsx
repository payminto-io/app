"use client"

import * as React from "react"
import { Check, Copy } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export type CopyButtonProps = {
  value: string
  label?: string
  className?: string
  size?: "sm" | "default" | "lg" | "icon"
  variant?: "default" | "outline" | "ghost" | "secondary"
  successMessage?: string
}

export function CopyButton({
  value,
  label,
  className,
  size = "sm",
  variant = "outline",
  successMessage = "Copied to clipboard",
}: CopyButtonProps) {
  const [copied, setCopied] = React.useState(false)

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      toast.success(successMessage)
      setTimeout(() => setCopied(false), 1600)
    } catch {
      toast.error("Could not copy")
    }
  }

  return (
    <Button
      type="button"
      size={size}
      variant={variant}
      onClick={handleCopy}
      aria-label={label === "" ? (copied ? "Copied" : "Copy") : undefined}
      className={cn("gap-1.5", className)}
    >
      {copied ? (
        <Check className="size-3.5" />
      ) : (
        <Copy className="size-3.5" />
      )}
      {label ?? (copied ? "Copied" : "Copy")}
    </Button>
  )
}
