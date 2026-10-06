import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'

/** First 200 departments for pickers and name maps (ceiling: a search endpoint past 200). */
export function useAllDepartments(slug: string) {
  return useQuery({
    queryKey: [slug, 'departments', 'all'],
    queryFn: () =>
      unwrap(
        api.GET('/universities/{slug}/departments', {
          params: { path: { slug }, query: { limit: 200 } },
        }),
      ),
    staleTime: 60_000,
  })
}

/** First 200 members for the audit-log "Changed by" filter (ceiling: a search endpoint past 200). */
export function useAllMembers(slug: string) {
  return useQuery({
    queryKey: [slug, 'memberships', 'all'],
    queryFn: () =>
      unwrap(
        api.GET('/universities/{slug}/memberships', {
          params: { path: { slug }, query: { limit: 200 } },
        }),
      ),
    staleTime: 60_000,
  })
}

/** Invalidates `[slug, resource]` (and its 'all' variants), the audit log, plus any extra keys. */
export function useRefresh(slug: string, resource: string) {
  const qc = useQueryClient()
  return (...extra: (readonly unknown[])[]) =>
    Promise.all([[slug, resource], [slug, 'audit-log'], ...extra].map((queryKey) => qc.invalidateQueries({ queryKey })))
}
