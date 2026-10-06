"use client"

import * as React from "react"
import { Check, ChevronLeft, ChevronRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export type WizardStepDef = {
  id: string
  title: string
  description?: string
}

export type WizardProps = {
  steps: WizardStepDef[]
  /** Render content for the active step. */
  children: (ctx: {
    stepIndex: number
    step: WizardStepDef
    next: () => void
    back: () => void
  }) => React.ReactNode
  onComplete?: () => void | Promise<void>
  /** Per-step validation. Return false to block "Next". */
  canAdvance?: (stepIndex: number) => boolean
  className?: string
  completeLabel?: string
}

export function Wizard({
  steps,
  children,
  onComplete,
  canAdvance,
  className,
  completeLabel = "Create",
}: WizardProps) {
  const [stepIndex, setStepIndex] = React.useState(0)
  const [submitting, setSubmitting] = React.useState(false)
  const isLast = stepIndex === steps.length - 1

  const next = React.useCallback(async () => {
    if (canAdvance && !canAdvance(stepIndex)) return
    if (isLast) {
      if (onComplete) {
        try {
          setSubmitting(true)
          await onComplete()
        } finally {
          setSubmitting(false)
        }
      }
      return
    }
    setStepIndex((i) => Math.min(steps.length - 1, i + 1))
  }, [canAdvance, isLast, onComplete, stepIndex, steps.length])

  const back = React.useCallback(
    () => setStepIndex((i) => Math.max(0, i - 1)),
    []
  )

  const progress = ((stepIndex + 1) / steps.length) * 100

  return (
    <div className={cn("space-y-6", className)}>
      {/* Step indicator */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          {steps.map((s, i) => {
            const done = i < stepIndex
            const active = i === stepIndex
            return (
              <div key={s.id} className="flex flex-1 items-center gap-2 min-w-0">
                <div
                  className={cn(
                    "flex size-7 shrink-0 items-center justify-center rounded-full border text-[12px] font-semibold",
                    done && "border-primary bg-primary text-white",
                    active && "border-primary text-primary",
                    !done && !active && "border-border text-muted-foreground"
                  )}
                >
                  {done ? <Check className="size-3.5" /> : i + 1}
                </div>
                <div className="min-w-0">
                  <div
                    className={cn(
                      "truncate text-[12px] font-medium",
                      active ? "text-foreground" : "text-muted-foreground"
                    )}
                  >
                    {s.title}
                  </div>
                </div>
                {i < steps.length - 1 && (
                  <div className="hidden md:block flex-1 h-px bg-border" />
                )}
              </div>
            )
          })}
        </div>
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className="h-full bg-primary transition-all"
            style={{ width: `${progress}%` }}
          />
        </div>
      </div>

      {/* Step body */}
      <div className="rounded-xl border border-border bg-card p-6">
        {children({ stepIndex, step: steps[stepIndex], next, back })}
      </div>

      {/* Footer nav */}
      <div className="flex items-center justify-between">
        <Button
          type="button"
          variant="outline"
          onClick={back}
          disabled={stepIndex === 0 || submitting}
        >
          <ChevronLeft className="size-4" />
          Back
        </Button>
        <Button type="button" onClick={next} disabled={submitting}>
          {isLast ? (submitting ? "Working…" : completeLabel) : "Next"}
          {!isLast && <ChevronRight className="size-4" />}
        </Button>
      </div>
    </div>
  )
}
