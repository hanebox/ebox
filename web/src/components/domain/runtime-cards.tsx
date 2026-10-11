import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { IconBox, IconContainer, IconDatabase, IconNetwork, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'
import { Link } from '@tanstack/react-router'

import { api } from '@/api/endpoints'
import type { ContainerInfo } from '@/api/types'
import { Badge, StatusDot } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatBytes, formatPercent } from '@/lib/format'

const resourceIcons: Record<string, typeof IconContainer> = {
  application: IconBox,
  database: IconDatabase,
  service: IconContainer,
  proxy: IconNetwork,
}

const resourceLabels: Record<string, string> = {
  application: 'application',
  database: 'database',
  service: 'service',
  proxy: 'proxy',
}

function containerLink(container: ContainerInfo) {
  const resourceType = container.labels?.['ebox.resource-type'] ?? ''
  const resourceId = container.labels?.['ebox.resource-id'] ?? ''

  if (resourceType === 'application' && resourceId) {
    return { to: '/applications/$applicationId' as const, params: { applicationId: resourceId } }
  }
  if (resourceType === 'proxy') {
    return { to: '/proxy' as const, params: {} }
  }

  return null
}

export function ResourcesCard() {
  const containers = useQuery({
    queryKey: ['containers'],
    queryFn: api.containers,
    refetchInterval: 5000,
  })

  const list = containers.data ?? []

  return (
    <Card className="p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <CardTitle>Resources</CardTitle>
          <CardDescription>Managed containers with live CPU and memory usage.</CardDescription>
        </div>
        <Badge variant="outline">{list.length}</Badge>
      </div>

      {containers.isPending ? (
        <div className="mt-3 flex flex-col gap-2">
          <Skeleton className="h-9" />
          <Skeleton className="h-9" />
        </div>
      ) : null}

      {containers.isSuccess && list.length === 0 ? (
        <p className="hatch mt-3 rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
          Nothing deployed yet. Applications and services you create appear here.
        </p>
      ) : null}

      {list.length > 0 ? (
        <div className="mt-3 flex flex-col divide-y divide-border">
          {list.map((container) => {
            const resourceType = container.labels?.['ebox.resource-type'] ?? 'container'
            const Icon = resourceIcons[resourceType] ?? IconContainer
            const link = containerLink(container)
            const name = container.labels?.['ebox.name'] ?? container.name

            return (
              <div key={container.id} className="flex items-center justify-between gap-3 py-2.5">
                <div className="flex min-w-0 items-center gap-2.5">
                  <span className="flex size-7 shrink-0 items-center justify-center rounded-tile border border-border bg-surface-raised text-muted">
                    <Icon aria-hidden className="size-3.5" />
                  </span>
                  <div className="flex min-w-0 flex-col gap-0.5">
                    {link ? (
                      <Link
                        to={link.to}
                        params={link.params}
                        className="truncate text-ui text-foreground hover:underline"
                      >
                        {name}
                      </Link>
                    ) : (
                      <span className="truncate text-ui text-foreground">{name}</span>
                    )}
                    <span className="truncate text-2xs text-subtle">
                      {resourceLabels[resourceType] ?? resourceType} · {container.image}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-3">
                  {container.stats ? (
                    <div className="hidden items-center gap-3 text-2xs text-muted sm:flex">
                      <span className="tabular">{formatPercent(container.stats.cpu_percent)}</span>
                      <span className="tabular">{formatBytes(container.stats.memory_usage)}</span>
                    </div>
                  ) : null}
                  <Badge
                    variant={container.state === 'running' ? 'success' : 'default'}
                    className="gap-1.5"
                  >
                    {container.state === 'running' ? <StatusDot variant="success" /> : null}
                    {container.state}
                  </Badge>
                </div>
              </div>
            )
          })}
        </div>
      ) : null}
    </Card>
  )
}

export function DiskUsageCard() {
  const queryClient = useQueryClient()
  const [pruneOpen, setPruneOpen] = useState(false)
  const [targets, setTargets] = useState({
    images: true,
    build_cache: true,
    containers: false,
    volumes: false,
  })

  const usage = useQuery({
    queryKey: ['system-disk'],
    queryFn: api.diskUsage,
    refetchInterval: 30_000,
  })

  const prune = useMutation({
    mutationFn: () =>
      api.prune(
        Object.entries(targets)
          .filter(([, enabled]) => enabled)
          .map(([target]) => target),
      ),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['system-disk'] })
      void queryClient.invalidateQueries({ queryKey: ['containers'] })
      toast.success(`Reclaimed ${formatBytes(result.space_reclaimed)}`, 'Prune finished.')
      setPruneOpen(false)
    },
    onError: (error) =>
      toast.error('Prune failed', error instanceof ApiError ? error.message : undefined),
  })

  const data = usage.data

  return (
    <Card className="p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <CardTitle>Disk usage</CardTitle>
          <CardDescription>Images, containers, volumes and build cache.</CardDescription>
        </div>
        <Button variant="ghost" size="sm" onClick={() => setPruneOpen(true)}>
          <IconTrash aria-hidden className="size-3.5" />
          Prune
        </Button>
      </div>

      {usage.isPending ? (
        <div className="mt-3 flex flex-col gap-2">
          <Skeleton className="h-6" />
          <Skeleton className="h-6" />
        </div>
      ) : null}

      {usage.isError ? (
        <p className="mt-3 rounded-control border border-danger/40 bg-danger-soft p-2 text-2xs text-danger">
          Docker is not reachable.
        </p>
      ) : null}

      {data ? (
        <dl className="mt-3 flex flex-col gap-2">
          {[
            { label: 'Images', key: 'images' as const },
            { label: 'Containers', key: 'containers' as const },
            { label: 'Volumes', key: 'volumes' as const },
            { label: 'Build cache', key: 'build_cache' as const },
          ].map((row) => (
            <div key={row.key} className="flex items-center justify-between gap-3">
              <dt className="flex items-center gap-2 text-xs text-subtle">
                {row.label}
                <span className="text-subtle/70">{data[row.key].count} items</span>
              </dt>
              <dd className="tabular flex items-center gap-3 text-xs text-foreground">
                <span>{formatBytes(data[row.key].size)}</span>
                {data[row.key].reclaimable > 0 ? (
                  <span className="text-warning">
                    {formatBytes(data[row.key].reclaimable)} reclaimable
                  </span>
                ) : null}
              </dd>
            </div>
          ))}
          <div className="flex items-center justify-between gap-3 border-t border-border pt-2">
            <dt className="text-xs font-medium text-foreground">Total</dt>
            <dd className="tabular text-xs text-foreground">{formatBytes(data.total_size)}</dd>
          </div>
        </dl>
      ) : null}

      <ConfirmDialog
        open={pruneOpen}
        onOpenChange={setPruneOpen}
        title="Prune unused Docker data"
        description="ebox managed containers, images in use and the ebox image namespace are never removed."
        confirmLabel="Run prune"
        loading={prune.isPending}
        onConfirm={() => prune.mutate()}
      >
        <div className="flex flex-col gap-3">
          {[
            { key: 'images' as const, label: 'Unused images' },
            { key: 'build_cache' as const, label: 'Build cache' },
            { key: 'containers' as const, label: 'Stopped containers (unmanaged)' },
            { key: 'volumes' as const, label: 'Unused volumes' },
          ].map((row) => (
            <div key={row.key} className="flex items-center justify-between gap-3">
              <label className="text-xs text-muted" htmlFor={`prune-${row.key}`}>
                {row.label}
              </label>
              <Switch
                id={`prune-${row.key}`}
                checked={targets[row.key]}
                onCheckedChange={(checked) => setTargets({ ...targets, [row.key]: checked })}
              />
            </div>
          ))}
        </div>
      </ConfirmDialog>
    </Card>
  )
}
