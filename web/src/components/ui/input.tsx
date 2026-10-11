import { forwardRef, type InputHTMLAttributes } from 'react'

import { cn } from '@/lib/cn'

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  invalid?: boolean
}

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, invalid = false, ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      aria-invalid={invalid || undefined}
      className={cn(
        'h-8.5 w-full rounded-lg border bg-surface px-3 text-sm text-foreground transition-colors',
        'placeholder:text-subtle',
        'disabled:pointer-events-none disabled:opacity-60',
        invalid ? 'border-danger' : 'border-border hover:border-border-strong',
        className,
      )}
      {...props}
    />
  )
})
