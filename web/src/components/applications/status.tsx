import { Badge, StatusDot, type BadgeVariant } from '@/components/ui/badge'
import type { ApplicationStatus, DeploymentStatus } from '@/api/types'

const deploymentVariants: Record<DeploymentStatus, { label: string; variant: BadgeVariant; pulse?: boolean }> = {
  queued: { label: 'queued', variant: 'default' },
  building: { label: 'building', variant: 'info', pulse: true },
  deploying: { label: 'deploying', variant: 'info', pulse: true },
  success: { label: 'success', variant: 'success' },
  failed: { label: 'failed', variant: 'danger' },
  cancelled: { label: 'cancelled', variant: 'warning' },
}

export function DeploymentStatusBadge({ status }: { status: DeploymentStatus }) {
  const config = deploymentVariants[status] ?? deploymentVariants.queued

  return (
    <Badge variant={config.variant} className="gap-1.5">
      {config.pulse ? <StatusDot variant={config.variant} pulse /> : null}
      {config.label}
    </Badge>
  )
}

export function DeploymentStatusDot({ status }: { status: DeploymentStatus }) {
  const config = deploymentVariants[status] ?? deploymentVariants.queued

  return <StatusDot variant={config.variant} pulse={config.pulse} />
}

const stateVariants: Record<string, { label: string; variant: BadgeVariant; pulse?: boolean }> = {
  running: { label: 'running', variant: 'success' },
  exited: { label: 'stopped', variant: 'default' },
  created: { label: 'created', variant: 'default' },
  paused: { label: 'paused', variant: 'warning' },
  restarting: { label: 'restarting', variant: 'info', pulse: true },
  dead: { label: 'dead', variant: 'danger' },
  missing: { label: 'not deployed', variant: 'outline' },
  unknown: { label: 'unknown', variant: 'outline' },
  deploying: { label: 'deploying', variant: 'info', pulse: true },
}

export function runtimeLabel(state: string, deploying?: boolean) {
  if (deploying) {
    return stateVariants.deploying
  }

  return stateVariants[state] ?? stateVariants.unknown
}

export function RuntimeStateBadge({ state, deploying }: { state: string; deploying?: boolean }) {
  const config = runtimeLabel(state, deploying)

  return (
    <Badge variant={config.variant} className="gap-1.5">
      <StatusDot variant={config.variant} pulse={config.pulse} />
      {config.label}
    </Badge>
  )
}

export function ApplicationStateBadge({
  runtimeState,
  deployStatus,
  appStatus,
}: {
  runtimeState: string
  deployStatus?: DeploymentStatus
  appStatus?: ApplicationStatus
}) {
  const deploying =
    deployStatus === 'queued' ||
    deployStatus === 'building' ||
    deployStatus === 'deploying' ||
    appStatus === 'deploying'

  return <RuntimeStateBadge state={runtimeState} deploying={deploying} />
}
