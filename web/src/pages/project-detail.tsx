import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from '@tanstack/react-router'
import { ArrowLeft, Database, FolderKanban, Layers, Pencil, Plus, Trash2 } from 'lucide-react'
import { useState } from 'react'

import { api } from '@/api/endpoints'
import { ApplicationCard } from '@/components/applications/application-card'
import { PageHeader } from '@/components/layout/page-header'
import { ProjectDialog } from '@/components/projects/project-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, Section } from '@/components/ui/card'
import { ConfirmDialog } from '@/components/ui/confirm-dialog'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'
import { formatDateTime } from '@/lib/format'

export function ProjectDetailPage() {
  const { projectId } = useParams({ strict: false }) as { projectId: string }
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [editOpen, setEditOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const project = useQuery({
    queryKey: ['project', projectId],
    queryFn: () => api.project(projectId),
  })

  const databases = useQuery({
    queryKey: ['databases', projectId],
    queryFn: () => api.databases(projectId),
    refetchInterval: 15_000,
  })

  const services = useQuery({
    queryKey: ['services', projectId],
    queryFn: () => api.services(projectId),
    refetchInterval: 15_000,
  })

  const applications = useQuery({
    queryKey: ['applications', projectId],
    queryFn: () => api.applications(projectId),
  })

  const deleteMutation = useMutation({
    mutationFn: () => api.deleteProject(projectId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['projects'] })
      toast.success('Project deleted', project.data?.name)
      void navigate({ to: '/projects' })
    },
    onError: (error) => {
      toast.error(
        'Could not delete the project',
        error instanceof ApiError ? error.message : undefined,
      )
    },
  })

  return (
    <div className="flex flex-col gap-6">
      <div>
        <Link
          to="/projects"
          className="inline-flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeft aria-hidden className="size-3.5" />
          All projects
        </Link>
      </div>

      {project.isPending ? <Skeleton className="h-20" /> : null}

      {project.isError ? (
        <EmptyState
          icon={<FolderKanban aria-hidden />}
          title="Project not found"
          description="It may have been deleted. Head back to the projects list."
          action={
            <Link to="/projects">
              <Button variant="secondary">Back to projects</Button>
            </Link>
          }
        />
      ) : null}

      {project.data ? (
        <>
          <PageHeader
            title={project.data.name}
            description={project.data.description || undefined}
            actions={
              <>
                <Button variant="secondary" size="sm" onClick={() => setEditOpen(true)}>
                  <Pencil aria-hidden className="size-3.5" />
                  Edit
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setDeleteOpen(true)}>
                  <Trash2 aria-hidden className="size-3.5 text-danger" />
                  <span className="text-danger">Delete</span>
                </Button>
              </>
            }
          />

          <Card>
            <CardContent className="grid gap-4 sm:grid-cols-3">
              <div className="flex flex-col gap-0.5">
                <span className="text-[11px] tracking-wide text-subtle uppercase">Identifier</span>
                <span className="font-mono text-xs text-foreground">{project.data.uuid}</span>
              </div>
              <div className="flex flex-col gap-0.5">
                <span className="text-[11px] tracking-wide text-subtle uppercase">Created</span>
                <span className="text-xs text-foreground">
                  {formatDateTime(project.data.created_at)}
                </span>
              </div>
              <div className="flex flex-col gap-0.5">
                <span className="text-[11px] tracking-wide text-subtle uppercase">Updated</span>
                <span className="text-xs text-foreground">
                  {formatDateTime(project.data.updated_at)}
                </span>
              </div>
            </CardContent>
          </Card>

          <Section
            title="Applications"
            description="Deploy from a source with build packs, domains and live logs."
            actions={
              <Link to="/projects/$projectId/new/application" params={{ projectId }}>
                <Button variant="primary" size="sm">
                  <Plus aria-hidden className="size-3.5" />
                  New application
                </Button>
              </Link>
            }
          >
            {applications.isPending ? (
              <div className="grid gap-3 lg:grid-cols-2">
                <Skeleton className="h-32" />
                <Skeleton className="h-32" />
              </div>
            ) : null}

            {applications.data && applications.data.length === 0 ? (
              <EmptyState
                icon={<FolderKanban aria-hidden />}
                title="No applications yet"
                description="Create an application to deploy code from one of your sources."
                action={
                  <Link to="/projects/$projectId/new/application" params={{ projectId }}>
                    <Button variant="primary">
                      <Plus aria-hidden className="size-3.5" />
                      New application
                    </Button>
                  </Link>
                }
              />
            ) : null}

            {applications.data && applications.data.length > 0 ? (
              <div className="grid gap-3 lg:grid-cols-2">
                {applications.data.map((application) => (
                  <ApplicationCard key={application.uuid} application={application} />
                ))}
              </div>
            ) : null}
          </Section>

          <Section
            title="Databases"
            description="Postgres, MySQL, MariaDB, Redis and MongoDB provisioned for this project."
            actions={
              <Link to="/databases">
                <Button variant="ghost" size="sm">
                  <Plus aria-hidden className="size-3.5" />
                  New database
                </Button>
              </Link>
            }
          >
            {(databases.data?.length ?? 0) === 0 ? (
              <p className="rounded-lg border border-dashed border-border px-4 py-6 text-center text-xs text-muted">
                No databases in this project yet.
              </p>
            ) : (
              <div className="grid gap-3 lg:grid-cols-2">
                {databases.data?.map((database) => (
                  <Link
                    key={database.uuid}
                    to="/databases/$databaseId"
                    params={{ databaseId: database.uuid }}
                    className="flex items-center justify-between gap-3 rounded-lg border border-border px-4 py-3 transition-colors hover:border-accent/60"
                  >
                    <div className="flex min-w-0 items-center gap-2">
                      <Database aria-hidden className="size-4 text-subtle" />
                      <div className="flex min-w-0 flex-col">
                        <span className="truncate text-sm text-foreground">{database.name}</span>
                        <span className="text-[11px] text-subtle">
                          {database.type} {database.version}
                          {database.port > 0 ? ` · host port ${database.port}` : ' · internal only'}
                        </span>
                      </div>
                    </div>
                    <Badge variant={database.runtime?.state === 'running' ? 'success' : 'outline'}>
                      {database.runtime?.state ?? database.status}
                    </Badge>
                  </Link>
                ))}
              </div>
            )}
          </Section>

          <Section
            title="Services"
            description="Compose stacks that belong to this project."
            actions={
              <Link to="/services">
                <Button variant="ghost" size="sm">
                  <Plus aria-hidden className="size-3.5" />
                  New service
                </Button>
              </Link>
            }
          >
            {(services.data?.length ?? 0) === 0 ? (
              <p className="rounded-lg border border-dashed border-border px-4 py-6 text-center text-xs text-muted">
                No compose services in this project yet.
              </p>
            ) : (
              <div className="grid gap-3 lg:grid-cols-2">
                {services.data?.map((service) => (
                  <Link
                    key={service.uuid}
                    to="/services/$serviceId"
                    params={{ serviceId: service.uuid }}
                    className="flex items-center justify-between gap-3 rounded-lg border border-border px-4 py-3 transition-colors hover:border-accent/60"
                  >
                    <div className="flex min-w-0 items-center gap-2">
                      <Layers aria-hidden className="size-4 text-subtle" />
                      <div className="flex min-w-0 flex-col">
                        <span className="truncate text-sm text-foreground">{service.name}</span>
                        <span className="text-[11px] text-subtle">
                          {service.runtime?.services?.length ?? 0} compose service(s)
                          {(service.runtime?.services?.length ?? 0) > 0
                            ? ` · ${service.runtime.services.filter((item) => item.state === 'running').length} running`
                            : ''}
                        </span>
                      </div>
                    </div>
                    <Badge
                      variant={
                        service.runtime?.state === 'running'
                          ? 'success'
                          : service.runtime?.state === 'partial'
                            ? 'warning'
                            : 'outline'
                      }
                    >
                      {service.runtime?.state ?? service.status}
                    </Badge>
                  </Link>
                ))}
              </div>
            )}
          </Section>


          <ProjectDialog open={editOpen} onOpenChange={setEditOpen} project={project.data} />

          <ConfirmDialog
            open={deleteOpen}
            onOpenChange={setDeleteOpen}
            title={`Delete ${project.data.name}?`}
            description="The project and its configuration will be removed. Applications inside it must be deleted first."
            confirmLabel="Delete project"
            danger
            loading={deleteMutation.isPending}
            onConfirm={() => deleteMutation.mutate()}
          />
        </>
      ) : null}
    </div>
  )
}
