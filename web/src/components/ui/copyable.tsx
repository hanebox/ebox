import { Check, Copy } from 'lucide-react'
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
        'group inline-flex max-w-full items-center gap-2 rounded-md border border-border bg-surface-raised px-2 py-1 text-left text-xs transition-colors hover:border-border-strong',
        className,
      )}
    >
      <span className={cn('truncate', mono && 'font-mono')}>{display ?? value}</span>
      {copied ? (
        <Check aria-hidden className="size-3 shrink-0 text-success" />
      ) : (
        <Copy
          aria-hidden
          className="size-3 shrink-0 text-subtle transition-colors group-hover:text-muted"
        />
      )}
      <span className="sr-only">{copied ? 'Copied' : 'Copy'}</span>
    </button>
  )
}
