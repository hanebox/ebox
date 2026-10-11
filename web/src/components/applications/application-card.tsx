import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Boxes, GitBranch, Rocket } from 'lucide-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { ApplicationWithRuntime } from '@/api/types'
import { ApplicationStateBadge } from '@/components/applications/status'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

const packLabels: Record<string, string> = {
  dockerfile: 'Dockerfile',
  static: 'Static',
  image: 'Image',
}

export function ApplicationCard({ application }: { application: ApplicationWithRuntime }) {
  const queryClient = useQueryClient()
  const [deploying, setDeploying] = useState(false)

  const deploy = useMutation({
    mutationFn: () => api.deployApplication(application.uuid),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] })
      void queryClient.invalidateQueries({ queryKey: ['application', application.uuid] })
      void queryClient.invalidateQueries({ queryKey: ['applications'] })
      toast.success('Deployment queued', application.name)
    },
    onError: (error) => {
      toast.error('Could not queue the deployment', error instanceof ApiError ? error.message : undefined)
    },
    onSettled: () => setDeploying(false),
  })

  const runtime = application.runtime

  return (
    <div className="flex flex-col gap-3 rounded-panel border border-border bg-surface p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-lg border border-border bg-surface-raised text-muted">
            {application.kind === 'image' ? (
              <Boxes aria-hidden className="size-4" />
            ) : (
              <Rocket aria-hidden className="size-4" />
            )}
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex flex-wrap items-center gap-2">
              <Link
                to="/applications/$applicationId"
                params={{ applicationId: application.uuid }}
                className="truncate text-sm font-medium text-foreground hover:underline"
              >
                {application.name}
              </Link>
              <Badge variant="outline">{packLabels[application.build_pack] ?? application.build_pack}</Badge>
              <ApplicationStateBadge
                runtimeState={runtime.state}
                deployStatus={runtime.latest_deployment?.status}
                appStatus={application.status}
              />
            </div>
            <p className="flex items-center gap-1.5 truncate font-mono text-[11px] text-muted">
              <GitBranch aria-hidden className="size-3 shrink-0" />
              {application.kind === 'image'
                ? (application.config.image || 'image')
                : `${application.repository} · ${application.branch}`}
            </p>
          </div>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          loading={deploy.isPending}
          onClick={() => {
            setDeploying(true)
            deploy.mutate()
          }}
          disabled={deploying}
        >
          <Rocket aria-hidden className="size-3.5" />
          Deploy
        </Button>
        <Link to="/applications/$applicationId" params={{ applicationId: application.uuid }}>
          <Button variant="ghost" size="sm">
            Open
          </Button>
        </Link>
      </div>
    </div>
  )
}
