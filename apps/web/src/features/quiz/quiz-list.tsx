import { useMutation } from '@tanstack/react-query'
import { ListChecks, Plus } from 'lucide-react'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { ButtonLink } from '../../components/button-link'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { RemoveDialog } from '../courses/remove-dialog'
import { fmtNum } from '../grades/grade'
import type { Section } from '../section/queries'
import { SectionFrame } from '../section/section-frame'
import { WindowDialog } from './quiz-dialog'
import { attemptsKey, useAssessmentRefresh, useAssessments, useAttempt, useAttempts, type Assessment } from './queries'
import { ReleaseButton } from './release'
import { useNow, windowLine, windowState } from './window'

// ponytail: the attempts page is a separate route file (6c); typed loosely until it exists
const attemptsTo = '/$slug/sections/$id/quizzes/$assessmentId/attempts' as never

const stateLabel = { upcoming: 'Upcoming', open: 'Open', closed: 'Closed' }

export function QuizzesPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {(section, access) => <Quizzes slug={slug} section={section} canEdit={access.canEdit} />}
    </SectionFrame>
  )
}

function meta(a: Assessment) {
  const n = a.questionCount ?? 0
  const weight = `Weight ${fmtNum(a.weight)}%.`
  if (n === 0) return `No questions yet. ${weight}`
  return `${n} ${n === 1 ? 'question' : 'questions'}, ${fmtNum(a.maxPoints)} ${a.maxPoints === 1 ? 'point' : 'points'}. ${weight}`
}

function QuizRow(props: {
  slug: string
  sectionId: string
  a: Assessment
  now: number
  canEdit: boolean
  onEdit: () => void
  onDelete: () => void
  onDone: (m: string) => void
  onError: (m: string) => void
}) {
  const { slug, sectionId, a, now, canEdit, onEdit, onDelete, onDone, onError } = props
  const attempt = useAttempt(slug, sectionId, a.id, 'me', { enabled: !canEdit })
  const state = windowState(a, now)
  const mine = attempt.data
  const submitted = !!mine?.submittedAt
  const score = a.published && mine?.score !== undefined ? ` Score ${fmtNum(mine.score)} / ${fmtNum(a.maxPoints)}.` : ''
  const params = { slug, id: sectionId, assessmentId: a.id }

  let student: React.ReactNode = null
  if (!canEdit && !attempt.isPending) {
    const start = state === 'open' && !attempt.isError && !mine && (a.questionCount ?? 0) > 0
    const cont = state === 'open' && !attempt.isError && !!mine && !submitted
    student = (
      <ButtonLink to="/$slug/sections/$id/quizzes/$assessmentId" params={params} size="sm" variant={start || cont ? 'primary' : 'secondary'} aria-label={`${start ? 'Start' : cont ? 'Continue' : 'View'} ${a.name}`}>
        {start ? 'Start' : cont ? 'Continue' : 'View'}
      </ButtonLink>
    )
  }

  return (
    <li>
      <article aria-labelledby={`quiz-${a.id}`} className="rounded border border-line bg-surface p-4 sm:p-6">
        <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
          <h3 id={`quiz-${a.id}`} className="min-w-0 text-base font-bold wrap-break-word">
            {a.name}
          </h3>
          <div className="flex flex-wrap items-center gap-2">
            <Badge tone="neutral">{stateLabel[state]}</Badge>
            {submitted && <Badge tone="neutral">Submitted</Badge>}
            {canEdit && a.published && <Badge tone="neutral">Scores released</Badge>}
            {canEdit && (a.questionCount ?? 0) === 0 && <Badge tone="warning">No questions</Badge>}
          </div>
        </div>
        <p className="mt-1 text-sm text-ink-muted">{windowLine(a)}</p>
        <p className="mt-1 text-sm text-ink-muted">
          {meta(a)}
          {score}
        </p>
        {canEdit ? (
          <div className="mt-3 flex flex-wrap items-center justify-between gap-2">
            <div className="flex flex-wrap items-center gap-2">
              <ButtonLink to="/$slug/sections/$id/quizzes/$assessmentId/questions" params={params} variant="ghost" size="sm" aria-label={`Questions for ${a.name}`}>
                Questions
              </ButtonLink>
              <ButtonLink to={attemptsTo} params={params} variant="ghost" size="sm" aria-label={`Attempts for ${a.name}`}>
                Attempts
              </ButtonLink>
              <ReleaseButton slug={slug} sectionId={sectionId} a={a} onDone={onDone} onError={onError} />
            </div>
            <div className="flex gap-1">
              <Button variant="ghost" size="sm" aria-label={`Edit ${a.name}`} onClick={onEdit}>
                Edit
              </Button>
              <Button variant="ghost-danger" size="sm" aria-label={`Delete ${a.name}`} onClick={onDelete}>
                Delete
              </Button>
            </div>
          </div>
        ) : (
          student && <div className="mt-3">{student}</div>
        )}
      </article>
    </li>
  )
}

function Quizzes({ slug, section, canEdit }: { slug: string; section: Section; canEdit: boolean }) {
  const sectionId = section.id
  const list = useAssessments(slug, sectionId)
  const refresh = useAssessmentRefresh(slug, sectionId)
  const now = useNow(30_000)
  const [dialog, setDialog] = useState<'new' | Assessment | null>(null)
  const [deleting, setDeleting] = useState<Assessment | null>(null)
  const [notice, setNotice] = useState('')
  const [pageError, setPageError] = useState('')
  const headingRef = useRef<HTMLHeadingElement>(null)
  const attempts = useAttempts(slug, sectionId, deleting?.id ?? '', !!deleting)
  const items = (list.data?.items ?? []).filter((a) => a.kind === 'quiz').sort((a, b) => a.position - b.position)

  const del = useMutation({
    mutationFn: (a: Assessment) =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/assessments/{assessmentId}', {
          params: { path: { slug, id: sectionId, assessmentId: a.id } },
        }),
      ),
    onSuccess: async (_r, a) => {
      await refresh(attemptsKey(slug, sectionId, a.id))
      setDeleting(null)
      setNotice(`${a.name} deleted.`)
      headingRef.current?.focus() // the trigger no longer exists
    },
  })

  const newButton = (
    <Button onClick={() => setDialog('new')}>
      <Plus aria-hidden="true" />
      New quiz
    </Button>
  )
  const done = (m: string) => {
    setPageError('')
    setNotice(m)
  }

  let body: React.ReactNode
  if (list.isPending) body = <Loading label="Loading quizzes…" />
  else if (list.error) {
    body =
      list.error.status === 403 ? (
        <ErrorBanner title="No access">Only this section's staff and enrolled students can open its quizzes.</ErrorBanner>
      ) : (
        <ErrorBanner title="Couldn't load quizzes" onRetry={() => void list.refetch()}>
          {errorMessage(list.error)}
        </ErrorBanner>
      )
  } else if (items.length === 0) {
    body = canEdit ? (
      <EmptyState
        icon={ListChecks}
        title="No quizzes yet"
        body="Create a quiz with multiple-choice and fill-in-the-blank questions. It opens and closes on its own, and answers still open at the close are submitted automatically."
        action={newButton}
      />
    ) : (
      <EmptyState icon={ListChecks} title="No quizzes yet" body="Staff haven't added a quiz to this section yet." />
    )
  } else {
    body = (
      <ul aria-labelledby="quizzes-h" className="flex flex-col gap-4">
        {items.map((a) => (
          <QuizRow
            key={a.id}
            slug={slug}
            sectionId={sectionId}
            a={a}
            now={now}
            canEdit={canEdit}
            onEdit={() => setDialog(a)}
            onDelete={() => {
              del.reset()
              setDeleting(a)
            }}
            onDone={done}
            onError={setPageError}
          />
        ))}
      </ul>
    )
  }

  const counting = attempts.isPending
  const started = attempts.data?.items.length ?? 0
  return (
    <>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-4">
        <h2 id="quizzes-h" ref={headingRef} tabIndex={-1} className="text-xl font-bold focus:outline-none">
          Quizzes
        </h2>
        {canEdit && items.length > 0 && newButton}
      </div>
      <StatusMessage message={notice} />
      {pageError && (
        <div className="mb-4">
          <ErrorBanner>{pageError}</ErrorBanner>
        </div>
      )}
      {body}
      {dialog && (
        <WindowDialog
          slug={slug}
          sectionId={sectionId}
          kind="quiz"
          assessment={dialog === 'new' ? undefined : dialog}
          onClose={() => setDialog(null)}
          onSaved={(m) => {
            setDialog(null)
            done(m)
          }}
        />
      )}
      {deleting && (
        <RemoveDialog
          open
          title={`Delete ${deleting.name}?`}
          description={
            counting
              ? 'Checking attempts…'
              : attempts.isError
                ? "Their attempts and scores are deleted too. This can't be undone."
                : started === 0
                ? "No student has started it. Its questions are deleted too. This can't be undone."
                : `${started} ${started === 1 ? 'student has' : 'students have'} started it. Their attempts and scores are deleted too. This can't be undone.`
          }
          confirmLabel="Delete quiz"
          pendingLabel="Deleting…"
          pending={del.isPending || counting}
          error={del.error}
          copy={{ forbidden: "Only this section's staff and administrators can change its quizzes." }}
          onConfirm={() => del.mutate(deleting)}
          onClose={() => setDeleting(null)}
        />
      )}
    </>
  )
}
