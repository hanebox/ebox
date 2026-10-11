import { Link } from '@tanstack/react-router'
import {
  IconArchive,
  IconBox,
  IconDatabase,
  IconFolders,
  IconGitBranch,
  IconLayoutDashboard,
  IconNetwork,
  IconRocket,
  IconSettings,
  IconTerminal2,
  type Icon,
} from '@tabler/icons-react'

import { Logo } from '@/components/ui/logo'
import { Tooltip } from '@/components/ui/tooltip'
import { UserMenu } from '@/components/layout/user-menu'
import { cn } from '@/lib/cn'

interface NavItem {
  to: string
  label: string
  icon: Icon
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
      { to: '/', label: 'Dashboard', icon: IconLayoutDashboard, exact: true },
      { to: '/deployments', label: 'Deployments', icon: IconRocket },
    ],
  },
  {
    label: 'Infrastructure',
    items: [
      { to: '/projects', label: 'Projects', icon: IconFolders },
      { to: '/databases', label: 'Databases', icon: IconDatabase },
      { to: '/services', label: 'Services', icon: IconBox },
      { to: '/backups', label: 'Backups', icon: IconArchive },
      { to: '/sources', label: 'Sources', icon: IconGitBranch },
      { to: '/proxy', label: 'Proxy', icon: IconNetwork },
    ],
  },
  {
    label: 'System',
    items: [
      { to: '/terminal', label: 'Terminal', icon: IconTerminal2 },
      { to: '/settings', label: 'Settings', icon: IconSettings },
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
        'group relative flex items-center gap-2.5 rounded-tile px-2.5 py-1.5 text-ui text-muted',
        'transition-colors duration-150 ease-out-quint',
        'hover:bg-surface-raised hover:text-foreground',
        'data-[status=active]:bg-surface-raised data-[status=active]:font-medium data-[status=active]:text-foreground',
        'before:absolute before:start-0 before:top-1/2 before:h-3.5 before:w-[2px] before:-translate-y-1/2 before:rounded-full before:bg-accent before:opacity-0 before:transition-opacity before:duration-150',
        'data-[status=active]:before:opacity-100',
      )}
    >
      <Icon aria-hidden className="size-4 shrink-0 text-subtle transition-colors group-hoverable:text-muted group-data-[status=active]:text-accent" />
      <span className="truncate">{item.label}</span>
      {item.soon ? (
        <span className="ms-auto rounded-full border border-border px-1.5 text-2xs leading-4 text-subtle">
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
    <aside className="sticky top-0 hidden h-screen w-[252px] shrink-0 p-2 pe-0 md:block">
      <div className="flex h-full flex-col rounded-panel border border-border bg-surface">
        <div className="flex h-14 shrink-0 items-center px-4">
          <Logo version={version} />
        </div>

        <nav className="flex flex-1 flex-col gap-6 overflow-y-auto px-1.5 py-1">
          {navGroups.map((group) => (
            <div key={group.label} className="flex flex-col gap-0.5">
              <p className="px-2.5 pb-1.5 text-2xs font-medium text-subtle">{group.label}</p>
              {group.items.map((item) => (
                <NavLink key={item.to} item={item} />
              ))}
            </div>
          ))}
        </nav>

        <div className="border-t border-border p-1.5">
          <UserMenu className="w-full" />
        </div>
      </div>
    </aside>
  )
}

export function MobileNav() {
  const items = navGroups.flatMap((group) => group.items)

  return (
    <nav className="sticky top-14 z-20 flex gap-1 overflow-x-auto border-b border-border bg-background/90 px-3 py-2 backdrop-blur-md md:hidden">
      {items.map((item) => (
        <Link
          key={item.to}
          to={item.to}
          activeOptions={{ exact: item.exact ?? false }}
          className={cn(
            'inline-flex shrink-0 items-center gap-1.5 rounded-full bg-surface px-2.5 py-1 text-xs text-muted shadow-border transition-colors duration-150 hover:text-foreground',
            'data-[status=active]:font-medium data-[status=active]:text-foreground',
          )}
        >
          <item.icon aria-hidden className="size-3.5" />
          {item.label}
        </Link>
      ))}
    </nav>
  )
}
