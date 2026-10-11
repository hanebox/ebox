import { Link } from '@tanstack/react-router'
import {
  Archive,
  Boxes,
  Database,
  FolderKanban,
  GitBranch,
  Network,
  LayoutDashboard,
  Rocket,
  Settings as SettingsIcon,
  SquareTerminal,
  type LucideIcon,
} from 'lucide-react'

import { Tooltip } from '@/components/ui/tooltip'
import { cn } from '@/lib/cn'

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  exact?: boolean
  soon?: string
}

interface NavGroup {
  label: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    label: 'Overview',
    items: [
      { to: '/', label: 'Dashboard', icon: LayoutDashboard, exact: true },
      { to: '/deployments', label: 'Deployments', icon: Rocket },
    ],
  },
  {
    label: 'Infrastructure',
    items: [
      { to: '/projects', label: 'Projects', icon: FolderKanban },
      { to: '/databases', label: 'Databases', icon: Database },
      { to: '/services', label: 'Services', icon: Boxes },
      { to: '/backups', label: 'Backups', icon: Archive },
      { to: '/sources', label: 'Sources', icon: GitBranch },
      { to: '/proxy', label: 'Proxy', icon: Network },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/terminal', label: 'Terminal', icon: SquareTerminal },
      { to: '/settings', label: 'Settings', icon: SettingsIcon },
    ],
  },
]

function NavLink({ item }: { item: NavItem }) {
  const Icon = item.icon

  const link = (
    <Link
      to={item.to}
      activeOptions={{ exact: item.exact ?? false }}
      className={cn(
        'group flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-sm text-muted transition-colors',
        'hover:bg-surface-raised hover:text-foreground',
        'data-[status=active]:bg-accent-soft data-[status=active]:font-medium data-[status=active]:text-foreground',
      )}
    >
      <Icon aria-hidden className="size-4 shrink-0" />
      <span className="truncate">{item.label}</span>
      {item.soon ? (
        <span className="ml-auto rounded border border-border px-1 text-[10px] leading-4 text-subtle">
          {item.soon}
        </span>
      ) : null}
    </Link>
  )

  if (!item.soon) {
    return link
  }

  return (
    <Tooltip content={`${item.label} arrives in milestone ${item.soon}`} side="right">
      {link}
    </Tooltip>
  )
}

export function Sidebar({ version }: { version?: string }) {
  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r border-border bg-surface md:flex">
      <div className="flex h-14 items-center gap-2.5 border-b border-border px-4">
        <div className="flex size-7 items-center justify-center rounded-lg bg-accent text-sm font-bold text-accent-foreground">
          e
        </div>
        <div className="flex items-baseline gap-1.5">
          <span className="text-sm font-semibold tracking-tight text-foreground">ebox</span>
          {version ? (
            <span className="font-mono text-[10px] text-subtle">{version}</span>
          ) : null}
        </div>
      </div>

      <nav className="flex flex-1 flex-col gap-5 overflow-y-auto p-3">
        {navGroups.map((group) => (
          <div key={group.label} className="flex flex-col gap-1">
            <p className="px-2.5 pb-1 text-[10px] font-semibold tracking-wider text-subtle uppercase">
              {group.label}
            </p>
            {group.items.map((item) => (
              <NavLink key={item.to} item={item} />
            ))}
          </div>
        ))}
      </nav>

      <div className="border-t border-border p-3">
        <p className="px-2.5 text-[10px] text-subtle">
          Personal server control plane
        </p>
      </div>
    </aside>
  )
}

export function MobileNav() {
  const items = navGroups.flatMap((group) => group.items)

  return (
    <nav className="flex gap-1 overflow-x-auto border-b border-border bg-surface px-2 py-2 md:hidden">
      {items.map((item) => (
        <NavLink key={item.to} item={item} />
      ))}
    </nav>
  )
}
