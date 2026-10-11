import { IconCheck, IconCopy } from '@tabler/icons-react'
import { useCallback, useRef, useState } from 'react'

import { cn } from '@/lib/cn'

export interface CopyableProps {
  value: string
  display?: string
  className?: string
  mono?: boolean
}

export function Copyable({ value, display, className, mono = true }: CopyableProps) {
  const [copied, setCopied] = useState(false)
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null)

  const copy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      if (timer.current) {
        clearTimeout(timer.current)
      }
      timer.current = setTimeout(() => setCopied(false), 1500)
    } catch {
      setCopied(false)
    }
  }, [value])

  return (
    <button
      type="button"
      onClick={copy}
      title="Copy to clipboard"
      className={cn(
        'hit-area group inline-flex max-w-full items-center gap-2 rounded-tile bg-surface-raised px-2 py-1 text-start text-xs shadow-border',
        'transition-[background-color,box-shadow] duration-150 hover:bg-surface-overlay hover:shadow-border-hover',
        className,
      )}
    >
      <span className={cn('truncate', mono && 'font-mono')}>{display ?? value}</span>
      <span className="relative inline-flex size-3 shrink-0 items-center justify-center">
        <IconCheck
          aria-hidden
          className={cn(
            'absolute inset-0 size-3 text-success transition-[opacity,filter,scale] duration-300 ease-icon',
            copied ? 'scale-100 opacity-100 blur-0' : 'scale-[0.25] opacity-0 blur-[4px]',
          )}
        />
        <IconCopy
          aria-hidden
          className={cn(
            'size-3 text-subtle transition-[opacity,filter,scale] duration-300 ease-icon group-hoverable:text-muted',
            copied ? 'scale-[0.25] opacity-0 blur-[4px]' : 'scale-100 opacity-100 blur-0',
          )}
        />
      </span>
      <span className="sr-only">{copied ? 'Copied' : 'Copy'}</span>
    </button>
  )
}
