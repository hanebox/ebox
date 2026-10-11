import { useQuery } from '@tanstack/react-query'
import { Rocket } from 'lucide-react'
import { useMemo } from 'react'

import { api } from '@/api/endpoints'
import { DeploymentTable } from '@/components/applications/deployment-table'
import { PageHeader } from '@/components/layout/page-header'
import { Card } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

export function DeploymentsPage() {
  const deployments = useQuery({
    queryKey: ['deployments', 'global'],
    queryFn: () => api.deployments(undefined, 50),
    refetchInterval: (query) => {
      const list = query.state.data ?? []
      const active = list.some((deployment) =>
        ['queued', 'building', 'deploying'].includes(deployment.status),
      )

      return active ? 4000 : 30_000
    },
  })

  const summary = useMemo(() => {
    const list = deployments.data ?? []

    return {
      total: list.length,
      running: list.filter((deployment) =>
        ['queued', 'building', 'deploying'].includes(deployment.status),
      ).length,
      failed: list.filter((deployment) => deployment.status === 'failed').length,
    }
  }, [deployments.data])

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Deployments"
        description="Every build and release, with streamed logs. Newest first."
      />

      <div className="flex flex-wrap gap-3 text-xs text-muted">
        <span className="rounded-lg border border-border px-3 py-1.5">{summary.total} recent</span>
        <span className="rounded-lg border border-border px-3 py-1.5">
          {summary.running} in progress
        </span>
        <span className="rounded-lg border border-border px-3 py-1.5">{summary.failed} failed</span>
      </div>

      {deployments.isPending ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-10" />
          <Skeleton className="h-10" />
          <Skeleton className="h-10" />
        </div>
      ) : null}

      {deployments.data ? (
        <Card className="p-4">
          <DeploymentTable deployments={deployments.data} showApplication />
        </Card>
      ) : null}

      {deployments.data && deployments.data.length === 0 ? (
        <p className="flex items-center gap-2 text-xs text-subtle">
          <Rocket aria-hidden className="size-3.5" />
          Nothing deployed yet. Create an application in a project to get started.
        </p>
      ) : null}
    </div>
  )
}
