import { useQuery } from '@tanstack/react-query'
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
import { Table } from '../../components/table'
import { formatDate } from '../../lib/format'
import { useUniversity } from '../auth/use-me'
import { canManageCourses, personName } from '../courses/access'
import { Schedule } from './schedule'
import { TermPicker, useTermChoice } from './term-picker'

type Term = components['schemas']['Term']
type MySection = components['schemas']['MySection']

function TermTable({ slug, term, sections }: { slug: string; term: Term; sections: MySection[] }) {
  const id = useId()
  return (
    <section aria-labelledby={id} className="mb-8">
      <h2 id={id} className="mb-2 text-base font-semibold">
        {term.name}
        <span className="ml-2 text-sm font-normal text-ink-muted">
          {formatDate(term.startsOn)} – {formatDate(term.endsOn)}
        </span>
      </h2>
      <Table caption={`My sections in ${term.name}`}>
        <thead>
          <tr>
            <th scope="col">Course</th>
            <th scope="col">Section</th>
            <th scope="col">My role</th>
            <th scope="col">Instructors</th>
          </tr>
        </thead>
        <tbody>
          {sections.map((s) => {
            const names = s.staff.filter((x) => x.role === 'instructor').map(personName)
            return (
              <tr key={s.id}>
                <td>
                  <Link
                    to="/$slug/sections/$id"
                    params={{ slug, id: s.id }}
                    className="font-medium text-accent hover:underline"
                  >
                    {s.course.code}
                  </Link>
                  <span className="block text-xs text-ink-muted">{s.course.title}</span>
                </td>
                <td>{s.number}</td>
                <td>
                  <RoleBadge role={s.myRole} />
                </td>
                <td>{names.length ? names.join(', ') : '—'}</td>
              </tr>
            )
          })}
        </tbody>
      </Table>
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
      <EmptyState title="No terms yet" body="Your schedule appears here once your university sets up its terms." />
    )
  else if (q.isPending) body = <Loading />
  else if (q.error) body = <ErrorBanner onRetry={() => void q.refetch()}>{errorMessage(q.error)}</ErrorBanner>
  else {
    const sections = q.data.terms.find((g) => g.term.id === term.id)?.sections ?? []
    if (sections.length === 0)
      body = (
        <EmptyState
          title={`No sections in ${term.name}`}
          body="Sections you teach, assist or take in this term appear here."
          action={
            <div className="flex justify-center gap-4">
              {/* ponytail: cast until the catalog route exists (P5-L2b); drop the cast then */}
              <Link {...({ to: '/$slug/catalog', params: { slug }, search: { termId: term.id } } as unknown as { to: '/' })} className="text-accent underline">
                Browse the catalog
              </Link>
              {canManageCourses(membership) && (
                <Link to="/$slug/courses" params={{ slug }} className="text-accent underline">
                  Go to Courses
                </Link>
              )}
            </div>
          }
        />
      )
    else if (view === 'schedule') body = <Schedule slug={slug} term={term} sections={sections} />
    else body = <TermTable slug={slug} term={term} sections={sections} />
  }
  return (
    <>
      <PageHeader title="My sections" description="Sections you teach, assist or take." />
      {terms.length > 0 && (
        <>
          <div role="search" className="mb-4 flex items-end gap-4">
            <TermPicker terms={terms} value={term?.id} onChange={(id) => onSearchChange({ termId: id })} />
            <div role="group" aria-label="View" className="flex">
              <Button size="sm" variant={view ? 'secondary' : 'primary'} aria-pressed={!view} onClick={() => onSearchChange({ view: undefined })}>
                List
              </Button>
              <Button size="sm" variant={view ? 'primary' : 'secondary'} aria-pressed={!!view} onClick={() => onSearchChange({ view: 'schedule' })}>
                Schedule
              </Button>
            </div>
          </div>
          {notInSession && term && (
            <p className="mb-4 text-sm text-ink-muted">No term is in session today. Showing {term.name}.</p>
          )}
        </>
      )}
      {body}
    </>
  )
}
