import { useQuery } from '@tanstack/react-query'
import { Container, RotateCw, Server } from 'lucide-react'
import { Suspense, lazy, useState } from 'react'

import { api } from '@/api/endpoints'
import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/cn'

const TerminalView = lazy(() =>
  import('@/components/terminal/terminal-view').then((module) => ({ default: module.TerminalView })),
)

export function TerminalPage() {
  const [target, setTarget] = useState('host')

  const targets = useQuery({
    queryKey: ['terminal-targets'],
    queryFn: api.terminalTargets,
    refetchInterval: 15_000,
  })

  const host = targets.data?.find((item) => item.kind === 'host') ?? null
  const containers = targets.data?.filter((item) => item.kind === 'container') ?? []

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Terminal"
        description="Shells into the host and every container ebox manages."
        actions={
          <Button variant="secondary" size="sm" loading={targets.isFetching} onClick={() => void targets.refetch()}>
            <RotateCw aria-hidden className="size-3.5" />
            Refresh targets
          </Button>
        }
      />

      <div className="grid gap-4 lg:grid-cols-[18rem_1fr]">
        <div className="flex flex-col gap-3">
          {targets.isPending ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-10" />
              <Skeleton className="h-10" />
            </div>
          ) : null}

          {host ? (
            <button
              type="button"
              onClick={() => setTarget(host.id)}
              className={cn(
                'flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-left transition-colors',
                target === host.id ? 'border-accent bg-accent-soft' : 'hover:bg-surface-raised',
              )}
            >
              <Server aria-hidden className="size-4 text-muted" />
              <span className="flex min-w-0 flex-col">
                <span className="text-xs text-foreground">{host.name}</span>
                <span className="text-[11px] text-subtle">host shell</span>
              </span>
            </button>
          ) : null}

          <div className="flex items-center justify-between px-1">
            <span className="text-[11px] tracking-wide text-subtle uppercase">Containers</span>
            <Badge variant="outline">{containers.length}</Badge>
          </div>

          <div className="flex max-h-[26rem] flex-col gap-2 overflow-auto pr-1">
            {containers.length === 0 ? (
              <p className="rounded-lg border border-dashed border-border px-3 py-4 text-center text-[11px] text-subtle">
                No containers yet. Deploy an application first.
              </p>
            ) : null}

            {containers.map((item) => (
              <button
                type="button"
                key={item.id}
                onClick={() => setTarget(item.id)}
                className={cn(
                  'flex items-center justify-between gap-2 rounded-lg border border-border px-3 py-2 text-left transition-colors',
                  target === item.id ? 'border-accent bg-accent-soft' : 'hover:bg-surface-raised',
                )}
              >
                <span className="flex min-w-0 items-center gap-2">
                  <Container aria-hidden className="size-4 shrink-0 text-muted" />
                  <span className="flex min-w-0 flex-col">
                    <span className="truncate text-xs text-foreground">{item.name}</span>
                    <span className="truncate text-[11px] text-subtle">{item.image}</span>
                  </span>
                </span>
                <Badge variant={item.state === 'running' ? 'success' : 'default'}>{item.state}</Badge>
              </button>
            ))}
          </div>
        </div>

        <Suspense fallback={<Skeleton className="h-[28rem]" />}>
          <TerminalView target={target} />
        </Suspense>
      </div>
    </div>
  )
}
