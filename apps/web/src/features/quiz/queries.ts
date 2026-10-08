import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'
import type { ApiError } from '../../api/errors'
import type { components } from '../../api/types'

type S = components['schemas']
export type Assessment = S['Assessment']
export type Question = S['Question']
export type QuestionsPut = S['QuestionsPut']
export type Attempt = S['Attempt']
export type AttemptRow = S['AttemptRow']
export type AttemptPut = S['AttemptPut']
export type AttemptReview = S['AttemptReview']
export type Submission = S['Submission']
export type SubmissionRow = S['SubmissionRow']

export const sectionBase = (slug: string, sectionId: string) => [slug, 'sections', sectionId] as const
export const assessmentsKey = (slug: string, id: string) => [...sectionBase(slug, id), 'assessments'] as const
export const questionsKey = (slug: string, id: string, aId: string) => [...sectionBase(slug, id), 'questions', aId] as const
export const attemptsKey = (slug: string, id: string, aId: string) => [...sectionBase(slug, id), 'attempts', aId] as const
export const attemptKey = (slug: string, id: string, aId: string, userId: string) =>
  [...attemptsKey(slug, id, aId), userId] as const
export const submissionsKey = (slug: string, id: string, aId: string) => [...sectionBase(slug, id), 'submissions', aId] as const
export const submissionKey = (slug: string, id: string, aId: string, userId: string) =>
  [...submissionsKey(slug, id, aId), userId] as const

const path = (slug: string, id: string, assessmentId: string) => ({ params: { path: { slug, id, assessmentId } } })

export function useAssessments(slug: string, sectionId: string) {
  return useQuery<S['AssessmentList'], ApiError>({
    queryKey: assessmentsKey(slug, sectionId),
    queryFn: () => unwrap(api.GET('/universities/{slug}/sections/{id}/assessments', { params: { path: { slug, id: sectionId } } })),
  })
}

export function useQuestions(slug: string, sectionId: string, aId: string, enabled = true) {
  return useQuery<{ items: Question[] }, ApiError>({
    queryKey: questionsKey(slug, sectionId, aId),
    queryFn: () => unwrap(api.GET('/universities/{slug}/sections/{id}/assessments/{assessmentId}/questions', path(slug, sectionId, aId))),
    enabled,
  })
}

/** Staff: one row per started attempt. */
export function useAttempts(slug: string, sectionId: string, aId: string, enabled = true) {
  return useQuery<{ items: AttemptRow[] }, ApiError>({
    queryKey: attemptsKey(slug, sectionId, aId),
    queryFn: () => unwrap(api.GET('/universities/{slug}/sections/{id}/assessments/{assessmentId}/attempts', path(slug, sectionId, aId))),
    enabled,
  })
}

/** One attempt; `userId` 'me' for the student's own. 404 resolves to null ("not started"). */
export function useAttempt(slug: string, sectionId: string, aId: string, userId: string, opts: { enabled?: boolean; refetchInterval?: number | false } = {}) {
  return useQuery<Attempt | null, ApiError>({
    queryKey: attemptKey(slug, sectionId, aId, userId),
    queryFn: async () => {
      try {
        return await unwrap(
          api.GET('/universities/{slug}/sections/{id}/assessments/{assessmentId}/attempts/{userId}', {
            params: { path: { slug, id: sectionId, assessmentId: aId, userId } },
          }),
        )
      } catch (e) {
        if ((e as ApiError).status === 404) return null
        throw e
      }
    },
    retry: false,
    ...opts,
  })
}

export function useSubmissions(slug: string, sectionId: string, aId: string, enabled = true) {
  return useQuery<{ items: SubmissionRow[] }, ApiError>({
    queryKey: submissionsKey(slug, sectionId, aId),
    queryFn: () => unwrap(api.GET('/universities/{slug}/sections/{id}/assessments/{assessmentId}/submissions', path(slug, sectionId, aId))),
    enabled,
  })
}

/** One submission; `userId` 'me' for the student's own. 404 resolves to null. */
export function useSubmission(slug: string, sectionId: string, aId: string, userId: string, enabled = true) {
  return useQuery<Submission | null, ApiError>({
    queryKey: submissionKey(slug, sectionId, aId, userId),
    queryFn: async () => {
      try {
        return await unwrap(
          api.GET('/universities/{slug}/sections/{id}/assessments/{assessmentId}/submissions/{userId}', {
            params: { path: { slug, id: sectionId, assessmentId: aId, userId } },
          }),
        )
      } catch (e) {
        if ((e as ApiError).status === 404) return null
        throw e
      }
    },
    retry: false,
    enabled,
  })
}

/** Invalidates assessments, gradebook, my-grades, audit log and the extra keys; await before a status line. */
export function useAssessmentRefresh(slug: string, sectionId: string) {
  const qc = useQueryClient()
  return (...extra: (readonly unknown[])[]) =>
    Promise.all(
      [assessmentsKey(slug, sectionId), [slug, 'gradebook'], [slug, 'my-grades'], [slug, 'audit-log'], ...extra].map((queryKey) =>
        qc.invalidateQueries({ queryKey }),
      ),
    )
}

export const patchAssessment = (slug: string, sectionId: string, assessmentId: string, body: S['AssessmentUpdate']) =>
  unwrap(api.PATCH('/universities/{slug}/sections/{id}/assessments/{assessmentId}', { params: { path: { slug, id: sectionId, assessmentId } }, body }))

export const putQuestions = (slug: string, sectionId: string, aId: string, body: QuestionsPut) =>
  unwrap(api.PUT('/universities/{slug}/sections/{id}/assessments/{assessmentId}/questions', { ...path(slug, sectionId, aId), body }))

export const startAttempt = (slug: string, sectionId: string, aId: string) =>
  unwrap(api.POST('/universities/{slug}/sections/{id}/assessments/{assessmentId}/attempts', path(slug, sectionId, aId)))

export const putAttempt = (slug: string, sectionId: string, aId: string, body: AttemptPut) =>
  unwrap(api.PUT('/universities/{slug}/sections/{id}/assessments/{assessmentId}/attempts/me', { ...path(slug, sectionId, aId), body }))

export const reviewAttempt = (slug: string, sectionId: string, aId: string, userId: string, body: AttemptReview) =>
  unwrap(
    api.PATCH('/universities/{slug}/sections/{id}/assessments/{assessmentId}/attempts/{userId}', {
      params: { path: { slug, id: sectionId, assessmentId: aId, userId } },
      body,
    }),
  )
