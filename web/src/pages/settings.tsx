import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'

import { api } from '@/api/endpoints'
import type { AuthResponse } from '@/api/types'
import { PageHeader } from '@/components/layout/page-header'
import { ApiTokensSection } from '@/components/settings/api-tokens'
import { NotificationsSection } from '@/components/settings/notifications'
import { UpdateSection } from '@/components/settings/update'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardDescription, CardTitle } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { Field } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsPanel, TabsTab } from '@/components/ui/tabs'
import { toast } from '@/components/ui/toast'
import { authQuery } from '@/hooks/use-auth'
import { usePanelSettings, useSessions, useSystemInfo } from '@/hooks/use-metrics'
import { ApiError } from '@/lib/api'
import { formatDateTime, formatRelative } from '@/lib/format'

const profileSchema = z.object({
  name: z.string().min(1, 'Your name is required.'),
  email: z.email('Enter a valid email address.'),
})

const passwordSchema = z.object({
  current_password: z.string().min(1, 'Enter your current password.'),
  password: z.string().min(10, 'Use at least 10 characters.'),
})

const panelSchema = z.object({
  panel_domain: z.string(),
  acme_email: z.string(),
  timezone: z.string(),
})

type ProfileValues = z.infer<typeof profileSchema>
type PasswordValues = z.infer<typeof passwordSchema>
type PanelValues = z.infer<typeof panelSchema>

export function SettingsPage() {
  const [tab, setTab] = useState('profile')

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Settings" description="Panel, account and security configuration." />

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTab value="profile">Profile</TabsTab>
          <TabsTab value="security">Security</TabsTab>
          <TabsTab value="sessions">Sessions</TabsTab>
          <TabsTab value="tokens">API tokens</TabsTab>
          <TabsTab value="notifications">Notifications</TabsTab>
          <TabsTab value="panel">Panel</TabsTab>
          <TabsTab value="about">About</TabsTab>
        </TabsList>

        <TabsPanel value="profile">
          <ProfileSection />
        </TabsPanel>
        <TabsPanel value="security">
          <SecuritySection />
        </TabsPanel>
        <TabsPanel value="sessions">
          <SessionsSection />
        </TabsPanel>
        <TabsPanel value="tokens">
          <ApiTokensSection />
        </TabsPanel>
        <TabsPanel value="notifications">
          <NotificationsSection />
        </TabsPanel>
        <TabsPanel value="panel">
          <PanelSection />
        </TabsPanel>
        <TabsPanel value="about">
          <div className="flex flex-col gap-4">
            <AboutSection />
            <UpdateSection />
          </div>
        </TabsPanel>
      </Tabs>
    </div>
  )
}

function SectionCard({
  title,
  description,
  children,
  wide = false,
}: {
  title: string
  description?: string
  children: ReactNode
  wide?: boolean
}) {
  return (
    <Card className={wide ? 'max-w-3xl' : 'max-w-xl'}>
      <div className="flex flex-col gap-1 p-4 pb-2">
        <CardTitle>{title}</CardTitle>
        {description ? <CardDescription>{description}</CardDescription> : null}
      </div>
      <div className="p-4 pt-2">{children}</div>
    </Card>
  )
}

function ProfileSection() {
  const queryClient = useQueryClient()
  const user = queryClient.getQueryData<AuthResponse>(authQuery.queryKey)?.user

  const form = useForm<ProfileValues>({
    resolver: zodResolver(profileSchema),
    values: { name: user?.name ?? '', email: user?.email ?? '' },
  })

  const mutation = useMutation({
    mutationFn: (values: ProfileValues) => api.updateAccount(values),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: authQuery.queryKey })
      toast.success('Profile updated')
    },
    onError: (error) => {
      toast.error('Update failed', error instanceof ApiError ? error.message : 'Unexpected error')
    },
  })

  return (
    <SectionCard title="Profile" description="Your display name and sign-in email.">
      <form
        className="flex flex-col gap-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
      >
        <Field label="Name" error={form.formState.errors.name?.message}>
          {({ id, invalid, ...aria }) => (
            <Input id={id} invalid={invalid} {...aria} {...form.register('name')} />
          )}
        </Field>
        <Field label="Email" error={form.formState.errors.email?.message}>
          {({ id, invalid, ...aria }) => (
            <Input id={id} invalid={invalid} type="email" {...aria} {...form.register('email')} />
          )}
        </Field>
        <div className="flex justify-end">
          <Button type="submit" variant="primary" loading={mutation.isPending}>
            Save profile
          </Button>
        </div>
      </form>
    </SectionCard>
  )
}

function SecuritySection() {
  const form = useForm<PasswordValues>({
    resolver: zodResolver(passwordSchema),
    defaultValues: { current_password: '', password: '' },
  })

  const mutation = useMutation({
    mutationFn: (values: PasswordValues) => api.updatePassword(values),
    onSuccess: () => {
      form.reset()
      toast.success('Password updated', 'Other sessions have been signed out.')
    },
    onError: (error) => {
      if (error instanceof ApiError && error.errors?.current_password) {
        form.setError('current_password', { message: error.errors.current_password[0] })
        return
      }
      toast.error('Update failed', error instanceof ApiError ? error.message : 'Unexpected error')
    },
  })

  return (
    <SectionCard
      title="Password"
      description="Changing the password signs out all other sessions."
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
      >
        <Field label="Current password" error={form.formState.errors.current_password?.message}>
          {({ id, invalid, ...aria }) => (
            <Input
              id={id}
              invalid={invalid}
              type="password"
              autoComplete="current-password"
              {...aria}
              {...form.register('current_password')}
            />
          )}
        </Field>
        <Field label="New password" error={form.formState.errors.password?.message}>
          {({ id, invalid, ...aria }) => (
            <Input
              id={id}
              invalid={invalid}
              type="password"
              autoComplete="new-password"
              {...aria}
              {...form.register('password')}
            />
          )}
        </Field>
        <div className="flex justify-end">
          <Button type="submit" variant="primary" loading={mutation.isPending}>
            Update password
          </Button>
        </div>
      </form>
    </SectionCard>
  )
}

function SessionsSection() {
  const sessions = useSessions()
  const queryClient = useQueryClient()
  const [pendingId, setPendingId] = useState<number | null>(null)

  const revoke = useMutation({
    mutationFn: (id: number) => api.revokeSession(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['sessions'] })
      toast.success('Session revoked')
      setPendingId(null)
    },
    onError: (error) => {
      toast.error('Revoke failed', error instanceof ApiError ? error.message : 'Unexpected error')
      setPendingId(null)
    },
  })

  return (
    <SectionCard
      wide
      title="Active sessions"
      description="Every device signed in to this panel. Revoke anything you do not recognize."
    >
      {sessions.isPending ? (
        <div className="flex flex-col gap-2">
          {Array.from({ length: 3 }).map((_, index) => (
            <Skeleton key={index} className="h-8" />
          ))}
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Device</TableHead>
              <TableHead>IP</TableHead>
              <TableHead>Last seen</TableHead>
              <TableHead>Expires</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(sessions.data ?? []).map((session) => (
              <TableRow key={session.id}>
                <TableCell className="text-xs text-muted">
                  <div className="flex items-center gap-2">
                    <span className="max-w-56 truncate">
                      {session.user_agent || 'Unknown client'}
                    </span>
                    {session.current ? <Badge variant="success">this device</Badge> : null}
                  </div>
                </TableCell>
                <TableCell className="font-mono text-xs whitespace-nowrap">
                  {session.ip || '—'}
                </TableCell>
                <TableCell className="text-xs whitespace-nowrap text-muted">
                  {formatRelative(session.last_seen_at)}
                </TableCell>
                <TableCell className="text-xs whitespace-nowrap text-muted">
                  {formatDateTime(session.expires_at)}
                </TableCell>
                <TableCell className="text-right">
                  {session.current ? null : (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-danger"
                      onClick={() => setPendingId(session.id)}
                    >
                      Revoke
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <ConfirmDialog
        open={pendingId !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingId(null)
          }
        }}
        title="Revoke session"
        description="That device will be signed out immediately."
        confirmLabel="Revoke"
        danger
        loading={revoke.isPending}
        onConfirm={() => {
          if (pendingId !== null) {
            revoke.mutate(pendingId)
          }
        }}
      />
    </SectionCard>
  )
}

function PanelSection() {
  const settings = usePanelSettings()
  const queryClient = useQueryClient()

  const form = useForm<PanelValues>({
    resolver: zodResolver(panelSchema),
    values: {
      panel_domain: settings.data?.panel_domain ?? '',
      acme_email: settings.data?.acme_email ?? '',
      timezone: settings.data?.timezone ?? '',
    },
  })

  const mutation = useMutation({
    mutationFn: (values: PanelValues) => api.updateSettings(values),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['settings'] })
      toast.success('Panel settings saved')
    },
    onError: (error) => {
      toast.error('Save failed', error instanceof ApiError ? error.message : 'Unexpected error')
    },
  })

  return (
    <SectionCard
      title="Panel"
      description="Domain and TLS details used when the reverse proxy layer arrives in M3."
    >
      <form
        className="flex flex-col gap-4"
        onSubmit={form.handleSubmit((values) => mutation.mutate(values))}
      >
        <Field
          label="Panel domain"
          description="Public hostname for this panel, e.g. panel.example.com."
        >
          {({ id, invalid, ...aria }) => (
            <Input
              id={id}
              invalid={invalid}
              placeholder="panel.example.com"
              {...aria}
              {...form.register('panel_domain')}
            />
          )}
        </Field>
        <Field
          label="ACME email"
          description="Let's Encrypt account email used when certificates are issued."
        >
          {({ id, invalid, ...aria }) => (
            <Input
              id={id}
              invalid={invalid}
              placeholder="you@example.com"
              {...aria}
              {...form.register('acme_email')}
            />
          )}
        </Field>
        <Field label="Timezone" description="Display timezone for dates in the panel.">
          {({ id, invalid, ...aria }) => (
            <Input
              id={id}
              invalid={invalid}
              placeholder="UTC"
              {...aria}
              {...form.register('timezone')}
            />
          )}
        </Field>
        <div className="flex justify-end">
          <Button type="submit" variant="primary" loading={mutation.isPending}>
            Save settings
          </Button>
        </div>
      </form>
    </SectionCard>
  )
}

function AboutSection() {
  const system = useSystemInfo()

  return (
    <SectionCard title="About" description="Versions of the pieces that make up this panel.">
      <dl className="flex flex-col gap-2 text-sm">
        <AboutRow label="Panel" value={system.data?.panel_version ?? '…'} />
        <AboutRow label="Commit" value={system.data?.panel_commit ?? '…'} />
        <AboutRow label="Go runtime" value={system.data?.go_version ?? '…'} />
        <AboutRow
          label="Docker"
          value={system.data?.docker.available ? (system.data.docker.version ?? '') : 'unavailable'}
        />
        <AboutRow
          label="Platform"
          value={
            system.data ? `${system.data.platform} ${system.data.platform_version}` : '…'
          }
        />
      </dl>
    </SectionCard>
  )
}

function AboutRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="text-xs text-subtle">{label}</dt>
      <dd className="truncate font-mono text-xs text-foreground">{value}</dd>
    </div>
  )
}
