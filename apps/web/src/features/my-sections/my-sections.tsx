import { useQuery } from '@tanstack/react-query'
import { CalendarDays, CalendarPlus, CalendarRange, Info, Table as TableIcon } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useId, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { RoleBadge } from '../../components/badge'
import { ButtonLink } from '../../components/button-link'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { Button } from '../../components/button'
import { PageHeader } from '../../components/page-header'
import { SegmentedControl } from '../../components/segmented-control'
import { Card } from '../../components/card'
import { formatDate } from '../../lib/format'
import { classCode, formatMeetings } from '../../lib/schedule'
import { useMe, useUniversity } from '../auth/use-me'
import { canManageCourses } from '../courses/access'
import { Schedule, instructors } from './schedule'
import { CalendarFeedDialog } from './calendar-feed-dialog'
import { TermPicker, useTermChoice } from './term-picker'
import { Upcoming } from './upcoming'

type Term = components['schemas']['Term']
type MySection = components['schemas']['MySection']

function TermCards({ slug, term, sections }: { slug: string; term: Term; sections: MySection[] }) {
  const id = useId()
  const meId = useMe().data?.user.id
  return (
    <section aria-labelledby={id} className="mb-8">
      <h2 id={id} className="mb-4 text-xl font-bold">
        {term.name}
        <span className="ml-2 text-sm font-normal text-ink-muted">
          {formatDate(term.startsOn)} – {formatDate(term.endsOn)}
        </span>
      </h2>
      <ul aria-label={`My sections in ${term.name}`} className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {sections.map((s) => {
          const names = instructors(s, meId)
          return (
            <Card
              key={s.id}
              code={classCode(s)}
              title={s.course.title}
              meta={[
                ...(names.length ? [names.join(', ')] : []),
                ...(s.meetings.length ? formatMeetings(s.meetings) : ['No meeting times']),
              ]}
              badge={<RoleBadge role={s.myRole} />}
              params={{ slug, id: s.id }}
              footer={
                s.myRole === 'instructor' || s.myRole === 'ta' ? (
                  <Link
                    to="/$slug/sections/$id/grades"
                    params={{ slug, id: s.id }}
                    className="inline-flex items-center gap-2 text-accent hover:text-accent-hover hover:underline"
                  >
                    <TableIcon aria-hidden="true" />
                    Gradebook
                  </Link>
                ) : undefined
              }
            />
          )
        })}
      </ul>
    </section>
  )
}

export function useMySections(slug: string) {
  return useQuery({
    queryKey: [slug, 'my-sections'],
    queryFn: () => unwrap(api.GET('/universities/{slug}/me/sections', { params: { path: { slug } } })),
  })
}

export function MySectionsPage(props: {
  slug: string
  termId?: string
  view?: 'schedule'
  onSearchChange: (p: { termId?: string; view?: 'schedule' }) => void
}) {
  const { slug, termId, view, onSearchChange } = props
  const { membership } = useUniversity()
  const { query: termsQuery, terms, term, picked, pick, notInSession } = useTermChoice(slug, termId)
  const q = useMySections(slug)
  const [feedOpen, setFeedOpen] = useState(false)
  let body: React.ReactNode
  if (termsQuery.isPending) body = <Loading variant="cards" />
  else if (termsQuery.error)
    body = (
      <ErrorBanner title="Couldn't load your sections" onRetry={() => void termsQuery.refetch()}>
        {errorMessage(termsQuery.error)}
      </ErrorBanner>
    )
  else if (!term)
    body = (
      <EmptyState
        icon={CalendarRange}
        title="No terms yet"
        body="Your schedule appears here once your university sets up its terms."
        action={
          membership.role === 'admin' ? (
            <ButtonLink variant="secondary" to="/$slug/admin/terms" params={{ slug }}>
              Set up terms
            </ButtonLink>
          ) : undefined
        }
      />
    )
  else if (q.isPending) body = <Loading variant="cards" />
  else if (q.error)
    body = (
      <ErrorBanner title="Couldn't load your sections" onRetry={() => void q.refetch()}>
        {errorMessage(q.error)}
      </ErrorBanner>
    )
  else {
    const sections = q.data.terms.find((g) => g.term.id === term.id)?.sections ?? []
    if (sections.length === 0)
      body = (
        <EmptyState
          icon={CalendarDays}
          title={`No sections in ${term.name}`}
          body="Sections you teach, assist or take in this term appear here."
          action={
            <div className="flex justify-center gap-4">
              <Link
                to="/$slug/catalog"
                params={{ slug }}
                search={{ termId: term.id }}
                className="text-accent underline hover:text-accent-hover"
              >
                Browse the catalog
              </Link>
              {canManageCourses(membership) && (
                <Link to="/$slug/courses" params={{ slug }} className="text-accent underline hover:text-accent-hover">
                  Go to Courses
                </Link>
              )}
            </div>
          }
        />
      )
    else if (view === 'schedule') body = <Schedule slug={slug} term={term} sections={sections} />
    else body = <TermCards slug={slug} term={term} sections={sections} />
  }
  return (
    <>
      <PageHeader
        title="My sections"
        description="Sections you teach, assist or take."
        action={
          <Button variant="secondary" onClick={() => setFeedOpen(true)}>
            <CalendarPlus aria-hidden="true" />
            Calendar feed
          </Button>
        }
      />
      {feedOpen && <CalendarFeedDialog slug={slug} onClose={() => setFeedOpen(false)} />}
      <Upcoming slug={slug} />
      {terms.length > 0 && (
        <>
          <div role="search" aria-label="Filters" className="mb-6 grid gap-4 md:flex md:flex-wrap md:items-end">
            <TermPicker slug={slug} terms={terms} more={!!termsQuery.data?.nextCursor} picked={picked} value={term?.id} onChange={(id, t) => { pick(t); onSearchChange({ termId: id }) }} />
            <div>
              <SegmentedControl
                label="View"
                options={[
                  { value: 'list', label: 'List' },
                  { value: 'schedule', label: 'Schedule' },
                ]}
                value={view ?? 'list'}
                onChange={(v) => onSearchChange({ view: v === 'schedule' ? 'schedule' : undefined })}
              />
            </div>
          </div>
          {notInSession && term && (
            <p className="mb-6 flex items-center gap-2 text-sm text-ink-muted">
              <Info size={16} />
              No term is in session today. Showing {term.name}.
            </p>
          )}
        </>
      )}
      {body}
    </>
  )
}
