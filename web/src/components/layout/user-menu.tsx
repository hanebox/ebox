import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { LogOut, Settings, UserRound } from 'lucide-react'

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

export function UserMenu() {
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

  return (
    <DropdownMenu
      trigger={
        <Button variant="ghost" size="sm" className="gap-2 px-1.5">
          <span className="flex size-5.5 items-center justify-center rounded-full bg-surface-raised text-[10px] font-semibold text-muted">
            {initials || <UserRound aria-hidden className="size-3" />}
          </span>
          <span className="hidden max-w-28 truncate sm:inline">{user.name}</span>
        </Button>
      }
    >
      <DropdownLabel>{user.email}</DropdownLabel>
      <DropdownSeparator />
      <DropdownItem onClick={() => navigate({ to: '/settings' })}>
        <Settings aria-hidden className="size-3.5" />
        Settings
      </DropdownItem>
      <DropdownItem danger onClick={() => logout.mutate()}>
        <LogOut aria-hidden className="size-3.5" />
        Sign out
      </DropdownItem>
    </DropdownMenu>
  )
}
