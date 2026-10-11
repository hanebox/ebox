import { IconLoader2 } from '@tabler/icons-react'
import type { HTMLAttributes } from 'react'

import { cn } from '@/lib/cn'

export function Skeleton({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('animate-pulse rounded-tile bg-surface-raised', className)} {...props} />
}

export function Spinner({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div className={cn('inline-flex items-center justify-center', className)} {...props}>
      <IconLoader2 aria-hidden className="size-4 animate-spin text-muted" />
    </div>
  )
}

export function PageLoader({ label = 'Loading…' }: { label?: string }) {
  return (
    <div className="flex min-h-64 flex-col items-center justify-center gap-3 text-muted">
      <IconLoader2 aria-hidden className="size-5 animate-spin" />
      <p className="text-xs">{label}</p>
    </div>
  )
}
