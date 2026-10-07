import { useQuery, type UseQueryResult } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'
import type { ApiError } from '../../api/errors'
import type { components } from '../../api/types'
import { useRefresh } from '../admin/queries'

type S = components['schemas']
export type GradeScaleRow = S['GradeScaleRow']
export type GradeScale = S['GradeScale']
export type GradeScaleUpdate = S['GradeScaleUpdate']
export type Assessment = S['Assessment']
export type Gradebook = S['Gradebook']
export type MyGrades = S['MyGrades']

export function useGradeScale(slug: string): UseQueryResult<GradeScale, ApiError> {
  return useQuery<GradeScale, ApiError>({
    queryKey: [slug, 'grade-scale'],
    queryFn: () => unwrap(api.GET('/universities/{slug}/grade-scale', { params: { path: { slug } } })),
    staleTime: 60_000,
  })
}

/** Invalidates the gradebook and my-grades caches (and the audit log). */
export function useGradesRefresh(slug: string) {
  const refresh = useRefresh(slug, 'gradebook')
  return () => refresh([slug, 'my-grades'])
}

/** After a scale PUT: gradebook and my-grades letters may change. */
export function useScaleRefresh(slug: string) {
  const refresh = useRefresh(slug, 'grade-scale')
  return () => refresh([slug, 'gradebook'], [slug, 'my-grades'])
}
