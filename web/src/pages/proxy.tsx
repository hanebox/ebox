import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { IconExternalLink, IconLock, IconNetwork, IconPlayerPlay, IconPlus, IconRefresh, IconShield, IconSquare, IconTool, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { ProxyMiddleware, ProxyStatus } from '@/api/types'
import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardTitle, Section } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Modal } from '@/components/ui/dialog'
import { EmptyState } from '@/components/ui/empty-state'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime } from '@/lib/format'

const middlewareTypes = [
  { value: 'basicAuth', label: 'Basic auth' },
  { value: 'redirectScheme', label: 'Redirect to HTTPS' },
  { value: 'headers', label: 'Custom headers' },
  { value: 'rateLimit', label: 'Rate limit' },
  { value: 'stripPrefix', label: 'Strip prefix' },
]

function StatusPill({ status }: { status: ProxyStatus }) {
  if (!status.installed) {
    return <Badge variant="outline">not configured</Badge>
  }
  if (status.running) {
    return <Badge variant="success">running</Badge>
  }

  return <Badge variant="danger">{status.state}</Badge>
}

function MiddlewareDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const queryClient = useQueryClient()
  const [type, setType] = useState('basicAuth')
  const [name, setName] = useState('')
  const [user, setUser] = useState('')
  const [password, setPassword] = useState('')
  const [scheme, setScheme] = useState('https')
  const [permanent, setPermanent] = useState(true)
  const [headers, setHeaders] = useState('')
  const [average, setAverage] = useState('100')
  const [burst, setBurst] = useState('200')
  const [prefixes, setPrefixes] = useState('')

  const save = useMutation({
    mutationFn: () => {
      let config: Record<string, unknown> = {}

      switch (type) {
        case 'basicAuth':
          config = { users: [{ username: user, password }] }
          break
        case 'redirectScheme':
          config = { scheme, permanent }
          break
        case 'headers':
          config = {
            customRequestHeaders: Object.fromEntries(
              headers
                .split('\n')
                .map((line) => line.split(':'))
                .filter((parts) => parts.length >= 2)
                .map((parts) => [parts[0].trim(), parts.slice(1).join(':').trim()]),
            ),
          }
          break
        case 'rateLimit':
          config = { average: Number(average), burst: Number(burst) }
          break
        case 'stripPrefix':
          config = { prefixes }
          break
      }

      return api.saveProxyMiddleware({ name, type, config })
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['proxy'] })
      toast.success('Middleware saved')
      onOpenChange(false)
      setName('')
    },
    onError: (error) =>
      toast.error('Could not save the middleware', error instanceof ApiError ? error.message : undefined),
  })

  return (
    <Modal open={open} onOpenChange={onOpenChange} title="New middleware" size="sm">
      <form
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault()
          save.mutate()
        }}
      >
        <Field label="Name">
          {(field) => (
            <Input
              {...field}
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="admin-auth"
              className="font-mono text-xs"
              autoFocus
            />
          )}
        </Field>

        <Field label="Type">
          {(field) => (
            <Select {...field} value={type} onChange={(event) => setType(event.target.value)}>
              {middlewareTypes.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </Select>
          )}
        </Field>

        {type === 'basicAuth' ? (
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Username">
              {(field) => (
                <Input
                  {...field}
                  value={user}
                  onChange={(event) => setUser(event.target.value)}
                  className="font-mono text-xs"
                />
              )}
            </Field>
            <Field label="Password">
              {(field) => (
                <Input
                  {...field}
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  className="font-mono text-xs"
                />
              )}
            </Field>
          </div>
        ) : null}

        {type === 'redirectScheme' ? (
          <div className="flex items-end gap-4">
            <Field label="Scheme">
              {(field) => (
                <Select {...field} value={scheme} onChange={(event) => setScheme(event.target.value)}>
                  <option value="https">https</option>
                  <option value="http">http</option>
                </Select>
              )}
            </Field>
            <div className="flex items-center gap-2 pb-2">
              <Switch id="permanent" checked={permanent} onCheckedChange={setPermanent} />
              <label className="text-xs text-muted" htmlFor="permanent">
                Permanent (301)
              </label>
            </div>
          </div>
        ) : null}

        {type === 'headers' ? (
          <Field label="Headers" description="One per line, Header: value">
            {(field) => (
              <Textarea
                {...field}
                value={headers}
                onChange={(event) => setHeaders(event.target.value)}
                rows={4}
                className="font-mono text-xs"
                placeholder={'X-Frame-Options: DENY\nStrict-Transport-Security: max-age=31536000'}
              />
            )}
          </Field>
        ) : null}

        {type === 'rateLimit' ? (
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Average (req/s)">
              {(field) => (
                <Input
                  {...field}
                  value={average}
                  onChange={(event) => setAverage(event.target.value)}
                  inputMode="numeric"
                  className="font-mono text-xs"
                />
              )}
            </Field>
            <Field label="Burst">
              {(field) => (
                <Input
                  {...field}
                  value={burst}
                  onChange={(event) => setBurst(event.target.value)}
                  inputMode="numeric"
                  className="font-mono text-xs"
                />
              )}
            </Field>
          </div>
        ) : null}

        {type === 'stripPrefix' ? (
          <Field label="Prefixes" description="Comma separated, for example /api">
            {(field) => (
              <Input
                {...field}
                value={prefixes}
                onChange={(event) => setPrefixes(event.target.value)}
                className="font-mono text-xs"
              />
            )}
          </Field>
        ) : null}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={save.isPending}>
            Save middleware
          </Button>
        </div>
      </form>
    </Modal>
  )
}

export function ProxyPage() {
  const queryClient = useQueryClient()
  const [deleteTarget, setDeleteTarget] = useState<ProxyMiddleware | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [settings, setSettings] = useState({ panel_domain: '', acme_email: '' })
  const [settingsLoaded, setSettingsLoaded] = useState(false)

  const overview = useQuery({
    queryKey: ['proxy', 'status'],
    queryFn: api.proxyStatus,
    refetchInterval: 15_000,
  })

  const routes = useQuery({
    queryKey: ['proxy', 'routes'],
    queryFn: api.proxyRoutes,
    enabled: overview.data?.status.api_reachable ?? false,
    refetchInterval: 20_000,
  })

  const certificates = useQuery({
    queryKey: ['proxy', 'certificates'],
    queryFn: api.proxyCertificates,
  })

  const middlewares = useQuery({
    queryKey: ['proxy', 'middlewares'],
    queryFn: api.proxyMiddlewares,
  })

  const logs = useQuery({
    queryKey: ['proxy', 'logs'],
    queryFn: () => api.proxyLogs(200),
    enabled: overview.data?.status.running ?? false,
  })

  if (overview.data && !settingsLoaded) {
    setSettings({
      panel_domain: overview.data.settings.panel_domain ?? '',
      acme_email: overview.data.settings.acme_email ?? '',
    })
    setSettingsLoaded(true)
  }

  const lifecycle = useMutation({
    mutationFn: (action: 'ensure' | 'start' | 'stop' | 'restart') => api.proxyLifecycle(action),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['proxy'] })
      toast.success(result.message)
    },
    onError: (error) =>
      toast.error('Proxy action failed', error instanceof ApiError ? error.message : undefined),
  })

  const saveSettings = useMutation({
    mutationFn: () => api.updateProxySettings(settings),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['proxy'] })
      toast.success('Proxy settings saved')
    },
    onError: (error) =>
      toast.error('Could not save the proxy settings', error instanceof ApiError ? error.message : undefined),
  })

  const removeMiddleware = useMutation({
    mutationFn: (name: string) => api.deleteProxyMiddleware(name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['proxy'] })
      toast.success('Middleware deleted')
      setDeleteTarget(null)
    },
    onError: (error) =>
      toast.error('Could not delete the middleware', error instanceof ApiError ? error.message : undefined),
  })

  const status = overview.data?.status

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Proxy"
        description="Traefik terminates TLS and routes every domain to its container."
        actions={
          <>
            {status?.installed ? (
              <>
                <Button
                  variant="secondary"
                  size="sm"
                  loading={lifecycle.isPending}
                  onClick={() => lifecycle.mutate(status.running ? 'stop' : 'start')}
                >
                  {status.running ? (
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
                <Button
                  variant="secondary"
                  size="sm"
                  loading={lifecycle.isPending}
                  onClick={() => lifecycle.mutate('restart')}
                >
                  <IconRefresh aria-hidden className="size-3.5" />
                  Restart
                </Button>
              </>
            ) : (
              <Button
                variant="primary"
                size="sm"
                loading={lifecycle.isPending}
                onClick={() => lifecycle.mutate('ensure')}
              >
                <IconTool aria-hidden className="size-3.5" />
                Set up proxy
              </Button>
            )}
          </>
        }
      />

      {overview.isPending ? <Skeleton className="h-32" /> : null}

      {status ? (
        <div className="grid gap-4 lg:grid-cols-[1.4fr_1fr]">
          <Card>
            <CardContent className="flex flex-col gap-3">
              <div className="flex flex-wrap items-center gap-2">
                <StatusPill status={status} />
                {status.version ? <Badge variant="outline">traefik {status.version}</Badge> : null}
                {status.api_reachable ? <Badge variant="info">API</Badge> : null}
              </div>
              <div className="grid gap-2 text-xs sm:grid-cols-2">
                <div className="flex flex-col gap-0.5">
                  <span className="text-2xs font-medium text-subtle">Container</span>
                  <span className="font-mono text-foreground">{status.container ?? '—'}</span>
                </div>
                <div className="flex flex-col gap-0.5">
                  <span className="text-2xs font-medium text-subtle">Image</span>
                  <span className="font-mono text-foreground">{status.image ?? '—'}</span>
                </div>
                <div className="flex flex-col gap-0.5">
                  <span className="text-2xs font-medium text-subtle">Entrypoints</span>
                  <span className="text-foreground">
                    {overview.data?.settings.http_port ?? 80} → 80, {overview.data?.settings.https_port ?? 443} → 443
                  </span>
                </div>
                <div className="flex flex-col gap-0.5">
                  <span className="text-2xs font-medium text-subtle">Dashboard</span>
                  {status.running ? (
                    <a
                      href="http://127.0.0.1:8080/dashboard/"
                      target="_blank"
                      rel="noreferrer"
                      className="inline-flex items-center gap-1 text-accent hover:underline"
                    >
                      127.0.0.1:8080 <IconExternalLink aria-hidden className="size-3" />
                    </a>
                  ) : (
                    <span className="text-subtle">—</span>
                  )}
                </div>
              </div>
              {!status.installed ? (
                <p className="rounded-lg border border-border bg-surface-raised p-2 text-2xs text-muted">
                  The proxy runs as a Traefik container on the ebox network. Set it up to publish
                  applications and the panel over HTTPS.
                </p>
              ) : null}
            </CardContent>
          </Card>

          <Card>
            <div className="flex flex-col gap-1 p-4 pb-2">
              <CardTitle>Proxy settings</CardTitle>
              <CardDescription>Used for the panel route and certificate requests.</CardDescription>
            </div>
            <form
              className="flex flex-col gap-3 p-4 pt-2"
              onSubmit={(event) => {
                event.preventDefault()
                saveSettings.mutate()
              }}
            >
              <Field label="Panel domain" description="host.docker.internal is used to reach the panel.">
                {(field) => (
                  <Input
                    {...field}
                    value={settings.panel_domain}
                    onChange={(event) => setSettings({ ...settings, panel_domain: event.target.value })}
                    placeholder="panel.example.com"
                    className="font-mono text-xs"
                  />
                )}
              </Field>
              <Field label="ACME email" description="Let's Encrypt account email.">
                {(field) => (
                  <Input
                    {...field}
                    value={settings.acme_email}
                    onChange={(event) => setSettings({ ...settings, acme_email: event.target.value })}
                    placeholder="ops@example.com"
                    className="font-mono text-xs"
                  />
                )}
              </Field>
              <div className="flex justify-end">
                <Button type="submit" variant="primary" size="sm" loading={saveSettings.isPending}>
                  Save and reload proxy
                </Button>
              </div>
            </form>
          </Card>
        </div>
      ) : null}

      <Section
        title="Routes"
        description="Routers published by the Docker provider (applications) and the file provider (panel, middlewares)."
      >
        {!status?.api_reachable ? (
          <EmptyState
            icon={<IconNetwork aria-hidden />}
            title="Proxy API not reachable"
            description="Start the proxy to inspect the live routing table."
          />
        ) : null}

        {status?.api_reachable && routes.data && routes.data.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Router</TableHead>
                <TableHead>Rule</TableHead>
                <TableHead>Service</TableHead>
                <TableHead>Entrypoints</TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>TLS</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {routes.data.map((route) => (
                <TableRow key={route.name}>
                  <TableCell className="font-mono text-xs whitespace-nowrap text-foreground">
                    {route.name}
                  </TableCell>
                  <TableCell className="max-w-80 truncate font-mono text-2xs text-muted">
                    {route.rule}
                  </TableCell>
                  <TableCell className="font-mono text-2xs text-muted">{route.service}</TableCell>
                  <TableCell className="text-2xs text-muted">{route.entrypoints.join(', ')}</TableCell>
                  <TableCell>
                    <Badge variant={route.provider === 'docker' ? 'info' : 'outline'}>{route.provider}</Badge>
                  </TableCell>
                  <TableCell>
                    {route.tls ? (
                      <span className="inline-flex items-center gap-1 text-2xs text-success">
                        <IconLock aria-hidden className="size-3" />
                        on
                      </span>
                    ) : (
                      <span className="text-2xs text-subtle">off</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : null}
      </Section>

      <Section title="Certificates" description="Parsed from the ACME store.">
        {certificates.data && certificates.data.length === 0 ? (
          <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
            No certificates yet. They are requested automatically for HTTPS domains once a domain
            points at this server.
          </p>
        ) : null}

        {certificates.data && certificates.data.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Domain</TableHead>
                <TableHead>SANs</TableHead>
                <TableHead>Issuer</TableHead>
                <TableHead>Expires</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {certificates.data.map((certificate) => (
                <TableRow key={certificate.domain}>
                  <TableCell className="font-mono text-xs text-foreground">{certificate.domain}</TableCell>
                  <TableCell className="text-2xs text-muted">{certificate.sans.join(', ')}</TableCell>
                  <TableCell className="text-2xs text-muted">{certificate.issuer}</TableCell>
                  <TableCell className="text-2xs whitespace-nowrap">
                    <Badge variant={certificate.days_left > 21 ? 'success' : 'warning'}>
                      {certificate.days_left} days
                    </Badge>
                    <span className="ms-2 text-subtle">{formatDateTime(certificate.not_after)}</span>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : null}
      </Section>

      <Section
        title="Middlewares"
        description="File-provider middlewares are editable here; Docker-provider ones come from container labels."
        actions={
          <Button variant="primary" size="sm" onClick={() => setDialogOpen(true)}>
            <IconPlus aria-hidden className="size-3.5" />
            New middleware
          </Button>
        }
      >
        {middlewares.data && middlewares.data.length === 0 ? (
          <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
            No middlewares defined yet.
          </p>
        ) : null}

        {middlewares.data && middlewares.data.length > 0 ? (
          <div className="flex flex-col gap-2">
            {middlewares.data.map((middleware) => (
              <div
                key={`${middleware.provider}-${middleware.name}`}
                className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5"
              >
                <div className="flex min-w-0 items-center gap-2">
                  <IconShield aria-hidden className="size-3.5 text-subtle" />
                  <span className="font-mono text-xs text-foreground">{middleware.name}</span>
                  <Badge variant="outline">{middleware.type}</Badge>
                  <Badge variant={middleware.provider === 'file' ? 'info' : 'outline'}>
                    {middleware.provider}
                  </Badge>
                </div>
                <div className="flex items-center gap-2">
                  <span className="max-w-72 truncate font-mono text-2xs text-subtle">
                    {JSON.stringify(middleware.config)}
                  </span>
                  {middleware.editable ? (
                    <Button
                      variant="ghost"
                      size="sm"
                      aria-label={`Delete ${middleware.name}`}
                      onClick={() => setDeleteTarget(middleware)}
                    >
                      <IconTrash aria-hidden className="size-3.5 text-danger" />
                    </Button>
                  ) : null}
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </Section>

      <Section title="Proxy logs" description="Last 200 lines from the Traefik container.">
        <div className="h-64 overflow-auto rounded-control border border-border bg-[#0a0a0d] px-3 py-2">
          <pre className="font-mono text-2xs leading-5 whitespace-pre-wrap text-[#c9c9cd]">
            {(logs.data?.lines ?? []).join('\n') || (status?.running ? 'Waiting for output…' : 'Proxy is not running.')}
          </pre>
        </div>
      </Section>

      <MiddlewareDialog open={dialogOpen} onOpenChange={setDialogOpen} />

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null)
          }
        }}
        title={`Delete ${deleteTarget?.name ?? ''}?`}
        description="Traefik reloads the file provider immediately; routers using this middleware lose it."
        confirmLabel="Delete middleware"
        danger
        loading={removeMiddleware.isPending}
        onConfirm={() => {
          if (deleteTarget) {
            removeMiddleware.mutate(deleteTarget.name)
          }
        }}
      />
    </div>
  )
}
