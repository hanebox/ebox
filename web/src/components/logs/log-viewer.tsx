import { useVirtualizer } from '@tanstack/react-virtual'
import { AnsiUp } from 'ansi_up'
import { ArrowDownToLine, Download, RotateCw, Search, WrapText } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'

import { resourceLogDownloadURL, resourceLogStreamURL } from '@/api/endpoints'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { cn } from '@/lib/cn'

const maxLines = 20000
const tailOptions = [100, 500, 1000, 5000]

export type LogResource = 'applications' | 'databases' | 'services'

export interface LogSource {
  id: string
  label: string
}

export interface LogViewerProps {
  resource: LogResource
  uuid: string
  sources?: LogSource[]
  className?: string
  height?: string
}

export function LogViewer({ resource, uuid, sources, className, height = 'h-[30rem]' }: LogViewerProps) {
  const [container, setContainer] = useState(sources?.[0]?.id ?? '')
  const [lines, setLines] = useState<string[]>([])
  const [tail, setTail] = useState(500)
  const [search, setSearch] = useState('')
  const [follow, setFollow] = useState(true)
  const [wrap, setWrap] = useState(true)
  const [status, setStatus] = useState<'connecting' | 'open' | 'closed'>('connecting')
  const [attempt, setAttempt] = useState(0)
  const viewportRef = useRef<HTMLDivElement | null>(null)

  const ansi = useMemo(() => {
    const instance = new AnsiUp()
    instance.escape_html = true

    return instance
  }, [])

  useEffect(() => {
    setLines([])
    setStatus('connecting')

    const source = new EventSource(resourceLogStreamURL(resource, uuid, tail, container || undefined), {
      withCredentials: true,
    })

    source.onopen = () => setStatus('open')
    source.onerror = () => {
      setStatus('closed')
      source.close()
    }

    source.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data) as { line: string }
        setLines((previous) => {
          const next = [...previous, payload.line]

          return next.length > maxLines ? next.slice(-maxLines) : next
        })
      } catch {
        // ignore malformed payloads
      }
    }

    return () => source.close()
  }, [resource, uuid, tail, container, attempt])

  const filtered = useMemo(() => {
    if (!search.trim()) {
      return lines.map((line, index) => ({ line, index }))
    }

    const needle = search.trim().toLowerCase()

    return lines
      .map((line, index) => ({ line, index }))
      .filter((entry) => stripAnsi(entry.line).toLowerCase().includes(needle))
  }, [lines, search])

  const virtualizer = useVirtualizer({
    count: filtered.length,
    getScrollElement: () => viewportRef.current,
    estimateSize: () => 20,
    overscan: 24,
  })

  useEffect(() => {
    if (!follow || filtered.length === 0) {
      return
    }

    virtualizer.scrollToIndex(filtered.length - 1, { align: 'end' })
  }, [filtered.length, follow, virtualizer])

  return (
    <div className={cn('flex flex-col overflow-hidden rounded-lg border border-border bg-[#0b0b12]', className)}>
      <div className="flex flex-wrap items-center gap-2 border-b border-border bg-surface-raised px-3 py-1.5">
        <div className="relative min-w-40 flex-1">
          <Search
            aria-hidden
            className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-subtle"
          />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search lines…"
            className="h-7 pl-8 text-xs"
          />
        </div>
        {sources && sources.length > 1 ? (
          <Select
            value={container}
            onChange={(event) => setContainer(event.target.value)}
            className="h-7 w-40 text-xs"
            aria-label="Log source"
          >
            {sources.map((source) => (
              <option key={source.id} value={source.id}>
                {source.label}
              </option>
            ))}
          </Select>
        ) : null}
        <Select
          value={String(tail)}
          onChange={(event) => setTail(Number(event.target.value))}
          className="h-7 w-28 text-xs"
          aria-label="Lines to load"
        >
          {tailOptions.map((option) => (
            <option key={option} value={option}>
              tail {option}
            </option>
          ))}
        </Select>
        <Button variant="ghost" size="sm" onClick={() => setFollow((value) => !value)}>
          <ArrowDownToLine aria-hidden className="size-3.5" />
          {follow ? 'Following' : 'Follow'}
        </Button>
        <Button variant="ghost" size="sm" onClick={() => setWrap((value) => !value)}>
          <WrapText aria-hidden className="size-3.5" />
          {wrap ? 'Wrapped' : 'Wrap'}
        </Button>
        <Button variant="ghost" size="sm" onClick={() => setAttempt((value) => value + 1)}>
          <RotateCw aria-hidden className="size-3.5" />
        </Button>
        <a href={resourceLogDownloadURL(resource, uuid, 5000, container || undefined)} download>
          <Button variant="ghost" size="sm">
            <Download aria-hidden className="size-3.5" />
            Download
          </Button>
        </a>
        <span className="text-[11px] text-subtle">
          {filtered.length}/{lines.length}
        </span>
        <span
          className={cn(
            'size-1.5 rounded-full',
            status === 'open' ? 'bg-success' : status === 'connecting' ? 'bg-info' : 'bg-danger',
          )}
          aria-label={status}
        />
      </div>

      <div
        ref={viewportRef}
        className={cn('overflow-auto px-3 py-2', height)}
        onScroll={(event) => {
          const element = event.currentTarget
          const atBottom = element.scrollHeight - element.scrollTop - element.clientHeight < 40
          if (!atBottom && follow) {
            setFollow(false)
          }
        }}
      >
        {filtered.length === 0 ? (
          <p className="py-8 text-center text-xs text-subtle">
            {lines.length === 0 ? 'Waiting for output…' : 'No lines match the filter.'}
          </p>
        ) : (
          <div style={{ height: `${virtualizer.getTotalSize()}px`, position: 'relative', width: '100%' }}>
            {virtualizer.getVirtualItems().map((item) => {
              const entry = filtered[item.index]

              return (
                <div
                  key={item.key}
                  className="flex gap-3 font-mono text-[11px] leading-5"
                  style={{
                    position: 'absolute',
                    top: 0,
                    left: 0,
                    width: '100%',
                    transform: `translateY(${item.start}px)`,
                  }}
                >
                  <span className="w-10 shrink-0 text-right text-[10px] text-subtle/60 select-none">
                    {entry.index + 1}
                  </span>
                  <span
                    className={cn('min-w-0 flex-1 text-[#c8c8d8]', wrap ? 'whitespace-pre-wrap break-all' : 'whitespace-pre')}
                    dangerouslySetInnerHTML={{ __html: ansi.ansi_to_html(entry.line) }}
                  />
                </div>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

const ansiPattern = new RegExp(`${String.fromCharCode(27)}\\[[0-9;]*[A-Za-z]`, 'g')

function stripAnsi(value: string): string {
  return value.replace(ansiPattern, '')
}
