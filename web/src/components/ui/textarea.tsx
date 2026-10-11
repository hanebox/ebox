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
        'w-full rounded-control border bg-surface-raised px-3 py-2 text-base text-foreground sm:text-sm',
        'transition-[border-color,background-color,box-shadow] duration-150 ease-out-quint',
        'placeholder:text-subtle',
        'hover:border-border-strong',
        'focus-visible:border-accent focus-visible:bg-surface focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-accent/45',
        'disabled:pointer-events-none disabled:opacity-60',
        invalid
          ? 'border-danger focus-visible:border-danger focus-visible:ring-danger/45'
          : 'border-border',
        className,
      )}
      {...props}
    />
  )
})
