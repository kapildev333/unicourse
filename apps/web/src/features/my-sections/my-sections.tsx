import { useQuery } from '@tanstack/react-query'
import { CalendarDays, CalendarRange, Info, List } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useId } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { RoleBadge } from '../../components/badge'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { Button } from '../../components/button'
import { Card } from '../../components/card'
import { formatDate } from '../../lib/format'
import { classCode, formatMeeting } from '../../lib/schedule'
import { useUniversity } from '../auth/use-me'
import { canManageCourses, personName } from '../courses/access'
import { Schedule } from './schedule'
import { TermPicker, useTermChoice } from './term-picker'

type Term = components['schemas']['Term']
type MySection = components['schemas']['MySection']

function TermCards({ slug, term, sections }: { slug: string; term: Term; sections: MySection[] }) {
  const id = useId()
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
          const names = s.staff.filter((x) => x.role === 'instructor').map(personName)
          return (
            <Card
              key={s.id}
              code={classCode(s)}
              title={s.course.title}
              meta={[
                ...(names.length ? [names.join(', ')] : []),
                ...(s.meetings.length ? s.meetings.map((m) => formatMeeting(m)) : ['No meeting times']),
              ]}
              badge={<RoleBadge role={s.myRole} />}
              params={{ slug, id: s.id }}
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
  const { query: termsQuery, terms, term, notInSession } = useTermChoice(slug, termId)
  const q = useMySections(slug)
  let body: React.ReactNode
  if (termsQuery.isPending) body = <Loading />
  else if (termsQuery.error)
    body = <ErrorBanner onRetry={() => void termsQuery.refetch()}>{errorMessage(termsQuery.error)}</ErrorBanner>
  else if (!term)
    body = (
      <EmptyState icon={CalendarRange} title="No terms yet" body="Your schedule appears here once your university sets up its terms." />
    )
  else if (q.isPending) body = <Loading />
  else if (q.error) body = <ErrorBanner onRetry={() => void q.refetch()}>{errorMessage(q.error)}</ErrorBanner>
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
              <Link to="/$slug/catalog" params={{ slug }} search={{ termId: term.id }} className="text-accent underline hover:text-accent-hover">
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
      <PageHeader title="My sections" description="Sections you teach, assist or take." />
      {terms.length > 0 && (
        <>
          <div role="search" aria-label="Filters" className="mb-6 flex flex-wrap items-end gap-4">
            <TermPicker terms={terms} value={term?.id} onChange={(id) => onSearchChange({ termId: id })} />
            <div role="group" aria-label="View" className="flex gap-2">
              <Button variant={view ? 'secondary' : 'primary'} aria-pressed={!view} onClick={() => onSearchChange({ view: undefined })}>
                <List size={16} />
                List
              </Button>
              <Button variant={view ? 'primary' : 'secondary'} aria-pressed={!!view} onClick={() => onSearchChange({ view: 'schedule' })}>
                <CalendarDays size={16} />
                Schedule
              </Button>
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
