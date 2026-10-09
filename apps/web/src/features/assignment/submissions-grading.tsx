import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, Inbox } from 'lucide-react'
import { useRef, useState } from 'react'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage, FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputSizedClass, textareaClass } from '../../components/form-field'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { UnsavedChangesGuard } from '../../components/unsaved-changes'
import { formatDateTime } from '../../lib/format'
import { fmtNum, parsePoints } from '../grades/grade'
import {
  submissionKey,
  submissionsKey,
  useAssessmentRefresh,
  useAssessments,
  useSubmission,
  useSubmissions,
  type Assessment,
  type Submission,
  type SubmissionRow,
} from '../quiz/queries'
import { ReleaseButton } from '../quiz/release'
import { PlainText, countHint } from '../section/plain-text'
import { SectionFrame } from '../section/section-frame'
import { gradeSubmission } from './queries'
import { SubmissionFiles } from './submission-form'

const backClass = 'mb-3 inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline'
const who = (r: Pick<SubmissionRow, 'displayName' | 'email'>) => r.displayName || r.email

export function SubmissionsPage(props: { slug: string; sectionId: string; assessmentId: string; userId?: string }) {
  const { slug, sectionId } = props
  return <SectionFrame slug={slug} sectionId={sectionId}>{() => <Submissions {...props} />}</SectionFrame>
}

function Submissions({ slug, sectionId, assessmentId, userId }: { slug: string; sectionId: string; assessmentId: string; userId?: string }) {
  const list = useAssessments(slug, sectionId)
  const subs = useSubmissions(slug, sectionId, assessmentId)
  const [notice, setNotice] = useState('')
  const [pageError, setPageError] = useState('')
  const a = list.data?.items.find((x) => x.id === assessmentId && x.kind === 'assignment')
  const params = { slug, id: sectionId, assessmentId }
  const rows = subs.data?.items ?? []

  const back = (
    <Link to="/$slug/sections/$id/assignments" params={{ slug, id: sectionId }} className={backClass}>
      <ArrowLeft aria-hidden="true" />
      All assignments
    </Link>
  )

  const failed = list.error ?? subs.error
  let body: React.ReactNode
  if (failed) {
    body =
      failed.status === 403 ? (
        <ErrorBanner title="No access">Only this section's staff can grade its assignments.</ErrorBanner>
      ) : (
        <ErrorBanner title="Couldn't load submissions" onRetry={() => void (list.error ? list.refetch() : subs.refetch())}>
          {errorMessage(failed)}
        </ErrorBanner>
      )
  } else if (list.isPending || subs.isPending) body = <Loading label="Loading submissions…" />
  else if (!a) body = <ErrorBanner title="Assignment not found">This assignment doesn't exist or was deleted.</ErrorBanner>
  else {
    const ungraded = rows.filter((r) => r.score === undefined).length
    const row = userId ? rows.find((r) => r.userId === userId) : undefined
    body = (
      <>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-4">
          <h2 className="min-w-0 text-xl font-bold wrap-break-word">{a.name}</h2>
          <ReleaseButton
            slug={slug}
            sectionId={sectionId}
            a={a}
            variant="secondary"
            note={ungraded > 0 ? `${ungraded} ${ungraded === 1 ? "submission isn't" : "submissions aren't"} graded yet.` : undefined}
            onDone={(m) => {
              setPageError('')
              setNotice(m)
            }}
            onError={setPageError}
          />
        </div>
        <StatusMessage message={notice} />
        {pageError && (
          <div className="mb-4">
            <ErrorBanner>{pageError}</ErrorBanner>
          </div>
        )}
        {userId ? (
          <Panel key={userId} slug={slug} sectionId={sectionId} a={a} userId={userId} row={row} />
        ) : rows.length === 0 ? (
          <EmptyState icon={Inbox} title="No submissions yet" body="Students' submissions appear here as soon as they save text or upload a file." />
        ) : (
          <Table stack caption={`Submissions for ${a.name}`}>
            <thead>
              <tr>
                <th scope="col">Student</th>
                <th scope="col">Submitted</th>
                <th scope="col">Files</th>
                <th scope="col">Score</th>
                <th scope="col">Graded</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.userId}>
                  <td data-label="Student">
                    <Link
                      to="/$slug/sections/$id/assignments/$assessmentId/submissions"
                      params={params}
                      search={{ user: r.userId }}
                      className="text-accent underline hover:text-accent-hover"
                    >
                      {who(r)}
                    </Link>
                    {r.displayName && <span className="block text-xs text-ink-muted">{r.email}</span>}
                  </td>
                  <td data-label="Submitted">{formatDateTime(r.submittedAt)}</td>
                  <td data-label="Files">{r.fileCount}</td>
                  <td data-label="Score">{r.score === undefined ? 'Not graded' : `${fmtNum(r.score)} / ${fmtNum(a.maxPoints)}`}</td>
                  <td data-label="Graded">{r.gradedAt ? formatDateTime(r.gradedAt) : 'Not graded'}</td>
                </tr>
              ))}
            </tbody>
          </Table>
        )}
      </>
    )
  }
  return (
    <>
      {back}
      {body}
    </>
  )
}

function Panel({ slug, sectionId, a, userId, row }: { slug: string; sectionId: string; a: Assessment; userId: string; row?: SubmissionRow }) {
  const sub = useSubmission(slug, sectionId, a.id, userId)
  const name = row ? who(row) : 'this student'
  const back = (
    <Link
      to="/$slug/sections/$id/assignments/$assessmentId/submissions"
      params={{ slug, id: sectionId, assessmentId: a.id }}
      search={{ user: undefined }}
      className={backClass}
    >
      <ArrowLeft aria-hidden="true" />
      All submissions
    </Link>
  )
  let body: React.ReactNode
  if (sub.error) {
    body = (
      <ErrorBanner title="Couldn't load the submission" onRetry={() => void sub.refetch()}>
        {errorMessage(sub.error)}
      </ErrorBanner>
    )
  } else if (sub.isPending) body = <Loading label="Loading submission…" />
  else if (!sub.data) body = <ErrorBanner title="Submission not found">This student has no submission, or it was removed.</ErrorBanner>
  else body = <GradeForm slug={slug} sectionId={sectionId} a={a} name={name} sub={sub.data} />
  return (
    <>
      {back}
      {body}
    </>
  )
}

function GradeForm({ slug, sectionId, a, name, sub }: { slug: string; sectionId: string; a: Assessment; name: string; sub: Submission }) {
  const qc = useQueryClient()
  const refresh = useAssessmentRefresh(slug, sectionId)
  const savedScore = sub.score === undefined ? '' : fmtNum(sub.score)
  const savedFeedback = sub.feedback ?? ''
  const [score, setScore] = useState(savedScore)
  const [feedback, setFeedback] = useState(savedFeedback)
  const [local, setLocal] = useState('')
  const [notice, setNotice] = useState('')
  const formRef = useRef<HTMLFormElement>(null)
  const dirty = score.trim() !== savedScore || feedback !== savedFeedback

  const save = useMutation({
    mutationFn: (points: number) => gradeSubmission(slug, sectionId, a.id, sub.userId, { score: points, feedback }),
    onSuccess: async (s) => {
      qc.setQueryData(submissionKey(slug, sectionId, a.id, sub.userId), s)
      await refresh(submissionsKey(slug, sectionId, a.id))
      setNotice(`Grade saved. ${name}: ${fmtNum(s.score ?? 0)} / ${fmtNum(a.maxPoints)}.`)
    },
  })
  const fe = useFieldErrors(save.error)
  useFocusFirstError(formRef, save.error)

  function onSave(e: React.FormEvent) {
    e.preventDefault()
    setNotice('')
    save.reset()
    if (!score.trim()) return setLocal('Enter a score.')
    const r = parsePoints(score, a.maxPoints)
    if ('error' in r) return setLocal(r.error)
    setLocal('')
    save.mutate(r.points!)
  }

  return (
    <section aria-labelledby="grade-name-h" className="flex max-w-3xl flex-col gap-4">
      <UnsavedChangesGuard when={dirty} />
      <div>
        <h3 id="grade-name-h" className="text-base font-bold wrap-break-word">
          {name}
        </h3>
        <p className="text-sm text-ink-muted">Submitted {formatDateTime(sub.submittedAt)}</p>
      </div>
      {sub.body ? (
        <PlainText text={sub.body} className="rounded border border-line bg-surface p-3 text-sm text-ink" />
      ) : (
        <p className="text-sm text-ink-muted">No text submitted.</p>
      )}
      {sub.files.length > 0 && <SubmissionFiles slug={slug} sectionId={sectionId} aId={a.id} userId={sub.userId} files={sub.files} />}
      <form ref={formRef} onSubmit={onSave} noValidate className="flex flex-col gap-4">
        {save.error && <FormErrors error={save.error} />}
        <FormField label={`Score for ${name}`} name="score" hint={`Out of ${fmtNum(a.maxPoints)}. Up to ${fmtNum(2 * a.maxPoints)} with bonus.`} error={local || fe.score}>
          <input inputMode="decimal" className={`${inputSizedClass} w-32`} value={score} onChange={(e) => setScore(e.target.value)} />
        </FormField>
        <FormField label={`Feedback for ${name}`} name="feedback" hint={countHint(feedback, 5000)} error={fe.feedback}>
          <textarea rows={5} className={textareaClass} value={feedback} onChange={(e) => setFeedback(e.target.value)} />
        </FormField>
        <div>
          <Button type="submit" disabled={save.isPending}>
            {save.isPending ? 'Saving…' : 'Save grade'}
          </Button>
        </div>
        <StatusMessage message={notice} />
      </form>
    </section>
  )
}
