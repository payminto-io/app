import { cn } from "@/lib/utils"

const FORMATS: Record<"date" | "datetime" | "time", Intl.DateTimeFormatOptions> = {
  date: { month: "short", day: "numeric", year: "numeric" },
  datetime: { month: "short", day: "numeric", year: "numeric", hour: "2-digit", minute: "2-digit" },
  time: { hour: "2-digit", minute: "2-digit", second: "2-digit" },
}

/** A timestamp in tabular figures with the full value on hover. Renders nothing for a missing or invalid value. */
export function DateTime({
  value,
  format = "datetime",
  className,
}: {
  value: string | number | Date | null | undefined
  format?: "date" | "datetime" | "time"
  className?: string
}) {
  if (value === null || value === undefined || value === "") return null
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return (
    <time dateTime={date.toISOString()} title={date.toISOString()} className={cn("num whitespace-nowrap", className)}>
      {date.toLocaleString("en-US", FORMATS[format])}
    </time>
  )
}
