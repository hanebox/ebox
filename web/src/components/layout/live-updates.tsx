import { useQueryClient } from '@tanstack/react-query'

import { useEventStream } from '@/hooks/use-event-stream'

/**
 * Keeps resource queries fresh by reacting to the panel event stream instead of
 * polling: container state changes, deployment updates and prune results.
 */
export function LiveUpdates() {
  const queryClient = useQueryClient()

  useEventStream((event) => {
    if (event.topic.startsWith('container.') || event.topic.startsWith('application.')) {
      void queryClient.invalidateQueries({ queryKey: ['applications'] })
      void queryClient.invalidateQueries({ queryKey: ['application'] })
      void queryClient.invalidateQueries({ queryKey: ['containers'] })

      return
    }

    if (event.topic.startsWith('deployment.')) {
      void queryClient.invalidateQueries({ queryKey: ['deployments'] })
      void queryClient.invalidateQueries({ queryKey: ['deployment'] })

      return
    }

    if (event.topic === 'system') {
      void queryClient.invalidateQueries({ queryKey: ['system-disk'] })
    }
  })

  return null
}
