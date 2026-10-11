import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { IconDatabase, IconPlus } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { DatabaseDefinition, DatabaseResponse } from '@/api/types'
import { PageHeader } from '@/components/layout/page-header'
import { Badge, StatusDot } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Modal } from '@/components/ui/dialog'
import { Select } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatRelative } from '@/lib/format'

const engineIcons: Record<string, string> = {
  postgres: 'PostgreSQL',
  mysql: 'MySQL',
  mariadb: 'MariaDB',
  redis: 'Redis',
  mongo: 'MongoDB',
}

function stateVariant(state: string) {
  switch (state) {
    case 'running':
      return 'success' as const
    case 'exited':
    case 'stopped':
      return 'warning' as const
    case 'missing':
      return 'outline' as const
    default:
      return 'default' as const
  }
}

function DatabaseCard({ database }: { database: DatabaseResponse }) {
  const state = database.runtime?.state ?? database.status

  return (
    <Link
      to="/databases/$databaseId"
      params={{ databaseId: database.uuid }}
      className="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 transition-colors hover:border-border-strong"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="flex size-8 items-center justify-center rounded-lg bg-accent-soft text-accent">
            <IconDatabase aria-hidden className="size-4" />
          </span>
          <div className="flex min-w-0 flex-col">
            <span className="truncate text-sm font-medium text-foreground">{database.name}</span>
            <span className="truncate text-2xs text-subtle">
              {engineIcons[database.type] ?? database.type} {database.version}
            </span>
          </div>
        </div>
        <Badge variant={stateVariant(state)} className="gap-1.5">
          {state === 'running' ? <StatusDot variant="success" /> : null}
          {state}
        </Badge>
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-2xs text-subtle">
        <span className="font-mono">{database.runtime?.container_name ?? '–'}</span>
        {database.port > 0 ? <span className="font-mono">host port {database.port}</span> : <span>internal only</span>}
        <span>{formatRelative(database.created_at)}</span>
      </div>
    </Link>
  )
}

function CreateDatabaseDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const queryClient = useQueryClient()
  const [project, setProject] = useState('')
  const [name, setName] = useState('')
  const [type, setType] = useState('postgres')
  const [version, setVersion] = useState('')
  const [expose, setExpose] = useState(false)
  const [port, setPort] = useState('')
  const [memory, setMemory] = useState('')
  const [cpus, setCpus] = useState('')

  const projects = useQuery({ queryKey: ['projects'], queryFn: api.projects })
  const definitions = useQuery({ queryKey: ['database-definitions'], queryFn: api.databaseDefinitions })

  const definition: DatabaseDefinition | undefined = definitions.data?.find((item) => item.type === type)

  const create = useMutation({
    mutationFn: () =>
      api.createDatabase({
        project: project || projects.data?.[0]?.uuid || '',
        name,
        type,
        version: version || definition?.versions[0] || '',
        port: expose && port ? Number(port) : 0,
        limits:
          memory || cpus
            ? { memory_mb: memory ? Number(memory) : 0, cpus: cpus ? Number(cpus) : 0 }
            : undefined,
      }),
    onSuccess: (database) => {
      void queryClient.invalidateQueries({ queryKey: ['databases'] })
      void queryClient.invalidateQueries({ queryKey: ['containers'] })
      toast.success(`${database.name} is ready`, 'Connection details are on the database page.')
      onOpenChange(false)
      setName('')
      setPort('')
    },
    onError: (error) =>
      toast.error('Could not create the database', error instanceof ApiError ? error.message : undefined),
  })

  return (
    <Modal open={open} onOpenChange={onOpenChange} title="New database" description="Provisioned with a generated password.">
      <form
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault()
          create.mutate()
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Name">
            {(field) => (
              <Input
                {...field}
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="app-db"
                autoFocus
              />
            )}
          </Field>

          <Field label="Project">
            {(field) => (
              <Select {...field} value={project} onChange={(event) => setProject(event.target.value)}>
                <option value="">Select a project…</option>
                {(projects.data ?? []).map((item) => (
                  <option key={item.uuid} value={item.uuid}>
                    {item.name}
                  </option>
                ))}
              </Select>
            )}
          </Field>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Engine">
            {(field) => (
              <Select
                {...field}
                value={type}
                onChange={(event) => {
                  setType(event.target.value)
                  setVersion('')
                }}
              >
                {(definitions.data ?? []).map((item) => (
                  <option key={item.type} value={item.type}>
                    {item.label}
                  </option>
                ))}
              </Select>
            )}
          </Field>

          <Field label="Version">
            {(field) => (
              <Select {...field} value={version} onChange={(event) => setVersion(event.target.value)}>
                {(definition?.versions ?? []).map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </Select>
            )}
          </Field>
        </div>

        <div className="rounded-lg border border-border p-3">
          <div className="flex items-center justify-between gap-3">
            <div className="flex flex-col">
              <span className="text-xs text-foreground">Expose on a host port</span>
              <span className="text-2xs text-subtle">
                Apps reach the database on the ebox network without this.
              </span>
            </div>
            <Switch checked={expose} onCheckedChange={setExpose} aria-label="Expose on a host port" />
          </div>
          {expose ? (
            <Field label="Host port" className="mt-3" description="Pick a free port, for example 5433.">
              {(field) => (
                <Input
                  {...field}
                  value={port}
                  onChange={(event) => setPort(event.target.value.replace(/[^0-9]/g, ''))}
                  placeholder="5433"
                  inputMode="numeric"
                  className="font-mono text-xs"
                />
              )}
            </Field>
          ) : null}
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Memory limit" description="MB, optional">
            {(field) => (
              <Input
                {...field}
                value={memory}
                onChange={(event) => setMemory(event.target.value.replace(/[^0-9]/g, ''))}
                placeholder="512"
                inputMode="numeric"
                className="font-mono text-xs"
              />
            )}
          </Field>
          <Field label="CPU limit" description="Cores, optional">
            {(field) => (
              <Input
                {...field}
                value={cpus}
                onChange={(event) => setCpus(event.target.value.replace(/[^0-9.]/g, ''))}
                placeholder="1.0"
                inputMode="decimal"
                className="font-mono text-xs"
              />
            )}
          </Field>
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={create.isPending}>
            {create.isPending ? 'Provisioning…' : 'Create database'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

export function DatabasesPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const databases = useQuery({ queryKey: ['databases'], queryFn: () => api.databases(), refetchInterval: 10_000 })

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Databases"
        description="Postgres, MySQL, MariaDB, Redis and MongoDB with generated credentials."
        actions={
          <Button variant="primary" size="sm" onClick={() => setDialogOpen(true)}>
            <IconPlus aria-hidden className="size-3.5" />
            New database
          </Button>
        }
      />

      {databases.isPending ? (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <Skeleton className="h-28" />
          <Skeleton className="h-28" />
        </div>
      ) : null}

      {databases.isSuccess && (databases.data?.length ?? 0) === 0 ? (
        <Card className="p-6">
          <EmptyState
            icon={<IconDatabase aria-hidden />}
            title="No databases yet"
            description="Create one to get a connection string your applications can use."
            action={
              <Button variant="primary" size="sm" onClick={() => setDialogOpen(true)}>
                <IconPlus aria-hidden className="size-3.5" />
                New database
              </Button>
            }
          />
        </Card>
      ) : null}

      {(databases.data?.length ?? 0) > 0 ? (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {databases.data?.map((database) => (
            <DatabaseCard key={database.uuid} database={database} />
          ))}
        </div>
      ) : null}

      <CreateDatabaseDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </div>
  )
}
