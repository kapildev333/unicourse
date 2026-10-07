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
export type Section = S['Section']
export type ScoresPut = S['ScoresUpdate']

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

export function useSection(slug: string, sectionId: string): UseQueryResult<Section, ApiError> {
  // same key as section-detail.tsx: shared cache
  return useQuery<Section, ApiError>({
    queryKey: [slug, 'sections', 'detail', sectionId],
    queryFn: () => unwrap(api.GET('/universities/{slug}/sections/{id}', { params: { path: { slug, id: sectionId } } })),
  })
}

export function useGradebook(slug: string, sectionId: string, enabled: boolean): UseQueryResult<Gradebook, ApiError> {
  return useQuery<Gradebook, ApiError>({
    queryKey: [slug, 'gradebook', sectionId],
    queryFn: () =>
      unwrap(api.GET('/universities/{slug}/sections/{id}/gradebook', { params: { path: { slug, id: sectionId } } })),
    enabled,
  })
}

export function useMyGrades(slug: string): UseQueryResult<MyGrades, ApiError> {
  return useQuery<MyGrades, ApiError>({
    queryKey: [slug, 'my-grades'],
    queryFn: () => unwrap(api.GET('/universities/{slug}/me/grades', { params: { path: { slug } } })),
  })
}

/** One PUT; scores.length <= 500 (callers chunk). points null removes a score. */
export function putScores(slug: string, sectionId: string, assessmentId: string, scores: ScoresPut['scores']) {
  return unwrap(
    api.PUT('/universities/{slug}/sections/{id}/assessments/{assessmentId}/scores', {
      params: { path: { slug, id: sectionId, assessmentId } },
      body: { scores },
    }),
  )
}
