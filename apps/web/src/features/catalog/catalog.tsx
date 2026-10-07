import { Link } from '@tanstack/react-router'
import { useEffect, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { LoadMore } from '../../components/load-more'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { Table } from '../../components/table'
import { classCode, clashesFor, formatMeeting } from '../../lib/schedule'
import { useCursorList } from '../../lib/use-cursor-list'
import { useAllDepartments } from '../admin/queries'
import { personName } from '../courses/access'
import { useMySections } from '../my-sections/my-sections'
import { TermPicker, useTermChoice } from '../my-sections/term-picker'

type Section = components['schemas']['Section']
type Params = { termId?: string; departmentId?: string; q?: string }

export function CatalogPage(props: { slug: string; onSearchChange: (p: Params) => void } & Params) {
  const { slug, termId, departmentId, q, onSearchChange } = props
  const { query: termsQuery, terms, term, notInSession } = useTermChoice(slug, termId)
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

  let body: React.ReactNode
  if (termsQuery.isPending) body = <Loading />
  else if (termsQuery.error)
    body = <ErrorBanner onRetry={() => void termsQuery.refetch()}>{errorMessage(termsQuery.error)}</ErrorBanner>
  else if (!term)
    body = <EmptyState title="No terms yet" body="The catalog appears here once your university sets up its terms." />
  else if (list.isPending) body = <Loading />
  else if (list.error) body = <ErrorBanner onRetry={list.refetch}>{errorMessage(list.error)}</ErrorBanner>
  else if (list.items.length === 0)
    body =
      q || departmentId ? (
        <EmptyState title="No sections match" body="Try another search or department." />
      ) : (
        <EmptyState
          title={`No sections in ${term.name}`}
          body="Sections appear here once your university schedules them."
        />
      )
  else
    body = (
      <>
        <Table caption={`Sections in ${term.name}`}>
          <thead>
            <tr>
              <th scope="col">Class</th>
              <th scope="col">Instructors</th>
              <th scope="col">Meets</th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((s) => {
              const names = s.staff.filter((x) => x.role === 'instructor').map(personName)
              const clash = clashesFor(s, mine)
              return (
                <tr key={s.id}>
                  <td>
                    <Link
                      to="/$slug/sections/$id"
                      params={{ slug, id: s.id }}
                      className="font-medium text-accent hover:underline"
                    >
                      {classCode(s)}
                    </Link>
                    <span className="block text-xs text-ink-muted">{s.course.title}</span>
                  </td>
                  <td>
                    {names.length ? (
                      names.join(', ')
                    ) : (
                      <>
                        <span className="text-ink-muted">—</span>
                        <span className="sr-only">none</span>
                      </>
                    )}
                  </td>
                  <td>
                    {s.meetings.length ? (
                      <ul>
                        {s.meetings.map((m) => (
                          <li key={m.id}>{formatMeeting(m)}</li>
                        ))}
                      </ul>
                    ) : (
                      <span className="text-ink-muted">No meeting times</span>
                    )}
                    {clash.length > 0 && (
                      <p className="mt-1 font-semibold text-danger">Clashes with {clash.map(classCode).join(', ')}</p>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </Table>
        <LoadMore
          hasNextPage={list.hasNextPage}
          isFetchingNextPage={list.isFetchingNextPage}
          onClick={list.fetchNextPage}
          shown={list.items.length}
        />
      </>
    )

  return (
    <>
      <PageHeader title="Catalog" description="Sections offered in a term." />
      {terms.length > 0 && (
        <>
          <div role="search" className="mb-4 flex flex-wrap items-end gap-4">
            <TermPicker terms={terms} value={term?.id} onChange={(id) => onSearchChange({ termId: id })} />
            <FormField label="Department" name="departmentId">
              <select
                className={`${inputClass} w-64`}
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
                className={`${inputClass} w-64`}
                value={text}
                onChange={(e) => setText(e.target.value)}
              />
            </FormField>
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
