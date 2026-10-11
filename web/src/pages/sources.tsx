import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { IconGitBranch, IconPlus, IconSettings } from '@tabler/icons-react'
import { useEffect, useState } from 'react'

import { api } from '@/api/endpoints'
import { GitSourceDialog } from '@/components/sources/git-source-dialog'
import { GithubAppDialog } from '@/components/sources/github-app-dialog'
import { SourceCard } from '@/components/sources/source-card'
import { PageHeader } from '@/components/layout/page-header'
import { Button } from '@/components/ui/button'
import { GitHubIcon } from '@/components/ui/icons'
import { EmptyState } from '@/components/ui/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from '@/components/ui/toast'
import { ApiError } from '@/lib/api'

const flashErrors: Record<string, { title: string; description: string }> = {
  state: {
    title: 'The connection attempt expired',
    description: 'The manifest flow must be completed within 20 minutes. Please try again.',
  },
  code: {
    title: 'GitHub did not return an authorization code',
    description: 'The app creation may have been cancelled on GitHub.',
  },
  manifest: {
    title: 'GitHub rejected the app manifest',
    description: 'The app could not be created. Please try again.',
  },
  save: {
    title: 'The app was created but could not be saved',
    description: 'Check the panel logs and try connecting again.',
  },
  source: {
    title: 'Unknown source',
    description: 'The installation was reported for a source this panel does not know.',
  },
  install: {
    title: 'Could not save the installation',
    description: 'GitHub redirected successfully, but the installation ID was not saved.',
  },
}

function useGithubFlash() {
  const queryClient = useQueryClient()

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const connected = params.get('connected')
    const installed = params.get('installed')
    const error = params.get('error')

    if (!connected && !installed && !error) {
      return
    }

    if (connected) {
      toast.success(
        'GitHub App connected',
        'Now install it on the repositories you want to deploy.',
      )
    }
    if (installed) {
      toast.success('Installation connected', 'Repositories are now available for this source.')
    }
    if (error) {
      const message = flashErrors[error] ?? {
        title: 'The GitHub flow failed',
        description: `Unknown reason: ${error}`,
      }
      toast.error(message.title, message.description)
    }

    void queryClient.invalidateQueries({ queryKey: ['sources'] })

    const url = new URL(window.location.href)
    for (const key of ['connected', 'installed', 'error']) {
      url.searchParams.delete(key)
    }
    window.history.replaceState({}, '', url.pathname + url.search)
  }, [queryClient])
}

function postManifestToGithub(actionUrl: string, manifest: Record<string, unknown>) {
  const form = document.createElement('form')
  form.method = 'POST'
  form.action = actionUrl

  const input = document.createElement('input')
  input.type = 'hidden'
  input.name = 'manifest'
  input.value = JSON.stringify(manifest)

  form.appendChild(input)
  document.body.appendChild(form)
  form.submit()
}

export function SourcesPage() {
  const [gitDialogOpen, setGitDialogOpen] = useState(false)
  const [githubDialogOpen, setGithubDialogOpen] = useState(false)

  useGithubFlash()

  const sources = useQuery({ queryKey: ['sources'], queryFn: api.sources })

  const manifestMutation = useMutation({
    mutationFn: () => api.startGithubManifest({ origin: window.location.origin }),
    onSuccess: (data) => {
      postManifestToGithub(data.action_url, data.manifest)
    },
    onError: (error) => {
      toast.error(
        'Could not start the GitHub flow',
        error instanceof ApiError ? error.message : undefined,
      )
    },
  })

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Sources"
        description="Connect a git provider, then deploy from any of its repositories."
        actions={
          <>
            <Button
              variant="primary"
              onClick={() => manifestMutation.mutate()}
              loading={manifestMutation.isPending}
            >
              <GitHubIcon className="size-3.5" />
              Connect GitHub App
            </Button>
            <Button variant="secondary" onClick={() => setGitDialogOpen(true)}>
              <IconPlus aria-hidden className="size-3.5" />
              Add git source
            </Button>
          </>
        }
      />

      {sources.isPending ? (
        <div className="grid gap-4 lg:grid-cols-2">
          <Skeleton className="h-40" />
          <Skeleton className="h-40" />
        </div>
      ) : null}

      {sources.data && sources.data.length === 0 ? (
        <EmptyState
          icon={<IconGitBranch aria-hidden />}
          title="No sources connected"
          description="Sources tell the panel where your code lives. Connect a GitHub App for one-click repositories, or paste any git URL."
          action={
            <div className="flex flex-col items-center gap-3">
              <div className="flex flex-wrap items-center justify-center gap-2">
                <Button
                  variant="primary"
                  onClick={() => manifestMutation.mutate()}
                  loading={manifestMutation.isPending}
                >
                  <GitHubIcon className="size-3.5" />
                  Connect GitHub App
                </Button>
                <Button variant="secondary" onClick={() => setGitDialogOpen(true)}>
                  <IconPlus aria-hidden className="size-3.5" />
                  Add git source
                </Button>
              </div>
              <button
                type="button"
                className="inline-flex items-center gap-1.5 text-xs text-subtle transition-colors hover:text-foreground"
                onClick={() => setGithubDialogOpen(true)}
              >
                <IconSettings aria-hidden className="size-3.5" />
                Prefer to create the GitHub App yourself? Set it up manually
              </button>
            </div>
          }
        />
      ) : null}

      {sources.data && sources.data.length > 0 ? (
        <>
          <div className="grid gap-4 lg:grid-cols-2">
            {sources.data.map((source) => (
              <SourceCard key={source.uuid} source={source} />
            ))}
          </div>
          <button
            type="button"
            className="inline-flex w-fit items-center gap-1.5 text-xs text-subtle transition-colors hover:text-foreground"
            onClick={() => setGithubDialogOpen(true)}
          >
            <IconSettings aria-hidden className="size-3.5" />
            Connect a GitHub App manually
          </button>
        </>
      ) : null}

      <GitSourceDialog open={gitDialogOpen} onOpenChange={setGitDialogOpen} />
      <GithubAppDialog open={githubDialogOpen} onOpenChange={setGithubDialogOpen} />
    </div>
  )
}
