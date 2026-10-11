import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AnsiUp } from 'ansi_up'
import { ArrowDownToLine, Eraser } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import { api } from '@/api/endpoints'
import type { DeploymentStatus } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/skeleton'
import { useEventStream } from '@/hooks/use-event-stream'
import { cn } from '@/lib/cn'

const maxRenderedLines = 4000

export interface DeploymentLogProps {
  deploymentId: string
  status: DeploymentStatus
  onStatusChange?: (status: DeploymentStatus) => void
  className?: string
}

interface LiveState {
  count: number
  lines: string[]
}

export function DeploymentLog({ deploymentId, status, onStatusChange, className }: DeploymentLogProps) {
  const queryClient = useQueryClient()
  const [live, setLive] = useState<LiveState>({ count: 0, lines: [] })
  const [follow, setFollow] = useState(true)
  const viewportRef = useRef<HTMLDivElement | null>(null)
  const snapshotSeqRef = useRef(0)
  const loadingGapRef = useRef(false)

  const ansi = useMemo(() => {
    const instance = new AnsiUp()
    instance.escape_html = true

    return instance
  }, [])

  const snapshot = useQuery({
    queryKey: ['deployment-log', deploymentId],
    queryFn: () => api.deploymentLogs(deploymentId, 0),
    refetchOnWindowFocus: false,
    staleTime: Infinity,
  })

  const snapshotLines = useMemo(() => snapshot.data?.lines ?? [], [snapshot.data])

  useEffect(() => {
    snapshotSeqRef.current = snapshot.data?.seq ?? 0
  }, [snapshot.data])

  const loadGap = useCallback(
    async (since: number) => {
      if (loadingGapRef.current) {
        return
      }
      loadingGapRef.current = true

      try {
        const page = await api.deploymentLogs(deploymentId, since)
        setLive((previous) => ({
          count: previous.count,
          lines: [...previous.lines, ...(page.lines ?? [])].slice(-maxRenderedLines),
        }))
      } catch {
        // The next event will trigger another attempt.
      } finally {
        loadingGapRef.current = false
      }
    },
    [deploymentId],
  )

  useEventStream((event) => {
    if (event.topic !== `deployment.${deploymentId}`) {
      return
    }

    if (event.type === 'log') {
      const data = event.data as { seq: number; line: string }
      const snapshotSeq = snapshotSeqRef.current

      if (data.seq < snapshotSeq) {
        return
      }

      setLive((previous) => {
        const expected = snapshotSeq + previous.lines.length
        if (data.seq < expected) {
          return previous
        }
        if (data.seq > expected) {
          void loadGap(expected)

          return previous
        }

        return { count: previous.count + 1, lines: [...previous.lines, data.line].slice(-maxRenderedLines) }
      })

      return
    }

    if (event.type === 'status') {
      const data = event.data as { status: DeploymentStatus }
      onStatusChange?.(data.status)
      void queryClient.invalidateQueries({ queryKey: ['deployment', deploymentId] })
      void queryClient.invalidateQueries({ queryKey: ['deployments'] })
      void queryClient.invalidateQueries({ queryKey: ['applications'] })
    }
  })

  const lines = useMemo(() => [...snapshotLines, ...live.lines], [snapshotLines, live.lines])

  useEffect(() => {
    if (!follow || !viewportRef.current) {
      return
    }

    viewportRef.current.scrollTop = viewportRef.current.scrollHeight
  }, [lines.length, follow])

  const visible = useMemo(
    () => (lines.length > maxRenderedLines ? lines.slice(-maxRenderedLines) : lines),
    [lines],
  )

  const html = useMemo(
    () => visible.map((line) => ansi.ansi_to_html(line)).join('\n'),
    [visible, ansi],
  )

  const running = status === 'queued' || status === 'building' || status === 'deploying'

  return (
    <div className={cn('flex flex-col overflow-hidden rounded-lg border border-border bg-[#0b0b12]', className)}>
      <div className="flex items-center justify-between gap-2 border-b border-border bg-surface-raised px-3 py-1.5">
        <div className="flex items-center gap-2 text-[11px] text-subtle">
          {running ? <Spinner className="[&_svg]:size-3" /> : null}
          {visible.length} lines
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="sm" onClick={() => setFollow((value) => !value)}>
            <ArrowDownToLine aria-hidden className="size-3.5" />
            {follow ? 'Following' : 'Follow'}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setLive({ count: 0, lines: [] })}
            disabled={live.lines.length === 0}
          >
            <Eraser aria-hidden className="size-3.5" />
            Clear live
          </Button>
        </div>
      </div>

      <div
        ref={viewportRef}
        className="h-[26rem] overflow-auto px-3 py-2"
        onScroll={(event) => {
          const element = event.currentTarget
          const atBottom = element.scrollHeight - element.scrollTop - element.clientHeight < 40
          if (!atBottom && follow) {
            setFollow(false)
          }
        }}
      >
        {snapshot.isPending ? (
          <p className="py-6 text-center text-xs text-subtle">Loading log…</p>
        ) : null}
        {!snapshot.isPending && visible.length === 0 ? (
          <p className="py-6 text-center text-xs text-subtle">No output yet.</p>
        ) : null}
        <pre
          className="font-mono text-[11px] leading-5 whitespace-pre-wrap text-[#c8c8d8]"
          dangerouslySetInnerHTML={{ __html: html }}
        />
      </div>
    </div>
  )
}
