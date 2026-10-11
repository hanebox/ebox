import { useQuery } from '@tanstack/react-query'

import { api } from '@/api/endpoints'

export const setupStatusQuery = {
  queryKey: ['setup', 'status'] as const,
  queryFn: () => api.setupStatus(),
  staleTime: 30_000,
}

export const authQuery = {
  queryKey: ['auth', 'me'] as const,
  queryFn: () => api.me(),
  retry: false,
  staleTime: 60_000,
}

export function useAuth() {
  return useQuery({ ...authQuery, retry: false })
}
