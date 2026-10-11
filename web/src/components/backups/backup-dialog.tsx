import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import { Button } from '@/components/ui/button'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Modal } from '@/components/ui/dialog'
import { Select } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTab } from '@/components/ui/tabs'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { schedulePresets } from '@/components/backups/shared'

export function BackupDialog({
  open,
  onOpenChange,
  databaseUuid,
  databaseName,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** Preselect a database, used from the database detail page. */
  databaseUuid?: string
  databaseName?: string
}) {
  const queryClient = useQueryClient()
  const [mode, setMode] = useState<'database' | 'volume'>('database')
  const [name, setName] = useState(databaseName ? `${databaseName}-nightly` : '')
  const [database, setDatabase] = useState(databaseUuid ?? '')
  const [volume, setVolume] = useState('')
  const [schedule, setSchedule] = useState('0 3 * * *')
  const [retention, setRetention] = useState('7')
  const [enabled, setEnabled] = useState(true)
  const [stopDuring, setStopDuring] = useState(false)

  const targets = useQuery({ queryKey: ['backup-targets'], queryFn: api.backupTargets })

  const create = useMutation({
    mutationFn: () =>
      api.createBackup({
        name,
        target: mode,
        resource_id: mode === 'database' ? database : undefined,
        volume_name: mode === 'volume' ? volume : undefined,
        schedule,
        retention: Number(retention) || 7,
        enabled,
        stop_during_backup: stopDuring,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['backups'] })
      toast.success('Backup scheduled')
      onOpenChange(false)
      setName('')
    },
    onError: (error) =>
      toast.error('Could not create the backup', error instanceof ApiError ? error.message : undefined),
  })

  return (
    <Modal open={open} onOpenChange={onOpenChange} title="New backup" description="Databases and volumes, on a cron schedule." size="lg">
      <form
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault()
          create.mutate()
        }}
      >
        <Tabs value={mode} onValueChange={(value) => setMode(value as 'database' | 'volume')}>
          <TabsList>
            <TabsTab value="database">Database</TabsTab>
            <TabsTab value="volume">Volume</TabsTab>
          </TabsList>
        </Tabs>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Name">
            {(field) => (
              <Input
                {...field}
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="nightly-app-db"
                autoFocus
              />
            )}
          </Field>

          {mode === 'database' ? (
            <Field label="Database">
              {(field) => (
                <Select {...field} value={database} onChange={(event) => setDatabase(event.target.value)}>
                  <option value="">Select a database…</option>
                  {(targets.data?.databases ?? []).map((item) => (
                    <option key={item.uuid} value={item.uuid}>
                      {item.name} ({item.type})
                    </option>
                  ))}
                </Select>
              )}
            </Field>
          ) : (
            <Field label="Volume">
              {(field) => (
                <Select {...field} value={volume} onChange={(event) => setVolume(event.target.value)}>
                  <option value="">Select a volume…</option>
                  {(targets.data?.volumes ?? []).map((item) => (
                    <option key={item.name} value={item.name}>
                      {item.name}
                    </option>
                  ))}
                </Select>
              )}
            </Field>
          )}
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Schedule" description="Standard five field cron.">
            {(field) => (
              <Input
                {...field}
                value={schedule}
                onChange={(event) => setSchedule(event.target.value)}
                placeholder="0 3 * * *"
                className="font-mono text-xs"
              />
            )}
          </Field>
          <Field label="Keep" description="Newest runs to keep.">
            {(field) => (
              <Input
                {...field}
                value={retention}
                onChange={(event) => setRetention(event.target.value.replace(/[^0-9]/g, ''))}
                inputMode="numeric"
                className="font-mono text-xs"
              />
            )}
          </Field>
        </div>

        <div className="flex flex-wrap gap-2">
          {schedulePresets.map((preset) => (
            <Button
              key={preset.value}
              type="button"
              variant={schedule === preset.value ? 'primary' : 'secondary'}
              size="sm"
              onClick={() => setSchedule(preset.value)}
            >
              {preset.label}
            </Button>
          ))}
        </div>

        {mode === 'volume' ? (
          <div className="flex items-center justify-between gap-3 rounded-lg border border-border p-3">
            <div className="flex flex-col">
              <span className="text-xs text-foreground">Stop the container during the backup</span>
              <span className="text-2xs text-subtle">
                Pauses containers using this volume so file level copies are consistent.
              </span>
            </div>
            <Switch checked={stopDuring} onCheckedChange={setStopDuring} aria-label="Stop the container during the backup" />
          </div>
        ) : null}

        <div className="flex items-center justify-between gap-3 rounded-lg border border-border p-3">
          <div className="flex flex-col">
            <span className="text-xs text-foreground">Enabled</span>
            <span className="text-2xs text-subtle">Disabled schedules only run when you press Run now.</span>
          </div>
          <Switch checked={enabled} onCheckedChange={setEnabled} aria-label="Enabled" />
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={create.isPending}>
            Save schedule
          </Button>
        </div>
      </form>
    </Modal>
  )
}

