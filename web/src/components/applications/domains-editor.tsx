import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Globe, Info, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { Domain } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Modal } from '@/components/ui/dialog'
import { EmptyState } from '@/components/ui/empty-state'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

export function DomainsEditor({
  applicationId,
  domains,
  onChange,
  resource = 'applications',
  uuid,
  live = true,
}: {
  applicationId?: string
  domains?: Domain[]
  onChange?: () => void
  resource?: 'applications' | 'services'
  uuid?: string
  /** Fetch the domain list from the API instead of using the passed value. */
  live?: boolean
}) {
  const targetId = uuid ?? applicationId ?? ''
  const liveDomains = useQuery({
    queryKey: ['resource', resource, targetId, 'domains'],
    queryFn: () => api.resourceDomains(resource, targetId),
    enabled: live,
  })
  const list = live ? (liveDomains.data ?? []) : (domains ?? [])
  const queryClient = useQueryClient()
  const [addOpen, setAddOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Domain | null>(null)
  const [draft, setDraft] = useState({ host: '', https: true, is_primary: false })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['resource', resource, targetId, 'domains'] })
    void queryClient.invalidateQueries({ queryKey: ['application', applicationId] })
    void queryClient.invalidateQueries({ queryKey: ['service', uuid] })
    onChange?.()
  }

  const add = useMutation({
    mutationFn: () => api.createResourceDomain(resource, targetId, draft),
    onSuccess: () => {
      invalidate()
      setAddOpen(false)
      setDraft({ host: '', https: true, is_primary: false })
      toast.success('Domain added', 'Routing applies on the next deployment.')
    },
    onError: (error) =>
      toast.error('Could not add the domain', error instanceof ApiError ? error.message : undefined),
  })

  const remove = useMutation({
    mutationFn: (id: number) => api.deleteResourceDomain(resource, targetId, id),
    onSuccess: () => {
      invalidate()
      toast.success('Domain removed')
      setDeleteTarget(null)
    },
    onError: (error) =>
      toast.error('Could not remove the domain', error instanceof ApiError ? error.message : undefined),
  })

  const makePrimary = useMutation({
    mutationFn: (domain: Domain) =>
      api.updateResourceDomain(resource, targetId, domain.id, {
        host: domain.host,
        https: domain.https,
        is_primary: true,
      }),
    onSuccess: () => {
      invalidate()
      toast.success('Primary domain updated')
    },
    onError: (error) =>
      toast.error('Could not update the domain', error instanceof ApiError ? error.message : undefined),
  })

  return (
    <Card className="max-w-3xl">
      <div className="flex flex-wrap items-start justify-between gap-3 p-4 pb-2">
        <div className="flex flex-col gap-1">
          <CardTitle>Domains</CardTitle>
          <CardDescription>
            Traffic is routed by Traefik through the <span className="font-mono">ebox</span> network
            once the proxy milestone lands.
          </CardDescription>
        </div>
        <Button variant="secondary" size="sm" onClick={() => setAddOpen(true)}>
          <Plus aria-hidden className="size-3.5" />
          Add domain
        </Button>
      </div>

      <div className="p-4 pt-2">
        {live && liveDomains.isPending ? <Skeleton className="h-8" /> : null}

        {list.length === 0 ? (
          <EmptyState
            icon={<Globe aria-hidden />}
            title="No domains yet"
            description="Add a hostname to publish this application."
          />
        ) : (
          <div className="flex flex-col divide-y divide-border/60 rounded-lg border border-border">
            {list.map((domain) => (
              <div key={domain.id} className="flex flex-wrap items-center justify-between gap-3 px-3 py-2.5">
                <div className="flex items-center gap-2">
                  <Globe aria-hidden className="size-3.5 text-subtle" />
                  <span className="font-mono text-xs text-foreground">{domain.host}</span>
                  {domain.https ? <Badge variant="success">https</Badge> : <Badge variant="outline">http</Badge>}
                  {domain.is_primary ? <Badge variant="info">primary</Badge> : null}
                </div>
                <div className="flex items-center gap-1">
                  {!domain.is_primary ? (
                    <Button
                      variant="ghost"
                      size="sm"
                      loading={makePrimary.isPending}
                      onClick={() => makePrimary.mutate(domain)}
                    >
                      Make primary
                    </Button>
                  ) : null}
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label={`Delete ${domain.host}`}
                    onClick={() => setDeleteTarget(domain)}
                  >
                    <Trash2 aria-hidden className="size-3.5 text-danger" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        )}

        <p className="mt-3 flex items-start gap-2 text-[11px] text-subtle">
          <Info aria-hidden className="mt-0.5 size-3.5 shrink-0" />
          Point an A record at this server. Certificates are issued automatically once Traefik is
          connected (proxy milestone).
        </p>
      </div>

      <Modal open={addOpen} onOpenChange={setAddOpen} title="Add domain" size="sm">
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!draft.host.trim()) {
              toast.error('A hostname is required')
              return
            }
            add.mutate()
          }}
        >
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium text-foreground" htmlFor="domain-host">
              Hostname
            </label>
            <Input
              id="domain-host"
              value={draft.host}
              onChange={(event) => setDraft({ ...draft, host: event.target.value })}
              placeholder="app.example.com"
              className="font-mono text-xs"
              autoFocus
            />
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Switch
                id="domain-https"
                checked={draft.https}
                onCheckedChange={(checked) => setDraft({ ...draft, https: checked })}
              />
              <label className="text-xs text-muted" htmlFor="domain-https">
                Request a certificate (HTTPS)
              </label>
            </div>
            <div className="flex items-center gap-2">
              <Switch
                id="domain-primary"
                checked={draft.is_primary}
                onCheckedChange={(checked) => setDraft({ ...draft, is_primary: checked })}
              />
              <label className="text-xs text-muted" htmlFor="domain-primary">
                Primary
              </label>
            </div>
          </div>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => setAddOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={add.isPending}>
              Add domain
            </Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null)
          }
        }}
        title={`Delete ${deleteTarget?.host ?? ''}?`}
        description="The route is removed on the next deployment."
        confirmLabel="Delete domain"
        danger
        loading={remove.isPending}
        onConfirm={() => {
          if (deleteTarget) {
            remove.mutate(deleteTarget.id)
          }
        }}
      />
    </Card>
  )
}
