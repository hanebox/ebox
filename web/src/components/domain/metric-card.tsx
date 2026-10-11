import type { ReactNode } from 'react'

import { Sparkline } from '@/components/charts/sparkline'
import { Card } from '@/components/ui/card'
import { cn } from '@/lib/cn'

export interface MetricCardProps {
  label: string
  value: string
  sub?: ReactNode
  data?: number[]
  color?: string
  className?: string
}

export function MetricCard({ label, value, sub, data, color, className }: MetricCardProps) {
  return (
    <Card className={cn('flex flex-col gap-2.5 overflow-hidden p-4', className)}>
      <p className="text-xs font-medium text-muted">{label}</p>
      <div className="flex flex-col gap-1">
        <p className="tabular text-2xl leading-none font-semibold tracking-[-0.02em] text-foreground">
          {value}
        </p>
        {sub ? <p className="truncate text-2xs text-subtle">{sub}</p> : null}
      </div>
      {data && data.length > 1 ? (
        <div className="-mx-4 -mb-4 mt-1">
          <Sparkline data={data} height={40} color={color} />
        </div>
      ) : (
        <div className="h-2" />
      )}
    </Card>
  )
}
