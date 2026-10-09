import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useSearch } from '@tanstack/react-router'
import { ArrowLeft, Check, ClipboardList, X } from 'lucide-react'
import { useRef, useState } from 'react'
import { ApiError } from '../../api/errors'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage, useFieldErrors } from '../../components/form-errors'
import { FormField, inputClass, inputSizedClass, textareaClass } from '../../components/form-field'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { UnsavedChangesGuard } from '../../components/unsaved-changes'
import { formatDateTime } from '../../lib/format'
import { personName } from '../courses/access'
import { fmtNum, parseDecimal } from '../grades/grade'
import { PlainText, countHint } from '../section/plain-text'
import { SectionFrame } from '../section/section-frame'
import {
  attemptKey,
  attemptsKey,
  reviewAttempt,
  useAssessmentRefresh,
  useAssessments,
  useAttempt,
  useAttempts,
  useQuestions,
  type Assessment,
  type Attempt,
  type AttemptRow,
  type Question,
} from './queries'
import { ReleaseButton } from './release'

const backClass = 'mb-3 inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline'
const linkTo = '/$slug/sections/$id/quizzes/$assessmentId/attempts'

export function AttemptsPage({ slug, sectionId, assessmentId, user: userProp }: { slug: string; sectionId: string; assessmentId: string; user?: string }) {
  // the route passes `user`; reading the search here keeps the page usable under any router
  const search = useSearch({ strict: false }) as { user?: string }
  const user = userProp ?? search.user
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {() => <Attempts slug={slug} sectionId={sectionId} assessmentId={assessmentId} user={user} />}
    </SectionFrame>
  )
}

function Attempts({ slug, sectionId, assessmentId, user }: { slug: string; sectionId: string; assessmentId: string; user?: string }) {
  const list = useAssessments(slug, sectionId)
  const attempts = useAttempts(slug, sectionId, assessmentId)
  const [notice, setNotice] = useState('')
  const [pageError, setPageError] = useState('')
  const params = { slug, id: sectionId, assessmentId }
  const quiz = list.data?.items.find((a) => a.id === assessmentId && a.kind === 'quiz')
  const failed = list.error ?? attempts.error

  let body: React.ReactNode
  if (failed) {
    body =
      failed.status === 403 ? (
        <ErrorBanner title="No access">Only this section's staff can review its attempts.</ErrorBanner>
      ) : (
        <ErrorBanner title="Couldn't load attempts" onRetry={() => void (list.error ? list.refetch() : attempts.refetch())}>
          {errorMessage(failed)}
        </ErrorBanner>
      )
  } else if (list.isPending || attempts.isPending) body = <Loading label="Loading attempts…" />
  else if (!quiz) body = <ErrorBanner title="Quiz not found">This quiz doesn't exist or was deleted.</ErrorBanner>
  else {
    const rows = attempts.data?.items ?? []
    const row = user ? rows.find((r) => r.userId === user) : undefined
    body =
      user && row ? (
        <AttemptPanel key={user} slug={slug} sectionId={sectionId} quiz={quiz} row={row} />
      ) : user ? (
        <>
          <Link to={linkTo} params={params} search={{ user: undefined }} className={backClass}>
            <ArrowLeft aria-hidden="true" />
            All attempts
          </Link>
          <ErrorBanner title="Attempt not found">This student has no attempt on this quiz.</ErrorBanner>
        </>
      ) : (
        <AttemptTable
          slug={slug}
          sectionId={sectionId}
          quiz={quiz}
          rows={rows}
          notice={notice}
          pageError={pageError}
          onDone={(m) => {
            setPageError('')
            setNotice(m)
          }}
          onError={(m) => {
            setNotice('')
            setPageError(m)
          }}
        />
      )
  }
  return (
    <>
      {!user && (
        <Link to="/$slug/sections/$id/quizzes" params={{ slug, id: sectionId }} className={backClass}>
          <ArrowLeft aria-hidden="true" />
          All quizzes
        </Link>
      )}
      {body}
    </>
  )
}

const submitted = (r: AttemptRow) => !!r.submittedAt
const unreviewed = (rows: AttemptRow[]) => rows.filter((r) => submitted(r) && !r.reviewedAt).length
const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`

function AttemptTable(props: {
  slug: string
  sectionId: string
  quiz: Assessment
  rows: AttemptRow[]
  notice: string
  pageError: string
  onDone: (m: string) => void
  onError: (m: string) => void
}) {
  const { slug, sectionId, quiz, rows, notice, pageError, onDone, onError } = props
  const count = unreviewed(rows)
  const total = rows.filter(submitted).length
  const line =
    total === 0
      ? 'No attempt is submitted yet.'
      : count === 0
        ? 'All submitted attempts are reviewed.'
        : `${count} of ${plural(total, 'submitted attempt', 'submitted attempts')} ${count === 1 ? "isn't" : "aren't"} reviewed yet.`
  return (
    <>
      <div className="mb-4 flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
        <div>
          <h2 className="text-xl font-bold">Attempts: {quiz.name}</h2>
          {rows.length > 0 && <p className="text-sm text-ink-muted">{line}</p>}
        </div>
        <ReleaseButton
          slug={slug}
          sectionId={sectionId}
          a={quiz}
          variant="secondary"
          note={count > 0 ? `${plural(count, "submitted attempt isn't", "submitted attempts aren't")} reviewed yet.` : undefined}
          onDone={onDone}
          onError={onError}
        />
      </div>
      {pageError && (
        <div className="mb-4">
          <ErrorBanner>{pageError}</ErrorBanner>
        </div>
      )}
      <StatusMessage message={notice} />
      {rows.length === 0 ? (
        <EmptyState icon={ClipboardList} title="No attempts yet" body="A student's attempt appears here as soon as they start the quiz." />
      ) : (
        <Table caption={`Attempts for ${quiz.name}`} stack>
          <thead>
            <tr>
              <th scope="col">Student</th>
              <th scope="col">Started</th>
              <th scope="col">Submitted</th>
              <th scope="col">Score</th>
              <th scope="col">Reviewed</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.userId}>
                <th scope="row" data-label="Student" className="font-bold max-sm:bg-surface">
                  <Link
                    to={linkTo}
                    params={{ slug, id: sectionId, assessmentId: quiz.id }}
                    search={{ user: r.userId }}
                    className="text-accent hover:underline"
                  >
                    {personName(r)}
                  </Link>
                  {r.displayName && <div className="text-xs font-normal text-ink-muted">{r.email}</div>}
                </th>
                <td data-label="Started">{formatDateTime(r.startedAt)}</td>
                <td data-label="Submitted">
                  {r.submittedAt ? (
                    <span className="inline-flex flex-wrap items-center gap-2">
                      {formatDateTime(r.submittedAt)}
                      {r.autoSubmitted && <Badge tone="neutral">Auto</Badge>}
                    </span>
                  ) : (
                    <span className="text-ink-muted">In progress</span>
                  )}
                </td>
                <td data-label="Score">
                  {r.score === undefined ? <span className="text-ink-muted">Not submitted</span> : `${fmtNum(r.score)} / ${fmtNum(quiz.maxPoints)}`}
                </td>
                <td data-label="Reviewed">
                  {r.reviewedAt ? formatDateTime(r.reviewedAt) : <span className="text-ink-muted">Not reviewed</span>}
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
      )}
    </>
  )
}

type Form = { points: Record<string, string>; feedback: Record<string, string>; overall: string }

const toForm = (qs: Question[], a: Attempt): Form => ({
  points: Object.fromEntries(qs.map((q) => [q.id, fmtNum(a.grading?.[q.id]?.points ?? 0)])),
  feedback: Object.fromEntries(qs.map((q) => [q.id, a.grading?.[q.id]?.feedback ?? ''])),
  overall: a.feedback ?? '',
})

function AttemptPanel({ slug, sectionId, quiz, row }: { slug: string; sectionId: string; quiz: Assessment; row: AttemptRow }) {
  const attempt = useAttempt(slug, sectionId, quiz.id, row.userId)
  const qs = useQuestions(slug, sectionId, quiz.id)
  const failed = attempt.error ?? qs.error
  const name = personName(row)
  const back = (
    <Link to={linkTo} params={{ slug, id: sectionId, assessmentId: quiz.id }} search={{ user: undefined }} className={backClass}>
      <ArrowLeft aria-hidden="true" />
      All attempts
    </Link>
  )
  let body: React.ReactNode
  if (failed) {
    body = (
      <ErrorBanner title="Couldn't load the attempt" onRetry={() => void (attempt.error ? attempt.refetch() : qs.refetch())}>
        {errorMessage(failed)}
      </ErrorBanner>
    )
  } else if (attempt.isPending || qs.isPending) body = <Loading label="Loading attempt…" />
  else if (!attempt.data) body = <ErrorBanner title="Attempt not found">This student has no attempt on this quiz.</ErrorBanner>
  else body = <Review slug={slug} sectionId={sectionId} quiz={quiz} row={row} name={name} attempt={attempt.data} questions={qs.data?.items ?? []} />
  return (
    <>
      {back}
      {body}
    </>
  )
}

function Review(props: { slug: string; sectionId: string; quiz: Assessment; row: AttemptRow; name: string; attempt: Attempt; questions: Question[] }) {
  const { slug, sectionId, quiz, row, name, attempt, questions } = props
  const qc = useQueryClient()
  const refresh = useAssessmentRefresh(slug, sectionId)
  const [initial, setInitial] = useState(() => toForm(questions, attempt))
  const [form, setForm] = useState(initial)
  const [local, setLocal] = useState<Record<string, string>>({})
  const [notice, setNotice] = useState('')
  const formRef = useRef<HTMLFormElement>(null)

  const changed = (q: Question) => form.points[q.id] !== initial.points[q.id] || form.feedback[q.id] !== initial.feedback[q.id]
  const dirty = form.overall !== initial.overall || questions.some(changed)
  const live = questions.reduce((s, q) => s + (parseDecimal(form.points[q.id]) ?? parseDecimal(initial.points[q.id]) ?? 0), 0)
  const maxTotal = questions.reduce((s, q) => s + q.points, 0)

  const m = useMutation({
    mutationFn: () => {
      const grading: Record<string, { points: number; feedback?: string }> = {}
      for (const q of questions.filter(changed))
        grading[q.id] = {
          points: parseDecimal(form.points[q.id])!,
          ...(form.feedback[q.id] !== initial.feedback[q.id] ? { feedback: form.feedback[q.id] } : {}),
        }
      return reviewAttempt(slug, sectionId, quiz.id, row.userId, {
        ...(Object.keys(grading).length > 0 ? { grading } : {}),
        ...(form.overall !== initial.overall ? { feedback: form.overall } : {}),
      })
    },
    onSuccess: async (saved) => {
      qc.setQueryData(attemptKey(slug, sectionId, quiz.id, row.userId), saved)
      await refresh(attemptsKey(slug, sectionId, quiz.id))
      const next = toForm(questions, saved)
      setInitial(next)
      setForm(next)
      setLocal({})
      setNotice(`Review saved. ${name}: ${fmtNum(saved.score ?? 0)} / ${fmtNum(quiz.maxPoints)}.`)
    },
    onError: (e) => {
      setNotice('')
      if (e instanceof ApiError && e.status === 409) void qc.invalidateQueries({ queryKey: attemptKey(slug, sectionId, quiz.id, row.userId) })
    },
  })
  const server = useFieldErrors(m.error)
  const conflict = m.error instanceof ApiError && m.error.status === 409
  const fieldErr = (f: string) => local[f] ?? server[f]
  const hasFieldErr = Object.keys(local).length > 0 || (m.error instanceof ApiError && m.error.status === 400 && m.error.details.length > 0 && m.error.details.every((d) => /^(grading\.[^.]+\.(points|feedback)|feedback)$/.test(d.field)))

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const errs: Record<string, string> = {}
    for (const q of questions) {
      const p = parseDecimal(form.points[q.id])
      if (p === undefined || p < 0 || p > q.points) errs[`grading.${q.id}.points`] = `Enter 0 to ${fmtNum(q.points)} points.`
    }
    setLocal(errs)
    if (Object.keys(errs).length > 0) {
      queueMicrotask(() => formRef.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus())
      return
    }
    if (!dirty) {
      setNotice('Nothing to save.')
      return
    }
    setNotice('')
    m.mutate()
  }

  const times = attempt.submittedAt
    ? attempt.autoSubmitted
      ? `Started ${formatDateTime(attempt.startedAt)} · Submitted automatically at ${formatDateTime(attempt.submittedAt)}`
      : `Started ${formatDateTime(attempt.startedAt)} · Submitted ${formatDateTime(attempt.submittedAt)}`
    : `Started ${formatDateTime(attempt.startedAt)}`

  return (
    <>
      <UnsavedChangesGuard when={dirty} />
      <h3 className="text-lg font-bold">
        {name}
      </h3>
      <p className="text-sm text-ink-muted">{times}</p>
      {attempt.submittedAt ? (
        <p className="mb-4 text-sm font-bold">
          Score {fmtNum(Number(live.toFixed(2)))} / {fmtNum(maxTotal)}
          {dirty && <span className="font-normal text-ink-muted"> (unsaved)</span>}
        </p>
      ) : (
        <p className="mt-4 rounded border border-line bg-surface px-4 py-3 text-sm text-ink-muted">
          This attempt isn't submitted yet. You can review it once the student submits or the quiz closes.
        </p>
      )}
      {attempt.submittedAt && (
        <>
          {conflict && (
            <div className="mb-4">
              <ErrorBanner>This attempt isn't submitted yet, so it can't be reviewed.</ErrorBanner>
            </div>
          )}
          {m.error && !conflict && !hasFieldErr && (
            <div className="mb-4">
              <ErrorBanner>{errorMessage(m.error)}</ErrorBanner>
            </div>
          )}
          <form ref={formRef} noValidate onSubmit={submit}>
            <fieldset disabled={m.isPending} className="contents">
              <ol className="flex flex-col gap-4">
                {questions.map((q, i) => (
                  <li key={q.id}>
                    <QuestionReview
                      q={q}
                      n={i + 1}
                      attempt={attempt}
                      points={form.points[q.id]}
                      feedback={form.feedback[q.id]}
                      pointsError={fieldErr(`grading.${q.id}.points`)}
                      feedbackError={server[`grading.${q.id}.feedback`]}
                      onPoints={(v) => setForm((f) => ({ ...f, points: { ...f.points, [q.id]: v } }))}
                      onFeedback={(v) => setForm((f) => ({ ...f, feedback: { ...f.feedback, [q.id]: v } }))}
                    />
                  </li>
                ))}
              </ol>
              <div className="mt-6">
                <FormField label={`Feedback for ${name}`} name="feedback" error={server.feedback} hint={countHint(form.overall, 5000)}>
                  <textarea
                    rows={3}
                    className={textareaClass}
                    value={form.overall}
                    onChange={(e) => setForm((f) => ({ ...f, overall: e.target.value }))}
                  />
                </FormField>
              </div>
            </fieldset>
            <div className="sticky bottom-0 mt-6 flex flex-wrap items-center justify-between gap-4 border-t border-line bg-canvas py-3">
              <div>
                <StatusMessage message={notice} />
              </div>
              <Button type="submit" disabled={m.isPending}>
                {m.isPending ? 'Saving…' : 'Save review'}
              </Button>
            </div>
          </form>
        </>
      )}
    </>
  )
}

function answerText(q: Question, a: Attempt): string {
  const ans = a.answers[q.id]
  if (q.kind === 'blank') return ans?.text?.trim() ? ans.text : 'No answer'
  const picked = (ans?.choices ?? []).map((i) => q.choices[i]?.text).filter(Boolean)
  return picked.length > 0 ? picked.join(', ') : 'No answer'
}

function QuestionReview(props: {
  q: Question
  n: number
  attempt: Attempt
  points: string
  feedback: string
  pointsError?: string
  feedbackError?: string
  onPoints: (v: string) => void
  onFeedback: (v: string) => void
}) {
  const { q, n, attempt, points, feedback, pointsError, feedbackError, onPoints, onFeedback } = props
  const mark = attempt.grading?.[q.id]
  const key = q.kind === 'blank' ? (q.answers ?? []).join(', ') : q.choices.filter((c) => c.correct).map((c) => c.text).join(', ')
  return (
    <article aria-labelledby={`rq-${q.id}`} className="rounded border border-line bg-surface p-4 sm:p-6">
      <h4 id={`rq-${q.id}`} className="mb-2 text-base font-bold">
        Question {n}, {plural(q.points, 'point', 'points')}
      </h4>
      <PlainText text={q.prompt} className="max-w-prose text-base text-ink" />
      <dl className="mt-3 flex flex-col gap-1 text-sm">
        <div>
          <dt className="inline font-bold">Answer: </dt>
          <dd className="inline">{answerText(q, attempt)}</dd>
        </div>
        <div>
          <dt className="inline font-bold">{q.kind === 'blank' ? 'Accepted answers: ' : q.multiple ? 'Correct answers: ' : 'Correct answer: '}</dt>
          <dd className="inline">{key}</dd>
        </div>
        {mark && (
          <div className="flex flex-wrap items-center gap-2">
            <dt className="font-bold">Graded: </dt>
            <dd className="inline-flex items-center gap-2">
              {fmtNum(mark.points)} of {fmtNum(q.points)} points
              <Badge tone="neutral">
                {mark.correct ? <Check aria-hidden="true" /> : <X aria-hidden="true" />}
                {mark.correct ? 'Correct' : 'Incorrect'}
              </Badge>
            </dd>
          </div>
        )}
      </dl>
      <div className="mt-4 flex flex-wrap items-start gap-4">
        <FormField label={`Points for question ${n}`} name={`points-${n}`} error={pointsError} hint={`Out of ${fmtNum(q.points)}.`}>
          <input type="text" inputMode="decimal" className={`${inputSizedClass} w-24`} value={points} onChange={(e) => onPoints(e.target.value)} />
        </FormField>
        <div className="min-w-48 flex-1">
          <FormField label={`Feedback on question ${n}`} name={`feedback-${n}`} error={feedbackError} hint={countHint(feedback, 2000)}>
            <input type="text" className={inputClass} value={feedback} onChange={(e) => onFeedback(e.target.value)} />
          </FormField>
        </div>
      </div>
    </article>
  )
}
