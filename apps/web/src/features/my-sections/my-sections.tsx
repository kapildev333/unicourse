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
import { Table } from '../../components/table'
import { formatDate } from '../../lib/format'
import { personName } from '../courses/access'

type TermGroup = components['schemas']['MySections']['terms'][number]

function TermTable({ slug, group }: { slug: string; group: TermGroup }) {
  const id = useId()
  const { term, sections } = group
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

export function MySectionsPage({ slug }: { slug: string }) {
  const q = useQuery({
    queryKey: [slug, 'my-sections'],
    queryFn: () => unwrap(api.GET('/universities/{slug}/me/sections', { params: { path: { slug } } })),
  })
  const groups = q.data?.terms.filter((g) => g.sections.length > 0) ?? []
  let body: React.ReactNode
  if (q.isPending) body = <Loading />
  else if (q.error) body = <ErrorBanner onRetry={() => void q.refetch()}>{errorMessage(q.error)}</ErrorBanner>
  else if (groups.length === 0)
    body = <EmptyState title="No sections yet" body="Sections you teach, assist or take will appear here." />
  else body = groups.map((g) => <TermTable key={g.term.id} slug={slug} group={g} />)
  return (
    <>
      <PageHeader title="My sections" description="Sections you teach, assist or take." />
      {body}
    </>
  )
}
