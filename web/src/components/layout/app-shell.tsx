import { Link, Outlet } from '@tanstack/react-router'

import { LiveUpdates } from '@/components/layout/live-updates'
import { MobileNav, Sidebar } from '@/components/layout/sidebar'
import { StreamIndicator } from '@/components/layout/stream-indicator'
import { ThemeToggle } from '@/components/layout/theme-toggle'
import { UserMenu } from '@/components/layout/user-menu'
import { LogoMark } from '@/components/ui/logo'
import { useEventStream } from '@/hooks/use-event-stream'
import { useSystemInfo } from '@/hooks/use-metrics'

export function AppShell() {
  const system = useSystemInfo()
  useEventStream()

  return (
    <div className="flex min-h-screen bg-background">
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:start-3 focus:top-3 focus:z-50 focus:rounded-control focus:bg-surface focus:px-3 focus:py-2 focus:text-ui focus:font-medium focus:text-foreground focus:shadow-popover"
      >
        Skip to content
      </a>
      <LiveUpdates />
      <Sidebar version={system.data?.panel_version} />

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 flex h-14 items-center gap-2 bg-background/80 px-4 backdrop-blur-md sm:px-6">
          <Link to="/" className="me-auto flex items-center gap-2 md:hidden">
            <LogoMark className="size-6" />
            <span className="text-sm font-semibold tracking-[-0.02em] text-foreground">ebox</span>
          </Link>

          <div className="ms-auto flex items-center gap-1.5">
            <StreamIndicator />
            <ThemeToggle />
            <UserMenu compact className="md:hidden" />
          </div>
        </header>

        <MobileNav />

        <main id="main" className="mx-auto w-full max-w-[1400px] flex-1 px-4 py-6 sm:px-6 sm:py-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
