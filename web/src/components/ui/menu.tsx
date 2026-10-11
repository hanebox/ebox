import { Menu as BaseMenu } from '@base-ui/react/menu'
import type { ReactElement, ReactNode } from 'react'

import { cn } from '@/lib/cn'

export interface DropdownMenuProps {
  trigger: ReactElement
  children: ReactNode
  align?: 'start' | 'center' | 'end'
  sideOffset?: number
}

export function DropdownMenu({
  trigger,
  children,
  align = 'end',
  sideOffset = 6,
}: DropdownMenuProps) {
  return (
    <BaseMenu.Root>
      <BaseMenu.Trigger render={trigger} />
      <BaseMenu.Portal>
        <BaseMenu.Positioner align={align} sideOffset={sideOffset} className="z-50">
          <BaseMenu.Popup
            className={cn(
              'min-w-44 rounded-control border border-border bg-surface-overlay p-1 shadow-popover outline-none',
              'origin-[var(--transform-origin)] transition-[opacity,transform] duration-100 ease-out-quint',
              'data-ending-style:scale-[0.97] data-ending-style:opacity-0 data-starting-style:scale-[0.97] data-starting-style:opacity-0',
            )}
          >
            <BaseMenu.Viewport>{children}</BaseMenu.Viewport>
          </BaseMenu.Popup>
        </BaseMenu.Positioner>
      </BaseMenu.Portal>
    </BaseMenu.Root>
  )
}

export interface DropdownItemProps {
  children: ReactNode
  onClick?: () => void
  danger?: boolean
  disabled?: boolean
  closeOnClick?: boolean
}

export function DropdownItem({
  children,
  onClick,
  danger = false,
  disabled = false,
  closeOnClick = true,
}: DropdownItemProps) {
  return (
    <BaseMenu.Item
      onClick={closeOnClick ? onClick : undefined}
      disabled={disabled}
      className={cn(
        'hit-area flex cursor-pointer items-center gap-2 rounded-tile px-2 py-1.5 text-ui outline-none',
        'data-highlighted:bg-surface-raised',
        danger ? 'text-danger data-highlighted:text-danger' : 'text-foreground',
        'data-disabled:pointer-events-none data-disabled:opacity-50',
        '[&_svg]:size-3.5 [&_svg]:text-subtle',
        danger && '[&_svg]:text-danger',
      )}
    >
      {children}
    </BaseMenu.Item>
  )
}

export function DropdownSeparator() {
  return <BaseMenu.Separator className="my-1 h-px bg-border" />
}

export function DropdownLabel({ children }: { children: ReactNode }) {
  return (
    <BaseMenu.GroupLabel className="truncate px-2 py-1 text-2xs text-subtle">
      {children}
    </BaseMenu.GroupLabel>
  )
}
