import {
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
} from '@tanstack/react-router'

import { AppShell } from '@/components/layout/app-shell'
import { PageLoader } from '@/components/ui/skeleton'
import { authQuery, setupStatusQuery } from '@/hooks/use-auth'
import { ApiError } from '@/lib/api'
import { queryClient } from '@/lib/query'
import { ApplicationDetailPage } from '@/pages/application-detail'
import { DashboardPage } from '@/pages/dashboard'
import { DeploymentDetailPage } from '@/pages/deployment-detail'
import { DeploymentsPage } from '@/pages/deployments'
import { LoginPage } from '@/pages/login'
import { NewApplicationPage } from '@/pages/new-application'
import { ProxyPage } from '@/pages/proxy'
import { TerminalPage } from '@/pages/terminal'
import { NotFoundPage } from '@/pages/not-found'
import { ProjectDetailPage } from '@/pages/project-detail'
import { ProjectsPage } from '@/pages/projects'
import { SettingsPage } from '@/pages/settings'
import { SetupPage } from '@/pages/setup'
import { BackupsPage } from '@/pages/backups'
import { DatabaseDetailPage } from '@/pages/database-detail'
import { DatabasesPage } from '@/pages/databases'
import { ServiceDetailPage } from '@/pages/service-detail'
import { ServicesPage } from '@/pages/services'
import { SourceDetailPage } from '@/pages/source-detail'
import { SourcesPage } from '@/pages/sources'

async function ensureInitialized() {
  return queryClient.ensureQueryData(setupStatusQuery)
}

async function requireAuth() {
  const status = await ensureInitialized()
  if (!status.initialized) {
    throw redirect({ to: '/setup' })
  }

  if (queryClient.getQueryData(authQuery.queryKey)) {
    return
  }

  try {
    await queryClient.ensureQueryData(authQuery)
  } catch (error) {
    queryClient.removeQueries({ queryKey: authQuery.queryKey })
    if (error instanceof ApiError && error.status === 401) {
      throw redirect({ to: '/login' })
    }
    throw error
  }
}

const rootRoute = createRootRoute({
  component: () => <Outlet />,
  notFoundComponent: NotFoundPage,
})

const appLayoutRoute = createRoute({
  id: 'app',
  getParentRoute: () => rootRoute,
  component: AppShell,
  beforeLoad: requireAuth,
})

const dashboardRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/',
  component: DashboardPage,
})

const settingsRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/settings',
  component: SettingsPage,
})

const projectsRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/projects',
  component: ProjectsPage,
})

const projectDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/projects/$projectId',
  component: ProjectDetailPage,
})

const sourcesRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/sources',
  component: SourcesPage,
})

const sourceDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/sources/$sourceId',
  component: SourceDetailPage,
})

const deploymentsRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/deployments',
  component: DeploymentsPage,
})

const deploymentDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/deployments/$deploymentId',
  component: DeploymentDetailPage,
})

const newApplicationRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/projects/$projectId/new/application',
  component: NewApplicationPage,
})

const applicationDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/applications/$applicationId',
  component: ApplicationDetailPage,
})

const proxyRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/proxy',
  component: ProxyPage,
})

const terminalRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/terminal',
  component: TerminalPage,
})

const databasesRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/databases',
  component: DatabasesPage,
})

const servicesRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/services',
  component: ServicesPage,
})

const serviceDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/services/$serviceId',
  component: ServiceDetailPage,
})

const databaseDetailRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/databases/$databaseId',
  component: DatabaseDetailPage,
})

const backupsRoute = createRoute({
  getParentRoute: () => appLayoutRoute,
  path: '/backups',
  component: BackupsPage,
})

const setupRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/setup',
  component: SetupPage,
  beforeLoad: async () => {
    const status = await ensureInitialized()
    if (status.initialized) {
      throw redirect({ to: '/' })
    }
  },
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginPage,
  beforeLoad: async () => {
    const status = await ensureInitialized()
    if (!status.initialized) {
      throw redirect({ to: '/setup' })
    }
    if (queryClient.getQueryData(authQuery.queryKey)) {
      throw redirect({ to: '/' })
    }
  },
})

const routeTree = rootRoute.addChildren([
  appLayoutRoute.addChildren([
    dashboardRoute,
    settingsRoute,
    projectsRoute,
    projectDetailRoute,
    newApplicationRoute,
    applicationDetailRoute,
    sourcesRoute,
    sourceDetailRoute,
    deploymentsRoute,
    deploymentDetailRoute,
    proxyRoute,
    terminalRoute,
    databasesRoute,
    databaseDetailRoute,
    servicesRoute,
    serviceDetailRoute,
    backupsRoute,
  ]),
  setupRoute,
  loginRoute,
])

export const router = createRouter({
  routeTree,
  defaultPendingComponent: () => (
    <div className="flex min-h-screen items-center justify-center">
      <PageLoader />
    </div>
  ),
  defaultPreload: 'intent',
  scrollRestoration: true,
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
