import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'
import { currentTerm } from '../../lib/schedule'

/** First 200 courses (matching q when given) for pickers; pickers search past 200. */
export function useAllCourses(slug: string, departmentId?: string, enabled = true, q = '') {
  return useQuery({
    enabled,
    queryKey: [slug, 'courses', 'all', { departmentId, q }],
    queryFn: () =>
      unwrap(
        api.GET('/universities/{slug}/courses', {
          params: {
            path: { slug },
            query: { limit: 200, departmentId, q: q || undefined },
          },
        }),
      ),
    staleTime: 60_000,
  })
}

/** First 200 terms (matching q when given) for pickers; pickers search past 200. */
export function useAllTerms(slug: string, q = '', enabled = true) {
  return useQuery({
    enabled,
    queryKey: [slug, 'terms', 'all', q],
    queryFn: () =>
      unwrap(
        api.GET('/universities/{slug}/terms', {
          params: { path: { slug }, query: { limit: 200, q: q || undefined } },
        }),
      ),
    staleTime: 60_000,
  })
}

export function todayIso(d = new Date()): string {
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

type T = { id: string; startsOn: string; endsOn: string }

/** Current term, else soonest upcoming, else latest-starting, else undefined. */
export function defaultTermId(terms: T[], today: string): string | undefined {
  return (currentTerm(terms, today) ?? [...terms].sort((a, b) => a.startsOn.localeCompare(b.startsOn)).at(-1))?.id
}
