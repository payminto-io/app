import * as React from "react"
import { Input as InputPrimitive } from "@base-ui/react/input"

import { cn } from "@/lib/utils"

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <InputPrimitive
      type={type}
      data-slot="input"
      className={cn(
        "h-9 w-full min-w-0 rounded-sm border border-line-strong bg-surface px-3 py-1 text-body text-ink transition-[border-color,box-shadow] duration-120 ease-out outline-none",
        "file:inline-flex file:h-6 file:border-0 file:bg-transparent file:text-body-sm file:font-medium file:text-ink",
        "placeholder:text-ink-faint hover:border-ink-faint",
        "focus-visible:border-tide focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tide",
        "disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-surface-sunken disabled:text-ink-faint",
        "aria-invalid:border-bad aria-invalid:focus-visible:outline-bad",
        "md:text-body",
        className
      )}
      {...props}
    />
  )
}

export { Input }
