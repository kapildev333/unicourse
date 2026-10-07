import { useMutation } from '@tanstack/react-query'
import { Upload, Users } from 'lucide-react'
import { useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { LoadMore } from '../../components/load-more'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { formatDate } from '../../lib/format'
import { useCursorList } from '../../lib/use-cursor-list'
import { useRefresh } from '../admin/queries'
import { personName } from './access'
import { CsvImportDialog, importSummary } from './csv-import-dialog'
import { RemoveDialog } from './remove-dialog'

type Enrollment = components['schemas']['Enrollment']

export function Roster(props: { slug: string; sectionId: string; canEdit: boolean }) {
  const { slug, sectionId, canEdit } = props
  const refresh = useRefresh(slug, 'sections')
  const [removing, setRemoving] = useState<Enrollment | null>(null)
  const [importing, setImporting] = useState(false)
  const [notice, setNotice] = useState('')
  const list = useCursorList<Enrollment>([slug, 'sections', 'detail', sectionId, 'enrollments'], (cursor) =>
    unwrap(
      api.GET('/universities/{slug}/sections/{id}/enrollments', {
        params: { path: { slug, id: sectionId }, query: { cursor, limit: 50 } },
      }),
    ),
  )
  const remove = useMutation({
    mutationFn: (e: Enrollment) =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/enrollments/{userId}', {
          params: { path: { slug, id: sectionId, userId: e.userId } },
        }),
      ),
    onSuccess: async (_, e) => {
      await refresh([slug, 'my-sections'])
      setRemoving(null)
      setNotice(`${personName(e)} removed from the roster.`)
    },
  })
  const importButton = (
    <Button variant="secondary" onClick={() => setImporting(true)}>
      <Upload />
      Import roster
    </Button>
  )

  let body: React.ReactNode
  if (list.isPending) body = <Loading variant="table" columns={['Name', 'Email', 'Added']} />
  else if (list.error)
    body = (
      <ErrorBanner title="Couldn't load the roster" onRetry={list.refetch}>
        {errorMessage(list.error)}
      </ErrorBanner>
    )
  else if (list.items.length === 0)
    body = (
      <EmptyState
        title="No students yet"
        icon={Users}
        body={
          canEdit
            ? 'Import a CSV of student emails to build the roster.'
            : 'Students enrolled in this section will appear here.'
        }
      />
    )
  else
    body = (
      <>
        <Table caption="Roster" stack>
          <thead>
            <tr>
              <th scope="col">Name</th>
              <th scope="col">Email</th>
              <th scope="col">Added</th>
              {canEdit && (
                <th scope="col">
                  <span className="sr-only">Actions</span>
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {list.items.map((e) => (
              <tr key={e.userId}>
                <td data-label="Name" className="font-medium">
                  {e.displayName ? (
                    e.displayName
                  ) : (
                    <>
                      <span className="text-ink-muted">—</span>
                      <span className="sr-only">no name</span>
                    </>
                  )}
                </td>
                <td data-label="Email">{e.email}</td>
                <td data-label="Added">{formatDate(e.createdAt)}</td>
                {canEdit && (
                  <td className="text-right">
                    <Button
                      variant="ghost-danger"
                      size="sm"
                      aria-label={`Remove ${personName(e)} from the roster`}
                      onClick={() => {
                        remove.reset()
                        setRemoving(e)
                      }}
                    >
                      Remove
                    </Button>
                  </td>
                )}
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
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 id="roster-h" className="text-xl font-bold">
          Roster
          {!list.isPending && !list.error && (
            <>
              {' '}
              <span className="font-normal text-ink-muted">
                {list.items.length}
                {list.hasNextPage ? '+' : ''}
              </span>
            </>
          )}
        </h2>
        {canEdit && importButton}
      </div>
      <StatusMessage message={notice} />
      {body}
      <RemoveDialog
        open={removing !== null}
        title={`Remove ${removing ? personName(removing) : ''} from this section?`}
        description="They lose access to this section. Their account and other sections are not affected."
        confirmLabel="Remove"
        pendingLabel="Removing…"
        pending={remove.isPending}
        error={remove.error}
        onConfirm={() => removing && remove.mutate(removing)}
        onClose={() => setRemoving(null)}
      />
      {canEdit && (
        <CsvImportDialog
          open={importing}
          onClose={() => setImporting(false)}
          title="Import roster"
          target={{ kind: 'enrollments', slug, sectionId }}
          columns={['email', 'displayName']}
          hint="Emails not yet in the university get an invitation."
          onDone={async (r) => {
            await refresh([slug, 'my-sections'], [slug, 'invitations'])
            setImporting(false)
            setNotice(importSummary('enrollments', r.counts))
          }}
        />
      )}
    </>
  )
}
