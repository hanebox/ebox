import { IconLoader2 } from '@tabler/icons-react'
import { forwardRef, type ButtonHTMLAttributes } from 'react'

import { cn } from '@/lib/cn'

export type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger' | 'success'
export type ButtonSize = 'sm' | 'md' | 'lg' | 'icon'

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'bg-primary text-primary-foreground hover:bg-primary-hover',
  secondary:
    'bg-surface-raised text-foreground shadow-border hover:bg-surface-overlay hover:shadow-border-hover',
  outline: 'bg-transparent text-foreground shadow-border hover:bg-surface-raised',
  ghost: 'bg-transparent text-muted hover:bg-surface-raised hover:text-foreground',
  danger: 'bg-danger text-background hover:bg-danger/90',
  success: 'bg-success text-background hover:bg-success/90',
}

const sizeClasses: Record<ButtonSize, string> = {
  sm: 'h-7 gap-1.5 rounded-tile px-2.5 text-xs has-[>svg:first-child]:ps-2 has-[>svg:last-child]:pe-2',
  md: 'h-8.5 gap-1.5 rounded-control px-3 text-ui has-[>svg:first-child]:ps-2.5 has-[>svg:last-child]:pe-2.5',
  lg: 'h-10 gap-2 rounded-control px-4 text-sm has-[>svg:first-child]:ps-3.5 has-[>svg:last-child]:pe-3.5',
  icon: 'size-8.5 rounded-control',
}

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant
  size?: ButtonSize
  loading?: boolean
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { className, variant = 'secondary', size = 'md', loading = false, disabled, children, ...props },
  ref,
) {
  return (
    <button
      ref={ref}
      disabled={disabled || loading}
      className={cn(
        'hit-area press inline-flex shrink-0 items-center justify-center font-medium whitespace-nowrap',
        'transition-[background-color,border-color,color,opacity,transform,box-shadow] duration-150 ease-out-quint',
        'disabled:pointer-events-none disabled:opacity-45',
        '[&_svg]:shrink-0 [&_svg]:stroke-[2]',
        variantClasses[variant],
        sizeClasses[size],
        className,
      )}
      {...props}
    >
      {loading ? <IconLoader2 aria-hidden className="size-3.5 animate-spin" /> : null}
      {children}
    </button>
  )
})
