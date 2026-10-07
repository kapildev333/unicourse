import { Link } from '@tanstack/react-router'
import { useId } from 'react'
import { Disclosure } from '../../components/disclosure'
import { roleLabel } from '../../components/badge'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { formatDate } from '../../lib/format'
import { classCode } from '../../lib/schedule'
import { useMySections } from '../my-sections/my-sections'
import { fmt1, fmt2, fmtNum } from './grade'
import { useGradeScale, useMyGrades, type MyGrades } from './queries'
import { ScaleTable } from './scale-editor'

type Term = MyGrades['terms'][number]
type GradeSection = Term['sections'][number]

function GradeBlock({ s }: { s: GradeSection }) {
  const { final, running } = s
  if (final?.status === 'graded')
    return (
      <>
        <p className="text-2xl font-bold">{final.letter}</p>
        <p className="text-sm text-ink-muted">
          {fmt2(final.points ?? 0)} grade points · {fmt2(final.percent ?? 0)}%
        </p>
      </>
    )
  if (final)
    return (
      <>
        <p className="text-base font-bold">{final.status === 'incomplete' ? 'Incomplete' : 'Withdrawn'}</p>
        <p className="text-sm text-ink-muted">Not counted in GPA</p>
      </>
    )
  if (running)
    return (
      <>
        <p className="text-base font-bold">Current {running.letter}</p>
        <p className="text-sm text-ink-muted">{fmt2(running.percent)}% so far</p>
        <p className="text-sm text-ink-muted">Counts published scores only.</p>
      </>
    )
  return <p className="text-sm text-ink-muted">No grade yet</p>
}

function SectionCard({ slug, s }: { slug: string; s: GradeSection }) {
  return (
    <li className="rounded border border-line bg-surface p-4">
      <h3 className="text-base font-bold">
        <Link to="/$slug/sections/$id" params={{ slug, id: s.section.id }} className="text-accent hover:underline">
          {classCode({ course: s.course, number: s.section.number })}
        </Link>
        <span className="block font-bold text-ink">{s.course.title}</span>
      </h3>
      <p className="text-sm text-ink-muted">{fmt1(s.credits)} credits</p>
      <div className="mt-3 min-h-14">
        <GradeBlock s={s} />
      </div>
      {s.assessments.length === 0 ? (
        <p className="mt-3 text-sm text-ink-muted">No published scores yet.</p>
      ) : (
        <Disclosure className="mt-3 text-sm" summary={`Published scores (${s.assessments.length})`}>
          <ul className="mt-2 flex flex-col gap-1 text-sm">
            {s.assessments.map((a) => (
              <li key={a.id}>
                {a.points === undefined
                  ? `${a.name} · Not graded yet · ${fmtNum(a.weight)}%`
                  : `${a.name} · ${fmtNum(a.points)} / ${fmtNum(a.maxPoints)} · ${fmtNum(a.weight)}%`}
                {a.dueOn && ` · Due ${formatDate(a.dueOn)}`}
              </li>
            ))}
          </ul>
        </Disclosure>
      )}
    </li>
  )
}

function TermBlock({ slug, t }: { slug: string; t: Term }) {
  const id = useId()
  const sections = [...t.sections].sort((a, b) => a.course.code.localeCompare(b.course.code))
  return (
    <section aria-labelledby={id} className="mb-8">
      <div className="mb-3 flex flex-wrap items-baseline gap-x-4">
        <h2 id={id} className="text-xl font-bold">
          {t.term.name}
        </h2>
        <span className="text-sm text-ink-muted">
          {t.gpa === undefined ? 'No term GPA yet' : `Term GPA ${fmt2(t.gpa)} · ${fmt1(t.credits)} credits`}
        </span>
      </div>
      <ul aria-label={`Sections in ${t.term.name}`} className="grid items-start gap-6 md:grid-cols-2 lg:grid-cols-3">
        {sections.map((s) => (
          <SectionCard key={s.section.id} slug={slug} s={s} />
        ))}
      </ul>
    </section>
  )
}

function Gradebooks({ slug }: { slug: string }) {
  const id = useId()
  const mine = useMySections(slug).data
  const staffed = (mine?.terms ?? []).flatMap((g) =>
    g.sections.filter((s) => s.myRole === 'instructor' || s.myRole === 'ta').map((s) => ({ s, term: g.term })),
  )
  if (staffed.length === 0) return null
  return (
    <section aria-labelledby={id}>
      <h2 id={id} className="mb-3 text-xl font-bold">
        Gradebooks
      </h2>
      <ul className="mb-8 flex flex-col gap-1 text-sm">
        {staffed.map(({ s, term }) => (
          <li key={s.id}>
            <Link to="/$slug/sections/$id/grades" params={{ slug, id: s.id }} className="text-accent underline">
              {classCode(s)}
            </Link>{' '}
            · {term.name} · {roleLabel(s.myRole)}
          </li>
        ))}
      </ul>
    </section>
  )
}

function ScaleDisclosure({ slug }: { slug: string }) {
  const q = useGradeScale(slug)
  return (
    <Disclosure className="mt-8 text-sm" summary="How letters and grade points are assigned">
      <div className="mt-3">
        {q.isPending ? (
          <Loading />
        ) : q.error ? (
          <ErrorBanner title="Couldn't load the grade scale" onRetry={() => void q.refetch()}>
            {errorMessage(q.error)}
          </ErrorBanner>
        ) : (
          <ScaleTable rows={q.data.rows} caption="Grade scale" />
        )}
      </div>
    </Disclosure>
  )
}

export function MyGradesPage({ slug }: { slug: string }) {
  const q = useMyGrades(slug)
  const header = <PageHeader title="My grades" description="Final grades, published scores and GPA for every term." />
  let body
  if (q.isPending) body = <Loading variant="cards" label="Loading your grades…" />
  else if (q.error) body = <ErrorBanner title="Couldn't load your grades" onRetry={() => void q.refetch()}>{errorMessage(q.error)}</ErrorBanner>
  else {
    const g = q.data
    const terms = [...g.terms].sort((a, b) => b.term.startsOn.localeCompare(a.term.startsOn))
    const inProgress = terms.flatMap((t) => t.sections).filter((s) => !s.final).reduce((n, s) => n + s.credits, 0)
    body =
      terms.length === 0 ? (
        <EmptyState
          title="No grades yet"
          body="Grades appear here once you're enrolled in a section and your instructors publish scores."
          action={
            <Link to="/$slug/catalog" params={{ slug }} className="text-accent underline">
              Browse the catalog
            </Link>
          }
        />
      ) : (
        <>
          <div className="mb-8 w-fit max-w-full rounded border border-line bg-surface p-4">
            <dl className="flex flex-wrap gap-x-12 gap-y-2">
              <div>
                <dt className="text-sm text-ink-muted">Cumulative GPA</dt>
                <dd className="text-3xl font-bold">{g.cumulativeGpa === undefined ? '—' : fmt2(g.cumulativeGpa)}</dd>
              </div>
              <div>
                <dt className="text-sm text-ink-muted">Graded credits</dt>
                <dd className="text-3xl font-bold">{fmt1(g.cumulativeCredits)}</dd>
              </div>
            </dl>
            <p className="mt-2 text-sm text-ink-muted">
              Counts finalized sections only.
              {inProgress > 0 && ` ${fmt1(inProgress)} credits in progress.`}
            </p>
            {g.cumulativeGpa === undefined && (
              <p className="mt-1 text-sm text-ink-muted">Your GPA appears once a section has a final grade.</p>
            )}
          </div>
          {terms.map((t) => (
            <TermBlock key={t.term.id} slug={slug} t={t} />
          ))}
        </>
      )
  }
  return (
    <>
      {header}
      <Gradebooks slug={slug} />
      {body}
      <ScaleDisclosure slug={slug} />
    </>
  )
}
