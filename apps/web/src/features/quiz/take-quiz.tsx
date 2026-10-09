import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { AlarmClock, ArrowLeft, Check, CircleAlert, Clock } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError } from '../../api/errors'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage, FormErrors } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { Loading } from '../../components/loading'
import { formatDateTime } from '../../lib/format'
import { fmtNum } from '../grades/grade'
import { PlainText } from '../section/plain-text'
import { SectionFrame } from '../section/section-frame'
import { announcement, countdownText } from './countdown'
import {
  attemptKey,
  putAttempt,
  questionsKey,
  startAttempt,
  useAssessmentRefresh,
  useAssessments,
  useAttempt,
  useQuestions,
  type Assessment,
  type Attempt,
  type Question,
} from './queries'
import { Results } from './results'
import { useNow, windowLine, windowState } from './window'

type Answers = Attempt['answers']
type SaveStatus = 'idle' | 'saving' | 'saved' | 'error'

const retryable = (e: unknown) => !(e instanceof ApiError) || e.status >= 500 || e.status === 429
const isConflict = (e: unknown): e is ApiError => e instanceof ApiError && e.status === 409

/**
 * Saves `answers` one second after the last change: one request in flight, a follow-up when the map changed meanwhile,
 * a retry every 5 s after a network or server error. Responses never touch the typed answers.
 * `onStop` receives any other failure (409, 400, ...); saving then stops until `resume`.
 */
export function useAutosave(save: (answers: Answers) => Promise<unknown>, answers: Answers, onStop: (e: unknown, unsaved: boolean) => void) {
  const [status, setStatus] = useState<SaveStatus>('idle')
  const latest = useRef(answers)
  const saved = useRef(answers)
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const inflight = useRef(false)
  const again = useRef(false)
  const stopped = useRef(false)
  const fns = useRef({ save, onStop })
  useEffect(() => {
    fns.current = { save, onStop }
  })

  const runRef = useRef<() => Promise<void>>(async () => {})
  const run = useCallback(async () => {
    timer.current = undefined
    if (stopped.current) return
    if (inflight.current) {
      again.current = true
      return
    }
    inflight.current = true
    setStatus('saving')
    const snap = latest.current
    try {
      await fns.current.save(snap)
      inflight.current = false
      if (stopped.current) return
      saved.current = snap
      if (again.current || latest.current !== snap) {
        again.current = false
        void runRef.current()
        return
      }
      setStatus('saved')
    } catch (e) {
      inflight.current = false
      if (stopped.current) return
      if (retryable(e)) {
        setStatus('error')
        timer.current = setTimeout(() => void runRef.current(), 5000)
      } else {
        stopped.current = true
        fns.current.onStop(e, latest.current !== saved.current)
      }
    }
  }, [])
  useEffect(() => {
    runRef.current = run
  }, [run])

  useEffect(() => {
    if (answers === latest.current) return
    latest.current = answers
    clearTimeout(timer.current)
    if (!stopped.current) timer.current = setTimeout(() => void run(), 1000)
  }, [answers, run])

  useEffect(() => {
    const warn = (e: BeforeUnloadEvent) => {
      if (latest.current !== saved.current && !stopped.current) e.preventDefault()
    }
    window.addEventListener('beforeunload', warn)
    return () => {
      window.removeEventListener('beforeunload', warn)
      // leaving with a change waiting: send it now, fire and forget
      if (timer.current && !stopped.current && latest.current !== saved.current) void fns.current.save(latest.current).catch(() => {})
    }
  }, [])

  return {
    status,
    unsaved: () => latest.current !== saved.current,
    stop: () => {
      stopped.current = true
      clearTimeout(timer.current)
      timer.current = undefined
    },
    resume: () => {
      stopped.current = false
    },
  }
}

const backClass = 'mb-3 inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline'
const pts = (n: number) => `${fmtNum(n)} ${n === 1 ? 'point' : 'points'}`

export function TakeQuizPage({ slug, sectionId, assessmentId }: { slug: string; sectionId: string; assessmentId: string }) {
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {(_section, access) => <TakeQuiz slug={slug} sectionId={sectionId} assessmentId={assessmentId} canEdit={access.canEdit} />}
    </SectionFrame>
  )
}

type Props = { slug: string; sectionId: string; assessmentId: string }

function TakeQuiz({ slug, sectionId, assessmentId, canEdit }: Props & { canEdit: boolean }) {
  const qc = useQueryClient()
  const list = useAssessments(slug, sectionId)
  const refetchList = list.refetch
  useEffect(() => {
    const t = setInterval(() => void refetchList(), 60_000)
    return () => clearInterval(t)
  }, [refetchList])
  const refresh = useAssessmentRefresh(slug, sectionId)
  const now = useNow(1000)
  const [left, setLeft] = useState(false) // the form was left: submitted, or closed and waiting for the server
  const [note, setNote] = useState(false)
  const [startClosed, setStartClosed] = useState(false)
  const [notReady, setNotReady] = useState(false)
  const a = list.data?.items.find((x) => x.id === assessmentId && x.kind === 'quiz')

  const attemptQ = useAttempt(slug, sectionId, assessmentId, 'me', { enabled: !canEdit })
  const refetchAttempt = attemptQ.refetch
  const attempt = attemptQ.data
  const submitted = !!attempt?.submittedAt
  const questionsQ = useQuestions(slug, sectionId, assessmentId, !canEdit && !!attempt)
  // a closed quiz refuses its questions to a student whose attempt is not yet submitted: wait for the sweeper
  const closedUnsubmitted = !!attempt && !submitted && questionsQ.error?.status === 409

  // the sweeper submits an attempt that outlived its window; poll until it shows
  const polling = !!attempt && !submitted && (left || closedUnsubmitted)
  useEffect(() => {
    if (!polling) return
    const t = setInterval(() => void refetchAttempt(), 3000)
    return () => clearInterval(t)
  }, [polling, refetchAttempt])

  useEffect(() => {
    if (submitted) void qc.invalidateQueries({ queryKey: questionsKey(slug, sectionId, assessmentId) })
  }, [submitted, qc, slug, sectionId, assessmentId])

  const start = useMutation({
    mutationFn: () => startAttempt(slug, sectionId, assessmentId),
    onSuccess: (r) => qc.setQueryData(attemptKey(slug, sectionId, assessmentId, 'me'), r),
    onError: (e) => {
      const m = e instanceof ApiError && e.status === 409 ? e.message : ''
      if (m === 'closed') setStartClosed(true)
      else if (m === 'the quiz has no questions') setNotReady(true)
      else if (m === 'not open yet') void list.refetch()
    },
  })

  const back = (
    <Link to="/$slug/sections/$id/quizzes" params={{ slug, id: sectionId }} className={backClass}>
      <ArrowLeft />
      All quizzes
    </Link>
  )

  if (list.isPending) return <Loading />
  if (list.error)
    return (
      <>
        {back}
        <ErrorBanner title="Couldn't load the quiz" onRetry={() => void list.refetch()}>
          {errorMessage(list.error)}
        </ErrorBanner>
      </>
    )
  if (!a)
    return (
      <>
        {back}
        <ErrorBanner title="Quiz not found">This quiz doesn't exist or was deleted.</ErrorBanner>
      </>
    )

  const state = startClosed ? 'closed' : windowState(a, now)
  const n = a.questionCount ?? 0
  const closesText = a.closesAt ? formatDateTime(a.closesAt) : ''
  const opensText = a.opensAt ? formatDateTime(a.opensAt) : ''
  const showForm = !!attempt && !submitted && !left && !closedUnsubmitted

  let body: React.ReactNode
  if (canEdit) {
    body = <p className="text-ink-muted">Staff review attempts from the Quizzes tab.</p>
  } else if (attemptQ.isPending) {
    body = <Loading />
  } else if (attemptQ.error) {
    body = (
      <ErrorBanner title="Couldn't load your attempt" onRetry={() => void attemptQ.refetch()}>
        {errorMessage(attemptQ.error)}
      </ErrorBanner>
    )
  } else if (!attempt) {
    if (state === 'closed') body = <p>This quiz closed on {closesText}. You didn't start it.</p>
    else if (state === 'upcoming') body = <p>This quiz opens {opensText}.</p>
    else if (n === 0 || notReady) body = <p>This quiz isn't ready yet. Check back later.</p>
    else
      body = (
        <div className="flex flex-col items-start gap-3">
          <p>You have one attempt. Your answers are saved as you go and submitted automatically when the quiz closes.</p>
          {start.error && (
            <ErrorBanner title="Couldn't start the quiz">
              {start.error instanceof ApiError && start.error.status === 409 && start.error.message === 'not open yet'
                ? `This quiz isn't open yet. It opens ${opensText}.`
                : errorMessage(start.error)}
            </ErrorBanner>
          )}
          <Button onClick={() => start.mutate()} disabled={start.isPending}>
            {start.isPending ? 'Starting…' : 'Start quiz'}
          </Button>
        </div>
      )
  } else if (showForm) {
    body = questionsQ.isPending ? (
      <Loading />
    ) : questionsQ.error ? (
      <ErrorBanner title="Couldn't load the questions" onRetry={() => void questionsQ.refetch()}>
        {errorMessage(questionsQ.error)}
      </ErrorBanner>
    ) : (
      <QuizForm
        slug={slug}
        sectionId={sectionId}
        assessmentId={assessmentId}
        assessment={a}
        attempt={attempt}
        questions={questionsQ.data.items}
        now={now}
        refetchList={async () => (await list.refetch()).data?.items.find((x) => x.id === assessmentId)}
        onSubmitted={(r) => {
          qc.setQueryData(attemptKey(slug, sectionId, assessmentId, 'me'), r)
          void refresh(questionsKey(slug, sectionId, assessmentId))
          setLeft(true)
        }}
        onClosed={(unsaved) => {
          setNote(unsaved)
          setLeft(true)
          void refetchAttempt()
        }}
      />
    )
  } else {
    body = (
      <>
        {note && (
          <div className="mb-4">
            <ErrorBanner title="Your last change wasn't saved">
              The quiz closed before it reached the server. Your answers as of the closing time were submitted.
            </ErrorBanner>
          </div>
        )}
        <Results assessment={a} attempt={attempt} questions={questionsQ.data?.items ?? []} focusHeading={left && submitted} />
      </>
    )
  }


  return (
    <>
      {back}
      <div className="max-w-3xl">
        <h2 className="text-xl font-bold wrap-break-word">{a.name}</h2>
        <p className="mt-1 text-sm text-ink-muted">{windowLine(a)}</p>
        <p className="mt-1 text-sm text-ink-muted">{n === 0 ? 'No questions yet.' : `${n} ${n === 1 ? 'question' : 'questions'}, ${pts(a.maxPoints)}.`}</p>
        {a.description && <PlainText text={a.description} className="mt-3 max-w-prose" />}
        <div className="mt-6">{body}</div>
      </div>
    </>
  )
}

function QuizForm(props: Props & {
  assessment: Assessment
  attempt: Attempt
  questions: Question[]
  now: number
  refetchList: () => Promise<Assessment | undefined>
  onSubmitted: (a: Attempt) => void
  onClosed: (unsaved: boolean) => void
}) {
  const { slug, sectionId, assessmentId, assessment, attempt, questions, now, refetchList, onSubmitted, onClosed } = props
  const [answers, setAnswers] = useState<Answers>(attempt.answers)
  const [confirm, setConfirm] = useState(false)
  const [pending, setPending] = useState(false)
  const [submitError, setSubmitError] = useState<unknown>(null)
  const [saveError, setSaveError] = useState('')
  const finishing = useRef(false)

  const auto = useAutosave(
    (a) => putAttempt(slug, sectionId, assessmentId, { answers: a }),
    answers,
    (e, unsaved) => {
      if (isConflict(e)) {
        finishing.current = true
        onClosed(!/already submitted/.test(e.message) && unsaved)
      } else if (e instanceof ApiError && e.status === 400) setSaveError("Your answers couldn't be saved. Reload the page.")
      else setSaveError(errorMessage(e))
    },
  )

  const closesAt = assessment.closesAt
  const remaining = closesAt ? Date.parse(closesAt) - now : Infinity
  const isZero = remaining <= 0

  const [announce, setAnnounce] = useState('')
  const prev = useRef(remaining)
  useEffect(() => {
    const text = announcement(prev.current, remaining)
    prev.current = remaining
    if (text) setAnnounce(text)
  }, [remaining])

  // At zero: stop autosave, make sure the server has the latest answers (no submit: the server closes the attempt), then wait.
  const atZero = async () => {
    if (finishing.current) return
    finishing.current = true
    auto.stop()
    const fresh = await refetchList()
    if (fresh?.closesAt && Date.parse(fresh.closesAt) > Date.now()) {
      finishing.current = false
      auto.resume()
      return
    }
    const unsaved = auto.unsaved()
    let lost = false
    try {
      await putAttempt(slug, sectionId, assessmentId, { answers })
    } catch (e) {
      lost = isConflict(e) && !/already submitted/.test(e.message) && unsaved
    }
    onClosed(lost)
  }
  const zeroRef = useRef(atZero)
  useEffect(() => {
    zeroRef.current = atZero
  })
  useEffect(() => {
    if (isZero) void zeroRef.current()
  }, [isZero])

  const submit = async () => {
    if (finishing.current) return
    finishing.current = true
    setPending(true)
    setSubmitError(null)
    const unsaved = auto.unsaved()
    auto.stop()
    try {
      onSubmitted(await putAttempt(slug, sectionId, assessmentId, { answers, submit: true }))
    } catch (e) {
      if (isConflict(e)) {
        setConfirm(false)
        onClosed(!/already submitted/.test(e.message) && unsaved)
        return
      }
      finishing.current = false
      auto.resume()
      setPending(false)
      setSubmitError(e)
    }
  }

  const set = (id: string, v: { choices?: number[]; text?: string }) =>
    setAnswers((cur) => {
      const next = { ...cur }
      if ((v.choices && v.choices.length === 0) || (v.text !== undefined && v.text === '')) delete next[id]
      else next[id] = v
      return next
    })

  const answered = Object.keys(answers).length
  const total = questions.length
  const urgent = remaining < 5 * 60_000
  const CountIcon = urgent ? AlarmClock : Clock

  return (
    <div>
      <div className="sticky top-0 z-10 -mx-1 mb-4 flex flex-wrap items-center justify-between gap-x-4 gap-y-1 border-b border-line bg-canvas px-1 py-2">
        {closesAt && (
          <p className={`flex items-center gap-2 text-sm ${urgent ? 'font-bold text-warning' : 'text-ink'}`}>
            <CountIcon aria-hidden="true" />
            {countdownText(remaining, closesAt)}
          </p>
        )}
        <p role="status" aria-atomic="true" className="text-sm text-ink-muted">
          {auto.status === 'saving' ? (
            <span aria-hidden="true">Saving…</span>
          ) : auto.status === 'saved' ? (
            <span className="inline-flex items-center gap-1">
              <Check aria-hidden="true" />
              Saved
            </span>
          ) : auto.status === 'error' ? (
            <span className="inline-flex items-center gap-1 text-danger">
              <CircleAlert aria-hidden="true" />
              Not saved. Retrying…
            </span>
          ) : (
            'Answers save automatically.'
          )}
        </p>
      </div>
      <p role="status" className="sr-only">
        {announce}
      </p>
      {saveError && (
        <div className="mb-4">
          <ErrorBanner>{saveError}</ErrorBanner>
        </div>
      )}
      <ol className="flex flex-col gap-4">
        {questions.map((q, i) => (
          <li key={q.id}>
            <fieldset className="min-w-0 rounded border border-line bg-surface p-4 sm:p-6">
              <legend className="mb-3 flex flex-col gap-1 p-0">
                <span className="text-sm text-ink-muted">
                  Question {i + 1} of {total}, {pts(q.points)}
                </span>
                <span className="font-bold whitespace-pre-wrap wrap-break-word">{q.prompt}</span>
                {q.kind === 'choice' && <span className="text-sm text-ink-muted">{q.multiple ? 'Choose all that apply.' : 'Choose one answer.'}</span>}
              </legend>
              {q.kind === 'choice' ? (
                <div className="flex flex-col">
                  {q.choices.map((c, ci) => {
                    const chosen = answers[q.id]?.choices ?? []
                    const on = chosen.includes(ci)
                    return (
                      <label key={ci} className="flex min-h-11 items-center gap-3">
                        <input
                          type={q.multiple ? 'checkbox' : 'radio'}
                          name={q.id}
                          value={ci}
                          checked={on}
                          className="size-4 shrink-0 accent-accent"
                          onChange={() => set(q.id, { choices: q.multiple ? (on ? chosen.filter((x) => x !== ci) : [...chosen, ci].sort((x, y) => x - y)) : [ci] })}
                        />
                        <span className="wrap-break-word">{c.text}</span>
                      </label>
                    )
                  })}
                </div>
              ) : (
                <FormField label="Your answer" name={q.id}>
                  <input
                    className={inputClass}
                    maxLength={1000}
                    autoComplete="off"
                    spellCheck={false}
                    value={answers[q.id]?.text ?? ''}
                    onChange={(e) => set(q.id, { text: e.target.value })}
                  />
                </FormField>
              )}
            </fieldset>
          </li>
        ))}
      </ol>
      <Button className="mt-6 w-full sm:w-auto" disabled={pending} onClick={() => setConfirm(true)}>
        Submit quiz
      </Button>
      <Dialog
        open={confirm}
        onClose={() => setConfirm(false)}
        title="Submit quiz?"
        description={`${answered === total ? `You answered all ${total} questions.` : `You answered ${answered} of ${total} questions.`} You can't change your answers after submitting.`}
      >
        {submitError !== null && (
          <div className="mb-3">
            <FormErrors error={submitError} />
          </div>
        )}
        <DialogFooter>
          <Button variant="secondary" disabled={pending} onClick={() => setConfirm(false)}>
            Cancel
          </Button>
          <Button disabled={pending} onClick={() => void submit()}>
            {pending ? 'Submitting…' : 'Submit quiz'}
          </Button>
        </DialogFooter>
      </Dialog>
    </div>
  )
}
