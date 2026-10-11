import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import {
  ArrowLeft,
  CircleStop,
  ExternalLink,
  GitBranch,
  Pencil,
  Play,
  RotateCcw,
  Rocket,
  Trash2,
} from 'lucide-react'
import { Suspense, lazy, useMemo, useState } from 'react'

import { api } from '@/api/endpoints'
import type { Deployment } from '@/api/types'
import { DeploymentStatusBadge } from '@/components/applications/status'
import { DeploymentTable } from '@/components/applications/deployment-table'
import { LogViewer } from '@/components/logs/log-viewer'
import { DomainsEditor } from '@/components/applications/domains-editor'
import { EnvVarsEditor } from '@/components/applications/env-vars-editor'
import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, Section } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Copyable } from '@/components/ui/copyable'
import { GitHubIcon } from '@/components/ui/icons'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsPanel, TabsTab } from '@/components/ui/tabs'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime, formatRelative } from '@/lib/format'

const TerminalView = lazy(() =>
  import('@/components/terminal/terminal-view').then((module) => ({ default: module.TerminalView })),
)

function MetaRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:gap-4">
      <span className="w-36 shrink-0 text-[11px] tracking-wide text-subtle uppercase">{label}</span>
      <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-foreground">{children}</div>
    </div>
  )
}

export function ApplicationDetailPage() {
  const { applicationId } = useParams({ strict: false }) as { applicationId: string }
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState('overview')
  const [deleteOpen, setDeleteOpen] = useState(false)

  const containers = useQuery({
    queryKey: ['application-containers', applicationId],
    queryFn: () => api.containers(),
    refetchInterval: 10_000,
  })

  const composeStack = useQuery({
    queryKey: ['application', applicationId, 'compose'],
    queryFn: () => api.applicationCompose(applicationId),
    retry: false,
  })

  const stackContainers = useMemo(
    () => (containers.data ?? []).filter((container) => container.labels?.['ebox.resource-id'] === applicationId),
    [containers.data, applicationId],
  )

  const logSources = useMemo(() => {
    const list = (containers.data ?? []).filter(
      (container) => container.labels?.['ebox.resource-id'] === applicationId,
    )

    if (list.length <= 1) {
      return undefined
    }

    return list.map((container) => ({
      id: container.labels?.['com.docker.compose.service'] ?? container.id,
      label: container.labels?.['com.docker.compose.service'] ?? container.name,
    }))
  }, [containers.data, applicationId])

  const applicationQuery = useQuery({
    queryKey: ['application', applicationId],
    queryFn: () => api.application(applicationId),
    refetchInterval: (query) => {
      const data = query.state.data
      const active =
        data?.runtime.state === 'unknown' ||
        ['queued', 'building', 'deploying'].includes(data?.runtime.latest_deployment?.status ?? '')

      return active ? 4000 : false
    },
  })

  const deployments = useQuery({
    queryKey: ['deployments', applicationId],
    queryFn: () => api.deployments(applicationId, 25),
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['application', applicationId] })
    void queryClient.invalidateQueries({ queryKey: ['deployments'] })
    void queryClient.invalidateQueries({ queryKey: ['deployments', applicationId] })
    void queryClient.invalidateQueries({ queryKey: ['applications'] })
  }

  const deploy = useMutation({
    mutationFn: () => api.deployApplication(applicationId),
    onSuccess: () => {
      invalidate()
      toast.success('Deployment queued')
      setTab('deployments')
    },
    onError: (error) =>
      toast.error('Could not queue the deployment', error instanceof ApiError ? error.message : undefined),
  })

  const action = useMutation({
    mutationFn: (value: 'start' | 'stop' | 'restart') => api.applicationAction(applicationId, value),
    onSuccess: (result) => {
      invalidate()
      toast.success(result.message)
    },
    onError: (error) =>
      toast.error('Container action failed', error instanceof ApiError ? error.message : undefined),
  })

  const rollback = useMutation({
    mutationFn: (deployment: Deployment) => api.rollbackApplication(applicationId, deployment.uuid),
    onSuccess: () => {
      invalidate()
      toast.success('Rollback queued')
      setTab('deployments')
    },
    onError: (error) =>
      toast.error('Could not queue the rollback', error instanceof ApiError ? error.message : undefined),
  })

  const remove = useMutation({
    mutationFn: () => api.deleteApplication(applicationId),
    onSuccess: () => {
      invalidate()
      toast.success('Application deleted')
      void navigate({ to: '/projects' })
    },
    onError: (error) =>
      toast.error('Could not delete the application', error instanceof ApiError ? error.message : undefined),
  })

  const application = applicationQuery
  const data = applicationQuery.data
  const runtime = data?.runtime
  const state = runtime?.state ?? 'unknown'
  const running = state === 'running'
  const latest = runtime?.latest_deployment

  return (
    <div className="flex flex-col gap-6">
      {data ? (
        <div>
          <Link
            to="/projects/$projectId"
            params={{ projectId: data.project?.uuid ?? '' }}
            className="inline-flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-foreground"
          >
            <ArrowLeft aria-hidden className="size-3.5" />
            {data.project?.name ?? 'Project'}
          </Link>
        </div>
      ) : null}

      {application.isPending ? <Skeleton className="h-20" /> : null}

      {data ? (
        <>
          <PageHeader
            title={data.name}
            description={
              data.kind === 'image'
                ? data.config.image
                : `${data.repository} · ${data.branch}`
            }
            actions={
              <>
                <Button variant="primary" size="sm" loading={deploy.isPending} onClick={() => deploy.mutate()}>
                  <Rocket aria-hidden className="size-3.5" />
                  Deploy
                </Button>
                {running ? (
                  <Button
                    variant="secondary"
                    size="sm"
                    loading={action.isPending}
                    onClick={() => action.mutate('restart')}
                  >
                    <RotateCcw aria-hidden className="size-3.5" />
                    Restart
                  </Button>
                ) : null}
                <Button
                  variant="secondary"
                  size="sm"
                  loading={action.isPending}
                  onClick={() => action.mutate(running ? 'stop' : 'start')}
                >
                  {running ? (
                    <>
                      <CircleStop aria-hidden className="size-3.5" />
                      Stop
                    </>
                  ) : (
                    <>
                      <Play aria-hidden className="size-3.5" />
                      Start
                    </>
                  )}
                </Button>
              </>
            }
          />

          <Tabs value={tab} onValueChange={setTab}>
            <TabsList>
              <TabsTab value="overview">Overview</TabsTab>
              <TabsTab value="deployments">Deployments</TabsTab>
              <TabsTab value="logs">Logs</TabsTab>
              <TabsTab value="terminal">Terminal</TabsTab>
              <TabsTab value="environment">Environment</TabsTab>
              <TabsTab value="domains">Domains</TabsTab>
              <TabsTab value="settings">Settings</TabsTab>
            </TabsList>

            <TabsPanel value="overview">
              <div className="flex flex-col gap-4">
                <Card>
                  <CardContent className="flex flex-col gap-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant={running ? 'success' : 'default'}>{state}</Badge>
                      {runtime?.health && runtime.health !== 'none' ? (
                        <Badge variant={runtime.health === 'healthy' ? 'success' : 'warning'}>
                          {runtime.health}
                        </Badge>
                      ) : null}
                      <Badge variant="outline">{data.build_pack}</Badge>
                      {data.auto_deploy ? <Badge variant="info">auto-deploy</Badge> : null}
                    </div>

                    <MetaRow label="Container">
                      {runtime?.container_name ? (
                        <Copyable value={runtime.container_name} />
                      ) : (
                        <span className="text-subtle">Not deployed yet</span>
                      )}
                    </MetaRow>
                    <MetaRow label="Image">
                      {runtime?.image ? (
                        <Copyable value={runtime.image} />
                      ) : (
                        <span className="text-subtle">—</span>
                      )}
                    </MetaRow>
                    <MetaRow label="Port">{data.config.expose_port}</MetaRow>
                    <MetaRow label="Domains">
                      {data.domains.length === 0 ? (
                        <span className="text-subtle">No domain configured</span>
                      ) : (
                        data.domains.map((domain) => (
                          <Badge key={domain.id} variant={domain.is_primary ? 'info' : 'outline'}>
                            {domain.host}
                          </Badge>
                        ))
                      )}
                    </MetaRow>
                    <MetaRow label="Deploy hook">
                      {data.deploy_url ? (
                        <Copyable value={window.location.origin + data.deploy_url} className="max-w-md" />
                      ) : (
                        <span className="text-subtle">—</span>
                      )}
                    </MetaRow>
                  </CardContent>
                </Card>

                {data.build_pack === 'compose' ? (
                  <Section
                    title="Compose stack"
                    description="Services defined in the repository's compose file."
                    actions={
                      composeStack.data?.entry_service ? (
                        <Badge variant="info">
                          ingress {composeStack.data.entry_service}:{data.config.expose_port}
                        </Badge>
                      ) : null
                    }
                  >
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Service</TableHead>
                          <TableHead>Container</TableHead>
                          <TableHead>State</TableHead>
                          <TableHead>Image</TableHead>
                          <TableHead>Ports</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(composeStack.data?.services ?? []).map((service) => {
                          const container = stackContainers.find(
                            (item) => item.labels?.['com.docker.compose.service'] === service.name,
                          )

                          return (
                            <TableRow key={service.name}>
                              <TableCell className="font-mono text-xs text-foreground">
                                {service.name}
                                {service.name === composeStack.data?.entry_service ? (
                                  <Badge variant="info" className="ml-2">
                                    ingress
                                  </Badge>
                                ) : null}
                              </TableCell>
                              <TableCell className="font-mono text-[11px] text-muted">
                                {container?.name ?? '—'}
                              </TableCell>
                              <TableCell>
                                <Badge variant={container?.state === 'running' ? 'success' : 'outline'}>
                                  {container?.state ?? 'missing'}
                                </Badge>
                              </TableCell>
                              <TableCell className="font-mono text-[11px] text-muted">
                                {container?.image ?? service.image ?? '(build)'}
                              </TableCell>
                              <TableCell className="font-mono text-[11px] text-muted">
                                {service.ports.join(', ') || '—'}
                              </TableCell>
                            </TableRow>
                          )
                        })}
                      </TableBody>
                    </Table>
                  </Section>
                ) : null}

                <Section
                  title="Latest deployment"
                  description="Live output and history are on the deployments tab."
                  actions={
                    latest ? (
                      <Link to="/deployments/$deploymentId" params={{ deploymentId: latest.uuid }}>
                        <Button variant="secondary" size="sm">
                          <ExternalLink aria-hidden className="size-3.5" />
                          Open log
                        </Button>
                      </Link>
                    ) : undefined
                  }
                >
                  {latest ? (
                    <div className="flex flex-wrap items-center gap-3 text-xs text-muted">
                      <DeploymentStatusBadge status={latest.status} />
                      <span className="font-mono">{latest.commit ? latest.commit.slice(0, 7) : '—'}</span>
                      <span>{latest.commit_message || 'no commit message'}</span>
                      <span>{formatRelative(latest.created_at)}</span>
                      {latest.error ? <span className="text-danger">{latest.error}</span> : null}
                    </div>
                  ) : (
                    <p className="text-xs text-muted">This application has not been deployed yet.</p>
                  )}
                </Section>
              </div>
            </TabsPanel>

            <TabsPanel value="deployments">
              <Section title="Deployments" description="Newest first. Logs stream live while a deploy runs.">
                <DeploymentTable
                  deployments={deployments.data ?? []}
                  onRollback={(deployment) => rollback.mutate(deployment)}
                />
              </Section>
            </TabsPanel>

            <TabsPanel value="logs">
              <Section
                title="Container logs"
                description="Live output from the running container, with search and download."
              >
                {running ? (
                  <LogViewer
                    resource="applications"
                    uuid={applicationId}
                    sources={logSources}
                  />
                ) : (
                  <p className="rounded-lg border border-dashed border-border px-4 py-6 text-center text-xs text-muted">
                    The container is not running. Start or deploy it to read logs.
                  </p>
                )}
              </Section>
            </TabsPanel>

            <TabsPanel value="terminal">
              <Section title="Terminal" description="A shell inside the container, running as its user.">
                {running && runtime?.container_id ? (
                  <Suspense fallback={<Skeleton className="h-[28rem]" />}>
                    <TerminalView target={runtime.container_id} />
                  </Suspense>
                ) : (
                  <p className="rounded-lg border border-dashed border-border px-4 py-6 text-center text-xs text-muted">
                    The container is not running. Start or deploy it to open a shell.
                  </p>
                )}
              </Section>
            </TabsPanel>

            <TabsPanel value="environment">
              <EnvVarsEditor applicationId={applicationId} />
            </TabsPanel>

            <TabsPanel value="domains">
              <DomainsEditor
                applicationId={applicationId}
                domains={data.domains}
                onChange={invalidate}
              />
            </TabsPanel>

            <TabsPanel value="settings">
              <div className="flex flex-col gap-4">
                <Card className="max-w-2xl">
                  <CardContent className="flex flex-col gap-3">
                    <div className="flex items-center gap-2">
                      {data.kind === 'image' ? (
                        <Badge variant="outline">image</Badge>
                      ) : (
                        <GitBranch aria-hidden className="size-3.5 text-muted" />
                      )}
                      <span className="text-sm font-medium text-foreground">Configuration</span>
                    </div>
                    <MetaRow label="Repository">{data.repository || '—'}</MetaRow>
                    <MetaRow label="Branch">{data.branch || '—'}</MetaRow>
                    <MetaRow label="Build pack">{data.build_pack}</MetaRow>
                    {data.config.dockerfile_path ? (
                      <MetaRow label="Dockerfile">{data.config.dockerfile_path}</MetaRow>
                    ) : null}
                    {data.config.build_command ? (
                      <MetaRow label="Build command">{data.config.build_command}</MetaRow>
                    ) : null}
                    {data.config.publish_directory ? (
                      <MetaRow label="Publish dir">{data.config.publish_directory}</MetaRow>
                    ) : null}
                    {data.source?.type === 'github_app' ? (
                      <MetaRow label="Source">
                        <span className="inline-flex items-center gap-1.5">
                          <GitHubIcon className="size-3.5" />
                          {data.source.name}
                        </span>
                      </MetaRow>
                    ) : null}
                    <MetaRow label="Created">{formatDateTime(data.created_at)}</MetaRow>
                  </CardContent>
                </Card>

                <Card className="max-w-2xl">
                  <CardContent className="flex flex-col gap-3">
                    <span className="text-sm font-medium text-foreground">Danger zone</span>
                    <p className="text-xs text-muted">
                      Deleting removes the container, its deployments history and its configuration.
                    </p>
                    <div>
                      <Button variant="danger" size="sm" onClick={() => setDeleteOpen(true)}>
                        <Trash2 aria-hidden className="size-3.5" />
                        Delete application
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </TabsPanel>
          </Tabs>

          <ConfirmDialog
            open={deleteOpen}
            onOpenChange={setDeleteOpen}
            title={`Delete ${data.name}?`}
            description="The container is stopped and removed. This cannot be undone."
            confirmLabel="Delete application"
            danger
            loading={remove.isPending}
            onConfirm={() => remove.mutate()}
          />
        </>
      ) : null}

      {application.isError ? (
        <Card className="max-w-xl">
          <CardContent className="flex flex-col gap-3">
            <span className="text-sm font-medium text-foreground">Application not found</span>
            <Link to="/projects">
              <Button variant="secondary" size="sm">
                <Pencil aria-hidden className="size-3.5" />
                Back to projects
              </Button>
            </Link>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}
