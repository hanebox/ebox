import { useId } from 'react'

import { cn } from '@/lib/cn'

export interface LogoMarkProps {
  className?: string
}

export function LogoMark({ className }: LogoMarkProps) {
  const id = useId()
  const gradientId = `${id}-gradient`
  const sheenId = `${id}-sheen`

  return (
    <svg viewBox="0 0 32 32" fill="none" aria-hidden className={cn('shrink-0', className)}>
      <defs>
        <linearGradient id={gradientId} x1="2" y1="1" x2="30" y2="31" gradientUnits="userSpaceOnUse">
          <stop stopColor="#7B6BFF" />
          <stop offset="0.52" stopColor="#A855F7" />
          <stop offset="1" stopColor="#EC4899" />
        </linearGradient>
        <linearGradient id={sheenId} x1="16" y1="1" x2="16" y2="31" gradientUnits="userSpaceOnUse">
          <stop stopColor="#FFFFFF" stopOpacity="0.34" />
          <stop offset="1" stopColor="#FFFFFF" stopOpacity="0" />
        </linearGradient>
      </defs>
      <rect x="1" y="1" width="30" height="30" rx="9" fill={`url(#${gradientId})`} />
      <rect x="1" y="1" width="30" height="30" rx="9" fill={`url(#${sheenId})`} />
      <rect
        x="1.5"
        y="1.5"
        width="29"
        height="29"
        rx="8.5"
        stroke="#FFFFFF"
        strokeOpacity="0.26"
      />
      <g fill="#FFFFFF" stroke="#FFFFFF" strokeWidth="1.2" strokeLinejoin="round">
        <path d="M16 7.9 23.4 12.1 16 16.3 8.6 12.1Z" opacity="0.96" />
        <path d="M23.4 12.1v7.8L16 24.1v-7.8Z" opacity="0.72" />
        <path d="M8.6 12.1v7.8L16 24.1v-7.8Z" opacity="0.48" />
      </g>
    </svg>
  )
}

export interface LogoProps {
  version?: string
  className?: string
}

export function Logo({ version, className }: LogoProps) {
  return (
    <span className={cn('flex items-center gap-2.5', className)}>
      <LogoMark className="size-7" />
      <span className="flex items-baseline gap-1.5">
        <span className="text-base leading-none font-semibold tracking-[-0.02em] text-foreground">
          ebox
        </span>
        {version ? (
          <span className="font-mono text-2xs leading-none text-subtle">{version}</span>
        ) : null}
      </span>
    </span>
  )
}
