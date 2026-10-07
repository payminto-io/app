import type { ComponentProps, ReactNode } from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";

/** Label above the control, then a hint or an error. Rules: DESIGN.md section 12. */
export function FormField({
  label,
  htmlFor,
  error,
  hint,
  children,
  className,
}: {
  label: string;
  htmlFor: string;
  error?: string;
  hint?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("space-y-1.5", className)}>
      <Label htmlFor={htmlFor}>{label}</Label>
      {children}
      {hint && !error ? <p className="text-caption text-ink-soft">{hint}</p> : null}
      {error ? (
        <p role="alert" className="text-label text-bad">
          {error}
        </p>
      ) : null}
    </div>
  );
}

export function TextInput(props: ComponentProps<"input">) {
  return <Input {...props} />;
}

export function TextArea(props: ComponentProps<"textarea">) {
  return <Textarea {...props} />;
}
