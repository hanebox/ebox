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
    <Card className={cn('flex flex-col gap-3 p-4', className)}>
      <div className="flex items-baseline justify-between gap-2">
        <p className="text-xs font-medium text-muted">{label}</p>
        {sub ? <p className="text-[11px] text-subtle">{sub}</p> : null}
      </div>
      <p className="font-mono text-xl font-semibold tracking-tight tabular-nums text-foreground">
        {value}
      </p>
      {data && data.length > 1 ? (
        <Sparkline data={data} height={34} color={color} />
      ) : (
        <div className="h-[34px]" />
      )}
    </Card>
  )
}
