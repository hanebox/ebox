import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from '@tanstack/react-router'
import { IconArrowLeft, IconCheck, IconCopy, IconDatabase, IconEye, IconEyeOff, IconPlayerPlay, IconRotateClockwise2, IconSquare, IconTrash } from '@tabler/icons-react'
import { Suspense, lazy, useState } from 'react'

import { api } from '@/api/endpoints'
import { DatabaseBackups } from '@/components/backups/database-backups'
import type { DatabaseConnection } from '@/api/types'
import { LogViewer } from '@/components/logs/log-viewer'
import { PageHeader } from '@/components/layout/page-header'
import { Badge, StatusDot } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardTitle, Section } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsPanel, TabsTab } from '@/components/ui/tabs'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

const TerminalView = lazy(() =>
  import('@/components/terminal/terminal-view').then((module) => ({ default: module.TerminalView })),
)

function CopyRow({ label, value, mono = true }: { label: string; value: string; mono?: boolean }) {
  const [copied, setCopied] = useState(false)

  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-border bg-surface-raised px-3 py-2">
      <div className="flex min-w-0 flex-col">
        <span className="text-2xs font-medium text-subtle">{label}</span>
        <span className={`truncate text-xs text-foreground ${mono ? 'font-mono' : ''}`}>{value || '–'}</span>
      </div>
      <Button
        variant="ghost"
        size="sm"
        aria-label={`Copy ${label}`}
        onClick={() => {
          void navigator.clipboard.writeText(value).then(() => {
            setCopied(true)
            setTimeout(() => setCopied(false), 1500)
          })
        }}
      >
        {copied ? <IconCheck aria-hidden className="size-3.5 text-success" /> : <IconCopy aria-hidden className="size-3.5" />}
      </Button>
    </div>
  )
}

export function DatabaseDetailPage() {
  const { databaseId } = useParams({ strict: false }) as { databaseId: string }
  const queryClient = useQueryClient()
  const [tab, setTab] = useState('connection')
  const [reveal, setReveal] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const database = useQuery({
    queryKey: ['database', databaseId],
    queryFn: () => api.database(databaseId),
    refetchInterval: 10_000,
  })

  const credentials = useQuery({
    queryKey: ['database', databaseId, 'credentials'],
    queryFn: () => api.databaseCredentials(databaseId),
    enabled: reveal,
  })

  const lifecycle = useMutation({
    mutationFn: (action: 'start' | 'stop' | 'restart') => api.databaseLifecycle(databaseId, action),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['database', databaseId] })
      void queryClient.invalidateQueries({ queryKey: ['containers'] })
      toast.success(result.message)
    },
    onError: (error) =>
      toast.error('Action failed', error instanceof ApiError ? error.message : undefined),
  })

  const remove = useMutation({
    mutationFn: () => api.deleteDatabase(databaseId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['databases'] })
      void queryClient.invalidateQueries({ queryKey: ['containers'] })
      toast.success('Database deleted')
      window.history.back()
    },
    onError: (error) =>
      toast.error('Could not delete the database', error instanceof ApiError ? error.message : undefined),
  })

  const connection: DatabaseConnection | undefined = reveal ? credentials.data : database.data?.connection
  const runtime = database.data?.runtime
  const running = runtime?.state === 'running'

  if (database.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48" />
      </div>
    )
  }

  if (database.isError || !database.data) {
    return (
      <Card className="p-6">
        <p className="text-sm text-danger">Database not found.</p>
      </Card>
    )
  }

  const record = database.data

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <Link
          to="/databases"
          className="inline-flex items-center gap-1 text-xs text-subtle transition-colors hover:text-foreground"
        >
          <IconArrowLeft aria-hidden className="size-3.5" />
          {record.project?.name ?? 'Databases'}
        </Link>
        <PageHeader
          title={record.name}
          description={`${record.definition?.label ?? record.type} ${record.version} · ${record.runtime?.container_name ?? 'not created yet'}`}
          actions={
            <>
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
        <Badge variant={running ? 'success' : 'default'} className="gap-1.5">
          {running ? <StatusDot variant="success" /> : null}
          {runtime?.state ?? record.status}
        </Badge>
        {runtime?.health && runtime.health !== 'none' ? (
          <Badge variant={runtime.health === 'healthy' ? 'success' : 'warning'}>{runtime.health}</Badge>
        ) : null}
        <Badge variant="outline">
          <IconDatabase aria-hidden className="me-1 size-3" />
          {record.type} {record.version}
        </Badge>
        {record.port > 0 ? <Badge variant="info">host port {record.port}</Badge> : <Badge variant="outline">internal only</Badge>}
      </div>

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTab value="connection">Connection</TabsTab>
          <TabsTab value="backups">Backups</TabsTab>
          <TabsTab value="logs">Logs</TabsTab>
          <TabsTab value="terminal">Terminal</TabsTab>
          <TabsTab value="settings">Settings</TabsTab>
        </TabsList>

        <TabsPanel value="connection">
          <Section
            title="Connection"
            description="Applications on the ebox network connect with the internal host."
            actions={
              <Button variant="secondary" size="sm" onClick={() => setReveal((value) => !value)}>
                {reveal ? <IconEyeOff aria-hidden className="size-3.5" /> : <IconEye aria-hidden className="size-3.5" />}
                {reveal ? 'Hide secrets' : 'Reveal secrets'}
              </Button>
            }
          >
            {credentials.isFetching ? <Skeleton className="h-32" /> : null}

            <div className="grid gap-2 sm:grid-cols-2">
              <CopyRow label="Internal host" value={connection?.host ?? ''} />
              <CopyRow label="Internal port" value={String(connection?.port ?? '')} />
              <CopyRow label="Username" value={connection?.username ?? ''} />
              <CopyRow label="Database" value={connection?.database ?? ''} />
              <CopyRow label="Password" value={connection?.password ?? '•••••••• (reveal to copy)'} />
              <CopyRow label="Connection URL" value={connection?.url ?? '•••••••• (reveal to copy)'} />
              {connection?.exposed ? (
                <>
                  <CopyRow
                    label="Public address"
                    value={`${connection.public_host ?? ''}:${connection.public_port ?? ''}`}
                  />
                  <CopyRow label="Public URL" value={connection.public_url ?? ''} />
                </>
              ) : null}
            </div>

            <p className="mt-3 text-2xs text-subtle">
              Tip: open an application's Environment tab and use “Attach database” to copy these values as
              environment variables.
            </p>
          </Section>
        </TabsPanel>

        <TabsPanel value="backups">
          <Section title="Backups" description="Scheduled and on-demand dumps of this database.">
            <DatabaseBackups databaseId={databaseId} databaseName={record.name} />
          </Section>
        </TabsPanel>

        <TabsPanel value="logs">
          <Section title="Logs" description="Container output from the database engine.">
            {running ? (
              <LogViewer resource="databases" uuid={databaseId} />
            ) : (
              <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
                The database container is not running.
              </p>
            )}
          </Section>
        </TabsPanel>

        <TabsPanel value="terminal">
          <Section title="Terminal" description="A shell inside the database container.">
            {running && runtime?.container_id ? (
              <Suspense fallback={<Skeleton className="h-[28rem]" />}>
                <TerminalView target={runtime.container_id} />
              </Suspense>
            ) : (
              <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
                The database container is not running.
              </p>
            )}
          </Section>
        </TabsPanel>

        <TabsPanel value="settings">
          <div className="flex flex-col gap-4">
            <Section title="Limits" description="Applied on the next start or restart.">
              <div className="grid gap-3 sm:grid-cols-2">
                <Field label="Memory">
                  {(field) => (
                    <Input {...field} readOnly value={record.limits?.memory_mb ? `${record.limits.memory_mb} MB` : 'unlimited'} />
                  )}
                </Field>
                <Field label="CPU">
                  {(field) => (
                    <Input {...field} readOnly value={record.limits?.cpus ? `${record.limits.cpus} cores` : 'unlimited'} />
                  )}
                </Field>
              </div>
            </Section>

            <Card>
              <div className="flex flex-col gap-1 p-4 pb-2">
                <CardTitle>Danger zone</CardTitle>
                <CardDescription>
                  Deleting removes the container and its data volume. This cannot be undone.
                </CardDescription>
              </div>
              <CardContent className="flex justify-end pt-0">
                <Button variant="danger" size="sm" onClick={() => setDeleteOpen(true)}>
                  <IconTrash aria-hidden className="size-3.5" />
                  Delete database
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
        description="The container and the data volume holding the database files are removed."
        confirmLabel="Delete database and data"
        danger
        loading={remove.isPending}
        onConfirm={() => remove.mutate()}
      />
    </div>
  )
}
