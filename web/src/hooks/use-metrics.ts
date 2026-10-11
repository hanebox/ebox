import { useQuery } from '@tanstack/react-query'

import { api } from '@/api/endpoints'

export function useHostMetrics(window: '20m' | '24h' = '20m') {
  return useQuery({
    queryKey: ['metrics', 'host', window],
    queryFn: () => api.hostMetrics(window),
    refetchInterval: 3000,
    staleTime: 2000,
  })
}

export function useSystemInfo() {
  return useQuery({
    queryKey: ['system', 'info'],
    queryFn: () => api.systemInfo(),
    refetchInterval: 30_000,
  })
}

export function useSessions() {
  return useQuery({
    queryKey: ['sessions'],
    queryFn: () => api.sessions(),
  })
}

export function usePanelSettings() {
  return useQuery({
    queryKey: ['settings'],
    queryFn: () => api.settings(),
  })
}
