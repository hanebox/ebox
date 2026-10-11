import type { HTMLAttributes } from 'react'

import { cn } from '@/lib/cn'

export type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info' | 'outline'

const variantClasses: Record<BadgeVariant, string> = {
  default: 'bg-surface-raised text-muted',
  success: 'bg-success-soft text-success',
  warning: 'bg-warning-soft text-warning',
  danger: 'bg-danger-soft text-danger',
  info: 'bg-info-soft text-info',
  outline: 'border border-border text-muted',
}

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant
}

export function Badge({ className, variant = 'default', ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex h-5 items-center rounded-md px-1.5 text-[11px] font-medium whitespace-nowrap',
        variantClasses[variant],
        className,
      )}
      {...props}
    />
  )
}

const dotClasses: Record<BadgeVariant, string> = {
  default: 'bg-subtle',
  success: 'bg-success',
  warning: 'bg-warning',
  danger: 'bg-danger',
  info: 'bg-info',
  outline: 'bg-subtle',
}

export interface StatusDotProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant
  pulse?: boolean
}

export function StatusDot({ className, variant = 'default', pulse = false, ...props }: StatusDotProps) {
  return (
    <span className={cn('relative inline-flex size-2 shrink-0', className)} {...props}>
      {pulse ? (
        <span
          className={cn(
            'absolute inline-flex h-full w-full animate-ping rounded-full opacity-60',
            dotClasses[variant],
          )}
        />
      ) : null}
      <span className={cn('relative inline-flex size-2 rounded-full', dotClasses[variant])} />
    </span>
  )
}
