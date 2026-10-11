import { IconChevronDown } from '@tabler/icons-react'
import { forwardRef, type SelectHTMLAttributes } from 'react'

import { cn } from '@/lib/cn'

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  invalid?: boolean
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { className, invalid = false, children, ...props },
  ref,
) {
  return (
    <div className="relative">
      <select
        ref={ref}
        aria-invalid={invalid || undefined}
        className={cn(
          'hit-area h-8.5 w-full appearance-none rounded-control border bg-surface-raised ps-3 pe-8 text-base text-foreground sm:text-sm',
          'transition-[border-color,background-color,box-shadow] duration-150 ease-out-quint',
          'hover:border-border-strong',
          'focus-visible:border-accent focus-visible:bg-surface focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-accent/45',
          'disabled:pointer-events-none disabled:opacity-60',
          invalid
            ? 'border-danger focus-visible:border-danger focus-visible:ring-danger/45'
            : 'border-border',
          className,
        )}
        {...props}
      >
        {children}
      </select>
      <IconChevronDown
        aria-hidden
        className="pointer-events-none absolute end-2.5 top-1/2 size-3.5 -translate-y-1/2 text-subtle"
      />
    </div>
  )
})
