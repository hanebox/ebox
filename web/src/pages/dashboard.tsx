import { useMemo, useState } from 'react'
import { RefreshCw } from 'lucide-react'

import { AreaChart } from '@/components/charts/area-chart'
import { DiskUsageCard, ResourcesCard } from '@/components/domain/runtime-cards'
import { MetricCard } from '@/components/domain/metric-card'
import { PageHeader } from '@/components/layout/page-header'
import { Badge, StatusDot } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTab } from '@/components/ui/tabs'
import { Tooltip } from '@/components/ui/tooltip'
import { useHostMetrics, useSystemInfo } from '@/hooks/use-metrics'
import { formatBytes, formatPercent, formatRate, formatUptime } from '@/lib/format'

type MetricKey = 'cpu' | 'memory' | 'disk' | 'load'

const metricConfig: Record<
  MetricKey,
  { label: string; color: string; format: (value: number) => string; unit: string }
> = {
  cpu: { label: 'CPU', color: 'var(--accent)', format: (v) => formatPercent(v), unit: '%' },
  memory: { label: 'Memory', color: 'var(--info)', format: (v) => formatPercent(v), unit: '%' },
  disk: { label: 'Disk', color: 'var(--warning)', format: (v) => formatPercent(v), unit: '%' },
  load: { label: 'Load', color: 'var(--success)', format: (v) => v.toFixed(2), unit: '' },
}

export function DashboardPage() {
  const [metric, setMetric] = useState<MetricKey>('cpu')
  const [range, setRange] = useState<'20m' | '24h'>('20m')

  const metrics = useHostMetrics(range)
  const system = useSystemInfo()

  const sample = metrics.data?.current
  const history = useMemo(() => metrics.data?.history ?? [], [metrics.data])

  const series = useMemo(() => {
    switch (metric) {
      case 'cpu':
        return history.map((point) => ({ at: point.at, value: point.cpu_percent }))
      case 'memory':
        return history.map((point) => ({ at: point.at, value: point.mem_percent }))
      case 'disk':
        return history.map((point) => ({ at: point.at, value: point.disk_percent }))
      case 'load':
        return history.map((point) => ({ at: point.at, value: point.load1 }))
    }
  }, [history, metric])

  const config = metricConfig[metric]

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Dashboard"
        description="Live host and runtime overview."
        actions={
          <Tooltip content="Refetch metrics now">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => {
                void metrics.refetch()
                void system.refetch()
              }}
              aria-label="Refresh"
            >
              <RefreshCw aria-hidden className="size-4" />
            </Button>
          </Tooltip>
        }
      />

      {!sample ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <Skeleton key={index} className="h-32" />
          ))}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            label="CPU"
            value={formatPercent(sample.cpu_percent)}
            sub={`Load ${sample.load1.toFixed(2)}`}
            data={history.map((point) => point.cpu_percent)}
            color="var(--accent)"
          />
          <MetricCard
            label="Memory"
            value={formatPercent(sample.mem_percent)}
            sub={`${formatBytes(sample.mem_used)} / ${formatBytes(sample.mem_total)}`}
            data={history.map((point) => point.mem_percent)}
            color="var(--info)"
          />
          <MetricCard
            label="Disk"
            value={formatPercent(sample.disk_percent)}
            sub={`${formatBytes(sample.disk_used)} / ${formatBytes(sample.disk_total)}`}
            data={history.map((point) => point.disk_percent)}
            color="var(--warning)"
          />
          <MetricCard
            label="Network"
            value={formatRate(sample.net_rx_rate)}
            sub={`↑ ${formatRate(sample.net_tx_rate)}`}
            data={history.map((point) => point.net_rx_rate)}
            color="var(--success)"
          />
        </div>
      )}

      <div className="grid gap-4 lg:grid-cols-3">
        <Card className="flex flex-col lg:col-span-2">
          <div className="flex flex-wrap items-center justify-between gap-3 p-4 pb-2">
            <Tabs value={metric} onValueChange={(value) => setMetric(value as MetricKey)}>
              <TabsList>
                {(Object.keys(metricConfig) as MetricKey[]).map((key) => (
                  <TabsTab key={key} value={key}>
                    {metricConfig[key].label}
                  </TabsTab>
                ))}
              </TabsList>
            </Tabs>

            <div className="flex items-center gap-1 rounded-lg border border-border p-0.5">
              {(['20m', '24h'] as const).map((option) => (
                <button
                  key={option}
                  type="button"
                  onClick={() => setRange(option)}
                  className={
                    option === range
                      ? 'rounded-md bg-surface-raised px-2 py-1 text-xs font-medium text-foreground'
                      : 'rounded-md px-2 py-1 text-xs text-muted transition-colors hover:text-foreground'
                  }
                >
                  {option}
                </button>
              ))}
            </div>
          </div>

          <div className="px-4 pb-4">
            <AreaChart
              data={series}
              color={config.color}
              formatValue={config.format}
              height={180}
              maxOverride={config.unit === '%' ? 100 : undefined}
            />
          </div>
        </Card>

        <Card className="flex flex-col gap-3 p-4">
          <CardTitle>System</CardTitle>
          {system.data ? (
            <dl className="flex flex-col gap-2 text-sm">
              <SystemRow label="Hostname" value={system.data.hostname} />
              <SystemRow
                label="Platform"
                value={`${system.data.platform} ${system.data.platform_version}`}
              />
              <SystemRow label="Kernel" value={system.data.kernel_version} />
              <SystemRow label="Architecture" value={system.data.arch} />
              <SystemRow label="CPUs" value={String(system.data.cpus)} />
              <SystemRow label="Memory" value={formatBytes(system.data.mem_total)} />
              <SystemRow label="Uptime" value={formatUptime(system.data.uptime)} />
              <SystemRow label="Go" value={system.data.go_version} />
              <SystemRow label="Panel" value={system.data.panel_version} />
              <div className="flex items-center justify-between gap-3">
                <dt className="text-xs text-subtle">Docker</dt>
                <dd>
                  {system.data.docker.available ? (
                    <Badge variant="success">
                      <StatusDot variant="success" className="mr-1.5" />
                      {system.data.docker.version}
                    </Badge>
                  ) : (
                    <Tooltip
                      content={system.data.docker.error ?? 'Docker is not available on this host'}
                    >
                      <Badge variant="danger">unavailable</Badge>
                    </Tooltip>
                  )}
                </dd>
              </div>
            </dl>
          ) : (
            <div className="flex flex-col gap-2">
              {Array.from({ length: 6 }).map((_, index) => (
                <Skeleton key={index} className="h-5" />
              ))}
            </div>
          )}
        </Card>
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <ResourcesCard />
        <DiskUsageCard />
      </div>

      <Card className="p-4">
        <CardTitle>Roadmap</CardTitle>
        <CardDescription className="mt-1">
          The panel is being built in milestones. What is live today and what is next:
        </CardDescription>
        <ul className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
          <RoadmapRow
            milestone="M0"
            done
            title="Foundation"
            description="Auth, sessions, live host metrics, event stream, settings."
          />
          <RoadmapRow
            milestone="M1"
            done
            title="Projects & Sources"
            description="Projects, GitHub App sources, repository and branch pickers."
          />
          <RoadmapRow
            milestone="M2"
            done
            title="Applications & Deployments"
            description="Dockerfile / Static / Image deploys with live build logs."
          />
          <RoadmapRow
            milestone="M3"
            done
            title="Terminal, Logs & Proxy"
            description="Browser terminals, container logs, Traefik routes and certificates."
          />
          <RoadmapRow
            milestone="M4"
            done
            title="Databases & Services"
            description="One-click Postgres / MySQL / MariaDB / Redis / Mongo and compose services."
          />
          <RoadmapRow
            milestone="M5"
            done
            title="Ops polish"
            description="Notifications, scheduled backups, installer and self-update."
          />
        </ul>
      </Card>
    </div>
  )
}

function SystemRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="text-xs text-subtle">{label}</dt>
      <dd className="truncate font-mono text-xs text-foreground">{value}</dd>
    </div>
  )
}

function RoadmapRow({
  milestone,
  title,
  description,
  done = false,
}: {
  milestone: string
  title: string
  description: string
  done?: boolean
}) {
  return (
    <li className="flex items-start gap-3 rounded-lg border border-border p-3">
      <Badge variant={done ? 'success' : 'outline'}>{milestone}</Badge>
      <div className="flex min-w-0 flex-col">
        <span className="text-sm font-medium text-foreground">{title}</span>
        <span className="text-xs text-muted">{description}</span>
      </div>
    </li>
  )
}
