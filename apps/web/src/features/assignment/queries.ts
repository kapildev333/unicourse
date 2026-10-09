import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { sendWithProgress } from '../../lib/upload'
import type { Submission } from '../quiz/queries'

type S = components['schemas']

const base = (slug: string, sectionId: string, aId: string) =>
  `/api/v1/universities/${encodeURIComponent(slug)}/sections/${encodeURIComponent(sectionId)}/assessments/${encodeURIComponent(aId)}/submissions`

/** Download URL; `userId` 'me' for the student's own. The browser sends the session cookie. */
export const submissionFileHref = (slug: string, sectionId: string, aId: string, userId: string, fileId: string) =>
  `${base(slug, sectionId, aId)}/${userId}/files/${fileId}/download`

const path = (slug: string, id: string, assessmentId: string) => ({ slug, id, assessmentId })

export const putSubmission = (slug: string, sectionId: string, aId: string, body: string) =>
  unwrap(api.PUT('/universities/{slug}/sections/{id}/assessments/{assessmentId}/submissions/me', { params: { path: path(slug, sectionId, aId) }, body: { body } }))

/** One file with byte progress as a whole percent; resolves with the updated submission. */
export function uploadSubmissionFile(slug: string, sectionId: string, aId: string, file: File, onPercent: (pct: number) => void) {
  const form = new FormData()
  form.append('file', file)
  return sendWithProgress<Submission>(`${window.location.origin}${base(slug, sectionId, aId)}/me/files`, form, (sent, total) =>
    onPercent(Math.floor((sent / total) * 100)),
  )
}

export const removeSubmissionFile = (slug: string, sectionId: string, aId: string, fileId: string) =>
  unwrap(
    api.DELETE('/universities/{slug}/sections/{id}/assessments/{assessmentId}/submissions/me/files/{fileId}', {
      params: { path: { ...path(slug, sectionId, aId), fileId } },
    }),
  )

export const gradeSubmission = (slug: string, sectionId: string, aId: string, userId: string, body: S['SubmissionGrade']) =>
  unwrap(
    api.PATCH('/universities/{slug}/sections/{id}/assessments/{assessmentId}/submissions/{userId}', {
      params: { path: { ...path(slug, sectionId, aId), userId } },
      body,
    }),
  )
