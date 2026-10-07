"use client"

import * as React from "react"
import { Eye, EyeOff } from "lucide-react"

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

/** A secret shown once: masked by default, reveal and copy, then hidden for good after confirming. */
export function ApiKeyReveal({ secret, label = "API key", className, onConfirm }: ApiKeyRevealProps) {
  const [revealed, setRevealed] = React.useState(false)
  const [confirmed, setConfirmed] = React.useState(false)

  const display = confirmed || !revealed ? maskSecret(secret) : secret

  return (
    <div className={cn("space-y-3", className)}>
      <div className="space-y-1.5">
        <div className="text-label font-medium text-ink">{label}</div>
        <div className="flex items-center gap-1 rounded-sm border border-line bg-surface-sunken py-1 pr-1 pl-3">
          <code className="min-w-0 flex-1 truncate font-mono text-body-sm text-ink">{display}</code>
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="size-7"
            disabled={confirmed}
            aria-label={revealed && !confirmed ? "Hide key" : "Show key"}
            onClick={() => setRevealed((r) => !r)}
          >
            {revealed && !confirmed ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
          </Button>
          <CopyButton value={secret} variant="ghost" size="icon" label="" className="size-7" />
        </div>
        <p className={cn("text-caption", confirmed ? "text-ink-soft" : "text-wait")}>
          {confirmed ? "Hidden. If it is lost, generate a new key." : "Shown once. Store it somewhere safe now."}
        </p>
      </div>

      {!confirmed ? (
        <Button
          type="button"
          className="w-full"
          onClick={() => {
            setConfirmed(true)
            setRevealed(false)
            onConfirm?.()
          }}
        >
          I have copied it
        </Button>
      ) : null}
    </div>
  )
}
