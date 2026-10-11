import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { IconBellRinging, IconPlus, IconSend, IconTrash } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { NotificationChannel, NotificationChannelType } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { EmptyState } from '@/components/ui/empty-state'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Modal } from '@/components/ui/dialog'
import { Select } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Textarea } from '@/components/ui/textarea'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime } from '@/lib/format'
import { cn } from '@/lib/cn'

const channelTypes: Array<{ value: NotificationChannelType; label: string }> = [
  { value: 'email', label: 'Email (SMTP)' },
  { value: 'discord', label: 'Discord webhook' },
  { value: 'slack', label: 'Slack webhook' },
  { value: 'telegram', label: 'Telegram bot' },
  { value: 'webhook', label: 'Custom webhook' },
]

const eventLabels: Record<string, string> = {
  'deployment.success': 'Deployment succeeded',
  'deployment.failed': 'Deployment failed',
  'backup.success': 'Backup succeeded',
  'backup.failed': 'Backup failed',
  'container.down': 'Container stopped',
  'certificate.expiring': 'Certificate expiring',
  'disk.high': 'Disk almost full',
  'update.available': 'Update available',
}

function fieldsFor(type: NotificationChannelType) {
  switch (type) {
    case 'email':
      return [
        { key: 'host', label: 'SMTP host', placeholder: 'smtp.example.com' },
        { key: 'port', label: 'Port', placeholder: '587' },
        { key: 'username', label: 'Username', placeholder: 'alerts@example.com' },
        { key: 'password', label: 'Password', secret: true },
        { key: 'from', label: 'From address', placeholder: 'ebox@example.com' },
        { key: 'to', label: 'Recipients', placeholder: 'you@example.com, ops@example.com' },
        { key: 'encryption', label: 'Encryption', placeholder: 'starttls, tls or none' },
      ]
    case 'discord':
    case 'slack':
      return [
        { key: 'webhook_url', label: 'Webhook URL', placeholder: 'https://…', secret: type === 'slack' },
        { key: 'username', label: 'Display name (optional)', placeholder: 'ebox' },
      ]
    case 'telegram':
      return [
        { key: 'bot_token', label: 'Bot token', secret: true },
        { key: 'chat_id', label: 'Chat id', placeholder: '-1001234567890' },
      ]
    default:
      return [
        { key: 'webhook_url', label: 'Endpoint URL', placeholder: 'https://example.com/ebox' },
        { key: 'secret', label: 'Signing secret (optional)', secret: true },
      ]
  }
}

function ChannelDialog({
  open,
  onOpenChange,
  channel,
  events,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  channel?: NotificationChannel
  events: string[]
}) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(channel?.name ?? '')
  const [type, setType] = useState<NotificationChannelType>(channel?.type ?? 'discord')
  const [enabled, setEnabled] = useState(channel?.enabled ?? true)
  const [config, setConfig] = useState<Record<string, string>>(channel?.config ?? {})
  const [subscribed, setSubscribed] = useState<string[]>(channel?.subscribed_events ?? [])

  const save = useMutation({
    mutationFn: () =>
      channel
        ? api.updateNotificationChannel(channel.uuid, { name, type, enabled, config, events: subscribed })
        : api.createNotificationChannel({ name, type, enabled, config, events: subscribed }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['notification-channels'] })
      toast.success(channel ? 'Channel updated' : 'Channel added')
      onOpenChange(false)
    },
    onError: (error) =>
      toast.error('Could not save the channel', error instanceof ApiError ? error.message : undefined),
  })

  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title={channel ? `Edit ${channel.name}` : 'New notification channel'}
      description="Secrets are encrypted at rest and never leave the panel in plain text."
      size="lg"
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={(event) => {
          event.preventDefault()
          save.mutate()
        }}
      >
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label="Name">
            {(field) => (
              <Input
                {...field}
                value={name}
                onChange={(event) => setName(event.target.value)}
                placeholder="ops-alerts"
                autoFocus
              />
            )}
          </Field>
          <Field label="Type">
            {(field) => (
              <Select
                {...field}
                value={type}
                onChange={(event) => {
                  setType(event.target.value as NotificationChannelType)
                  setConfig({})
                }}
              >
                {channelTypes.map((item) => (
                  <option key={item.value} value={item.value}>
                    {item.label}
                  </option>
                ))}
              </Select>
            )}
          </Field>
        </div>

        {fieldsFor(type).map((item) => (
          <Field key={item.key} label={item.label}>
            {(field) => (
              <Input
                {...field}
                type={item.secret ? 'password' : 'text'}
                value={config[item.key] ?? ''}
                onChange={(event) => setConfig((current) => ({ ...current, [item.key]: event.target.value }))}
                placeholder={item.placeholder}
                className="font-mono text-xs"
              />
            )}
          </Field>
        ))}

        <div className="flex flex-col gap-2 rounded-lg border border-border p-3">
          <div className="flex items-center justify-between gap-3">
            <div className="flex flex-col">
              <span className="text-xs text-foreground">Events</span>
              <span className="text-2xs text-subtle">None selected means every event.</span>
            </div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setSubscribed(subscribed.length === events.length ? [] : events)}
            >
              {subscribed.length === events.length ? 'Clear' : 'Select all'}
            </Button>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            {events.map((event) => {
              const checked = subscribed.includes(event)

              return (
                <button
                  key={event}
                  type="button"
                  onClick={() =>
                    setSubscribed((current) =>
                      checked ? current.filter((item) => item !== event) : [...current, event],
                    )
                  }
                  className={cn(
                    'flex items-center gap-2 rounded-md border px-2 py-1.5 text-start text-2xs transition-colors',
                    checked
                      ? 'border-accent/60 bg-accent-soft text-foreground'
                      : 'border-border text-muted hover:border-accent/40',
                  )}
                >
                  <span
                    className={cn(
                      'size-1.5 rounded-full',
                      checked ? 'bg-accent' : 'bg-subtle',
                    )}
                  />
                  {eventLabels[event] ?? event}
                </button>
              )
            })}
          </div>
        </div>

        <div className="flex items-center justify-between gap-3 rounded-lg border border-border p-3">
          <div className="flex flex-col">
            <span className="text-xs text-foreground">Enabled</span>
            <span className="text-2xs text-subtle">Disabled channels stay configured but receive nothing.</span>
          </div>
          <Switch checked={enabled} onCheckedChange={setEnabled} aria-label="Enabled" />
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={save.isPending}>
            {channel ? 'Save channel' : 'Add channel'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}

export function NotificationsSection() {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState<NotificationChannel | undefined>(undefined)
  const [deleteTarget, setDeleteTarget] = useState<NotificationChannel | null>(null)
  const [broadcastOpen, setBroadcastOpen] = useState(false)
  const [broadcastTitle, setBroadcastTitle] = useState('')
  const [broadcastBody, setBroadcastBody] = useState('')

  const channels = useQuery({ queryKey: ['notification-channels'], queryFn: api.notificationChannels })
  const events = useQuery({ queryKey: ['notification-events'], queryFn: api.notificationEvents })
  const history = useQuery({
    queryKey: ['notification-history'],
    queryFn: () => api.notificationHistory(25),
    refetchInterval: 20_000,
  })

  const toggle = useMutation({
    mutationFn: (channel: NotificationChannel) => api.toggleNotificationChannel(channel.uuid, !channel.enabled),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['notification-channels'] }),
  })

  const test = useMutation({
    mutationFn: (channel: NotificationChannel) => api.testNotificationChannel(channel.uuid),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['notification-history'] })
      toast.success(result.message)
    },
    onError: (error) =>
      toast.error('Test failed', error instanceof ApiError ? error.message : undefined),
  })

  const remove = useMutation({
    mutationFn: (channel: NotificationChannel) => api.deleteNotificationChannel(channel.uuid),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['notification-channels'] })
      toast.success('Channel deleted')
      setDeleteTarget(null)
    },
  })

  const broadcast = useMutation({
    mutationFn: () => api.sendNotification(broadcastTitle, broadcastBody),
    onSuccess: (result) => {
      toast.success(result.message)
      setBroadcastOpen(false)
      setBroadcastTitle('')
      setBroadcastBody('')
    },
  })

  const rows = channels.data ?? []

  return (
    <div className="flex flex-col gap-4">
      <Card>
        <div className="flex flex-wrap items-start justify-between gap-3 p-4 pb-2">
          <div className="flex flex-col gap-1">
            <CardTitle>Notification channels</CardTitle>
            <CardDescription>
              Where ebox reports deployments, backups, certificates and containers that stop on their own.
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Button variant="secondary" size="sm" onClick={() => setBroadcastOpen(true)}>
              <IconSend aria-hidden className="size-3.5" />
              Send message
            </Button>
            <Button
              variant="primary"
              size="sm"
              onClick={() => {
                setEditing(undefined)
                setDialogOpen(true)
              }}
            >
              <IconPlus aria-hidden className="size-3.5" />
              Add channel
            </Button>
          </div>
        </div>

        <CardContent>
          {channels.isPending ? <Skeleton className="h-24" /> : null}

          {channels.isSuccess && rows.length === 0 ? (
            <EmptyState
              icon={<IconBellRinging aria-hidden />}
              title="No channels yet"
              description="Add a webhook or an email target to hear about failures."
            />
          ) : null}

          <div className="flex flex-col gap-2">
            {rows.map((channel) => (
              <div
                key={channel.uuid}
                className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-border px-3 py-2"
              >
                <div className="flex min-w-0 flex-col">
                  <div className="flex items-center gap-2">
                    <span className="truncate text-xs font-medium text-foreground">{channel.name}</span>
                    <Badge variant="outline">{channel.type}</Badge>
                    {channel.enabled ? <Badge variant="success">on</Badge> : <Badge variant="outline">off</Badge>}
                  </div>
                  <span className="truncate text-2xs text-subtle">
                    {(channel.subscribed_events ?? []).length === 0
                      ? 'all events'
                      : channel.subscribed_events.map((event) => eventLabels[event] ?? event).join(', ')}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <Switch
                    checked={channel.enabled}
                    onCheckedChange={() => toggle.mutate(channel)}
                    aria-label={`Toggle ${channel.name}`}
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    loading={test.isPending && test.variables?.uuid === channel.uuid}
                    onClick={() => test.mutate(channel)}
                  >
                    Test
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setEditing(channel)
                      setDialogOpen(true)
                    }}
                  >
                    Edit
                  </Button>
                  <Button variant="ghost" size="sm" aria-label="Delete channel" onClick={() => setDeleteTarget(channel)}>
                    <IconTrash aria-hidden className="size-3.5" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <div className="flex flex-col gap-1 p-4 pb-2">
          <CardTitle>Recent deliveries</CardTitle>
          <CardDescription>The last 25 messages, including failures.</CardDescription>
        </div>
        <CardContent>
          {history.isPending ? <Skeleton className="h-24" /> : null}
          {history.isSuccess && (history.data?.length ?? 0) === 0 ? (
            <p className="hatch rounded-control border border-border px-4 py-6 text-center text-xs text-muted">
              Nothing delivered yet.
            </p>
          ) : null}
          {(history.data?.length ?? 0) > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>When</TableHead>
                  <TableHead>Channel</TableHead>
                  <TableHead>Event</TableHead>
                  <TableHead>Title</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {history.data?.map((delivery) => (
                  <TableRow key={delivery.id}>
                    <TableCell className="text-2xs text-muted">{formatDateTime(delivery.created_at)}</TableCell>
                    <TableCell className="text-xs text-foreground">{delivery.channel}</TableCell>
                    <TableCell className="font-mono text-2xs text-subtle">{delivery.event}</TableCell>
                    <TableCell className="max-w-[22rem] truncate text-xs text-foreground" title={delivery.title}>
                      {delivery.title}
                    </TableCell>
                    <TableCell>
                      <Badge variant={delivery.status === 'sent' ? 'success' : 'danger'}>
                        {delivery.status}
                        {delivery.status === 'failed' && delivery.attempts > 0 ? ` · ${delivery.attempts}x` : ''}
                      </Badge>
                      {delivery.error ? (
                        <span className="mt-1 block max-w-[16rem] truncate text-2xs text-danger" title={delivery.error}>
                          {delivery.error}
                        </span>
                      ) : null}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : null}
        </CardContent>
      </Card>

      {dialogOpen ? (
        <ChannelDialog
          key={editing?.uuid ?? 'new'}
          open={dialogOpen}
          onOpenChange={setDialogOpen}
          channel={editing}
          events={events.data?.events ?? []}
        />
      ) : null}

      <Modal
        open={broadcastOpen}
        onOpenChange={setBroadcastOpen}
        title="Send a message"
        description="Delivered to every enabled channel."
      >
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            broadcast.mutate()
          }}
        >
          <Field label="Title">
            {(field) => (
              <Input {...field} value={broadcastTitle} onChange={(event) => setBroadcastTitle(event.target.value)} />
            )}
          </Field>
          <Field label="Message">
            {(field) => (
              <Textarea {...field} rows={4} value={broadcastBody} onChange={(event) => setBroadcastBody(event.target.value)} />
            )}
          </Field>
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => setBroadcastOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" variant="primary" loading={broadcast.isPending}>
              Send
            </Button>
          </div>
        </form>
      </Modal>

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={`Delete ${deleteTarget?.name ?? 'channel'}?`}
        description="Its delivery history is removed too."
        confirmLabel="Delete channel"
        danger
        loading={remove.isPending}
        onConfirm={() => deleteTarget && remove.mutate(deleteTarget)}
      />
    </div>
  )
}
