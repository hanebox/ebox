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
      <BaseTabs.Indicator className="absolute bottom-0 left-0 h-0.5 rounded-full bg-accent transition-all duration-150" />
    </BaseTabs.List>
  )
}

export function TabsTab({ value, children }: { value: string; children: ReactNode }) {
  return (
    <BaseTabs.Tab
      value={value}
      className={cn(
        'relative -mb-px inline-flex h-8.5 cursor-default items-center gap-1.5 border-b-2 border-transparent px-3 text-sm text-muted transition-colors select-none',
        'hover:text-foreground',
        'data-selected:border-accent data-selected:text-foreground',
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
