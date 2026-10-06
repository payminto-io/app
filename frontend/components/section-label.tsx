import { cn } from "@/lib/utils"

interface SectionLabelProps {
  children: React.ReactNode
  className?: string
  as?: "div" | "label" | "span" | "h3"
}

/**
 * UPPERCASE tracked label used for section headers and form field labels,
 * matching PayRam's "SELECT MEMBER", "AMOUNT", "ASSETS", "GENERAL" style.
 */
export function SectionLabel({
  children,
  className,
  as: Tag = "div",
}: SectionLabelProps) {
  return <Tag className={cn("pm-label", className)}>{children}</Tag>
}
