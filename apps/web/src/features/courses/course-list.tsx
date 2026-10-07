import { useMutation } from '@tanstack/react-query'
import { BookOpen, Plus, Upload } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Button } from '../../components/button'
import { ListCount } from '../../components/list-count'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { LoadMore } from '../../components/load-more'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { useCursorList } from '../../lib/use-cursor-list'
import { useAllDepartments, useRefresh } from '../admin/queries'
import { useUniversity } from '../auth/use-me'
import { canManageDepartment } from './access'
import { CourseDialog } from './course-dialog'
import { CsvImportDialog, importSummary } from './csv-import-dialog'
import { RemoveDialog } from './remove-dialog'

type Course = components['schemas']['Course']

const listCopy = { forbidden: "You don't have permission to view this." }
const deleteCopy = { conflict: 'This course still has sections. Delete its sections first.' }

export function CourseListPage(props: {
  slug: string
  departmentId?: string
  onDepartmentChange: (id: string) => void
}) {
  const { slug, departmentId, onDepartmentChange } = props
  const { membership } = useUniversity()
  const departments = useAllDepartments(slug).data?.items ?? []
  const refresh = useRefresh(slug, 'courses')
  const [editing, setEditing] = useState<Course | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Course | null>(null)
  const [importing, setImporting] = useState(false)
  const [notice, setNotice] = useState('')

  // a dept_admin lands on their first department so the URL always shows the filter
  const firstManaged = membership.role === 'dept_admin' ? membership.departmentIds[0] : undefined
  const defaulted = useRef(false)
  useEffect(() => {
    if (defaulted.current) return
    defaulted.current = true // once, so choosing "All departments" sticks
    if (!departmentId && firstManaged) onDepartmentChange(firstManaged)
  }, [departmentId, firstManaged, onDepartmentChange])

  const list = useCursorList<Course>([slug, 'courses', 'list', { departmentId }], (cursor) =>
    unwrap(
      api.GET('/universities/{slug}/courses', {
        params: { path: { slug }, query: { cursor, limit: 50, departmentId } },
      }),
    ),
  )
  const del = useMutation({
    mutationFn: (c: Course) =>
      unwrap(api.DELETE('/universities/{slug}/courses/{id}', { params: { path: { slug, id: c.id } } })),
    onSuccess: async (_, c) => {
      await refresh()
      setDeleting(null)
      setNotice(`Course “${c.code}” deleted.`)
    },
  })

  const deptCode = new Map(departments.map((d) => [d.id, d.code]))
  const importButton = (
    <Button variant="secondary" onClick={() => setImporting(true)}>
      <Upload />
      Import courses
    </Button>
  )
  const newButton = (
    <Button onClick={() => setEditing('new')}>
      <Plus />
      New course
    </Button>
  )
  // the empty state holds the pair, so the header pair is hidden while the list is empty (one primary on screen)
  const empty = !list.isPending && !list.error && list.items.length === 0
  const actions = empty ? undefined : (
    <div className="flex gap-2">
      {importButton}
      {newButton}
    </div>
  )

  let body: React.ReactNode
  if (list.isPending) body = <Loading variant="table" columns={['Code', 'Title', 'Department', '']} />
  else if (list.error)
    body = (
      <ErrorBanner title="Couldn't load courses" onRetry={list.refetch}>
        {errorMessage(list.error, listCopy)}
      </ErrorBanner>
    )
  else if (list.items.length === 0)
    body = (
      <EmptyState
        title="No courses yet"
        icon={BookOpen}
        body={
          departmentId
            ? 'This department has no courses. Create one or import a CSV.'
            : 'Create a course or import your catalog from a CSV file.'
        }
        action={
          <div className="flex flex-wrap justify-center gap-2">
            {newButton}
            {importButton}
          </div>
        }
      />
    )
  else
    body = (
      <>
        <Table caption="Courses" stack>
          <thead>
            <tr>
              <th scope="col">Code</th>
              <th scope="col">Title</th>
              <th scope="col">Department</th>
              <th scope="col">
                <span className="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((c) => (
              <tr key={c.id}>
                <td data-label="Code" className="font-medium">
                  {c.code}
                </td>
                <td data-label="Title">{c.title}</td>
                <td data-label="Department">{deptCode.get(c.departmentId) ?? '—'}</td>
                <td className="text-right whitespace-nowrap">
                  {canManageDepartment(membership, c.departmentId) && (
                    <>
                      <Button variant="ghost" size="sm" aria-label={`Edit ${c.code}`} onClick={() => setEditing(c)}>
                        Edit
                      </Button>
                      <Button
                        variant="ghost-danger"
                        size="sm"
                        aria-label={`Delete ${c.code}`}
                        onClick={() => {
                          del.reset()
                          setDeleting(c)
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
          showCount={false}
        />
      </>
    )

  return (
    <>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <p className="text-base text-ink-muted">Catalog courses by department.</p>
        {actions}
      </div>
      {(!empty || departmentId) && (
        <div role="search" className="mb-6 flex flex-wrap items-end gap-4">
          <FormField label="Department" name="departmentId">
            <select
              className={`${inputClass} w-full sm:w-72`}
              value={departmentId ?? ''}
              onChange={(e) => onDepartmentChange(e.target.value)}
            >
              <option value="">All departments</option>
              {departments.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.code} · {d.name}
                </option>
              ))}
            </select>
          </FormField>
          {!list.isPending && !list.error && list.items.length > 0 && (
            <ListCount shown={list.items.length} className="ml-auto" />
          )}
        </div>
      )}
      <StatusMessage message={notice} />
      {body}
      {editing !== null && (
        <CourseDialog
          key={editing === 'new' ? 'new' : editing.id}
          slug={slug}
          mode={editing === 'new' ? 'create' : 'edit'}
          course={editing === 'new' ? undefined : editing}
          defaultDepartmentId={departmentId}
          onClose={() => setEditing(null)}
          onSaved={(code) => {
            setNotice(`Course “${code}” ${editing === 'new' ? 'created' : 'updated'}.`)
            setEditing(null)
          }}
        />
      )}
      <RemoveDialog
        open={deleting !== null}
        title={`Delete ${deleting?.code ?? ''}?`}
        description={`This permanently deletes “${deleting?.title ?? ''}”. A course that still has sections can't be deleted: delete its sections first.`}
        confirmLabel="Delete"
        pendingLabel="Deleting…"
        pending={del.isPending}
        error={del.error}
        copy={deleteCopy}
        onConfirm={() => deleting && del.mutate(deleting)}
        onClose={() => setDeleting(null)}
      />
      <CsvImportDialog
        open={importing}
        onClose={() => setImporting(false)}
        title="Import courses"
        target={{ kind: 'courses', slug }}
        columns={['departmentCode', 'code', 'title']}
        onDone={async (r) => {
          await refresh()
          setImporting(false)
          setNotice(importSummary('courses', r.counts))
        }}
      />
    </>
  )
}
