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
        'hit-area h-8.5 w-full rounded-control border bg-surface-raised px-3 text-base text-foreground sm:text-sm',
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
