import { Tabs as BaseTabs } from '@base-ui/react/tabs'
import type { ReactNode } from 'react'

import { cn } from '@/lib/cn'

export interface TabsProps {
  value: string
  onValueChange: (value: string) => void
  children: ReactNode
  className?: string
}

export function Tabs({ value, onValueChange, children, className }: TabsProps) {
  return (
    <BaseTabs.Root
      value={value}
      onValueChange={(next) => onValueChange(String(next))}
      className={cn('flex flex-col gap-4', className)}
    >
      {children}
    </BaseTabs.Root>
  )
}

export function TabsList({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <BaseTabs.List
      className={cn(
        'relative inline-flex items-center gap-1 border-b border-border pb-px',
        className,
      )}
    >
      {children}
      <BaseTabs.Indicator className="absolute bottom-0 left-[var(--active-tab-left)] h-0.5 w-[var(--active-tab-width)] rounded-full bg-accent transition-[left,width] duration-200 ease-out-quint" />
    </BaseTabs.List>
  )
}

export function TabsTab({ value, children }: { value: string; children: ReactNode }) {
  return (
    <BaseTabs.Tab
      value={value}
      className={cn(
        'hit-area relative -mb-px inline-flex h-8.5 cursor-pointer items-center gap-1.5 border-b-2 border-transparent px-2.5 text-ui text-muted transition-colors duration-150',
        'hover:text-foreground',
        'data-selected:text-foreground',
      )}
    >
      {children}
    </BaseTabs.Tab>
  )
}

export function TabsPanel({
  value,
  children,
  className,
}: {
  value: string
  children: ReactNode
  className?: string
}) {
  return (
    <BaseTabs.Panel value={value} className={cn('outline-none', className)}>
      {children}
    </BaseTabs.Panel>
  )
}
