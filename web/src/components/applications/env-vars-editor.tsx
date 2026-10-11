import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Eye, EyeOff, FileUp, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { EnvVar } from '@/api/types'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Copyable } from '@/components/ui/copyable'
import { Modal } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

export function EnvVarsEditor({
  applicationId,
  resource = 'applications',
  uuid,
}: {
  applicationId?: string
  resource?: 'applications' | 'services'
  uuid?: string
}) {
  const targetId: string = uuid ?? applicationId ?? ''
  const queryClient = useQueryClient()
  const [revealed, setRevealed] = useState<Record<number, boolean>>({})
  const [draft, setDraft] = useState({ key: '', value: '', is_build_time: false })
  const [importOpen, setImportOpen] = useState(false)
  const [importContent, setImportContent] = useState('')
  const [exportOpen, setExportOpen] = useState(false)
  const [exportContent, setExportContent] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<EnvVar | null>(null)

  const env = useQuery({
    queryKey: ['resource', resource, targetId, 'env'],
    queryFn: () => api.resourceEnv(resource, targetId),
  })

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['resource', resource, targetId, 'env'] })
  }

  const add = useMutation({
    mutationFn: () => api.createResourceEnv(resource, targetId, draft),
    onSuccess: () => {
      invalidate()
      setDraft({ key: '', value: '', is_build_time: false })
      toast.success('Variable saved', 'Apply it with the next deployment.')
    },
    onError: (error) =>
      toast.error('Could not save the variable', error instanceof ApiError ? error.message : undefined),
  })

  const remove = useMutation({
    mutationFn: (id: number) => api.deleteResourceEnv(resource, targetId, id),
    onSuccess: () => {
      invalidate()
      toast.success('Variable removed')
      setDeleteTarget(null)
    },
    onError: (error) =>
      toast.error('Could not remove the variable', error instanceof ApiError ? error.message : undefined),
  })

  const importVars = useMutation({
    mutationFn: () => api.importResourceEnv(resource, targetId, importContent),
    onSuccess: (result) => {
      invalidate()
      setImportOpen(false)
      setImportContent('')
      toast.success(`${result.imported} variables imported`)
    },
    onError: (error) =>
      toast.error('Import failed', error instanceof ApiError ? error.message : undefined),
  })

  const openExport = async () => {
    try {
      const result = await api.exportResourceEnv(resource, targetId)
      setExportContent(result.content || '')
      setExportOpen(true)
    } catch (error) {
      toast.error('Export failed', error instanceof ApiError ? error.message : undefined)
    }
  }

  return (
    <Card className="max-w-4xl">
      <div className="flex flex-wrap items-start justify-between gap-3 p-4 pb-2">
        <div className="flex flex-col gap-1">
          <CardTitle>Environment variables</CardTitle>
          <CardDescription>
            Injected into the container at deploy time. Build-time variables are also passed to the
            build.
          </CardDescription>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" size="sm" onClick={() => setImportOpen(true)}>
            <FileUp aria-hidden className="size-3.5" />
            Import .env
          </Button>
          <Button variant="ghost" size="sm" onClick={() => void openExport()}>
            Export
          </Button>
        </div>
      </div>

      <div className="p-4 pt-2">
        {env.isPending ? (
          <div className="flex flex-col gap-2">
            <Skeleton className="h-8" />
            <Skeleton className="h-8" />
          </div>
        ) : null}

        {env.data && env.data.length === 0 ? (
          <p className="rounded-lg border border-dashed border-border px-4 py-6 text-center text-xs text-muted">
            No variables yet.
          </p>
        ) : null}

        {env.data && env.data.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Key</TableHead>
                <TableHead>Value</TableHead>
                <TableHead>Scope</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {env.data.map((item) => (
                <TableRow key={item.id}>
                  <TableCell className="font-mono text-xs whitespace-nowrap text-foreground">
                    {item.key}
                  </TableCell>
                  <TableCell className="max-w-80 text-xs text-muted">
                    <div className="flex items-center gap-2">
                      <span className="truncate font-mono">
                        {revealed[item.id] ? item.value : '••••••••'}
                      </span>
                      <button
                        type="button"
                        aria-label={revealed[item.id] ? 'Hide value' : 'Reveal value'}
                        className="shrink-0 text-subtle transition-colors hover:text-foreground"
                        onClick={() =>
                          setRevealed((previous) => ({ ...previous, [item.id]: !previous[item.id] }))
                        }
                      >
                        {revealed[item.id] ? (
                          <EyeOff aria-hidden className="size-3.5" />
                        ) : (
                          <Eye aria-hidden className="size-3.5" />
                        )}
                      </button>
                    </div>
                  </TableCell>
                  <TableCell className="text-xs whitespace-nowrap text-muted">
                    {item.is_build_time ? 'build + runtime' : 'runtime'}
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setDeleteTarget(item)}
                      aria-label={`Delete ${item.key}`}
                    >
                      <Trash2 aria-hidden className="size-3.5 text-danger" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : null}

        <form
          className="mt-4 flex flex-wrap items-end gap-2 border-t border-border pt-4"
          onSubmit={(event) => {
            event.preventDefault()
            if (!draft.key.trim()) {
              toast.error('A key is required')
              return
            }
            add.mutate()
          }}
        >
          <div className="flex min-w-48 flex-1 flex-col gap-1.5">
            <label className="text-xs text-subtle" htmlFor="env-key">
              Key
            </label>
            <Input
              id="env-key"
              value={draft.key}
              onChange={(event) => setDraft({ ...draft, key: event.target.value.toUpperCase() })}
              placeholder="API_KEY"
              className="font-mono text-xs"
            />
          </div>
          <div className="flex min-w-48 flex-1 flex-col gap-1.5">
            <label className="text-xs text-subtle" htmlFor="env-value">
              Value
            </label>
            <Input
              id="env-value"
              value={draft.value}
              onChange={(event) => setDraft({ ...draft, value: event.target.value })}
              placeholder="value"
              className="font-mono text-xs"
            />
          </div>
          <div className="flex items-center gap-2 pb-2">
            <Switch
              id="env-build"
              checked={draft.is_build_time}
              onCheckedChange={(checked) => setDraft({ ...draft, is_build_time: checked })}
            />
            <label className="text-xs text-muted" htmlFor="env-build">
              Build-time
            </label>
          </div>
          <Button type="submit" variant="secondary" loading={add.isPending} className="mb-0.5">
            <Plus aria-hidden className="size-3.5" />
            Add
          </Button>
        </form>
      </div>

      <Modal
        open={importOpen}
        onOpenChange={setImportOpen}
        title="Import .env"
        description="Paste the contents of a .env file. Existing keys are overwritten."
      >
        <div className="flex flex-col gap-4">
          <Textarea
            value={importContent}
            onChange={(event) => setImportContent(event.target.value)}
            rows={8}
            className="font-mono text-xs"
            placeholder={'APP_ENV=production\nDATABASE_URL=postgres://…'}
          />
          <div className="flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setImportOpen(false)}>
              Cancel
            </Button>
            <Button variant="primary" loading={importVars.isPending} onClick={() => importVars.mutate()}>
              Import
            </Button>
          </div>
        </div>
      </Modal>

      <Modal
        open={exportOpen}
        onOpenChange={setExportOpen}
        title="Export .env"
        description="Copy these values into your local environment."
      >
        <div className="flex flex-col gap-4">
          {exportContent ? (
            <Copyable value={exportContent} display="Copy all variables" mono={false} />
          ) : (
            <p className="text-xs text-muted">No variables to export.</p>
          )}
          <Textarea value={exportContent} readOnly rows={8} className="font-mono text-xs" />
          <div className="flex justify-end">
            <Button variant="primary" onClick={() => setExportOpen(false)}>
              Done
            </Button>
          </div>
        </div>
      </Modal>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteTarget(null)
          }
        }}
        title={`Delete ${deleteTarget?.key ?? ''}?`}
        description="The variable is removed from the next deployment."
        confirmLabel="Delete variable"
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
