import { useMutation } from '@tanstack/react-query'
import { FileText, Plus } from 'lucide-react'
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
import { WindowDialog } from '../quiz/quiz-dialog'
import {
  submissionsKey,
  useAssessmentRefresh,
  useAssessments,
  useSubmission,
  useSubmissions,
  type Assessment,
} from '../quiz/queries'
import { ReleaseButton } from '../quiz/release'
import { useNow, windowLine, windowState } from '../quiz/window'
import type { Section } from '../section/queries'
import { SectionFrame } from '../section/section-frame'

const stateLabel = { upcoming: 'Upcoming', open: 'Open', closed: 'Closed' }

export function AssignmentsPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {(section, access) => <Assignments slug={slug} section={section} canEdit={access.canEdit} />}
    </SectionFrame>
  )
}

function AssignmentRow(props: {
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
  const sub = useSubmission(slug, sectionId, a.id, 'me', !canEdit)
  const state = windowState(a, now)
  const mine = sub.data
  const score = a.published && mine?.score !== undefined ? ` Score ${fmtNum(mine.score)} / ${fmtNum(a.maxPoints)}.` : ''
  const params = { slug, id: sectionId, assessmentId: a.id }

  return (
    <li>
      <article aria-labelledby={`assignment-${a.id}`} className="rounded border border-line bg-surface p-4 sm:p-6">
        <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
          <h3 id={`assignment-${a.id}`} className="min-w-0 text-base font-bold wrap-break-word">
            {a.name}
          </h3>
          <div className="flex flex-wrap items-center gap-2">
            <Badge tone="neutral">{stateLabel[state]}</Badge>
            {mine && <Badge tone="neutral">Submitted</Badge>}
            {canEdit && a.published && <Badge tone="neutral">Scores released</Badge>}
          </div>
        </div>
        <p className="mt-1 text-sm text-ink-muted">{windowLine(a, 'assignment')}</p>
        <p className="mt-1 text-sm text-ink-muted">
          {fmtNum(a.maxPoints)} {a.maxPoints === 1 ? 'point' : 'points'}. Weight {fmtNum(a.weight)}%.{score}
        </p>
        {canEdit ? (
          <div className="mt-3 flex flex-wrap items-center justify-between gap-2">
            <div className="flex flex-wrap items-center gap-2">
              <ButtonLink
                to="/$slug/sections/$id/assignments/$assessmentId/submissions"
                params={params}
                variant="ghost"
                size="sm"
                aria-label={`Submissions for ${a.name}`}
              >
                Submissions
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
          !sub.isPending && (
            <div className="mt-3">
              <ButtonLink
                to="/$slug/sections/$id/assignments/$assessmentId"
                params={params}
                size="sm"
                variant={state === 'open' ? 'primary' : 'secondary'}
                aria-label={`View ${a.name}`}
              >
                View
              </ButtonLink>
            </div>
          )
        )}
      </article>
    </li>
  )
}

function Assignments({ slug, section, canEdit }: { slug: string; section: Section; canEdit: boolean }) {
  const sectionId = section.id
  const list = useAssessments(slug, sectionId)
  const refresh = useAssessmentRefresh(slug, sectionId)
  const now = useNow(30_000)
  const [dialog, setDialog] = useState<'new' | Assessment | null>(null)
  const [deleting, setDeleting] = useState<Assessment | null>(null)
  const [notice, setNotice] = useState('')
  const [pageError, setPageError] = useState('')
  const headingRef = useRef<HTMLHeadingElement>(null)
  const submissions = useSubmissions(slug, sectionId, deleting?.id ?? '', !!deleting)
  const items = (list.data?.items ?? []).filter((a) => a.kind === 'assignment').sort((a, b) => a.position - b.position)

  const del = useMutation({
    mutationFn: (a: Assessment) =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/assessments/{assessmentId}', {
          params: { path: { slug, id: sectionId, assessmentId: a.id } },
        }),
      ),
    onSuccess: async (_r, a) => {
      await refresh(submissionsKey(slug, sectionId, a.id))
      setDeleting(null)
      setNotice(`${a.name} deleted.`)
      headingRef.current?.focus() // the trigger no longer exists
    },
  })

  const newButton = (
    <Button onClick={() => setDialog('new')}>
      <Plus aria-hidden="true" />
      New assignment
    </Button>
  )
  const done = (m: string) => {
    setPageError('')
    setNotice(m)
  }

  let body: React.ReactNode
  if (list.isPending) body = <Loading label="Loading assignments…" />
  else if (list.error) {
    body =
      list.error.status === 403 ? (
        <ErrorBanner title="No access">Only this section's staff and enrolled students can open its assignments.</ErrorBanner>
      ) : (
        <ErrorBanner title="Couldn't load assignments" onRetry={() => void list.refetch()}>
          {errorMessage(list.error)}
        </ErrorBanner>
      )
  } else if (items.length === 0) {
    body = canEdit ? (
      <EmptyState
        icon={FileText}
        title="No assignments yet"
        body="Create an assignment with a due date. Students submit text and files until it is due, and you grade each submission."
        action={newButton}
      />
    ) : (
      <EmptyState icon={FileText} title="No assignments yet" body="Staff haven't added an assignment to this section yet." />
    )
  } else {
    body = (
      <ul aria-labelledby="assignments-h" className="flex flex-col gap-4">
        {items.map((a) => (
          <AssignmentRow
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

  const counting = submissions.isPending
  const n = submissions.data?.items.length ?? 0
  return (
    <>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-4">
        <h2 id="assignments-h" ref={headingRef} tabIndex={-1} className="text-xl font-bold focus:outline-none">
          Assignments
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
          kind="assignment"
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
              ? 'Checking submissions…'
              : n === 0
                ? "No student has submitted. This can't be undone."
                : `${n} ${n === 1 ? 'submission' : 'submissions'} and their files and scores are deleted too. This can't be undone.`
          }
          confirmLabel="Delete assignment"
          pendingLabel="Deleting…"
          pending={del.isPending || counting}
          error={del.error}
          copy={{ forbidden: "Only this section's staff and administrators can change its assignments." }}
          onConfirm={() => del.mutate(deleting)}
          onClose={() => setDeleting(null)}
        />
      )}
    </>
  )
}
