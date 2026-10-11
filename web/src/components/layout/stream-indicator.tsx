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
      <span className="inline-flex h-7 items-center gap-2 rounded-full bg-surface px-2.5 text-2xs text-muted shadow-border">
        <StatusDot variant={meta.variant} pulse={status !== 'open'} />
        {meta.text}
      </span>
    </Tooltip>
  )
}
