import * as React from "react"

import { cn } from "@/lib/utils"

function Textarea({ className, ...props }: React.ComponentProps<"textarea">) {
  return (
    <textarea
      data-slot="textarea"
      className={cn(
        "flex field-sizing-content min-h-20 w-full rounded-sm border border-line-strong bg-surface px-3 py-2 text-body text-ink transition-[border-color] duration-120 ease-out outline-none placeholder:text-ink-faint hover:border-ink-faint focus-visible:border-tide focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tide disabled:cursor-not-allowed disabled:bg-surface-sunken disabled:text-ink-faint aria-invalid:border-bad",
        className
      )}
      {...props}
    />
  )
}

export { Textarea }
