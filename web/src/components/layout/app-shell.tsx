import { Outlet } from '@tanstack/react-router'

import { LiveUpdates } from '@/components/layout/live-updates'
import { Sidebar, MobileNav } from '@/components/layout/sidebar'
import { StreamIndicator } from '@/components/layout/stream-indicator'
import { ThemeToggle } from '@/components/layout/theme-toggle'
import { UserMenu } from '@/components/layout/user-menu'
import { useEventStream } from '@/hooks/use-event-stream'
import { useSystemInfo } from '@/hooks/use-metrics'

export function AppShell() {
  const system = useSystemInfo()
  useEventStream()

  return (
    <div className="flex min-h-screen bg-background">
      <LiveUpdates />
      <Sidebar version={system.data?.panel_version} />

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 flex h-14 items-center justify-end gap-2 border-b border-border bg-background/85 px-4 backdrop-blur">
          <StreamIndicator />
          <ThemeToggle />
          <UserMenu />
        </header>

        <MobileNav />

        <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
