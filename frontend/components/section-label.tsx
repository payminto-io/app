import { cn } from "@/lib/utils"

interface SectionLabelProps {
  children: React.ReactNode
  className?: string
  as?: "div" | "label" | "span" | "h3"
}

/** 12px medium sentence-case label for sections and fields. */
export function SectionLabel({
  children,
  className,
  as: Tag = "div",
}: SectionLabelProps) {
  return <Tag className={cn("text-label font-medium text-ink-soft", className)}>{children}</Tag>
}
