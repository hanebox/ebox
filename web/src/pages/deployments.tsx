import { useQuery } from '@tanstack/react-query'
import { IconRocket } from '@tabler/icons-react'
import { useMemo } from 'react'

import { api } from '@/api/endpoints'
import { DeploymentTable } from '@/components/applications/deployment-table'
import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
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

      <div className="flex flex-wrap items-center gap-2">
        <Badge variant="default">{summary.total} recent</Badge>
        <Badge variant={summary.running > 0 ? 'info' : 'default'}>
          {summary.running} in progress
        </Badge>
        <Badge variant={summary.failed > 0 ? 'danger' : 'default'}>{summary.failed} failed</Badge>
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
          <IconRocket aria-hidden className="size-3.5" />
          Nothing deployed yet. Create an application in a project to get started.
        </p>
      ) : null}
    </div>
  )
}
