import { useMutation } from '@tanstack/react-query'
import { CalendarRange, Plus } from 'lucide-react'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { FormErrors, errorMessage, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { LoadMore } from '../../components/load-more'
import { ListCount } from '../../components/list-count'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { Badge } from '../../components/badge'
import { formatDate } from '../../lib/format'
import { useCursorList } from '../../lib/use-cursor-list'
import { useMe } from '../auth/use-me'
import { useRefresh } from './queries'
import { PageHeader } from '../../components/page-header'

type Term = components['schemas']['Term']

// ISO dates compare as strings; both bounds inclusive, local today.
function termState(t: Term): 'Current' | 'Upcoming' | 'Past' {
  const n = new Date()
  const today = `${n.getFullYear()}-${String(n.getMonth() + 1).padStart(2, '0')}-${String(n.getDate()).padStart(2, '0')}`
  return today < t.startsOn ? 'Upcoming' : today > t.endsOn ? 'Past' : 'Current'
}

const formCopy = {
  'conflict:name': 'A term with this name already exists.',
  'validation_failed:name': 'Enter a name of up to 100 characters.',
  'validation_failed:startsOn': 'Enter a start date.',
  'validation_failed:endsOn': 'End date must be after the start date.',
}
const listCopy = { forbidden: "You don't have permission to view this." }
const deleteCopy = {
  conflict: 'This term still has sections. Delete them first.',
}

function TermForm(props: { slug: string; term: Term | null; onDone: (notice?: string) => void; onCancel: () => void }) {
  const { slug, term, onDone, onCancel } = props
  const refresh = useRefresh(slug, 'terms')
  const [name, setName] = useState(term?.name ?? '')
  const [startsOn, setStartsOn] = useState(term?.startsOn ?? '')
  const [endsOn, setEndsOn] = useState(term?.endsOn ?? '')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const m = useMutation({
    mutationFn: (body: { name?: string; startsOn?: string; endsOn?: string }) =>
      term
        ? unwrap(
            api.PATCH('/universities/{slug}/terms/{id}', {
              params: { path: { slug, id: term.id } },
              body,
            }),
          )
        : unwrap(
            api.POST('/universities/{slug}/terms', {
              params: { path: { slug } },
              body: body as { name: string; startsOn: string; endsOn: string },
            }),
          ),
    onSuccess: async (_, body) => {
      await refresh()
      onDone(`Term “${body.name ?? term?.name}” ${term ? 'updated' : 'created'}.`)
    },
  })
  const fields = useFieldErrors(m.error, formCopy)
  useFocusFirstError(formRef, m.error, bannerRef)

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const n = name.trim()
    if (!term) return m.mutate({ name: n, startsOn, endsOn })
    const body = {
      ...(n !== term.name && { name: n }),
      ...(startsOn !== term.startsOn && { startsOn }),
      ...(endsOn !== term.endsOn && { endsOn }),
    }
    if (Object.keys(body).length === 0) return onDone()
    m.mutate(body)
  }

  return (
    <form ref={formRef} noValidate onSubmit={submit} className="flex flex-col gap-4">
      {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={m.error} copy={formCopy} />}
      <FormField label="Name" name="name" error={fields.name}>
        <input type="text" className={inputClass} value={name} onChange={(e) => setName(e.target.value)} />
      </FormField>
      <FormField label="Starts on" name="startsOn" error={fields.startsOn}>
        <input type="date" className={inputClass} value={startsOn} onChange={(e) => setStartsOn(e.target.value)} />
      </FormField>
      <FormField label="Ends on" name="endsOn" error={fields.endsOn}>
        <input type="date" className={inputClass} value={endsOn} onChange={(e) => setEndsOn(e.target.value)} />
      </FormField>
      <DialogFooter>
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={m.isPending}>
          {m.isPending ? 'Saving…' : term ? 'Save changes' : 'Create term'}
        </Button>
      </DialogFooter>
    </form>
  )
}

function DeleteForm(props: { slug: string; term: Term; onDone: (notice?: string) => void; onCancel: () => void }) {
  const { slug, term, onDone, onCancel } = props
  const refresh = useRefresh(slug, 'terms')
  const m = useMutation({
    mutationFn: () =>
      unwrap(
        api.DELETE('/universities/{slug}/terms/{id}', {
          params: { path: { slug, id: term.id } },
        }),
      ),
    onSuccess: async () => {
      await refresh()
      onDone(`Term “${term.name}” deleted.`)
    },
  })
  return (
    <div className="flex flex-col gap-4">
      {m.error != null && <ErrorBanner title="">{errorMessage(m.error, deleteCopy)}</ErrorBanner>}
      <DialogFooter>
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button variant="danger" disabled={m.isPending} onClick={() => m.mutate()}>
          {m.isPending ? 'Deleting…' : 'Delete'}
        </Button>
      </DialogFooter>
    </div>
  )
}

export function TermsPage({ slug }: { slug: string }) {
  const list = useCursorList<Term>([slug, 'terms'], (cursor) =>
    unwrap(
      api.GET('/universities/{slug}/terms', {
        params: { path: { slug }, query: { cursor, limit: 50 } },
      }),
    ),
  )
  // D23 hides only the tabs: the page itself is read-only for a department admin
  const canManage = useMe().data?.memberships.find((m) => m.university.slug === slug)?.role === 'admin'
  const [editing, setEditing] = useState<Term | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Term | null>(null)
  const [notice, setNotice] = useState('')
  const done = (n?: string) => {
    setEditing(null)
    setDeleting(null)
    if (n) setNotice(n)
  }
  const create = !canManage ? undefined : (
    <Button disabled={list.isPending} onClick={() => setEditing('new')}>
      <Plus size={16} />
      New term
    </Button>
  )
  const header = <PageHeader as="h2" title="Terms" description="Academic terms that sections belong to." action={create} />

  let body: React.ReactNode
  if (list.isPending) body = <Loading variant="table" columns={['Name', 'Starts', 'Ends', 'Status', '']} />
  else if (list.error) body = <ErrorBanner title="Couldn't load terms" onRetry={list.refetch}>{errorMessage(list.error, listCopy)}</ErrorBanner>
  else if (list.items.length === 0)
    body = (
      <EmptyState
        icon={CalendarRange}
        title="No terms yet"
        body={canManage ? 'Add a term such as “Fall 2026” before creating sections.' : 'A university admin adds terms.'}
        action={create}
      />
    )
  else
    body = (
      <>
        <ListCount shown={list.items.length} className="mb-2 text-right" />
        <Table caption="Terms" stack>
          <thead>
            <tr>
              <th scope="col">Name</th>
              <th scope="col">Starts</th>
              <th scope="col">Ends</th>
              <th scope="col">Status</th>
              {canManage && (
                <th scope="col">
                  <span className="sr-only">Actions</span>
                </th>
              )}
            </tr>
          </thead>
          <tbody>
            {list.items.map((d) => (
              <tr key={d.id}>
                <td data-label="Name">{d.name}</td>
                <td data-label="Starts">{formatDate(d.startsOn)}</td>
                <td data-label="Ends">{formatDate(d.endsOn)}</td>
                <td data-label="Status">
                  <Badge tone="neutral">{termState(d)}</Badge>
                </td>
                {canManage && (
                  <td className="text-right whitespace-nowrap">
                    <Button variant="ghost" size="sm" aria-label={`Edit ${d.name}`} onClick={() => setEditing(d)}>
                      Edit
                    </Button>
                    <Button
                      variant="ghost-danger"
                      size="sm"
                      aria-label={`Delete ${d.name}`}
                      onClick={() => setDeleting(d)}
                    >
                      Delete
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
          showCount={false}
        />
      </>
    )

  return (
    <>
      {header}
      <StatusMessage message={notice} />
      {body}
      <Dialog
        open={editing !== null}
        onClose={() => setEditing(null)}
        title={editing === 'new' || editing === null ? 'New term' : 'Edit term'}
      >
        {editing !== null && (
          <TermForm
            key={editing === 'new' ? 'new' : editing.id}
            slug={slug}
            term={editing === 'new' ? null : editing}
            onDone={done}
            onCancel={() => setEditing(null)}
          />
        )}
      </Dialog>
      <Dialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        title="Delete term?"
        description={deleting ? `${deleting.name} will be removed. This can't be undone.` : undefined}
      >
        {deleting && (
          <DeleteForm key={deleting.id} slug={slug} term={deleting} onDone={done} onCancel={() => setDeleting(null)} />
        )}
      </Dialog>
    </>
  )
}
