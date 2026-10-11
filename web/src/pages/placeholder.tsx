import type { LucideIcon } from 'lucide-react'

import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { EmptyState } from '@/components/ui/empty-state'

export interface PlaceholderPageProps {
  title: string
  description: string
  milestone: string
  icon: LucideIcon
  features: string[]
}

export function PlaceholderPage({
  title,
  description,
  milestone,
  icon: Icon,
  features,
}: PlaceholderPageProps) {
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title={title}
        description={description}
        actions={<Badge variant="outline">arrives in {milestone}</Badge>}
      />

      <EmptyState
        icon={<Icon />}
        title={`${title} is on the way`}
        description={`This section is scheduled for milestone ${milestone}. The foundation it builds on is already live.`}
      />

      <Card className="p-4">
        <p className="text-xs font-semibold tracking-wider text-subtle uppercase">
          Planned capabilities
        </p>
        <ul className="mt-2 grid gap-1.5 text-sm text-muted sm:grid-cols-2">
          {features.map((feature) => (
            <li key={feature} className="flex items-start gap-2">
              <span aria-hidden className="mt-1.5 size-1 shrink-0 rounded-full bg-subtle" />
              {feature}
            </li>
          ))}
        </ul>
      </Card>
    </div>
  )
}
