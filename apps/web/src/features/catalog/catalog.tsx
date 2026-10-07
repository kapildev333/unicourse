import { CalendarRange, Info, Library, SearchX } from 'lucide-react'
import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Badge, roleLabel } from '../../components/badge'
import { Button } from '../../components/button'
import { Card } from '../../components/card'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { LoadMore } from '../../components/load-more'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { classCode, clashesFor, formatMeetings } from '../../lib/schedule'
import { useCursorList } from '../../lib/use-cursor-list'
import { useMe } from '../auth/use-me'
import { useAllDepartments } from '../admin/queries'
import { personName } from '../courses/access'
import { useMySections } from '../my-sections/my-sections'
import { TermPicker, useTermChoice } from '../my-sections/term-picker'

type Section = components['schemas']['Section']
type Params = { termId?: string; departmentId?: string; q?: string }

export function CatalogPage(props: { slug: string; onSearchChange: (p: Params) => void } & Params) {
  const { slug, termId, departmentId, q, onSearchChange } = props
  const { query: termsQuery, terms, term, notInSession } = useTermChoice(slug, termId)
  const isAdmin = useMe().data?.memberships.find((m) => m.university.slug === slug)?.role === 'admin'
  const departments = useAllDepartments(slug).data?.items ?? []
  const [text, setText] = useState(q ?? '')
  // q changed from outside (nav click, back button): follow it, so the debounce below finds nothing to write
  const [seenQ, setSeenQ] = useState(q)
  if (seenQ !== q) {
    setSeenQ(q)
    if ((q ?? '') !== text.trim()) setText(q ?? '')
  }
  useEffect(() => {
    const t = setTimeout(() => {
      const v = text.trim()
      if (v !== (q ?? '')) onSearchChange({ q: v || undefined })
    }, 300)
    return () => clearTimeout(t)
  }, [text, q, onSearchChange])

  const list = useCursorList<Section>(
    [slug, 'sections', 'catalog', { termId: term?.id, departmentId, q }],
    (cursor) =>
      unwrap(
        api.GET('/universities/{slug}/sections', {
          params: { path: { slug }, query: { cursor, limit: 50, termId: term!.id, departmentId, q } },
        }),
      ),
    !!term,
  )
  const mine = useMySections(slug).data?.terms.find((g) => g.term.id === term?.id)?.sections ?? []
  const clearFilters = () => {
    setText('')
    onSearchChange({ departmentId: undefined, q: undefined })
  }
  const filtered = !!(q || departmentId)

  let body: React.ReactNode
  if (termsQuery.isPending) body = <Loading variant="cards" />
  else if (termsQuery.error)
    body = (
      <ErrorBanner title="Couldn't load sections" onRetry={() => void termsQuery.refetch()}>
        {errorMessage(termsQuery.error)}
      </ErrorBanner>
    )
  else if (!term)
    body = (
      <EmptyState
        icon={CalendarRange}
        title="No terms yet"
        body="The catalog appears here once your university sets up its terms."
        action={
          isAdmin ? (
            <Link to="/$slug/admin/terms" params={{ slug }} className="text-accent underline hover:text-accent-hover">
              Set up terms
            </Link>
          ) : undefined
        }
      />
    )
  else if (list.isPending) body = <Loading variant="cards" />
  else if (list.error)
    body = (
      <ErrorBanner title="Couldn't load sections" onRetry={list.refetch}>
        {errorMessage(list.error)}
      </ErrorBanner>
    )
  else if (list.items.length === 0)
    body = filtered ? (
      <EmptyState
        icon={SearchX}
        title={`No sections match "${q ?? departments.find((d) => d.id === departmentId)?.name ?? ''}"`}
        body="Try another search or department."
        action={
          <Button variant="secondary" onClick={clearFilters}>
            Clear filters
          </Button>
        }
      />
    ) : (
      <EmptyState
        icon={Library}
        title={`No sections in ${term.name}`}
        body="Sections appear here once your university schedules them."
      />
    )
  else
    body = (
      <>
        <p role="status" className="mb-4 text-sm text-ink-muted">
          Showing {list.items.length} {list.items.length === 1 ? 'section' : 'sections'}
        </p>
        <ul aria-label={`Sections in ${term.name}`} className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {list.items.map((s) => {
            const names = s.staff.filter((x) => x.role === 'instructor').map(personName)
            const clash = clashesFor(s, mine)
            const myRole = mine.find((x) => x.id === s.id)?.myRole
            return (
              <Card
                key={s.id}
                code={classCode(s)}
                title={s.course.title}
                meta={[
                  ...(names.length ? [names.join(', ')] : []),
                  ...(s.meetings.length ? formatMeetings(s.meetings) : ['No meeting times']),
                ]}
                note={clash.length ? `Clashes with ${clash.map(classCode).join(', ')}` : undefined}
                badge={
                  myRole && (
                    <Badge tone={myRole === 'student' ? 'neutral' : 'accent'}>
                      {myRole === 'student' ? 'Enrolled' : roleLabel(myRole)}
                    </Badge>
                  )
                }
                params={{ slug, id: s.id }}
              />
            )
          })}
        </ul>
        <LoadMore
          hasNextPage={list.hasNextPage}
          isFetchingNextPage={list.isFetchingNextPage}
          onClick={list.fetchNextPage}
          shown={list.items.length}
          showCount={false}
        />
      </>
    )

  return (
    <>
      <PageHeader title="Catalog" description="Sections offered in a term." />
      {terms.length > 0 && (
        <>
          <div role="search" aria-label="Filters" className="mb-6 grid gap-4 md:flex md:flex-wrap md:items-end">
            <TermPicker terms={terms} value={term?.id} onChange={(id) => onSearchChange({ termId: id })} />
            <FormField label="Department" name="departmentId">
              <select
                className={`${inputClass} w-full md:w-64`}
                value={departmentId ?? ''}
                onChange={(e) => onSearchChange({ departmentId: e.target.value || undefined })}
              >
                <option value="">All departments</option>
                {departments.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.code} — {d.name}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="Search" name="q">
              <input
                type="search"
                maxLength={100}
                placeholder="Course code or title"
                className={`${inputClass} w-full md:w-64`}
                value={text}
                onChange={(e) => setText(e.target.value)}
              />
            </FormField>
            {filtered && (
              <Button variant="ghost" size="sm" className="md:self-end" onClick={clearFilters}>
                Clear filters
              </Button>
            )}
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
