import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { AlarmClock, ArrowLeft, Download, Upload } from 'lucide-react'
import { createElement, useEffect, useRef, useState } from 'react'
import { ApiError } from '../../api/errors'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { ButtonLink } from '../../components/button-link'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage, FormErrors, useFieldErrors } from '../../components/form-errors'
import { FormField, textareaClass } from '../../components/form-field'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { formatDateTime } from '../../lib/format'
import { fmtNum } from '../grades/grade'
import { assessmentsKey, submissionKey, useAssessments, useSubmission, type Assessment, type Submission } from '../quiz/queries'
import { assignmentState, useNow, windowLine } from '../quiz/window'
import { accept, fileIcon, formatSize } from '../section/files'
import { PlainText, countHint } from '../section/plain-text'
import { useUploadPolicy } from '../section/queries'
import { SectionFrame } from '../section/section-frame'
import { finalLine, runUploads, type Failed } from '../section/upload'
import { putSubmission, removeSubmissionFile, submissionFileHref, uploadSubmissionFile } from './queries'

const MAX_FILES = 10
const lengthCopy = { 'validation_failed:body': 'Use at most 20,000 characters of plain text.' }
const noCopy = {}
const backClass = 'mb-3 inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline'
const linkClass =
  'inline-flex items-center gap-1 text-sm text-accent underline hover:text-accent-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus'
const stateLabel = { upcoming: 'Upcoming', open: 'Open', late: 'Past due', closed: 'Closed' }

export function AssignmentPage({ slug, sectionId, assessmentId }: { slug: string; sectionId: string; assessmentId: string }) {
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {(_section, access) => <Assignment slug={slug} sectionId={sectionId} assessmentId={assessmentId} canEdit={access.canEdit} />}
    </SectionFrame>
  )
}

/** Files with a Download link each; `onRemove` adds a Remove button (the student's own open submission). */
export function SubmissionFiles(props: {
  slug: string
  sectionId: string
  aId: string
  userId: string
  files: Submission['files']
  onRemove?: (f: Submission['files'][number]) => void
  busy?: boolean
}) {
  const { slug, sectionId, aId, userId, files, onRemove, busy } = props
  return (
    <ul className="divide-y divide-line rounded border border-line">
      {files.map((f) => (
        <li key={f.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-3 py-2">
          {createElement(fileIcon(f.contentType), { 'aria-hidden': true, className: 'shrink-0 text-ink-muted' })}
          <span className="min-w-0 flex-1 text-sm font-bold wrap-anywhere">
            {f.fileName} <span className="font-normal text-ink-muted">{formatSize(f.sizeBytes)}</span>
          </span>
          <a href={submissionFileHref(slug, sectionId, aId, userId, f.id)} download aria-label={`Download ${f.fileName}`} className={linkClass}>
            <Download aria-hidden="true" />
            Download
          </a>
          {onRemove && (
            <Button variant="ghost-danger" size="sm" disabled={busy} aria-label={`Remove ${f.fileName}`} onClick={() => onRemove(f)}>
              Remove
            </Button>
          )}
        </li>
      ))}
    </ul>
  )
}

function Assignment({ slug, sectionId, assessmentId, canEdit }: { slug: string; sectionId: string; assessmentId: string; canEdit: boolean }) {
  const list = useAssessments(slug, sectionId)
  const sub = useSubmission(slug, sectionId, assessmentId, 'me', !canEdit)
  const now = useNow(30_000)
  const [lapsed, setLapsed] = useState(false) // a write got 409: the server says closed whatever the clock says
  const a = list.data?.items.find((x) => x.id === assessmentId && x.kind === 'assignment')

  const back = (
    <Link to="/$slug/sections/$id/assignments" params={{ slug, id: sectionId }} className={backClass}>
      <ArrowLeft aria-hidden="true" />
      All assignments
    </Link>
  )
  let body: React.ReactNode
  const failed = list.error ?? sub.error
  if (failed) {
    body =
      failed.status === 403 ? (
        <ErrorBanner title="No access">Only this section's staff and enrolled students can open its assignments.</ErrorBanner>
      ) : (
        <ErrorBanner title="Couldn't load the assignment" onRetry={() => void (list.error ? list.refetch() : sub.refetch())}>
          {errorMessage(failed)}
        </ErrorBanner>
      )
  } else if (list.isPending || (!canEdit && sub.isPending)) body = <Loading label="Loading assignment…" />
  else if (!a) body = <ErrorBanner title="Assignment not found">This assignment doesn't exist or was deleted.</ErrorBanner>
  else {
    const state = lapsed ? 'closed' : assignmentState(a, now)
    body = (
      <>
        {lapsed && (
          <div className="mb-4">
            <ErrorBanner title={a.lateUntil ? 'Late submissions are closed' : 'The due date has passed'}>Your submission can no longer be changed.</ErrorBanner>
          </div>
        )}
        <Detail
        slug={slug}
        sectionId={sectionId}
        a={a}
        state={state}
        canEdit={canEdit}
        submission={sub.data ?? null}
        onLapsed={() => setLapsed(true)}
      />
      </>
    )
  }
  return (
    <div className="max-w-3xl">
      {back}
      {body}
    </div>
  )
}

function Detail(props: {
  slug: string
  sectionId: string
  a: Assessment
  state: 'upcoming' | 'open' | 'late' | 'closed'
  canEdit: boolean
  submission: Submission | null
  onLapsed: () => void
}) {
  const { slug, sectionId, a, state, canEdit, submission, onLapsed } = props
  const lead = (
    <>
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
        <h2 className="min-w-0 text-xl font-bold wrap-break-word">{a.name}</h2>
        <Badge tone={state === 'late' ? 'warning' : 'neutral'}>{stateLabel[state]}</Badge>
      </div>
      <p className="mt-1 text-sm text-ink-muted">{windowLine(a, 'assignment')}</p>
      <p className="mt-1 text-sm text-ink-muted">
        {fmtNum(a.maxPoints)} {a.maxPoints === 1 ? 'point' : 'points'}. Weight {fmtNum(a.weight)}%.
      </p>
      {a.description && <PlainText text={a.description} className="mt-4 text-sm text-ink" />}
    </>
  )
  if (canEdit)
    return (
      <>
        {lead}
        <div className="mt-4">
          <ButtonLink to="/$slug/sections/$id/assignments/$assessmentId/submissions" params={{ slug, id: sectionId, assessmentId: a.id }} variant="secondary">
            Submissions
          </ButtonLink>
        </div>
      </>
    )
  return (
    <>
      {lead}
      <div className="mt-6">
        {state === 'upcoming' ? (
          <p className="text-sm text-ink">This assignment opens {a.opensAt ? formatDateTime(a.opensAt) : 'soon'}.</p>
        ) : state === 'open' || state === 'late' ? (
          <>
            {state === 'late' && a.lateUntil && (
              <p className="mb-4 flex items-start gap-2 text-sm text-ink">
                <AlarmClock className="mt-0.5 shrink-0" aria-hidden="true" />
                {a.latePenaltyPercent
                  ? `Late submissions are accepted until ${formatDateTime(a.lateUntil)} with a ${fmtNum(a.latePenaltyPercent)}% penalty.`
                  : `Late submissions are accepted until ${formatDateTime(a.lateUntil)}.`}
              </p>
            )}
            <Form slug={slug} sectionId={sectionId} a={a} submission={submission} onLapsed={onLapsed} />
          </>
        ) : (
          <ReadOnly slug={slug} sectionId={sectionId} a={a} submission={submission} />
        )}
      </div>
      {a.published && submission && <Grade a={a} submission={submission} />}
    </>
  )
}

const submitted = (s: Submission | null) => (
  <p className="text-sm text-ink-muted">{s ? `Submitted ${s.late ? 'late on ' : ''}${formatDateTime(s.submittedAt)}` : 'Not submitted yet.'}</p>
)

function ReadOnly({ slug, sectionId, a, submission: s }: { slug: string; sectionId: string; a: Assessment; submission: Submission | null }) {
  return (
    <section aria-labelledby="your-submission-h" className="flex flex-col gap-3">
      <div className="flex flex-wrap items-baseline justify-between gap-x-4">
        <h3 id="your-submission-h" className="text-base font-bold">
          Your submission
        </h3>
        {submitted(s)}
      </div>
      {s ? (
        <>
          {s.body ? <PlainText text={s.body} className="rounded border border-line bg-surface p-3 text-sm text-ink" /> : <p className="text-sm text-ink-muted">No text submitted.</p>}
          {s.files.length > 0 && <SubmissionFiles slug={slug} sectionId={sectionId} aId={a.id} userId="me" files={s.files} />}
          <p className="text-sm text-ink">
            {a.lateUntil ? 'The late submission window has closed.' : 'The due date has passed.'} Your submission can no longer be changed.
          </p>
        </>
      ) : (
        <p className="text-sm text-ink">You didn't submit this assignment.</p>
      )}
    </section>
  )
}

function Grade({ a, submission: s }: { a: Assessment; submission: Submission }) {
  return (
    <section aria-labelledby="grade-h" className="mt-6 flex flex-col gap-2">
      <h3 id="grade-h" className="text-base font-bold">
        Grade
      </h3>
      {s.score === undefined ? (
        <p className="text-sm text-ink">Not graded.</p>
      ) : (
        <>
          <p className="text-xl font-bold">
            Score {fmtNum(s.score)} / {fmtNum(a.maxPoints)}
            {s.late && s.penaltyPercent > 0 && s.finalScore !== undefined && `, ${s.finalScore.toFixed(2)} after the ${fmtNum(s.penaltyPercent)}% late penalty`}
          </p>
          {s.feedback && (
            <>
              <p className="text-sm font-medium text-ink">Feedback from staff</p>
              <PlainText text={s.feedback} className="text-sm text-ink" />
            </>
          )}
        </>
      )}
    </section>
  )
}

type Upload = { line: string; file?: { name: string; pct: number }; failed: Failed[]; busy: boolean }

function Form({ slug, sectionId, a, submission, onLapsed }: { slug: string; sectionId: string; a: Assessment; submission: Submission | null; onLapsed: () => void }) {
  const qc = useQueryClient()
  const policy = useUploadPolicy().data
  const [text, setText] = useState(submission?.body ?? '')
  const [local, setLocal] = useState('')
  const [notice, setNotice] = useState('')
  const [pageError, setPageError] = useState('')
  const [upload, setUpload] = useState<Upload | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const files = submission?.files ?? []
  const busy = !!upload?.busy
  const key = submissionKey(slug, sectionId, a.id, 'me')

  useEffect(() => {
    if (!busy) return
    const stop = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', stop)
    return () => window.removeEventListener('beforeunload', stop)
  }, [busy])

  // A 409 on any write: the window closed. Tell the student, then let the closed view take over.
  const lapse = () => {
    void qc.invalidateQueries({ queryKey: assessmentsKey(slug, sectionId) })
    onLapsed()
  }
  const closed = (e: unknown) => e instanceof ApiError && e.status === 409

  const save = useMutation({
    mutationFn: () => putSubmission(slug, sectionId, a.id, text.trim()),
    onSuccess: (s) => {
      qc.setQueryData(key, s)
      setNotice('Submission saved.')
    },
    onError: (e) => closed(e) && lapse(),
  })
  const tooLong = save.error instanceof ApiError && save.error.details.some((d) => d.field === 'body' && d.message.includes('20000'))
  const fe = useFieldErrors(save.error, tooLong ? lengthCopy : noCopy)

  const remove = useMutation({
    mutationFn: (f: Submission['files'][number]) => removeSubmissionFile(slug, sectionId, a.id, f.id),
    onSuccess: async (_r, f) => {
      await qc.invalidateQueries({ queryKey: key })
      setNotice(`${f.fileName} removed.`)
    },
    onError: (e) => (closed(e) ? lapse() : setPageError(errorMessage(e))),
  })

  function onSave() {
    setNotice('')
    setPageError('')
    if (!text.trim() && files.length === 0) return setLocal('Write an answer or upload a file.')
    setLocal('')
    save.mutate()
  }

  async function onUpload(picked: File[]) {
    setNotice('')
    setPageError('')
    const room = Math.max(0, MAX_FILES - files.length)
    const send = picked.slice(0, room)
    const refused: Failed[] = picked.slice(room).map((f) => ({ name: f.name, reason: 'A submission can have at most 10 files.' }))
    setUpload({ line: '', failed: [], busy: true })
    const r = await runUploads(
      send,
      policy,
      (f, pct) =>
        uploadSubmissionFile(slug, sectionId, a.id, f, pct).catch((e) => {
          if (closed(e)) lapse()
          throw e
        }),
      (line, file) => setUpload((u) => u && { ...u, line, file }),
    )
    await qc.invalidateQueries({ queryKey: key })
    setUpload({ line: finalLine({ done: r.done, total: picked.length }, 'your submission'), failed: [...r.failed, ...refused], busy: false })
  }

  const failed = upload?.failed ?? []
  return (
    <section aria-labelledby="your-submission-h" className="flex flex-col gap-4">
      <div className="flex flex-wrap items-baseline justify-between gap-x-4">
        <h3 id="your-submission-h" className="text-base font-bold">
          Your submission
        </h3>
        {submitted(submission)}
      </div>
      {pageError && <ErrorBanner>{pageError}</ErrorBanner>}
      {save.error && !closed(save.error) && !fe.body && <FormErrors error={save.error} />}
      <FormField label="Your answer" name="body" hint={countHint(text, 20000)} error={local || fe.body}>
        <textarea rows={8} className={textareaClass} value={text} onChange={(e) => setText(e.target.value)} />
      </FormField>
      <div>
        <Button disabled={save.isPending} onClick={onSave}>
          {save.isPending ? 'Saving…' : 'Save'}
        </Button>
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-baseline justify-between gap-4">
          <h4 className="text-base font-bold">Files</h4>
          <span className="text-sm text-ink-muted">
            {files.length} of {MAX_FILES}
          </span>
        </div>
        {files.length > 0 ? (
          <SubmissionFiles slug={slug} sectionId={sectionId} aId={a.id} userId="me" files={files} busy={remove.isPending || busy} onRemove={(f) => remove.mutate(f)} />
        ) : (
          <p className="text-sm text-ink-muted">No files attached.</p>
        )}
        <div>
          <Button variant="secondary" disabled={busy} onClick={() => inputRef.current?.click()}>
            <Upload aria-hidden="true" />
            {busy ? 'Uploading…' : 'Upload files'}
          </Button>
          <input
            ref={inputRef}
            type="file"
            multiple
            accept={policy ? accept(policy) : undefined}
            hidden
            aria-label="Files for your submission"
            onChange={(e) => {
              const picked = Array.from(e.target.files ?? [])
              e.target.value = ''
              if (picked.length) void onUpload(picked)
            }}
          />
        </div>
        {policy && (
          <p className="text-xs text-ink-muted">
            Up to {MAX_FILES} files, each up to {formatSize(policy.maxBytes)}: PDF, Word, PowerPoint, Excel, images (PNG, JPG, GIF, WebP), ZIP, TXT, Markdown and CSV.
          </p>
        )}
        <p role="status" className="text-sm text-ink">
          {upload?.line}
        </p>
        {upload?.busy && upload.file && (
          <progress value={upload.file.pct} max={100} aria-label={`Uploading ${upload.file.name}`} className="block w-full accent-accent" />
        )}
        {failed.length > 0 && (
          <ErrorBanner title={`${failed.length} ${failed.length === 1 ? 'file' : 'files'} not uploaded`}>
            <ul className="list-disc pl-5">
              {failed.map((f, i) => (
                <li key={i}>
                  <span className="font-bold">{f.name}</span>: {f.reason}
                </li>
              ))}
            </ul>
          </ErrorBanner>
        )}
      </div>
      <StatusMessage message={notice} />
    </section>
  )
}
