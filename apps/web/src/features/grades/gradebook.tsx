import { useMutation } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { ApiError } from '../../api/errors'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { formatDate } from '../../lib/format'
import { useMe, useUniversity } from '../auth/use-me'
import { personName } from '../courses/access'
import { RemoveDialog } from '../courses/remove-dialog'
import { AssessmentDialog } from './assessment-dialog'
import { FinalizeDialog, finalizeCopy, OverrideDialog } from './finalize-dialog'
import { fmt2, fmtNum, gradebookAccess, parsePoints, round2, sentence, totalsHundred, weightTotal } from './grade'
import {
  downloadGradebookCsv,
  putScores,
  useGradebook,
  useGradeScale,
  useGradesRefresh,
  useSection,
  type Assessment,
  type Gradebook,
} from './queries'
import { ScoreCell, type Cell } from './score-cell'
import { ScoresImportDialog } from './scores-import-dialog'

type Student = Gradebook['students'][number]

const scoreCopy: Record<string, string> = {
  'validation_failed:points': 'Enter a number from 0 to twice the maximum points.',
  'validation_failed:userId': 'This student is no longer enrolled. Reload the page.',
  forbidden: "You can't enter scores in this section.",
}
const sectionCopy = {
  not_found: "This section doesn't exist or was deleted.",
  forbidden: "You aren't staff on this section.",
}
const capName = 'max-w-48 truncate sm:w-px sm:max-w-none sm:overflow-visible sm:whitespace-nowrap'
const key = (s: Student, a: Assessment) => `${s.userId}:${a.id}`
const serverText = (s: Student, a: Assessment) => (s.scores[a.id] === undefined ? '' : fmtNum(s.scores[a.id]))

function PublishToggle(props: {
  slug: string
  sectionId: string
  a: Assessment
  onDone: (message: string) => void
  onError: (message: string) => void
}) {
  const { slug, sectionId, a, onDone, onError } = props
  const refresh = useGradesRefresh(slug)
  const m = useMutation({
    mutationFn: () =>
      unwrap(
        api.PATCH('/universities/{slug}/sections/{id}/assessments/{assessmentId}', {
          params: { path: { slug, id: sectionId, assessmentId: a.id } },
          body: { published: !a.published },
        }),
      ),
    onSuccess: async () => {
      await refresh()
      onDone(
        a.published
          ? `${a.name} unpublished. Students no longer see its scores.`
          : `${a.name} published. Students can now see its scores.`,
      )
    },
    onError: (e) => onError(`Couldn't change ${a.name}: ${errorMessage(e)}`),
  })
  return (
    <label className="mt-1 inline-flex items-center gap-1.5 text-xs font-normal normal-case">
      <input
        type="checkbox"
        className="accent-accent"
        checked={a.published}
        disabled={m.isPending}
        onChange={() => m.mutate()}
      />
      Published<span className="sr-only"> {a.name}</span>
    </label>
  )
}

function FinalCell({ s }: { s: Student }) {
  const f = s.final
  if (!f) return <>—</>
  return (
    <>
      {f.status === 'graded' ? (
        <>
          <span className="font-bold">{f.letter}</span>
          {f.points !== undefined && <span className="text-ink-muted"> {fmt2(f.points)}</span>}
        </>
      ) : (
        <span>{f.status === 'incomplete' ? 'Incomplete' : 'Withdrawn'}</span>
      )}
      {f.overrideReason && (
        <>
          {' '}
          <Badge tone="neutral" className="align-middle">
            <span title={f.overrideReason}>Changed</span>
            <span className="sr-only">: {f.overrideReason}</span>
          </Badge>
        </>
      )}
    </>
  )
}

export function GradebookPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  const { membership } = useUniversity()
  const me = useMe().data
  const sq = useSection(slug, sectionId)
  const section = sq.data
  const access = section && me ? gradebookAccess(membership, me.user.id, section) : undefined
  const gq = useGradebook(slug, sectionId, !!access?.view)
  const scale = useGradeScale(slug).data
  const refreshGrades = useGradesRefresh(slug)

  const [cells, setCellsState] = useState<Record<string, Cell>>({})
  const cellsRef = useRef(cells)
  const patch = (fn: (c: Record<string, Cell>) => Record<string, Cell>) => {
    cellsRef.current = fn(cellsRef.current)
    setCellsState(cellsRef.current)
  }
  const [announce, setAnnounce] = useState('')
  const [message, setMessage] = useState('')
  const [pageError, setPageError] = useState('')
  const [dialog, setDialog] = useState<'new' | Assessment | null>(null)
  const [importing, setImporting] = useState(false)
  const [finalizing, setFinalizing] = useState(false)
  const [reopening, setReopening] = useState(false)
  const [changing, setChanging] = useState<Student>()
  const reopen = useMutation({
    mutationFn: () =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/final-grades', { params: { path: { slug, id: sectionId } } }),
      ),
    onSuccess: async () => {
      await refreshGrades()
      setReopening(false)
      setPageError('')
      setMessage('Grades reopened. Finalize again when scores are ready.')
    },
  })
  const exportCsv = useMutation({
    mutationFn: (fileName: string) => downloadGradebookCsv(slug, sectionId, fileName),
    onSuccess: () => setPageError(''),
    onError: (e) => setPageError(`Couldn't export the gradebook: ${errorMessage(e)}`),
  })

  const back = (label: string) => (
    <Link
      to="/$slug/sections/$id"
      params={{ slug, id: sectionId }}
      className="mb-2 inline-block text-sm text-accent hover:underline"
    >
      ← {label}
    </Link>
  )

  if (sq.isPending || !me) return <Loading />
  if (sq.error) {
    const title = sq.error.status === 404 ? 'Section not found' : sq.error.status === 403 ? 'No access' : undefined
    return (
      <>
        {back('Section')}
        <ErrorBanner
          title={title ?? "Couldn't load the section"}
          onRetry={title ? undefined : () => void sq.refetch()}
        >
          {errorMessage(sq.error, sectionCopy)}
        </ErrorBanner>
      </>
    )
  }
  const sec = sq.data
  const code = `${sec.course.code}-${sec.number}`
  const noAccess = (
    <ErrorBanner title="No access">
      Only this section&apos;s staff and administrators can open its gradebook.
    </ErrorBanner>
  )
  if (!access?.view)
    return (
      <>
        {back(`${sec.course.code} section ${sec.number}`)}
        {noAccess}
      </>
    )
  if (gq.isPending)
    return (
      <>
        {back(`${sec.course.code} section ${sec.number}`)}
        <Loading variant="table" columns={['Student', 'Percent', 'Letter']} label="Loading the gradebook…" />
      </>
    )
  if (gq.error)
    return (
      <>
        {back(`${sec.course.code} section ${sec.number}`)}
        {gq.error.status === 403 ? (
          noAccess
        ) : (
          <ErrorBanner title="Couldn't load the gradebook" onRetry={() => void gq.refetch()}>
            {errorMessage(gq.error)}
          </ErrorBanner>
        )}
      </>
    )

  const { assessments, students, stats } = gq.data
  const finalized = students.some((s) => s.final)
  const total = weightTotal(assessments)
  const finalizeBlock = !scale
    ? 'Loading the grade scale…'
    : students.length === 0
      ? 'No students to grade.'
      : assessments.length === 0
        ? 'Add assessments before you finalize.'
        : !totalsHundred(assessments)
          ? `Weights must total 100% to finalize. They total ${fmtNum(weightTotal(assessments))}%.`
          : ''

  // ---- cell state (design 3.3) ----
  const setDraft = (k: string, draft: string) =>
    patch((c) => ({ ...c, [k]: { draft, sent: c[k]?.sent, status: 'idle' } }))
  const drop = (k: string) => patch((c) => Object.fromEntries(Object.entries(c).filter(([x]) => x !== k)))

  async function save(a: Assessment, entries: { student: Student; text: string }[], note = '') {
    const batch: { student: Student; text: string; points: number | null }[] = []
    let firstError = ''
    for (const e of entries) {
      const k = key(e.student, a)
      const r = parsePoints(e.text, a.maxPoints)
      if ('error' in r) {
        patch((c) => ({
          ...c,
          [k]: { draft: e.text, status: 'error', message: r.error },
        }))
        firstError ||= `Not saved: ${personName(e.student)}, ${a.name}. ${r.error}`
      } else if (r.points === null && serverText(e.student, a) === '') drop(k)
      else batch.push({ ...e, points: r.points })
    }
    if (batch.length === 0) {
      if (firstError || note) setAnnounce(`${firstError} ${note}`.trim())
      return
    }
    // a new batch clears earlier "Saved" labels
    patch((c) => {
      const next: Record<string, Cell> = {}
      for (const [k, v] of Object.entries(c)) if (v.status !== 'saved') next[k] = v
      for (const b of batch)
        next[key(b.student, a)] = {
          draft: b.text,
          sent: b.text,
          status: 'pending',
        }
      return next
    })
    try {
      for (let i = 0; i < batch.length; i += 500)
        await putScores(
          slug,
          sectionId,
          a.id,
          batch.slice(i, i + 500).map((b) => ({ userId: b.student.userId, points: b.points })),
        )
      patch((c) => {
        const next = { ...c }
        for (const b of batch)
          if (c[key(b.student, a)]?.draft === b.text)
            next[key(b.student, a)] = { draft: b.text, sent: b.text, status: 'saved' }
        return next
      })
      setPageError('')
      setAnnounce(
        batch.length === 1
          ? `Saved: ${personName(batch[0].student)}, ${a.name}. ${note}`.trim()
          : `Saved ${batch.length} scores for ${a.name}. ${note}`.trim(),
      )
      await refreshGrades()
    } catch (e) {
      const messages: Record<number, string> = {}
      if (e instanceof ApiError && e.code === 'validation_failed')
        for (const d of e.details) {
          const m = /^scores\[(\d+)\]\.(\w+)$/.exec(d.field)
          if (m) messages[Number(m[1])] ||= scoreCopy[`validation_failed:${m[2]}`] ?? sentence(d.message)
        }
      const mapped = Object.keys(messages).length > 0
      patch((c) => {
        const next = { ...c }
        batch.forEach((b, i) => {
          if (c[key(b.student, a)]?.draft !== b.text) return
          next[key(b.student, a)] = {
            draft: b.text,
            status: 'error',
            message: mapped ? (messages[i] ?? 'Not saved: another score in this paste has an error.') : 'Not saved.',
          }
        })
        return next
      })
      if (!mapped) setPageError(errorMessage(e, scoreCopy))
      const first = batch[0]
      setAnnounce(
        batch.length === 1
          ? `Not saved: ${personName(first.student)}, ${a.name}. ${messages[0] ?? 'Not saved.'}`
          : `${batch.length} scores not saved for ${a.name}.`,
      )
    }
  }

  function commit(s: Student, a: Assessment) {
    const k = key(s, a)
    const c = cellsRef.current[k]
    if (!c || c.status === 'pending' || c.draft === (c.sent ?? serverText(s, a))) return
    if (c.draft === serverText(s, a) && c.sent === undefined) return drop(k)
    void save(a, [{ student: s, text: c.draft }])
  }

  function paste(a: Assessment, row: number, lines: string[]) {
    const batch: { student: Student; text: string }[] = []
    let dropped = 0
    lines.forEach((text, i) => {
      const student = students[row + i]
      if (!student) dropped++
      else if (text !== '') batch.push({ student, text })
    })
    for (const b of batch) setDraft(key(b.student, a), b.text)
    void save(
      a,
      batch,
      dropped > 0 ? `${dropped} pasted values didn't fit below the last student and were ignored.` : '',
    )
  }

  const letters = Object.entries(stats.letters)
    .filter(([, n]) => n > 0)
    .sort(([a], [b]) => {
    const pos = (l: string) => scale?.rows.find((r) => r.letter === l)?.minPercent ?? -1
    return pos(b) - pos(a)
  })
  const editing = access.assessments
  const missingCount = (s: Student) => assessments.filter((a) => a.published && s.scores[a.id] === undefined).length
  const scoredCount = (a: Assessment) => students.filter((s) => s.scores[a.id] !== undefined).length
  const newButton = (
    <Button variant="secondary" onClick={() => setDialog('new')}>
      New assessment
    </Button>
  )

  return (
    <>
      {back(`${sec.course.code} section ${sec.number}`)}
      <PageHeader
        title="Gradebook"
        action={
          access.finalize &&
          (finalized ? (
            <Button variant="secondary" onClick={() => setReopening(true)}>
              Reopen grades
            </Button>
          ) : (
            <div>
              <Button
                disabled={!!finalizeBlock}
                aria-describedby={finalizeBlock ? 'finalize-reason' : undefined}
                onClick={() => setFinalizing(true)}
              >
                Finalize grades
              </Button>
              {finalizeBlock && (
                <p id="finalize-reason" className="mt-1 text-xs text-ink-muted">
                  {finalizeBlock}
                </p>
              )}
            </div>
          ))
        }
        description={
          <>
            {`${sec.course.code} · Section ${sec.number} · ${sec.course.title}, ${sec.term.name}`}
            {finalized && (
              <>
                {' '}
                <Badge tone="neutral" className="align-middle">
                  Finalized
                </Badge>
              </>
            )}
          </>
        }
      />
      <div className="mb-4 flex flex-wrap items-center gap-4">
        <span className="text-sm">Weights total {fmtNum(total)}%</span>
        {!totalsHundred(assessments) && <Badge tone="warning">Must total 100% to finalize</Badge>}
        <div className="ml-auto flex flex-wrap gap-2">
          {editing && newButton}
          {access.scores && assessments.length > 0 && (
            <Button variant="secondary" onClick={() => setImporting(true)}>
              Import scores
            </Button>
          )}
          <Button
            variant="secondary"
            disabled={exportCsv.isPending}
            onClick={() => exportCsv.mutate(`${sec.course.code}-${sec.number}-gradebook.csv`)}
          >
            {exportCsv.isPending ? 'Exporting…' : 'Export CSV'}
          </Button>
        </div>
      </div>
      <p className="mb-4 text-sm text-ink-muted">
        {finalized
          ? access.finalize
            ? 'Final grades are recorded. Changing a score now does not change them; reopen grades to recompute.'
            : 'Final grades are recorded. Score changes do not change them until an instructor reopens grades.'
          : 'Percent counts only assessments with a score. Missing scores count as 0 when you finalize.'}
      </p>
      <StatusMessage message={message} />
      <p role="status" className="sr-only">
        {announce}
      </p>
      {pageError && (
        <div className="mb-4">
          <ErrorBanner>{pageError}</ErrorBanner>
        </div>
      )}

      {students.length === 0 ? (
        <EmptyState
          title="No students yet"
          body="Students enrolled in this section appear here."
          action={
            <Link to="/$slug/sections/$id" params={{ slug, id: sectionId }} className="text-accent underline">
              Go to the section
            </Link>
          }
        />
      ) : assessments.length === 0 ? (
        <EmptyState
          title="No assessments yet"
          body={
            editing
              ? 'Add an assessment, such as a midterm worth 40%, to start entering scores.'
              : "The instructor hasn't added assessments yet."
          }
          action={editing ? newButton : undefined}
        />
      ) : (
        <Table caption={`Gradebook for ${code}`}>
          <thead>
            <tr>
              <th scope="col" className={`sticky left-0 z-10 bg-canvas ${capName}`}>
                Student
              </th>
              {assessments.map((a) => (
                <th key={a.id} scope="col" className="text-right">
                  {editing ? (
                    <Button variant="ghost" size="sm" aria-label={`Edit ${a.name}`} onClick={() => setDialog(a)}>
                      {a.name}
                    </Button>
                  ) : (
                    <span className="block">{a.name}</span>
                  )}
                  <span className="block font-normal normal-case">
                    {fmtNum(a.maxPoints)} pts · {fmtNum(a.weight)}%
                  </span>
                  {a.dueOn && <span className="block font-normal normal-case">Due {formatDate(a.dueOn)}</span>}
                  {editing ? (
                    <PublishToggle
                      slug={slug}
                      sectionId={sectionId}
                      a={a}
                      onDone={(m) => {
                        setPageError('')
                        setMessage(m)
                      }}
                      onError={setPageError}
                    />
                  ) : (
                    <Badge tone={a.published ? 'accent' : 'neutral'}>{a.published ? 'Published' : 'Draft'}</Badge>
                  )}
                </th>
              ))}
              <th scope="col" className="text-right">
                {finalized ? 'Running %' : 'Percent'}
              </th>
              <th scope="col">{finalized ? 'Running letter' : 'Letter'}</th>
              {finalized && <th scope="col">Final</th>}
            </tr>
          </thead>
          <tbody>
            {students.map((s, row) => (
              <tr key={s.userId}>
                <th
                  scope="row"
                  title={personName(s)}
                  className={`sticky left-0 z-10 bg-surface! font-bold ${capName}`}
                >
                  {personName(s)}
                  {s.displayName && <span className="block truncate text-xs font-normal text-ink-muted">{s.email}</span>}
                </th>
                {assessments.map((a, col) => {
                  const k = key(s, a)
                  const c = cells[k]
                  if (!access.scores)
                    return (
                      <td key={a.id} className="text-right">
                        {s.scores[a.id] === undefined ? (
                          a.published ? (
                            <span className="text-xs text-ink-muted">Missing</span>
                          ) : (
                            <>
                              —<span className="sr-only">No score</span>
                            </>
                          )
                        ) : (
                          fmtNum(s.scores[a.id])
                        )}
                      </td>
                    )
                  return (
                    <ScoreCell
                      key={a.id}
                      label={`Score for ${personName(s)}, ${a.name}`}
                      row={row}
                      col={col}
                      value={c ? c.draft : serverText(s, a)}
                      status={c?.status ?? 'idle'}
                      message={c?.message}
                      missing={a.published && !c && s.scores[a.id] === undefined}
                      onChange={(t) => setDraft(k, t)}
                      onCommit={() => commit(s, a)}
                      onRevert={() => cellsRef.current[k]?.status !== 'pending' && drop(k)}
                      onPasteLines={(lines) => paste(a, row, lines)}
                    />
                  )
                })}
                <td className="text-right">
                  {s.percent === undefined ? '—' : `${fmt2(s.percent)}%`}
                  {missingCount(s) > 0 && <span className="text-ink-muted"> ({missingCount(s)} missing)</span>}
                </td>
                <td>{s.letter ?? '—'}</td>
                {finalized && (
                  <td className="whitespace-nowrap">
                    <FinalCell s={s} />
                    {access.finalize && s.final && (
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={`Change final grade for ${personName(s)}`}
                        onClick={() => setChanging(s)}
                      >
                        Change
                      </Button>
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <th scope="row" className="sticky left-0 z-10 bg-surface">
                Mean
              </th>
              <td colSpan={assessments.length} className="text-ink-muted">
                {letters.length > 0 && `Letters: ${letters.map(([l, n]) => `${l} ${n}`).join(' · ')}`}
              </td>
              <td className="text-right">{stats.mean === undefined ? '—' : `${fmt2(stats.mean)}%`}</td>
              <td />
              {finalized && <td />}
            </tr>
            <tr>
              <th scope="row" className="sticky left-0 z-10 bg-surface">
                Median
              </th>
              <td colSpan={assessments.length}>
                {(membership.role === 'admin' || membership.role === 'dept_admin') && (
                  <Link to="/$slug/grade-scale" params={{ slug }} className="text-accent underline">
                    Grade scale
                  </Link>
                )}
              </td>
              <td className="text-right">{stats.median === undefined ? '—' : `${fmt2(stats.median)}%`}</td>
              <td />
              {finalized && <td />}
            </tr>
          </tfoot>
        </Table>
      )}
      {finalizing && scale && (
        <FinalizeDialog
          slug={slug}
          sectionId={sectionId}
          code={code}
          gradebook={gq.data}
          scale={scale}
          onClose={() => setFinalizing(false)}
          onDone={(m) => {
            setFinalizing(false)
            setPageError('')
            setMessage(m)
          }}
        />
      )}
      {changing && scale && (
        <OverrideDialog
          slug={slug}
          sectionId={sectionId}
          student={changing}
          scale={scale}
          onClose={() => setChanging(undefined)}
          onDone={(m) => {
            setChanging(undefined)
            setPageError('')
            setMessage(m)
          }}
        />
      )}
      <RemoveDialog
        open={reopening}
        title={`Reopen grades for ${code}?`}
        description={`This deletes the ${students.filter((s) => s.final).length} final grades. Students stop seeing their final letters until you finalize again. Scores and assessments are kept.`}
        confirmLabel="Reopen grades"
        pendingLabel="Reopening…"
        pending={reopen.isPending}
        error={reopen.error}
        copy={finalizeCopy}
        onConfirm={() => reopen.mutate()}
        onClose={() => {
          setReopening(false)
          reopen.reset()
        }}
      />
      {importing && (
        <ScoresImportDialog
          slug={slug}
          sectionId={sectionId}
          assessments={assessments}
          onClose={() => setImporting(false)}
          onDone={(m) => {
            setImporting(false)
            setPageError('')
            setMessage(m)
          }}
        />
      )}
      {dialog && (
        <AssessmentDialog
          slug={slug}
          sectionId={sectionId}
          assessment={dialog === 'new' ? undefined : dialog}
          scoredCount={dialog === 'new' ? 0 : scoredCount(dialog)}
          otherWeight={dialog === 'new' ? total : round2(total - dialog.weight)}
          onClose={() => setDialog(null)}
          onSaved={(m) => {
            setDialog(null)
            setPageError('')
            setMessage(m)
          }}
        />
      )}
    </>
  )
}
