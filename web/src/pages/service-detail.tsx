import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { IconArrowLeft, IconPlayerPlay, IconRocket, IconRotateClockwise2, IconSquare, IconStack2, IconTrash } from '@tabler/icons-react'
import { Suspense, lazy, useState } from 'react'

import { api } from '@/api/endpoints'
import { DomainsEditor } from '@/components/applications/domains-editor'
import { EnvVarsEditor } from '@/components/applications/env-vars-editor'
import { LogViewer } from '@/components/logs/log-viewer'
import { PageHeader } from '@/components/layout/page-header'
import { Badge, StatusDot } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardTitle, Section } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsPanel, TabsTab } from '@/components/ui/tabs'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime, formatRelative } from '@/lib/format'

const TerminalView = lazy(() =>
  import('@/components/terminal/terminal-view').then((module) => ({ default: module.TerminalView })),
)

export function ServiceDetailPage() {
  const { serviceId } = useParams({ strict: false }) as { serviceId: string }
  const queryClient = useQueryClient()
  const [tab, setTab] = useState('overview')
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [terminalTarget, setTerminalTarget] = useState('')

  const service = useQuery({
    queryKey: ['service', serviceId],
    queryFn: () => api.service(serviceId),
    refetchInterval: 10_000,
  })

  const containers = useQuery({
    queryKey: ['service', serviceId, 'containers'],
    queryFn: () => api.serviceContainers(serviceId),
    refetchInterval: 10_000,
  })

  const lifecycle = useMutation({
    mutationFn: (action: 'deploy' | 'start' | 'stop' | 'restart') => api.serviceLifecycle(serviceId, action),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['service', serviceId] })
      void queryClient.invalidateQueries({ queryKey: ['services'] })
      void queryClient.invalidateQueries({ queryKey: ['containers'] })
      toast.success(result.message)
    },
    onError: (error) => toast.error('Action failed', error instanceof ApiError ? error.message : undefined),
  })

  const remove = useMutation({
    mutationFn: () => api.deleteService(serviceId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['services'] })
      void queryClient.invalidateQueries({ queryKey: ['containers'] })
      toast.success('Service deleted')
      window.history.back()
    },
    onError: (error) =>
      toast.error('Could not delete the service', error instanceof ApiError ? error.message : undefined),
  })

  if (service.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48" />
      </div>
    )
  }

  if (service.isError || !service.data) {
    return (
      <Card className="p-6">
        <p className="text-sm text-danger">Service not found.</p>
      </Card>
    )
  }

  const record = service.data
  const runtimeState = record.runtime?.state ?? record.status
  const running = runtimeState === 'running' || runtimeState === 'partial'
  const runningContainers = (containers.data ?? []).filter((container) => container.state === 'running')

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <Link
          to="/services"
          className="inline-flex items-center gap-1 text-xs text-subtle transition-colors hover:text-foreground"
        >
          <IconArrowLeft aria-hidden className="size-3.5" />
          {record.project?.name ?? 'Services'}
        </Link>
        <PageHeader
          title={record.name}
          description={
            record.source === 'git'
              ? `${record.repository || 'repository'} · ${record.branch} · ${record.compose_path}`
              : 'inline compose file'
          }
          actions={
            <>
              <Button variant="primary" size="sm" loading={lifecycle.isPending} onClick={() => lifecycle.mutate('deploy')}>
                <IconRocket aria-hidden className="size-3.5" />
                Deploy
              </Button>
              <Button
                variant="secondary"
                size="sm"
                loading={lifecycle.isPending}
                onClick={() => lifecycle.mutate(running ? 'stop' : 'start')}
              >
                {running ? (
                  <>
                    <IconSquare aria-hidden className="size-3.5" />
                    Stop
                  </>
                ) : (
                  <>
                    <IconPlayerPlay aria-hidden className="size-3.5" />
                    Start
                  </>
                )}
              </Button>
              <Button variant="secondary" size="sm" loading={lifecycle.isPending} onClick={() => lifecycle.mutate('restart')}>
                <IconRotateClockwise2 aria-hidden className="size-3.5" />
                Restart
              </Button>
            </>
          }
        />
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Badge
          variant={
            runtimeState === 'running'
              ? 'success'
              : runtimeState === 'partial'
                ? 'warning'
                : runtimeState === 'failed'
                  ? 'danger'
                  : 'outline'
          }
          className="gap-1.5"
        >
          {runtimeState === 'running' ? <StatusDot variant="success" /> : null}
          {runtimeState}
        </Badge>
        <Badge variant="outline">
          <IconStack2 aria-hidden className="me-1 size-3" />
          {record.definitions?.length ?? 0} compose service(s)
        </Badge>
        {record.entry_service ? (
          <Badge variant="info">
            ingress {record.entry_service}:{record.entry_port}
          </Badge>
        ) : null}
        <span className="text-2xs text-subtle">
          {record.last_deployed_at ? `deployed ${formatRelative(record.last_deployed_at)}` : 'never deployed'}
        </span>
      </div>

      {record.last_error ? (
        <p className="rounded-lg border border-danger/40 bg-danger-soft p-3 font-mono text-2xs whitespace-pre-wrap text-danger">
          {record.last_error}
        </p>
      ) : null}

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTab value="overview">Overview</TabsTab>
          <TabsTab value="logs">Logs</TabsTab>
          <TabsTab value="terminal">Terminal</TabsTab>
          <TabsTab value="environment">Environment</TabsTab>
          <TabsTab value="domains">Domains</TabsTab>
          <TabsTab value="settings">Settings</TabsTab>
        </TabsList>

        <TabsPanel value="overview">
          <Section
            title="Containers"
            description="Every compose service in this stack, as docker reports it."
          >
            {runningContainers.length === 0 && (containers.data?.length ?? 0) === 0 ? (
              <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
                The stack is not running. Deploy or start it to see containers.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Compose service</TableHead>
                    <TableHead>Container</TableHead>
                    <TableHead>State</TableHead>
                    <TableHead>Image</TableHead>
                    <TableHead>Ports</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(containers.data ?? []).map((container) => (
                    <TableRow key={container.id}>
                      <TableCell className="font-mono text-xs text-foreground">{container.service}</TableCell>
                      <TableCell className="font-mono text-2xs text-muted">{container.name}</TableCell>
                      <TableCell>
                        <Badge variant={container.state === 'running' ? 'success' : 'outline'}>
                          {container.state}
                          {container.health && container.health !== 'none' ? ` · ${container.health}` : ''}
                        </Badge>
                      </TableCell>
                      <TableCell className="font-mono text-2xs text-muted">{container.image}</TableCell>
                      <TableCell className="font-mono text-2xs text-muted">
                        {(container.ports ?? []).join(', ') || '–'}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Section>

          {(record.definitions?.length ?? 0) > 0 ? (
            <Section title="Compose services" description="Parsed from the compose file.">
              <div className="flex flex-col gap-2">
                {record.definitions?.map((item) => (
                  <div
                    key={item.name}
                    className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border px-3 py-2"
                  >
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs text-foreground">{item.name}</span>
                      {item.name === record.entry_service ? <Badge variant="info">ingress</Badge> : null}
                      {item.build ? <Badge variant="outline">build</Badge> : null}
                    </div>
                    <div className="flex flex-wrap items-center gap-3 text-2xs text-subtle">
                      <span className="font-mono">{item.image || 'built from context'}</span>
                      {item.ports.length > 0 ? <span className="font-mono">ports {item.ports.join(', ')}</span> : null}
                      {item.depends_on.length > 0 ? (
                        <span className="font-mono">after {item.depends_on.join(', ')}</span>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            </Section>
          ) : null}
        </TabsPanel>

        <TabsPanel value="logs">
          <Section title="Logs" description="Pick a compose service to follow its output.">
            {running ? (
              <LogViewer
                resource="services"
                uuid={serviceId}
                sources={(containers.data ?? []).map((container) => ({
                  id: container.service,
                  label: container.service,
                }))}
              />
            ) : (
              <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
                The stack is not running.
              </p>
            )}
          </Section>
        </TabsPanel>

        <TabsPanel value="terminal">
          <Section title="Terminal" description="A shell inside one of the stack containers.">
            {runningContainers.length === 0 ? (
              <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
                The stack is not running.
              </p>
            ) : (
              <div className="flex flex-col gap-3">
                <div className="flex flex-wrap gap-2">
                  {runningContainers.map((container) => (
                    <Button
                      key={container.id}
                      variant={terminalTarget === container.id ? 'primary' : 'secondary'}
                      size="sm"
                      onClick={() => setTerminalTarget(container.id)}
                    >
                      {container.service}
                    </Button>
                  ))}
                </div>
                <Suspense fallback={<Skeleton className="h-[28rem]" />}>
                  <TerminalView target={terminalTarget || runningContainers[0].id} />
                </Suspense>
              </div>
            )}
          </Section>
        </TabsPanel>

        <TabsPanel value="environment">
          <Section
            title="Environment"
            description="Passed to every container and used for compose interpolation."
          >
            <EnvVarsEditor resource="services" uuid={serviceId} />
          </Section>
        </TabsPanel>

        <TabsPanel value="domains">
          <Section
            title="Domains"
            description={`Routed to the ingress service (${record.entry_service || 'auto'}) through the proxy.`}
          >
            <DomainsEditor resource="services" uuid={serviceId} />
          </Section>
        </TabsPanel>

        <TabsPanel value="settings">
          <div className="flex flex-col gap-4">
            <Section title="Deployment" description="Details of the last deployment.">
              <dl className="grid gap-2 text-xs sm:grid-cols-2">
                <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
                  <dt className="text-subtle">Compose project</dt>
                  <dd className="font-mono text-foreground">ebox-{record.uuid.slice(0, 8)}</dd>
                </div>
                <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
                  <dt className="text-subtle">Last deployed</dt>
                  <dd className="text-foreground">
                    {record.last_deployed_at ? formatDateTime(record.last_deployed_at) : 'never'}
                  </dd>
                </div>
                <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
                  <dt className="text-subtle">Source</dt>
                  <dd className="font-mono text-foreground">
                    {record.source === 'git' ? record.source_record?.name ?? 'git' : 'inline'}
                  </dd>
                </div>
                <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2">
                  <dt className="text-subtle">Compose file</dt>
                  <dd className="font-mono text-foreground">
                    {record.source === 'git' ? record.compose_path : 'stored in the panel'}
                  </dd>
                </div>
              </dl>
            </Section>

            {record.source === 'inline' ? (
              <Section title="Compose file" description="Edit the stored stack definition.">
                <pre className="max-h-80 overflow-auto rounded-control border border-border bg-[#0a0a0d] p-3 font-mono text-2xs text-[#c9c9cd]">
                  {record.compose}
                </pre>
              </Section>
            ) : null}

            <Card>
              <div className="flex flex-col gap-1 p-4 pb-2">
                <CardTitle>Danger zone</CardTitle>
                <CardDescription>
                  Deleting brings the stack down, removes its containers and deletes the stored compose file.
                </CardDescription>
              </div>
              <CardContent className="flex justify-end pt-0">
                <Button variant="danger" size="sm" onClick={() => setDeleteOpen(true)}>
                  <IconTrash aria-hidden className="size-3.5" />
                  Delete service
                </Button>
              </CardContent>
            </Card>
          </div>
        </TabsPanel>
      </Tabs>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={`Delete ${record.name}?`}
        description="Every container in the stack is stopped and removed."
        confirmLabel="Delete service"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
    </div>
  )
}
