import { useMutation } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { CalendarRange, Layers, Plus, Upload } from 'lucide-react'
import { useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { LoadMore } from '../../components/load-more'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { formatDate } from '../../lib/format'
import { useCursorList } from '../../lib/use-cursor-list'
import { useRefresh } from '../admin/queries'
import { useUniversity } from '../auth/use-me'
import { canManageDepartment, personName } from './access'
import { CsvImportDialog, importSummary } from './csv-import-dialog'
import { defaultTermId, todayIso, useAllTerms } from './queries'
import { RemoveDialog } from './remove-dialog'
import { SectionDialog } from './section-dialog'

type Section = components['schemas']['Section']

const listCopy = { forbidden: "You don't have permission to view this." }

function Chips({ section, staffRole }: { section: Section; staffRole: 'instructor' | 'ta' }) {
  const staff = section.staff.filter((s) => s.role === staffRole)
  const pending = section.pendingStaff.filter((s) => s.role === staffRole)
  if (staff.length + pending.length === 0)
    return (
      <>
        <span className="text-ink-muted">—</span>
        <span className="sr-only">none</span>
      </>
    )
  return (
    <ul className="flex flex-wrap gap-1">
      {staff.map((s) => (
        <li key={s.userId}>
          <Badge tone="neutral">{personName(s)}</Badge>
        </li>
      ))}
      {pending.map((s) => (
        <li key={s.email}>
          <Badge tone="neutral" className="border-dashed">
            {s.email} (invited)
          </Badge>
        </li>
      ))}
    </ul>
  )
}

export function SectionListPage(props: { slug: string; termId?: string; onTermChange: (id: string) => void }) {
  const { slug, termId, onTermChange } = props
  const { membership } = useUniversity()
  const termsQuery = useAllTerms(slug)
  const terms = [...(termsQuery.data?.items ?? [])].sort((a, b) => b.startsOn.localeCompare(a.startsOn))
  const effectiveId = terms.some((t) => t.id === termId) ? termId : defaultTermId(terms, todayIso())
  const term = terms.find((t) => t.id === effectiveId)
  const refresh = useRefresh(slug, 'sections')
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<Section | null>(null)
  const [deleting, setDeleting] = useState<Section | null>(null)
  const [importing, setImporting] = useState(false)
  const [notice, setNotice] = useState('')

  const list = useCursorList<Section>([slug, 'sections', 'list', { termId: effectiveId }], (cursor) =>
    unwrap(
      api.GET('/universities/{slug}/sections', {
        params: { path: { slug }, query: { cursor, limit: 50, termId: effectiveId } },
      }),
    ),
    !!term,
  )
  const del = useMutation({
    mutationFn: (s: Section) =>
      unwrap(api.DELETE('/universities/{slug}/sections/{id}', { params: { path: { slug, id: s.id } } })),
    onSuccess: async (_, s) => {
      await refresh([slug, 'my-sections'])
      setDeleting(null)
      setNotice(`Section ${s.course.code} ${s.number} deleted.`)
    },
  })

  const actions = (
    <div className="flex gap-2">
      <Button variant="secondary" onClick={() => setImporting(true)}>
        <Upload />
        Import sections
      </Button>
      <Button disabled={!term} onClick={() => setCreating(true)}>
        <Plus />
        New section
      </Button>
    </div>
  )

  let body: React.ReactNode
  if (termsQuery.isPending) body = <Loading />
  else if (termsQuery.error)
    body = <ErrorBanner onRetry={() => void termsQuery.refetch()}>{errorMessage(termsQuery.error, listCopy)}</ErrorBanner>
  else if (!term)
    body = (
      <EmptyState
        title="No terms yet"
        icon={CalendarRange}
        body="An administrator creates terms under Admin → Terms before sections can be added."
      />
    )
  else if (list.isPending) body = <Loading />
  else if (list.error) body = <ErrorBanner onRetry={list.refetch}>{errorMessage(list.error, listCopy)}</ErrorBanner>
  else if (list.items.length === 0)
    body = (
      <EmptyState
        title={`No sections in ${term.name}`}
        icon={Layers}
        body="Create sections one at a time or import them from a CSV file."
        action={(
          <Button onClick={() => setCreating(true)}>
            <Plus />
            New section
          </Button>
        )}
      />
    )
  else
    body = (
      <>
        <Table caption={`Sections in ${term.name}`}>
          <thead>
            <tr>
              <th scope="col">Course</th>
              <th scope="col">Section</th>
              <th scope="col">Instructors</th>
              <th scope="col">TAs</th>
              <th scope="col">
                <span className="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((s) => (
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
                  <Chips section={s} staffRole="instructor" />
                </td>
                <td>
                  <Chips section={s} staffRole="ta" />
                </td>
                <td className="text-right whitespace-nowrap">
                  {canManageDepartment(membership, s.course.departmentId) && (
                    <>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={`Edit ${s.course.code} section ${s.number}`}
                        onClick={() => setEditing(s)}
                      >
                        Edit
                      </Button>
                      <Button
                        variant="ghost-danger"
                        size="sm"
                        aria-label={`Delete ${s.course.code} section ${s.number}`}
                        onClick={() => {
                          del.reset()
                          setDeleting(s)
                        }}
                      >
                        Delete
                      </Button>
                    </>
                  )}
                </td>
              </tr>
            ))}
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
      <PageHeader title="Sections" description="Sections of each course in a term." action={actions} />
      <div role="search" className="mb-6">
        <FormField label="Term" name="termId">
          <select className={`${inputClass} w-64`} value={effectiveId ?? ''} onChange={(e) => onTermChange(e.target.value)}>
            {terms.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name} ({formatDate(t.startsOn)} – {formatDate(t.endsOn)})
              </option>
            ))}
          </select>
        </FormField>
      </div>
      <StatusMessage message={notice} />
      {body}
      {creating && term && (
        <SectionDialog
          slug={slug}
          term={term}
          onClose={() => setCreating(false)}
          onSaved={(code, number) => {
            setNotice(`Section ${code} ${number} created.`)
            setCreating(false)
          }}
        />
      )}
      {editing && term && (
        <SectionDialog
          slug={slug}
          term={term}
          section={editing}
          onClose={() => setEditing(null)}
          onSaved={(code, number) => {
            setNotice(`Section ${code} ${number} updated.`)
            setEditing(null)
          }}
        />
      )}
      <RemoveDialog
        open={deleting !== null}
        title={`Delete ${deleting?.course.code ?? ''} section ${deleting?.number ?? ''}?`}
        description="Its staff assignments and roster are removed too. This cannot be undone."
        confirmLabel="Delete"
        pendingLabel="Deleting…"
        pending={del.isPending}
        error={del.error}
        onConfirm={() => deleting && del.mutate(deleting)}
        onClose={() => setDeleting(null)}
      />
      <CsvImportDialog
        open={importing}
        onClose={() => setImporting(false)}
        title="Import sections"
        target={{ kind: 'sections', slug }}
        columns={['courseCode', 'termName', 'number', 'instructorEmails', 'taEmails', 'meetings']}
        hint="Separate several emails with semicolons. meetings lists weekly times separated by semicolons, for example Mon 10:00-11:15 @ Room 101; Wed 10:00-11:15 (days Mon to Sun, 24-hour times, location after @ optional). A section that already has meeting times keeps them."
        onDone={async (r) => {
          await refresh([slug, 'my-sections'], [slug, 'invitations'])
          setImporting(false)
          setNotice(importSummary('sections', r.counts))
        }}
      />
    </>
  )
}
