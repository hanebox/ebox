import { StatusDot } from '@/components/ui/badge'
import { Tooltip } from '@/components/ui/tooltip'
import { useStreamStatus } from '@/hooks/use-event-stream'
import type { StreamStatus } from '@/lib/events'

const labels: Record<StreamStatus, { text: string; variant: 'success' | 'warning' | 'danger' }> = {
  open: { text: 'Live', variant: 'success' },
  connecting: { text: 'Connecting…', variant: 'warning' },
  closed: { text: 'Offline', variant: 'danger' },
}

export function StreamIndicator() {
  const status = useStreamStatus()
  const meta = labels[status]

  return (
    <Tooltip content="Realtime event stream (SSE)">
      <span className="inline-flex h-8.5 items-center gap-2 rounded-lg border border-border px-2.5 text-xs text-muted">
        <StatusDot variant={meta.variant} pulse={status !== 'open'} />
        {meta.text}
      </span>
    </Tooltip>
  )
}
