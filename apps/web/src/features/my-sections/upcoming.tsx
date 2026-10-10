import { useQuery } from '@tanstack/react-query'
import { ClipboardCheck, FileText } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useId } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Badge } from '../../components/badge'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { formatDateTime } from '../../lib/format'
import { assignmentState, useNow, windowState } from '../quiz/window'

type Item = components['schemas']['UpcomingItem']

function useUpcoming(slug: string) {
  return useQuery({
    queryKey: [slug, 'upcoming'],
    queryFn: () => unwrap(api.GET('/universities/{slug}/me/upcoming', { params: { path: { slug } } })),
    staleTime: 60_000,
  })
}

const linkClass = 'font-bold text-accent hover:text-accent-hover hover:underline'

function Row({ slug, it, now }: { slug: string; it: Item; now: number }) {
  const staff = it.myRole !== 'student'
  const quiz = it.kind === 'quiz'
  const win = { opensAt: it.opensAt, closesAt: it.dueAt, lateUntil: it.lateUntil }
  const late = !staff && !quiz && assignmentState(win, now) === 'late'
  const waiting = windowState(win, now) === 'upcoming'
  const when = waiting
    ? `Opens ${formatDateTime(it.opensAt)}`
    : late && it.lateUntil
      ? `Late until ${formatDateTime(it.lateUntil)}`
      : `${quiz ? 'Closes' : 'Due'} ${formatDateTime(it.dueAt)}`
  const params = { slug, id: it.section.id, assessmentId: it.assessmentId }
  const Icon = quiz ? ClipboardCheck : FileText
  const name = staff ? (
    quiz ? (
      <Link to="/$slug/sections/$id/quizzes/$assessmentId/attempts" params={params} className={linkClass}>
        {it.name}
      </Link>
    ) : (
      <Link to="/$slug/sections/$id/assignments/$assessmentId/submissions" params={params} className={linkClass}>
        {it.name}
      </Link>
    )
  ) : quiz ? (
    <Link to="/$slug/sections/$id/quizzes/$assessmentId" params={params} className={linkClass}>
      {it.name}
    </Link>
  ) : (
    <Link to="/$slug/sections/$id/assignments/$assessmentId" params={params} className={linkClass}>
      {it.name}
    </Link>
  )
  return (
    <li className="flex items-start gap-3 rounded border border-line bg-surface p-3 text-sm">
      <Icon className="mt-0.5 shrink-0 text-ink-muted" aria-hidden="true" />
      <div className="min-w-0 flex-1">
        <div>{name}</div>
        <div className="text-ink-muted">
          {it.course.code}-{it.section.number} · {quiz ? 'Quiz' : 'Assignment'} · {when}
        </div>
      </div>
      {staff && it.submittedCount !== undefined && it.enrolledCount !== undefined && (
        <Badge tone="neutral" className="shrink-0">
          {it.submittedCount} of {it.enrolledCount} submitted
        </Badge>
      )}
      {!staff && it.started && <Badge tone="accent" className="shrink-0">In progress</Badge>}
      {late && <Badge tone="warning" className="shrink-0">Past due</Badge>}
    </li>
  )
}

export function Upcoming({ slug }: { slug: string }) {
  const id = useId()
  const q = useUpcoming(slug)
  const now = useNow(60_000)
  return (
    <section aria-labelledby={id} className="mb-8">
      <h2 id={id} className="mb-4 text-xl font-bold">
        Upcoming
      </h2>
      {q.isPending ? (
        <Loading />
      ) : q.error ? (
        <ErrorBanner title="Couldn't load what's due" onRetry={() => void q.refetch()}>
          {errorMessage(q.error)}
        </ErrorBanner>
      ) : q.data.items.length === 0 ? (
        <p className="text-sm text-ink-muted">Nothing due in the next 14 days.</p>
      ) : (
        <ul aria-label="Upcoming work" className="flex flex-col gap-2">
          {q.data.items.map((it) => (
            <Row key={`${it.assessmentId}`} slug={slug} it={it} now={now} />
          ))}
        </ul>
      )}
    </section>
  )
}
