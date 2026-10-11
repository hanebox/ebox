import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Layers, Plus } from 'lucide-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { ServiceResponse } from '@/api/types'
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
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsTab } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatRelative } from '@/lib/format'

const sampleCompose = `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  cache:
    image: redis:7-alpine
`

function stateVariant(state: string) {
  switch (state) {
    case 'running':
      return 'success' as const
    case 'partial':
      return 'warning' as const
    case 'failed':
      return 'danger' as const
    case 'stopped':
      return 'warning' as const
    default:
      return 'outline' as const
  }
}

function ServiceCard({ service }: { service: ServiceResponse }) {
  const state = service.runtime?.state ?? service.status

  return (
    <Link
      to="/services/$serviceId"
      params={{ serviceId: service.uuid }}
      className="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 transition-colors hover:border-accent/60"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="flex size-8 items-center justify-center rounded-lg bg-accent-soft text-accent">
            <Layers aria-hidden className="size-4" />
          </span>
          <div className="flex min-w-0 flex-col">
            <span className="truncate text-sm font-medium text-foreground">{service.name}</span>
            <span className="truncate text-[11px] text-subtle">
              {service.source === 'git' ? `${service.repository || 'repository'} · ${service.branch}` : 'inline compose'}
            </span>
          </div>
        </div>
        <Badge variant={stateVariant(state)} className="gap-1.5">
          {state === 'running' ? <StatusDot variant="success" /> : null}
          {state}
        </Badge>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {(service.runtime?.services ?? []).map((item) => (
          <span
            key={item.name}
            className="rounded-md border border-border px-1.5 py-0.5 font-mono text-[10px] text-muted"
          >
            {item.name}
            <span className={item.state === 'running' ? 'text-success' : 'text-subtle'}> · {item.state}</span>
          </span>
        ))}
      </div>

      <span className="text-[11px] text-subtle">
        {service.last_deployed_at ? `deployed ${formatRelative(service.last_deployed_at)}` : 'never deployed'}
      </span>
    </Link>
  )
}

function CreateServiceDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const queryClient = useQueryClient()
  const [mode, setMode] = useState<'inline' | 'git'>('inline')
  const [name, setName] = useState('')
  const [project, setProject] = useState('')
  const [compose, setCompose] = useState(sampleCompose)
  const [environment, setEnvironment] = useState('')
  const [source, setSource] = useState('')
  const [repository, setRepository] = useState('')
  const [branch, setBranch] = useState('main')
  const [composePath, setComposePath] = useState('docker-compose.yml')

  const projects = useQuery({ queryKey: ['projects'], queryFn: api.projects })
  const sources = useQuery({ queryKey: ['sources'], queryFn: api.sources })

  const create = useMutation({
    mutationFn: () =>
      api.createService({
        project: project || projects.data?.[0]?.uuid || '',
        name,
        source: mode,
        compose: mode === 'inline' ? compose : undefined,
        source_uuid: mode === 'git' ? source : undefined,
        repository: mode === 'git' ? repository : undefined,
        branch: mode === 'git' ? branch : undefined,
        compose_path: mode === 'git' ? composePath : undefined,
        environment,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['services'] })
      void queryClient.invalidateQueries({ queryKey: ['containers'] })
      toast.success('Service deployed')
      onOpenChange(false)
      setName('')
    },
    onError: (error) =>
      toast.error('Could not create the service', error instanceof ApiError ? error.message : undefined),
  })

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title="New service"
      description="A docker compose stack managed as one unit."
      size="lg"
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault()
          create.mutate()
        }}
      >
        <Tabs value={mode} onValueChange={(value) => setMode(value as 'inline' | 'git')}>
          <TabsList>
            <TabsTab value="inline">Paste compose</TabsTab>
            <TabsTab value="git">From a repository</TabsTab>
          </TabsList>
        </Tabs>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Name">
            {(field) => (
              <Input
                {...field}
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="demo-stack"
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

        {mode === 'inline' ? (
          <Field label="Compose file" description="ebox adds labels, the shared network and routing.">
            {(field) => (
              <Textarea
                {...field}
                value={compose}
                onChange={(event) => setCompose(event.target.value)}
                rows={10}
                className="font-mono text-xs"
              />
            )}
          </Field>
        ) : (
          <>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Source">
                {(field) => (
                  <Select {...field} value={source} onChange={(event) => setSource(event.target.value)}>
                    <option value="">Select a source…</option>
                    {(sources.data ?? []).map((item) => (
                      <option key={item.uuid} value={item.uuid}>
                        {item.name}
                      </option>
                    ))}
                  </Select>
                )}
              </Field>
              <Field label="Repository" description="Required for GitHub App sources.">
                {(field) => (
                  <Input
                    {...field}
                    value={repository}
                    onChange={(event) => setRepository(event.target.value)}
                    placeholder="owner/repo"
                    className="font-mono text-xs"
                  />
                )}
              </Field>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Branch">
                {(field) => (
                  <Input
                    {...field}
                    value={branch}
                    onChange={(event) => setBranch(event.target.value)}
                    className="font-mono text-xs"
                  />
                )}
              </Field>
              <Field label="Compose path">
                {(field) => (
                  <Input
                    {...field}
                    value={composePath}
                    onChange={(event) => setComposePath(event.target.value)}
                    className="font-mono text-xs"
                  />
                )}
              </Field>
            </div>
          </>
        )}

        <Field label="Environment" description="Written to .env for compose interpolation.">
          {(field) => (
            <Textarea
              {...field}
              value={environment}
              onChange={(event) => setEnvironment(event.target.value)}
              rows={4}
              placeholder={'APP_ENV=production\nTAG=latest'}
              className="font-mono text-xs"
            />
          )}
        </Field>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={create.isPending}>
            {create.isPending ? 'Deploying…' : 'Create and deploy'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

export function ServicesPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const services = useQuery({ queryKey: ['services'], queryFn: () => api.services(), refetchInterval: 10_000 })

  const rows = services.data ?? []

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Services"
        description="Compose stacks: databases, workers, queues and anything else you run alongside your apps."
        actions={
          <Button variant="primary" size="sm" onClick={() => setDialogOpen(true)}>
            <Plus aria-hidden className="size-3.5" />
            New service
          </Button>
        }
      />

      {services.isPending ? (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </div>
      ) : null}

      {services.isSuccess && rows.length === 0 ? (
        <Card className="p-6">
          <EmptyState
            icon={<Layers aria-hidden />}
            title="No services yet"
            description="Paste a compose file or point ebox at one in your repository."
            action={
              <Button variant="primary" size="sm" onClick={() => setDialogOpen(true)}>
                <Plus aria-hidden className="size-3.5" />
                New service
              </Button>
            }
          />
        </Card>
      ) : null}

      {rows.length > 0 ? (
        <>
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {rows.map((service) => (
              <ServiceCard key={service.uuid} service={service} />
            ))}
          </div>

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Service</TableHead>
                <TableHead>Compose service</TableHead>
                <TableHead>State</TableHead>
                <TableHead>Image</TableHead>
                <TableHead>Ports</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.flatMap((service) =>
                (service.runtime?.services ?? []).map((item) => (
                  <TableRow key={`${service.uuid}-${item.name}`}>
                    <TableCell className="text-xs text-foreground">{service.name}</TableCell>
                    <TableCell className="font-mono text-[11px] text-muted">{item.name}</TableCell>
                    <TableCell>
                      <Badge variant={item.state === 'running' ? 'success' : 'outline'}>{item.state}</Badge>
                    </TableCell>
                    <TableCell className="font-mono text-[11px] text-muted">{item.image ?? '—'}</TableCell>
                    <TableCell className="font-mono text-[11px] text-muted">
                      {(item.ports ?? []).join(', ') || '—'}
                    </TableCell>
                  </TableRow>
                )),
              )}
            </TableBody>
          </Table>
        </>
      ) : null}

      <CreateServiceDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </div>
  )
}
