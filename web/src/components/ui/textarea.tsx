import { forwardRef, type TextareaHTMLAttributes } from 'react'

import { cn } from '@/lib/cn'

export interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  invalid?: boolean
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className, invalid = false, ...props },
  ref,
) {
  return (
    <textarea
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cn(
        'w-full rounded-lg border bg-surface px-3 py-2 text-sm text-foreground transition-colors',
        'placeholder:text-subtle',
        'disabled:pointer-events-none disabled:opacity-60',
        invalid ? 'border-danger' : 'border-border hover:border-border-strong',
        className,
      )}
      {...props}
    />
  )
})
