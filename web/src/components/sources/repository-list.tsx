import { useQuery } from '@tanstack/react-query'
import { ChevronDown, ChevronRight, Lock, RefreshCw, Search } from 'lucide-react'
import { useMemo, useState } from 'react'

import { api } from '@/api/endpoints'
import type { Repository, Source } from '@/api/types'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { EmptyState } from '@/components/ui/empty-state'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { ApiError } from '@/lib/api'
import { formatRelative } from '@/lib/format'

function RepositoryRow({
  source,
  repository,
}: {
  source: Source
  repository: Repository
}) {
  const [expanded, setExpanded] = useState(false)

  const branches = useQuery({
    queryKey: ['source', source.uuid, 'branches', repository.full_name],
    queryFn: () => api.branches(source.uuid, repository.full_name, repository.default_branch),
    enabled: expanded,
    staleTime: 60_000,
  })

  return (
    <div className="flex flex-col">
      <button
        type="button"
        onClick={() => setExpanded((value) => !value)}
        className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-left transition-colors hover:bg-surface-raised"
      >
        {expanded ? (
          <ChevronDown aria-hidden className="size-3.5 shrink-0 text-subtle" />
        ) : (
          <ChevronRight aria-hidden className="size-3.5 shrink-0 text-subtle" />
        )}
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm text-foreground">{repository.full_name}</span>
            {repository.private ? (
              <Badge variant="outline" className="gap-1">
                <Lock aria-hidden className="size-2.5" />
                private
              </Badge>
            ) : (
              <Badge variant="default">public</Badge>
            )}
          </div>
          {repository.description ? (
            <span className="truncate text-xs text-subtle">{repository.description}</span>
          ) : null}
        </div>
        <div className="hidden shrink-0 items-center gap-3 sm:flex">
          <span className="font-mono text-[11px] text-muted">{repository.default_branch}</span>
          <span className="text-[11px] text-subtle">{formatRelative(repository.updated_at)}</span>
        </div>
      </button>

      {expanded ? (
        <div className="mb-2 ml-9 rounded-lg border border-border bg-surface-raised/50 p-2">
          {branches.isPending ? (
            <div className="flex flex-col gap-1.5 p-1">
              <Skeleton className="h-5 w-40" />
              <Skeleton className="h-5 w-32" />
            </div>
          ) : null}
          {branches.isError ? (
            <p className="p-1 text-xs text-danger">
              {branches.error instanceof ApiError
                ? branches.error.message
                : 'Could not load branches.'}
            </p>
          ) : null}
          {branches.data ? (
            <div className="flex flex-wrap gap-1.5 p-1">
              {branches.data.map((branch) => (
                <span
                  key={branch.name}
                  className="inline-flex items-center gap-1.5 rounded-md border border-border bg-surface px-2 py-0.5 font-mono text-[11px] text-muted"
                >
                  {branch.name}
                  {branch.default ? <Badge variant="info">default</Badge> : null}
                  {branch.protected ? <Badge variant="warning">protected</Badge> : null}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}

export function RepositoryList({ source }: { source: Source }) {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')

  const repositories = useQuery({
    queryKey: ['source', source.uuid, 'repositories', page],
    queryFn: () => api.repositories(source.uuid, page),
    enabled: Boolean(source.installation_id),
  })

  const all = useMemo(() => repositories.data?.repositories ?? [], [repositories.data])

  const filtered = useMemo(() => {
    if (!search.trim()) {
      return all
    }

    const needle = search.trim().toLowerCase()

    return all.filter((repository) => repository.full_name.toLowerCase().includes(needle))
  }, [all, search])

  if (!source.installation_id) {
    return (
      <EmptyState
        icon={<Lock aria-hidden />}
        title="No installation connected"
        description="Install the GitHub App on the repositories you want to deploy, then connect the installation."
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2">
        <div className="relative max-w-xs flex-1">
          <Search
            aria-hidden
            className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-subtle"
          />
          <Input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Filter repositories…"
            className="pl-8"
          />
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => void repositories.refetch()}
          loading={repositories.isFetching}
        >
          <RefreshCw aria-hidden className="size-3.5" />
          Refresh
        </Button>
        <span className="text-xs text-subtle">
          {repositories.data ? `${repositories.data.total} accessible` : null}
        </span>
      </div>

      {repositories.isPending ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-11" />
          <Skeleton className="h-11" />
          <Skeleton className="h-11" />
        </div>
      ) : null}

      {repositories.isError ? (
        <p className="rounded-lg border border-danger/40 bg-danger-soft p-3 text-xs text-danger">
          {repositories.error instanceof ApiError
            ? repositories.error.message
            : 'Could not load repositories.'}
        </p>
      ) : null}

      {repositories.data && filtered.length === 0 ? (
        <p className="px-1 py-4 text-xs text-subtle">
          {search ? 'No repositories match the filter.' : 'No repositories are accessible.'}
        </p>
      ) : null}

      {filtered.length > 0 ? (
        <div className="flex flex-col divide-y divide-border/60 rounded-lg border border-border">
          {filtered.map((repository) => (
            <RepositoryRow key={repository.full_name} source={source} repository={repository} />
          ))}
        </div>
      ) : null}

      {repositories.data && repositories.data.total > all.length ? (
        <div className="flex justify-center">
          <Button
            variant="secondary"
            size="sm"
            onClick={() => setPage((value) => value + 1)}
            loading={repositories.isFetching}
          >
            Load more repositories
          </Button>
        </div>
      ) : null}
    </div>
  )
}
