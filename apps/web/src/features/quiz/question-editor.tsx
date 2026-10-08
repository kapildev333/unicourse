import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowDown, ArrowLeft, ArrowUp, ListChecks, Lock, Plus, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { ApiError } from '../../api/errors'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass, inputSizedClass, textareaClass } from '../../components/form-field'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { UnsavedChangesGuard } from '../../components/unsaved-changes'
import { fmtNum, parseDecimal, sentence } from '../grades/grade'
import { PlainText, countHint } from '../section/plain-text'
import { SectionFrame } from '../section/section-frame'
import {
  attemptsKey,
  putQuestions,
  questionsKey,
  useAssessmentRefresh,
  useAssessments,
  useAttempts,
  useQuestions,
  type Assessment,
  type Question,
  type QuestionsPut,
} from './queries'

export type ChoiceDraft = { key: string; text: string; correct: boolean }
export type Draft = { key: string; kind: 'choice' | 'blank'; prompt: string; points: string; choices: ChoiceDraft[]; answers: string }
export type ItemErrors = Partial<Record<'prompt' | 'points' | 'choices' | 'answers', string>>

const newKey = () => crypto.randomUUID()
const msg = {
  prompt: 'Write a prompt of at most 2,000 characters.',
  points: 'Enter points greater than 0 and at most 1,000, like 2 or 2.5.',
  choices: 'Give 2 to 10 choices, each with text of at most 500 characters, and check at least one as correct.',
  answers: 'Give 1 to 20 accepted answers, one per line, each at most 200 characters.',
  count: 'A quiz can have at most 100 questions.',
  locked: "Questions are locked because a student has started this quiz. Changing questions after that isn't supported yet.",
}
const MAX_QUESTIONS = 100

export function toDrafts(items: Question[]): Draft[] {
  return items.map((q) => ({
    key: newKey(),
    kind: q.kind,
    prompt: q.prompt,
    points: fmtNum(q.points),
    choices: q.kind === 'choice' ? q.choices.map((c) => ({ key: newKey(), text: c.text, correct: !!c.correct })) : [],
    answers: (q.answers ?? []).join('\n'),
  }))
}

const lines = (s: string) => s.split('\n').map((l) => l.trim()).filter(Boolean)

export function toBody(drafts: Draft[]): QuestionsPut {
  return {
    items: drafts.map((d) => ({
      kind: d.kind,
      prompt: d.prompt.trim(),
      points: Number(d.points.trim()),
      ...(d.kind === 'choice'
        ? { choices: d.choices.map((c) => ({ text: c.text.trim(), correct: c.correct })) }
        : { answers: lines(d.answers) }),
    })),
  }
}

const len = (s: string) => [...s].length

function validate(drafts: Draft[]): Record<number, ItemErrors> {
  const out: Record<number, ItemErrors> = {}
  drafts.forEach((d, i) => {
    const e: ItemErrors = {}
    const p = d.prompt.trim()
    const pts = parseDecimal(d.points)
    if (!p || len(p) > 2000) e.prompt = msg.prompt
    if (pts === undefined || pts <= 0 || pts > 1000) e.points = msg.points
    if (d.kind === 'choice') {
      const ok = d.choices.length >= 2 && d.choices.length <= 10 && d.choices.every((c) => c.text.trim() && len(c.text.trim()) <= 500) && d.choices.some((c) => c.correct)
      if (!ok) e.choices = msg.choices
    } else {
      const a = lines(d.answers)
      if (a.length < 1 || a.length > 20 || a.some((x) => len(x) > 200)) e.answers = msg.answers
    }
    if (Object.keys(e).length > 0) out[i] = e
  })
  return out
}

const itemField = /^items\[(\d+)\]\.(prompt|points|choices|answers)/

/** `items[i].field` details to per-question errors; anything else is returned as `rest`. */
export function mapItemErrors(details: { field: string; message: string }[]) {
  const errors: Record<number, ItemErrors> = {}
  const rest: { field: string; message: string }[] = []
  for (const d of details) {
    const m = itemField.exec(d.field)
    if (!m) rest.push(d)
    else {
      const e = (errors[Number(m[1])] ??= {})
      const f = m[2] as keyof ItemErrors
      e[f] ??= msg[f]
    }
  }
  return { errors, rest }
}

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`
const totalLine = (n: number, points: number) => `${plural(n, 'question', 'questions')}, ${plural(Number(points.toFixed(2)), 'point', 'points')}`
const sum = (xs: number[]) => xs.reduce((s, x) => s + x, 0)
const backClass = 'mb-3 inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline'

export function QuestionsPage({ slug, sectionId, assessmentId }: { slug: string; sectionId: string; assessmentId: string }) {
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {() => <Questions slug={slug} sectionId={sectionId} assessmentId={assessmentId} />}
    </SectionFrame>
  )
}

function Back({ slug, sectionId }: { slug: string; sectionId: string }) {
  return (
    <Link to="/$slug/sections/$id/quizzes" params={{ slug, id: sectionId }} className={backClass}>
      <ArrowLeft aria-hidden="true" />
      All quizzes
    </Link>
  )
}

function Questions({ slug, sectionId, assessmentId }: { slug: string; sectionId: string; assessmentId: string }) {
  const list = useAssessments(slug, sectionId)
  const qs = useQuestions(slug, sectionId, assessmentId)
  const attempts = useAttempts(slug, sectionId, assessmentId)
  const quiz = list.data?.items.find((a) => a.id === assessmentId && a.kind === 'quiz')

  let body: React.ReactNode
  const failed = list.error ?? qs.error
  if (list.isPending || qs.isPending || attempts.isPending) {
    body = failed ? null : <Loading label="Loading questions…" />
  }
  if (failed) {
    body =
      failed.status === 403 ? (
        <ErrorBanner title="No access">Only this section's staff can change its quizzes.</ErrorBanner>
      ) : (
        <ErrorBanner title="Couldn't load questions" onRetry={() => void (list.error ? list.refetch() : qs.refetch())}>
          {errorMessage(failed)}
        </ErrorBanner>
      )
  } else if (!body) {
    body = !quiz ? (
      <ErrorBanner title="Quiz not found">This quiz doesn't exist or was deleted.</ErrorBanner>
    ) : (
      <Editor
        slug={slug}
        sectionId={sectionId}
        quiz={quiz}
        serverItems={qs.data!.items}
        started={(attempts.data?.items.length ?? 0) > 0}
      />
    )
  }
  return (
    <>
      <Back slug={slug} sectionId={sectionId} />
      {body}
    </>
  )
}

function Editor(props: { slug: string; sectionId: string; quiz: Assessment; serverItems: Question[]; started: boolean }) {
  const { slug, sectionId, quiz, serverItems, started } = props
  const qc = useQueryClient()
  const refresh = useAssessmentRefresh(slug, sectionId)
  const [initial, setInitial] = useState(() => toDrafts(serverItems))
  const [drafts, setDrafts] = useState(initial)
  const [local, setLocal] = useState<Record<number, ItemErrors>>({})
  const [locked, setLocked] = useState(false)
  const [notice, setNotice] = useState('')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const focusNext = useRef('')
  const isLocked = started || locked

  const m = useMutation({
    mutationFn: () => putQuestions(slug, sectionId, quiz.id, toBody(drafts)),
    onSuccess: async (saved) => {
      await refresh(questionsKey(slug, sectionId, quiz.id))
      const next = toDrafts(saved.items)
      setInitial(next)
      setDrafts(next)
      setLocal({})
      setNotice(`Questions saved: ${totalLine(saved.items.length, sum(saved.items.map((q) => q.points)))}.`)
    },
    onError: (e) => {
      setNotice('')
      if (e instanceof ApiError && e.status === 409) {
        setLocked(true)
        void qc.invalidateQueries({ queryKey: questionsKey(slug, sectionId, quiz.id) })
        void qc.invalidateQueries({ queryKey: attemptsKey(slug, sectionId, quiz.id) })
      }
    },
  })
  const mapped = m.error instanceof ApiError && m.error.status === 400 ? mapItemErrors(m.error.details) : { errors: {}, rest: [] }
  const errs: Record<number, ItemErrors> = { ...mapped.errors, ...local }
  const hasLocal = Object.keys(local).length > 0
  useFocusFirstError(formRef, hasLocal ? local : m.error, bannerRef)

  // after an add, move or remove the focused element may be gone: put focus where the design says
  useEffect(() => {
    const target = focusNext.current
    if (!target) return
    focusNext.current = ''
    let el: HTMLElement | null = null
    if (target.startsWith('prompt:')) el = document.querySelector(`[data-prompt="${target.slice(7)}"]`)
    else if (target.startsWith('move:')) {
      const [dir, key] = target.slice(5).split(':')
      el = [`move-${dir}-${key}`, `move-${dir === 'up' ? 'down' : 'up'}-${key}`]
        .map((i) => document.getElementById(i) as HTMLButtonElement | null)
        .find((b) => b && !b.disabled) ?? null
    } else el = document.getElementById(target)
    el?.focus()
  })

  const dirty = JSON.stringify(drafts) !== JSON.stringify(initial)
  const valid = drafts.map((d) => parseDecimal(d.points)).filter((p): p is number => p !== undefined && p > 0)
  const atMax = drafts.length >= MAX_QUESTIONS

  const update = (i: number, patch: Partial<Draft>) => setDrafts((ds) => ds.map((d, j) => (j === i ? { ...d, ...patch } : d)))
  const add = (kind: Draft['kind']) => {
    const d: Draft = {
      key: newKey(),
      kind,
      prompt: '',
      points: '1',
      choices: kind === 'choice' ? [0, 1].map(() => ({ key: newKey(), text: '', correct: false })) : [],
      answers: '',
    }
    focusNext.current = `prompt:${d.key}`
    setDrafts((ds) => [...ds, d])
  }
  const move = (i: number, dir: 'up' | 'down') => {
    const j = dir === 'up' ? i - 1 : i + 1
    focusNext.current = `move:${dir}:${drafts[i].key}`
    setDrafts((ds) => {
      const c = [...ds]
      ;[c[i], c[j]] = [c[j], c[i]]
      return c
    })
  }
  const remove = (i: number) => {
    focusNext.current = i > 0 ? `q-h-${drafts[i - 1].key}` : 'add-choice-question'
    setDrafts((ds) => ds.filter((_, j) => j !== i))
  }

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const v = validate(drafts)
    setLocal(v)
    if (Object.keys(v).length > 0) return
    m.mutate()
  }

  if (isLocked) {
    return (
      <>
        <Header quiz={quiz} total={totalLine(serverItems.length, sum(serverItems.map((q) => q.points)))} />
        <p className="mb-4 flex items-start gap-2 rounded border border-line bg-surface px-4 py-3 text-sm text-ink-muted">
          <Lock className="mt-0.5 shrink-0" aria-hidden="true" />
          {msg.locked}
        </p>
        <ol className="flex flex-col gap-4">
          {serverItems.map((q, i) => (
            <ReadOnlyCard key={q.id} q={q} n={i + 1} />
          ))}
        </ol>
      </>
    )
  }

  const addButtons = (
    <div className="flex flex-wrap gap-2">
      <Button id="add-choice-question" variant="secondary" disabled={atMax} title={atMax ? msg.count : undefined} onClick={() => add('choice')}>
        <Plus aria-hidden="true" />
        Add multiple choice
      </Button>
      <Button variant="secondary" disabled={atMax} title={atMax ? msg.count : undefined} onClick={() => add('blank')}>
        <Plus aria-hidden="true" />
        Add fill in the blank
      </Button>
    </div>
  )
  const unmapped = m.error && (!(m.error instanceof ApiError) || m.error.status !== 400 || mapped.rest.length > 0 || Object.keys(mapped.errors).length === 0) && !(m.error instanceof ApiError && m.error.status === 409)

  return (
    <>
      <UnsavedChangesGuard when={dirty} />
      <Header quiz={quiz} total={totalLine(drafts.length, sum(valid))} />
      {m.error instanceof ApiError && m.error.status === 409 ? null : unmapped && (
        <div className="mb-4">
          <ErrorBanner ref={bannerRef}>
            {errorMessage(m.error)}
            {mapped.rest.length > 0 && (
              <ul className="mt-1 list-disc pl-5">
                {mapped.rest.map((d, i) => (
                  <li key={i}>{d.field === 'items' && /at most 100/.test(d.message) ? msg.count : sentence(d.message)}</li>
                ))}
              </ul>
            )}
          </ErrorBanner>
        </div>
      )}
      <form ref={formRef} noValidate onSubmit={submit}>
        <fieldset disabled={m.isPending} className="contents">
          {drafts.length === 0 ? (
            <EmptyState
              icon={ListChecks}
              title="No questions yet"
              body="Add multiple-choice or fill-in-the-blank questions. Students can't start the quiz until it has at least one."
              action={addButtons}
            />
          ) : (
            <>
              <ol className="flex flex-col gap-4">
                {drafts.map((d, i) => (
                  <li key={d.key}>
                    <article aria-labelledby={`q-h-${d.key}`} className="rounded border border-line bg-surface p-4 sm:p-6">
                      <QuestionCard
                        d={d}
                        i={i}
                        last={i === drafts.length - 1}
                        errors={errs[i] ?? {}}
                        update={(patch) => update(i, patch)}
                        move={(dir) => move(i, dir)}
                        remove={() => remove(i)}
                      />
                    </article>
                  </li>
                ))}
              </ol>
              <div className="mt-4">{addButtons}</div>
            </>
          )}
        </fieldset>
        <div className="sticky bottom-0 mt-6 flex flex-wrap items-center justify-between gap-4 border-t border-line bg-canvas py-3">
          <div>
            <StatusMessage message={notice} />
          </div>
          <Button type="submit" disabled={!dirty || m.isPending}>
            {m.isPending ? 'Saving…' : 'Save questions'}
          </Button>
        </div>
      </form>
    </>
  )
}

function Header({ quiz, total }: { quiz: Assessment; total: string }) {
  return (
    <div className="mb-4 flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <h2 className="text-xl font-bold">Questions: {quiz.name}</h2>
      <p className="text-sm text-ink-muted">{total}</p>
    </div>
  )
}

const kindLabel = { choice: 'Multiple choice', blank: 'Fill in the blank' }

function QuestionCard(props: {
  d: Draft
  i: number
  last: boolean
  errors: ItemErrors
  update: (patch: Partial<Draft>) => void
  move: (dir: 'up' | 'down') => void
  remove: () => void
}) {
  const { d, i, last, errors, update, move, remove } = props
  const n = i + 1
  const setChoice = (j: number, patch: Partial<ChoiceDraft>) =>
    update({ choices: d.choices.map((c, k) => (k === j ? { ...c, ...patch } : c)) })
  const groupErr = `${d.key}-choices-error`
  return (
    <>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <h3 id={`q-h-${d.key}`} tabIndex={-1} className="text-base font-bold focus:outline-none">
            Question {n}
          </h3>
          <Badge tone="neutral">{kindLabel[d.kind]}</Badge>
        </div>
        <div className="flex items-center gap-1">
          <Button id={`move-up-${d.key}`} variant="ghost-neutral" size="sm" disabled={i === 0} aria-label={`Move question ${n} up`} title={`Move question ${n} up`} onClick={() => move('up')}>
            <ArrowUp aria-hidden="true" />
          </Button>
          <Button id={`move-down-${d.key}`} variant="ghost-neutral" size="sm" disabled={last} aria-label={`Move question ${n} down`} title={`Move question ${n} down`} onClick={() => move('down')}>
            <ArrowDown aria-hidden="true" />
          </Button>
          <Button variant="ghost-danger" size="sm" aria-label={`Remove question ${n}`} onClick={remove}>
            Remove
          </Button>
        </div>
      </div>
      <div className="flex flex-col gap-4">
        <FormField label="Prompt" name={`prompt-${n}`} error={errors.prompt} hint={countHint(d.prompt, 2000)}>
          <textarea
            {...{ 'data-prompt': d.key }}
            rows={3}
            className={textareaClass}
            value={d.prompt}
            onChange={(e) => update({ prompt: e.target.value })}
          />
        </FormField>
        <FormField label="Points" name={`points-${n}`} error={errors.points}>
          <input type="text" inputMode="decimal" className={`${inputSizedClass} w-32`} value={d.points} onChange={(e) => update({ points: e.target.value })} />
        </FormField>
        {d.kind === 'choice' ? (
          <fieldset aria-describedby={errors.choices ? groupErr : undefined} className="flex flex-col gap-2">
            <legend className="text-sm font-medium text-ink">Choices</legend>
            <p className="text-xs text-ink-muted">
              Check every correct choice. A student earns the points only by choosing exactly these. With more than one correct choice, students see check boxes and &quot;Choose all that apply.&quot;
            </p>
            {d.choices.map((c, j) => (
              <div key={c.key} className="flex flex-wrap items-center gap-3">
                <input
                  type="text"
                  maxLength={500}
                  aria-label={`Choice ${j + 1}`}
                  aria-invalid={errors.choices && j === 0 ? true : undefined}
                  className={`${inputClass} min-w-0 flex-1`}
                  value={c.text}
                  onChange={(e) => setChoice(j, { text: e.target.value })}
                />
                <label className="flex items-center gap-2 text-sm text-ink">
                  <input type="checkbox" className="size-4 accent-accent" checked={c.correct} onChange={(e) => setChoice(j, { correct: e.target.checked })} />
                  <span>
                    Correct<span className="sr-only"> (choice {j + 1})</span>
                  </span>
                </label>
                <Button
                  variant="ghost-danger"
                  size="sm"
                  disabled={d.choices.length <= 2}
                  aria-label={`Remove choice ${j + 1}`}
                  title={`Remove choice ${j + 1}`}
                  onClick={() => update({ choices: d.choices.filter((_, k) => k !== j) })}
                >
                  <X aria-hidden="true" />
                </Button>
              </div>
            ))}
            {d.choices.length < 10 && (
              <div>
                <Button variant="ghost" size="sm" onClick={() => update({ choices: [...d.choices, { key: newKey(), text: '', correct: false }] })}>
                  <Plus aria-hidden="true" />
                  Add choice
                </Button>
              </div>
            )}
            {errors.choices && (
              <p id={groupErr} role="alert" className="text-sm text-danger">
                <span className="sr-only">Error: </span>
                {errors.choices}
              </p>
            )}
          </fieldset>
        ) : (
          <FormField
            label="Accepted answers"
            name={`answers-${n}`}
            error={errors.answers}
            hint="One per line, up to 20. Capital letters and extra spaces are ignored when matching."
          >
            <textarea rows={3} className={textareaClass} value={d.answers} onChange={(e) => update({ answers: e.target.value })} />
          </FormField>
        )}
      </div>
    </>
  )
}

function ReadOnlyCard({ q, n }: { q: Question; n: number }) {
  return (
    <li>
      <article aria-labelledby={`ro-${q.id}`} className="rounded border border-line bg-surface p-4 sm:p-6">
        <div className="mb-2 flex flex-wrap items-center gap-2">
          <h3 id={`ro-${q.id}`} className="text-base font-bold">
            Question {n}
          </h3>
          <Badge tone="neutral">{kindLabel[q.kind]}</Badge>
          <span className="text-sm text-ink-muted">{plural(q.points, 'point', 'points')}</span>
        </div>
        <PlainText text={q.prompt} className="max-w-prose text-base text-ink" />
        {q.kind === 'choice' ? (
          <ul className="mt-3 flex flex-col gap-1 text-sm">
            {q.choices.map((c, j) => (
              <li key={j} className="flex flex-wrap items-center gap-2">
                <span>{c.text}</span>
                {c.correct && <Badge tone="neutral">Correct</Badge>}
              </li>
            ))}
          </ul>
        ) : (
          <p className="mt-3 text-sm text-ink-muted">Accepted answers: {(q.answers ?? []).join(', ')}</p>
        )}
      </article>
    </li>
  )
}
