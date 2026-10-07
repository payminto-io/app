import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { cva, type VariantProps } from "class-variance-authority"

import { cn } from "@/lib/utils"

const buttonVariants = cva(
  "tap group/button relative inline-flex shrink-0 items-center justify-center rounded-sm border border-transparent text-body font-medium whitespace-nowrap transition-[background-color,border-color,color,opacity] duration-120 ease-out outline-none select-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tide active:not-aria-[haspopup]:translate-y-px disabled:pointer-events-none disabled:opacity-40 aria-invalid:border-bad [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: "bg-ink text-ink-inverse hover:bg-ink/90",
        outline:
          "border-line-strong bg-surface text-ink hover:bg-surface-sunken aria-expanded:bg-surface-sunken",
        secondary:
          "bg-surface-sunken text-ink hover:bg-line aria-expanded:bg-line",
        ghost:
          "text-ink-soft hover:bg-surface-sunken hover:text-ink aria-expanded:bg-surface-sunken aria-expanded:text-ink",
        destructive:
          "bg-bad text-white hover:bg-bad/90 focus-visible:outline-bad",
        link: "h-auto px-0 text-tide underline-offset-4 hover:text-tide-strong hover:underline",
      },
      size: {
        default: "h-9 gap-2 px-3.5 has-data-[icon=inline-end]:pr-2.5 has-data-[icon=inline-start]:pl-2.5",
        xs: "h-7 gap-1 px-2 text-label [&_svg:not([class*='size-'])]:size-3.5",
        sm: "h-8 gap-1.5 px-2.5 text-body-sm [&_svg:not([class*='size-'])]:size-3.5",
        lg: "h-11 gap-2 px-4 text-body",
        icon: "size-9",
        "icon-xs": "size-7 [&_svg:not([class*='size-'])]:size-3.5",
        "icon-sm": "size-8 [&_svg:not([class*='size-'])]:size-3.5",
        "icon-lg": "size-11",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
