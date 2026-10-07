import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'
import { currentTerm } from '../../lib/schedule'

/** First 200 courses for pickers (ceiling: a search endpoint past 200). */
export function useAllCourses(slug: string, departmentId?: string, enabled = true) {
  return useQuery({
    enabled,
    queryKey: [slug, 'courses', 'all', { departmentId }],
    queryFn: () =>
      unwrap(
        api.GET('/universities/{slug}/courses', { params: { path: { slug }, query: { limit: 200, departmentId } } }),
      ),
    staleTime: 60_000,
  })
}

/** First 200 terms for pickers (ceiling: a search endpoint past 200). */
export function useAllTerms(slug: string) {
  return useQuery({
    queryKey: [slug, 'terms', 'all'],
    queryFn: () => unwrap(api.GET('/universities/{slug}/terms', { params: { path: { slug }, query: { limit: 200 } } })),
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
