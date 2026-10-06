import { useQuery } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'

/** First 200 courses for pickers (ceiling: a search endpoint past 200). */
export function useAllCourses(slug: string, departmentId?: string) {
  return useQuery({
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

/** Current term (earliest start if several), else soonest future, else latest-starting, else undefined. */
export function defaultTermId(terms: T[], today: string): string | undefined {
  const byStart = [...terms].sort((a, b) => a.startsOn.localeCompare(b.startsOn))
  return (
    byStart.find((t) => t.startsOn <= today && today <= t.endsOn) ??
    byStart.find((t) => t.startsOn > today) ??
    byStart[byStart.length - 1]
  )?.id
}
