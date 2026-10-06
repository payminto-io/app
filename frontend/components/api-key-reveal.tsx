"use client"

import * as React from "react"
import { Eye, EyeOff, ShieldAlert } from "lucide-react"

import { Button } from "@/components/ui/button"
import { CopyButton } from "@/components/copy-button"
import { cn } from "@/lib/utils"

export type ApiKeyRevealProps = {
  /** The plaintext secret. Shown once, then masked. */
  secret: string
  label?: string
  className?: string
  /** Called once the user confirms they have copied the key. */
  onConfirm?: () => void
}

function maskSecret(s: string): string {
  if (s.length <= 8) return "•".repeat(s.length)
  return `${s.slice(0, 4)}${"•".repeat(Math.max(8, s.length - 8))}${s.slice(-4)}`
}

export function ApiKeyReveal({
  secret,
  label = "API Key",
  className,
  onConfirm,
}: ApiKeyRevealProps) {
  const [revealed, setRevealed] = React.useState(false)
  const [confirmed, setConfirmed] = React.useState(false)

  const display = confirmed || !revealed ? maskSecret(secret) : secret

  return (
    <div
      className={cn(
        "rounded-xl border border-border bg-card p-4 space-y-3",
        className
      )}
    >
      <div className="flex items-center gap-2 text-[12px] font-semibold uppercase tracking-wider text-muted-foreground">
        <ShieldAlert className="size-3.5 text-[var(--pm-warning)]" />
        {label} — show once
      </div>

      <div className="flex items-center gap-2">
        <code className="flex-1 truncate rounded-md border border-border bg-muted/50 px-3 py-2 font-mono text-[13px]">
          {display}
        </code>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={confirmed}
          onClick={() => setRevealed((r) => !r)}
        >
          {revealed && !confirmed ? (
            <EyeOff className="size-3.5" />
          ) : (
            <Eye className="size-3.5" />
          )}
        </Button>
        <CopyButton value={secret} />
      </div>

      {!confirmed ? (
        <p className="text-[12px] text-muted-foreground">
          Save this key in a secure location. You won&apos;t be able to view it
          again.
        </p>
      ) : (
        <p className="text-[12px] text-[var(--pm-success)]">
          Key hidden. If you lost it, regenerate a new one.
        </p>
      )}

      {!confirmed ? (
        <Button
          type="button"
          size="sm"
          className="w-full"
          onClick={() => {
            setConfirmed(true)
            setRevealed(false)
            onConfirm?.()
          }}
        >
          I&apos;ve copied it
        </Button>
      ) : null}
    </div>
  )
}
