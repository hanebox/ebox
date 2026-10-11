import { useEffect, useId, useMemo, useRef, useState } from 'react'

import { cn } from '@/lib/cn'
import { formatDateTime } from '@/lib/format'

export interface AreaChartProps {
  data: { at: string; value: number }[]
  className?: string
  height?: number
  color?: string
  formatValue?: (value: number) => string
  maxOverride?: number
}

function useElementWidth<T extends HTMLElement>() {
  const ref = useRef<T | null>(null)
  const [width, setWidth] = useState(0)

  useEffect(() => {
    const element = ref.current
    if (!element) {
      return
    }

    setWidth(Math.round(element.getBoundingClientRect().width))

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0]
      if (entry) {
        setWidth(Math.round(entry.contentRect.width))
      }
    })
    observer.observe(element)

    return () => observer.disconnect()
  }, [])

  return { ref, width }
}

export function AreaChart({
  data,
  className,
  height = 160,
  color = 'var(--accent)',
  formatValue = (value) => value.toFixed(1),
  maxOverride,
}: AreaChartProps) {
  const id = useId()
  const { ref, width } = useElementWidth<HTMLDivElement>()

  const chart = useMemo(() => {
    if (width < 80 || data.length < 2) {
      return null
    }

    const values = data.map((point) => point.value)
    const min = Math.min(...values)
    const rawMax = maxOverride ?? Math.max(...values)
    const max = rawMax === min ? min + 1 : rawMax
    const padding = { top: 10, right: 8, bottom: 22, left: 44 }
    const innerWidth = width - padding.left - padding.right
    const innerHeight = height - padding.top - padding.bottom
    const step = innerWidth / (data.length - 1)

    const points = data.map((point, index) => {
      const x = padding.left + index * step
      const y = padding.top + innerHeight - ((point.value - min) / (max - min)) * innerHeight

      return { x, y }
    })

    const line = `M${points.map((point) => `${point.x.toFixed(2)},${point.y.toFixed(2)}`).join(' L')}`
    const bottom = padding.top + innerHeight
    const area = `${line} L${(padding.left + innerWidth).toFixed(2)},${bottom} L${padding.left},${bottom} Z`

    const gridLines = [0, 0.5, 1].map((ratio) => ({
      y: padding.top + innerHeight * ratio,
      label: formatValue(max - (max - min) * ratio),
    }))

    return {
      line,
      area,
      gridLines,
      padding,
      innerWidth,
      min,
      max,
      first: data[0].at,
      last: data[data.length - 1].at,
    }
  }, [data, formatValue, height, maxOverride, width])

  return (
    <div ref={ref} className={cn('w-full', className)} style={{ height }}>
      {chart ? (
        <svg width={width} height={height} role="img" className="block overflow-visible">
          <defs>
            <linearGradient id={`area-${id}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={color} stopOpacity="0.28" />
              <stop offset="100%" stopColor={color} stopOpacity="0" />
            </linearGradient>
          </defs>

          {chart.gridLines.map((line) => (
            <g key={line.y}>
              <line
                x1={chart.padding.left}
                x2={chart.padding.left + chart.innerWidth}
                y1={line.y}
                y2={line.y}
                stroke="var(--border)"
              />
              <text
                x={chart.padding.left - 6}
                y={line.y + 3}
                textAnchor="end"
                className="tabular fill-[var(--subtle)] text-2xs"
              >
                {line.label}
              </text>
            </g>
          ))}

          <path d={chart.area} fill={`url(#area-${id})`} />
          <path
            d={chart.line}
            fill="none"
            stroke={color}
            strokeWidth="1.5"
            strokeLinejoin="round"
            strokeLinecap="round"
          />

          <text x={chart.padding.left} y={height - 6} className="fill-[var(--subtle)] text-2xs">
            {formatDateTime(chart.first)}
          </text>
          <text
            x={chart.padding.left + chart.innerWidth}
            y={height - 6}
            textAnchor="end"
            className="fill-[var(--subtle)] text-2xs"
          >
            {formatDateTime(chart.last)}
          </text>
        </svg>
      ) : (
        <div className="hatch flex h-full items-center justify-center rounded-control border border-border">
          <span className="rounded-full border border-border bg-background px-3 py-1 text-2xs text-subtle">
            Collecting samples…
          </span>
        </div>
      )}
    </div>
  )
}
