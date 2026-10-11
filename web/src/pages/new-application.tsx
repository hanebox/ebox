import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import { IconArrowLeft, IconBox, IconFileCode2, IconLoader2, IconSearch, IconWorld } from '@tabler/icons-react'
import { useMemo, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { api } from '@/api/endpoints'
import type { Branch, Repository } from '@/api/types'
import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardTitle } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'
import { Field } from '@/components/ui/field'
import { GitHubIcon } from '@/components/ui/icons'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { cn } from '@/lib/cn'

const schema = z
  .object({
    name: z.string().min(1, 'Name is required').max(255),
    source: z.string().min(1, 'Select a source'),
    repository: z.string().optional(),
    branch: z.string().min(1, 'A branch is required'),
    build_pack: z.enum(['dockerfile', 'static', 'image', 'compose']),
    compose_path: z.string().optional(),
    compose_service: z.string().optional(),
    base_directory: z.string().optional(),
    dockerfile_path: z.string().optional(),
    build_command: z.string().optional(),
    publish_directory: z.string().optional(),
    image: z.string().optional(),
    expose_port: z.coerce.number().int().min(1, 'Ports start at 1').max(65535, 'Ports end at 65535'),
    domains: z.string().optional(),
    env: z.string().optional(),
    auto_deploy: z.boolean(),
  })
  .refine((values) => values.build_pack !== 'image' || Boolean(values.image?.trim()), {
    path: ['image'],
    message: 'A docker image is required.',
  })

type FormValues = z.input<typeof schema>
type ParsedValues = z.output<typeof schema>

function parseDomains(value?: string): string[] {
  if (!value) {
    return []
  }

  return value
    .split(/[\s,]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function parseEnv(value?: string): { key: string; value: string }[] {
  if (!value) {
    return []
  }

  return value
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#') && line.includes('='))
    .map((line) => {
      const index = line.indexOf('=')
      return { key: line.slice(0, index).trim(), value: line.slice(index + 1).trim() }
    })
    .filter((pair) => pair.key.length > 0)
}

function RepositoryPicker({
  sourceId,
  value,
  onSelect,
}: {
  sourceId: string
  value: string
  onSelect: (repository: Repository) => void
}) {
  const [search, setSearch] = useState('')
  const repositories = useQuery({
    queryKey: ['source', sourceId, 'repositories', 1],
    queryFn: () => api.repositories(sourceId, 1),
    enabled: Boolean(sourceId),
  })

  const filtered = useMemo(() => {
    const list = repositories.data?.repositories ?? []
    if (!search.trim()) {
      return list
    }

    const needle = search.trim().toLowerCase()

    return list.filter((repository) => repository.full_name.toLowerCase().includes(needle))
  }, [repositories.data, search])

  return (
    <div className="flex flex-col gap-2">
      <div className="relative">
        <IconSearch
          aria-hidden
          className="pointer-events-none absolute top-1/2 start-2.5 size-3.5 -translate-y-1/2 text-subtle"
        />
        <Input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Filter repositories…"
          className="ps-8"
        />
      </div>

      {repositories.isPending ? <Skeleton className="h-24" /> : null}

      {repositories.isError ? (
        <p className="rounded-lg border border-danger/40 bg-danger-soft p-2 text-xs text-danger">
          {repositories.error instanceof ApiError
            ? repositories.error.message
            : 'Could not load repositories.'}
        </p>
      ) : null}

      <div className="max-h-56 overflow-auto rounded-lg border border-border">
        {filtered.map((repository) => (
          <button
            type="button"
            key={repository.full_name}
            onClick={() => onSelect(repository)}
            className={cn(
              'flex w-full items-center justify-between gap-3 border-b border-border/60 px-3 py-2 text-start last:border-b-0 transition-colors',
              value === repository.full_name ? 'bg-accent-soft' : 'hover:bg-surface-raised',
            )}
          >
            <span className="flex min-w-0 flex-col">
              <span className="truncate text-xs text-foreground">{repository.full_name}</span>
              <span className="truncate text-2xs text-subtle">
                {repository.description || repository.default_branch}
              </span>
            </span>
            {repository.private ? <Badge variant="outline">private</Badge> : null}
          </button>
        ))}
        {filtered.length === 0 && !repositories.isPending ? (
          <p className="px-3 py-4 text-center text-xs text-subtle">No repositories found.</p>
        ) : null}
      </div>
    </div>
  )
}

function BranchPicker({
  sourceId,
  repository,
  defaultBranch,
  value,
  onChange,
}: {
  sourceId: string
  repository: string
  defaultBranch: string
  value: string
  onChange: (branch: string) => void
}) {
  const branches = useQuery({
    queryKey: ['source', sourceId, 'branches', repository],
    queryFn: () => api.branches(sourceId, repository, defaultBranch),
    enabled: Boolean(sourceId && repository),
    staleTime: 60_000,
  })

  if (!repository) {
    return null
  }

  return (
    <div className="flex flex-col gap-2">
      {branches.isPending ? (
        <div className="flex items-center gap-2 text-xs text-muted">
          <IconLoader2 aria-hidden className="size-3.5 animate-spin" />
          Loading branches…
        </div>
      ) : null}
      {branches.data && branches.data.length > 0 ? (
        <Select value={value} onChange={(event) => onChange(event.target.value)}>
          {branches.data.map((branch: Branch) => (
            <option key={branch.name} value={branch.name}>
              {branch.name}
              {branch.default ? ' (default)' : ''}
            </option>
          ))}
        </Select>
      ) : null}
    </div>
  )
}

export function NewApplicationPage() {
  const { projectId } = useParams({ strict: false }) as { projectId: string }
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [defaultBranch, setDefaultBranch] = useState('main')

  const project = useQuery({ queryKey: ['project', projectId], queryFn: () => api.project(projectId) })
  const sources = useQuery({ queryKey: ['sources'], queryFn: api.sources })

  const form = useForm<FormValues, unknown, ParsedValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      source: '',
      repository: '',
      branch: 'main',
      build_pack: 'dockerfile',
      compose_path: '',
      compose_service: '',
      base_directory: '',
      dockerfile_path: '',
      build_command: '',
      publish_directory: '',
      image: '',
      expose_port: 80,
      domains: '',
      env: '',
      auto_deploy: false,
    },
  })

  const values = useWatch({ control: form.control })
  const sourceUuid = values.source ?? ''
  const buildPack = values.build_pack ?? 'dockerfile'
  const selectedSource = useMemo(
    () => sources.data?.find((source) => source.uuid === sourceUuid),
    [sources.data, sourceUuid],
  )
  const isGithubSource = selectedSource?.type === 'github_app'

  const create = useMutation({
    mutationFn: (input: ParsedValues) =>
      api.createApplication({
        project: projectId,
        name: input.name,
        source: input.source,
        repository: input.repository,
        branch: input.branch,
        build_pack: input.build_pack,
        config: {
          base_directory: input.base_directory || '',
          dockerfile_path: input.dockerfile_path || '',
          build_command: input.build_command || '',
          publish_directory: input.publish_directory || '',
          image: input.image || '',
          expose_port: input.expose_port,
        },
        auto_deploy: input.auto_deploy,
        domains: parseDomains(input.domains),
        env: parseEnv(input.env),
        deploy: true,
      }),
    onSuccess: (application) => {
      void queryClient.invalidateQueries({ queryKey: ['applications'] })
      void queryClient.invalidateQueries({ queryKey: ['deployments'] })
      toast.success('Application created', 'The first deployment is queued.')
      void navigate({ to: '/applications/$applicationId', params: { applicationId: application.uuid } })
    },
    onError: (error) => {
      toast.error('Could not create the application', error instanceof ApiError ? error.message : undefined)
    },
  })

  return (
    <div className="flex flex-col gap-6">
      <div>
        <Link
          to="/projects/$projectId"
          params={{ projectId }}
          className="inline-flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-foreground"
        >
          <IconArrowLeft aria-hidden className="size-3.5" />
          {project.data?.name ?? 'Project'}
        </Link>
      </div>

      <PageHeader
        title="New application"
        description="Point the panel at a repository, pick a build pack and deploy."
      />

      {sources.data && sources.data.length === 0 ? (
        <EmptyState
          icon={<GitHubIcon className="size-6" />}
          title="No sources connected"
          description="Connect a GitHub App or add a git source before creating an application."
          action={
            <Link to="/sources">
              <Button variant="primary">Go to sources</Button>
            </Link>
          }
        />
      ) : null}

      {sources.data && sources.data.length > 0 ? (
        <form
          className="flex max-w-3xl flex-col gap-4"
          onSubmit={form.handleSubmit((input) => {
            const source = sources.data?.find((item) => item.uuid === input.source)

            if (source?.type === 'github_app' && !input.repository?.trim()) {
              form.setError('repository', { message: 'Select a repository.' })
              toast.error('Select a repository')

              return
            }

            create.mutate(input)
          })}
        >
          <Card>
            <div className="flex flex-col gap-1 p-4 pb-2">
              <CardTitle>Basics</CardTitle>
              <CardDescription>The name is what you will see across the panel.</CardDescription>
            </div>
            <div className="flex flex-col gap-4 p-4 pt-2">
              <Field label="Name" error={form.formState.errors.name?.message}>
                {(field) => (
                  <Input
                    {...field}
                    {...form.register('name')}
                    placeholder="api"
                    invalid={Boolean(form.formState.errors.name)}
                    autoFocus
                  />
                )}
              </Field>
            </div>
          </Card>

          <Card>
            <div className="flex flex-col gap-1 p-4 pb-2">
              <CardTitle>Source</CardTitle>
              <CardDescription>Where the code comes from.</CardDescription>
            </div>
            <div className="flex flex-col gap-4 p-4 pt-2">
              <Field label="Source" error={form.formState.errors.source?.message}>
                {(field) => (
                  <Select
                    {...field}
                    {...form.register('source')}
                    invalid={Boolean(form.formState.errors.source)}
                  >
                    <option value="">Select a source…</option>
                    {(sources.data ?? []).map((source) => (
                      <option key={source.uuid} value={source.uuid}>
                        {source.name} ({source.type === 'github_app' ? 'GitHub App' : 'git'})
                      </option>
                    ))}
                  </Select>
                )}
              </Field>

              {isGithubSource ? (
                <Field label="Repository" error={form.formState.errors.repository?.message}>
                  {() => (
                    <RepositoryPicker
                      sourceId={sourceUuid}
                      value={values.repository ?? ''}
                      onSelect={(repository) => {
                        form.setValue('repository', repository.full_name, { shouldValidate: true })
                        setDefaultBranch(repository.default_branch || 'main')
                        form.setValue('branch', repository.default_branch || 'main')
                      }}
                    />
                  )}
                </Field>
              ) : null}

              {isGithubSource && values.repository ? (
                <Field label="Branch">
                  {() => (
                    <BranchPicker
                      sourceId={sourceUuid}
                      repository={values.repository ?? ''}
                      defaultBranch={defaultBranch}
                      value={values.branch ?? 'main'}
                      onChange={(branch) => form.setValue('branch', branch)}
                    />
                  )}
                </Field>
              ) : (
                <Field
                  label="Branch"
                  description={
                    selectedSource && selectedSource.type === 'git'
                      ? `Cloned from ${selectedSource.git_url}`
                      : undefined
                  }
                  error={form.formState.errors.branch?.message}
                >
                  {(field) => <Input {...field} {...form.register('branch')} placeholder="main" />}
                </Field>
              )}
            </div>
          </Card>

          <Card>
            <div className="flex flex-col gap-1 p-4 pb-2">
              <CardTitle>Build</CardTitle>
              <CardDescription>How the panel turns your code into a container.</CardDescription>
            </div>
            <div className="flex flex-col gap-4 p-4 pt-2">
              <Field label="Build pack">
                {(field) => (
                  <Select {...field} {...form.register('build_pack')}>
                    <option value="dockerfile">Dockerfile</option>
                    <option value="static">Static site (node build → caddy)</option>
                    <option value="image">Docker image (pull and run)</option>
                    <option value="compose">Docker compose (from the repository)</option>
                  </Select>
                )}
              </Field>

              {buildPack === 'image' ? (
                <Field
                  label="Image"
                  description="Any public or locally available image, e.g. nginx:alpine."
                  error={form.formState.errors.image?.message}
                >
                  {(field) => (
                    <Input
                      {...field}
                      {...form.register('image')}
                      placeholder="nginx:alpine"
                      className="font-mono text-xs"
                      invalid={Boolean(form.formState.errors.image)}
                    />
                  )}
                </Field>
              ) : null}

              {buildPack === 'compose' ? (
                <div className="flex flex-col gap-4">
                  <Field
                    label="Compose file"
                    description="Detected automatically when left empty (docker-compose.yml or compose.yaml)."
                  >
                    {(field) => (
                      <Input
                        {...field}
                        {...form.register('compose_path')}
                        placeholder="docker-compose.yml"
                        className="font-mono text-xs"
                      />
                    )}
                  </Field>
                  <Field
                    label="Ingress service"
                    description="Which service receives the domains. Detected from published ports when empty."
                  >
                    {(field) => (
                      <Input
                        {...field}
                        {...form.register('compose_service')}
                        placeholder="web"
                        className="font-mono text-xs"
                      />
                    )}
                  </Field>
                </div>
              ) : null}

              {buildPack === 'dockerfile' ? (
                <div className="grid gap-4 sm:grid-cols-2">
                  <Field label="Dockerfile path" description="Relative to the build context.">
                    {(field) => (
                      <Input
                        {...field}
                        {...form.register('dockerfile_path')}
                        placeholder="Dockerfile"
                        className="font-mono text-xs"
                      />
                    )}
                  </Field>
                  <Field label="Base directory" description="Optional. Build from a subdirectory.">
                    {(field) => (
                      <Input
                        {...field}
                        {...form.register('base_directory')}
                        placeholder="."
                        className="font-mono text-xs"
                      />
                    )}
                  </Field>
                </div>
              ) : null}

              {buildPack === 'static' ? (
                <div className="flex flex-col gap-4">
                  <Field
                    label="Build command"
                    description="Leave empty to detect from the lockfile (bun, pnpm, yarn, npm)."
                  >
                    {(field) => (
                      <Input
                        {...field}
                        {...form.register('build_command')}
                        placeholder="npm ci && npm run build"
                        className="font-mono text-xs"
                      />
                    )}
                  </Field>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <Field label="Publish directory" description="Defaults to dist.">
                      {(field) => (
                        <Input
                          {...field}
                          {...form.register('publish_directory')}
                          placeholder="dist"
                          className="font-mono text-xs"
                        />
                      )}
                    </Field>
                    <Field label="Base directory" description="Optional. Build from a subdirectory.">
                      {(field) => (
                        <Input
                          {...field}
                          {...form.register('base_directory')}
                          placeholder="."
                          className="font-mono text-xs"
                        />
                      )}
                    </Field>
                  </div>
                </div>
              ) : null}
            </div>
          </Card>

          <Card>
            <div className="flex flex-col gap-1 p-4 pb-2">
              <CardTitle>Runtime</CardTitle>
              <CardDescription>Port, domains and deployment behaviour.</CardDescription>
            </div>
            <div className="flex flex-col gap-4 p-4 pt-2">
              <div className="grid gap-4 sm:grid-cols-2">
                <Field
                  label="Container port"
                  description="The port your app listens on inside the container."
                  error={form.formState.errors.expose_port?.message}
                >
                  {(field) => (
                    <Input
                      {...field}
                      {...form.register('expose_port')}
                      inputMode="numeric"
                      className="font-mono text-xs"
                      invalid={Boolean(form.formState.errors.expose_port)}
                    />
                  )}
                </Field>
                <Field label="Domains" description="Comma or space separated. Optional.">
                  {(field) => (
                    <Input
                      {...field}
                      {...form.register('domains')}
                      placeholder="app.example.com"
                      className="font-mono text-xs"
                    />
                  )}
                </Field>
              </div>

              <div className="flex items-center gap-3">
                <Switch
                  id="auto-deploy"
                  checked={values.auto_deploy ?? false}
                  onCheckedChange={(checked) => form.setValue('auto_deploy', checked)}
                />
                <div className="flex flex-col">
                  <label className="text-sm text-foreground" htmlFor="auto-deploy">
                    Deploy on push
                  </label>
                  <span className="text-2xs text-subtle">
                    Requires the GitHub App webhook (set up automatically by the manifest flow).
                  </span>
                </div>
              </div>
            </div>
          </Card>

          <Card>
            <div className="flex flex-col gap-1 p-4 pb-2">
              <CardTitle>Environment</CardTitle>
              <CardDescription>Optional. Paste a .env file to seed the variables.</CardDescription>
            </div>
            <div className="p-4 pt-2">
              <Textarea
                {...form.register('env')}
                rows={5}
                className="font-mono text-xs"
                placeholder={'APP_ENV=production\nAPI_KEY=…'}
              />
            </div>
          </Card>

          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="flex items-center gap-2 text-xs text-subtle">
              <IconBox aria-hidden className="size-3.5" />
              The first deployment starts right after creation.
            </p>
            <div className="flex items-center gap-2">
              <Link to="/projects/$projectId" params={{ projectId }}>
                <Button type="button" variant="ghost">
                  Cancel
                </Button>
              </Link>
              <Button type="submit" variant="primary" loading={create.isPending}>
                <IconFileCode2 aria-hidden className="size-3.5" />
                Create and deploy
              </Button>
            </div>
          </div>
        </form>
      ) : null}

      {sources.isPending ? <Skeleton className="h-40" /> : null}

      {sources.data && sources.data.length > 0 ? (
        <p className="flex items-center gap-2 text-xs text-subtle">
          <IconWorld aria-hidden className="size-3.5" />
          Domains are routed through Traefik once the proxy milestone is wired up.
        </p>
      ) : null}
    </div>
  )
}
