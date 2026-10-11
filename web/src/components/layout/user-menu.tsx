import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { IconLogout, IconSelector, IconSettings, IconUserCircle } from '@tabler/icons-react'

import { api } from '@/api/endpoints'
import { Button } from '@/components/ui/button'
import {
  DropdownItem,
  DropdownLabel,
  DropdownMenu,
  DropdownSeparator,
} from '@/components/ui/menu'
import { toast } from '@/components/ui/toast'
import { useAuth } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'
import { cn } from '@/lib/cn'

export function UserMenu({ compact = false, className }: { compact?: boolean; className?: string }) {
  const auth = useAuth()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const logout = useMutation({
    mutationFn: () => api.logout(),
    onSuccess: () => {
      queryClient.clear()
      navigate({ to: '/login' })
    },
    onError: (error) => {
      toast.error('Sign out failed', error instanceof ApiError ? error.message : 'Unexpected error')
    },
  })

  const user = auth.data?.user
  if (!user) {
    return null
  }

  const initials = user.name
    .split(/\s+/)
    .map((part) => part[0])
    .filter(Boolean)
    .slice(0, 2)
    .join('')
    .toUpperCase()

  const avatar = (
    <span className="flex size-6 shrink-0 items-center justify-center rounded-full border border-border bg-surface-raised text-2xs font-semibold text-muted">
      {initials || <IconUserCircle aria-hidden className="size-3.5" />}
    </span>
  )

  return (
    <DropdownMenu
      trigger={
        compact ? (
          <Button variant="ghost" size="icon" aria-label="Account menu" className={className}>
            {avatar}
          </Button>
        ) : (
          <Button
            variant="ghost"
            className={cn('h-11 justify-start gap-2.5 rounded-tile px-2', className)}
          >
            {avatar}
            <span className="flex min-w-0 flex-col items-start">
              <span className="w-full truncate text-ui leading-tight font-medium text-foreground">
                {user.name}
              </span>
              <span className="w-full truncate text-2xs leading-tight text-subtle">
                {user.email}
              </span>
            </span>
            <IconSelector aria-hidden className="ms-auto size-3.5 shrink-0 text-subtle" />
          </Button>
        )
      }
    >
      <DropdownLabel>{user.email}</DropdownLabel>
      <DropdownSeparator />
      <DropdownItem onClick={() => navigate({ to: '/settings' })}>
        <IconSettings aria-hidden className="size-3.5" />
        Settings
      </DropdownItem>
      <DropdownItem danger onClick={() => logout.mutate()}>
        <IconLogout aria-hidden className="size-3.5" />
        Sign out
      </DropdownItem>
    </DropdownMenu>
  )
}
