import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { IconChevronRight, IconFolders, IconPlus } from '@tabler/icons-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import type { Project } from '@/api/types'
import { PageHeader } from '@/components/layout/page-header'
import { ProjectDialog } from '@/components/projects/project-dialog'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { formatRelative } from '@/lib/format'

function ProjectCard({ project }: { project: Project }) {
  return (
    <Link
      to="/projects/$projectId"
      params={{ projectId: project.uuid }}
      className="group flex flex-col gap-3 rounded-panel border border-border bg-surface p-4 transition-colors hover:border-border-strong"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <p className="truncate text-sm font-medium text-foreground">{project.name}</p>
          <p className="line-clamp-2 min-h-8 text-xs text-muted">
            {project.description || 'No description yet.'}
          </p>
        </div>
        <IconChevronRight
          aria-hidden
          className="mt-0.5 size-4 shrink-0 text-subtle transition-transform group-hoverable:translate-x-0.5"
        />
      </div>
      <p className="text-2xs text-subtle">Created {formatRelative(project.created_at)}</p>
    </Link>
  )
}

export function ProjectsPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const projects = useQuery({ queryKey: ['projects'], queryFn: api.projects })

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Projects"
        description="Group your applications, databases and services."
        actions={
          <Button variant="primary" onClick={() => setDialogOpen(true)}>
            <IconPlus aria-hidden className="size-3.5" />
            New project
          </Button>
        }
      />

      {projects.isPending ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
          <Skeleton className="h-32" />
        </div>
      ) : null}

      {projects.data && projects.data.length === 0 ? (
        <EmptyState
          icon={<IconFolders aria-hidden />}
          title="No projects yet"
          description="Projects keep related resources together. Create one to get started."
          action={
            <Button variant="primary" onClick={() => setDialogOpen(true)}>
              <IconPlus aria-hidden className="size-3.5" />
              New project
            </Button>
          }
        />
      ) : null}

      {projects.data && projects.data.length > 0 ? (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {projects.data.map((project) => (
            <ProjectCard key={project.uuid} project={project} />
          ))}
        </div>
      ) : null}

      <ProjectDialog open={dialogOpen} onOpenChange={setDialogOpen} />
    </div>
  )
}
