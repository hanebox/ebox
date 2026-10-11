import { useId } from 'react'

import { cn } from '@/lib/cn'

export interface SparklineProps {
  data: number[]
  className?: string
  height?: number
  width?: number
  color?: string
  fill?: boolean
}

export function Sparkline({
  data,
  className,
  height = 36,
  width = 140,
  color = 'var(--accent)',
  fill = true,
}: SparklineProps) {
  const id = useId()

  if (data.length < 2) {
    return <div className={cn('rounded-md bg-surface-raised/60', className)} style={{ height }} />
  }

  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1
  const padding = 2
  const step = width / (data.length - 1)

  const points = data.map((value, index) => {
    const x = index * step
    const y = height - padding - ((value - min) / range) * (height - padding * 2)

    return `${x.toFixed(2)},${y.toFixed(2)}`
  })

  const line = `M${points.join(' L')}`
  const area = `${line} L${width},${height} L0,${height} Z`

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      preserveAspectRatio="none"
      className={cn('w-full', className)}
      style={{ height }}
      aria-hidden
    >
      {fill ? (
        <>
          <defs>
            <linearGradient id={`spark-${id}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity="0.25" />
              <stop offset="100%" stopColor={color} stopOpacity="0" />
            </linearGradient>
          </defs>
          <path d={area} fill={`url(#spark-${id})`} />
        </>
      ) : null}
      <path d={line} fill="none" stroke={color} strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  )
}
