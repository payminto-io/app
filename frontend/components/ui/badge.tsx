import { mergeProps } from "@base-ui/react/merge-props"
import { useRender } from "@base-ui/react/use-render"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const badgeVariants = cva(
  "group/badge inline-flex h-5 w-fit shrink-0 items-center justify-center gap-1 overflow-hidden rounded-xs border border-transparent px-1.5 text-label font-medium whitespace-nowrap transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tide [&>svg]:pointer-events-none [&>svg]:size-3!",
  {
    variants: {
      variant: {
        default: "bg-ink text-ink-inverse",
        secondary: "bg-surface-sunken text-ink-soft",
        destructive: "bg-bad-tint text-bad",
        outline: "border-line-strong text-ink",
        ghost: "text-ink-soft hover:bg-surface-sunken",
        link: "text-tide underline-offset-4 hover:underline",
        ok: "bg-ok-tint text-ok",
        wait: "bg-wait-tint text-wait",
        bad: "bg-bad-tint text-bad",
        note: "bg-note-tint text-note",
        mute: "bg-mute-tint text-mute",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
)

function Badge({
  className,
  variant = "default",
  render,
  ...props
}: useRender.ComponentProps<"span"> & VariantProps<typeof badgeVariants>) {
  return useRender({
    defaultTagName: "span",
    props: mergeProps<"span">(
      {
        className: cn(badgeVariants({ variant }), className),
      },
      props
    ),
    render,
    state: {
      slot: "badge",
      variant,
    },
  })
}

export { Badge, badgeVariants }
