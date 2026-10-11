import type { ReactNode } from 'react'

import { cn } from '@/lib/cn'

export interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: string
  action?: ReactNode
  className?: string
}

export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 rounded-panel border border-border px-6 py-10 text-center',
        className,
      )}
    >
      {icon ? (
        <div className="flex size-9 items-center justify-center rounded-tile border border-border bg-surface-raised text-muted [&_svg]:size-4">
          {icon}
        </div>
      ) : null}
      <div className="flex flex-col gap-1">
        <p className="text-ui font-medium text-balance text-foreground">{title}</p>
        {description ? (
          <p className="max-w-sm text-xs leading-relaxed text-pretty text-muted">{description}</p>
        ) : null}
      </div>
      {action ? <div className="mt-1">{action}</div> : null}
    </div>
  )
}
