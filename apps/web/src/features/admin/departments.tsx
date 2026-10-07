import { useMutation } from '@tanstack/react-query'
import { Building2, Plus } from 'lucide-react'
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
import { ListCount } from './list-count'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { useCursorList } from '../../lib/use-cursor-list'
import { useRefresh } from './queries'

type Department = components['schemas']['Department']

const formCopy = {
  'conflict:code': 'A department with this code already exists.',
  'validation_failed:code': 'Use 2–10 capital letters or digits.',
  'validation_failed:name': 'Enter a name of up to 100 characters.',
}
const listCopy = { forbidden: "You don't have permission to view this." }
const deleteCopy = {
  conflict: 'This department still has courses or department admins. Move or remove them first.',
}

function DepartmentForm(props: {
  slug: string
  department: Department | null
  onDone: (notice?: string) => void
  onCancel: () => void
}) {
  const { slug, department, onDone, onCancel } = props
  const refresh = useRefresh(slug, 'departments')
  const [name, setName] = useState(department?.name ?? '')
  const [code, setCode] = useState(department?.code ?? '')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const m = useMutation({
    mutationFn: (body: { name?: string; code?: string }) =>
      department
        ? unwrap(
            api.PATCH('/universities/{slug}/departments/{id}', {
              params: { path: { slug, id: department.id } },
              body,
            }),
          )
        : unwrap(
            api.POST('/universities/{slug}/departments', {
              params: { path: { slug } },
              body: body as { name: string; code: string },
            }),
          ),
    onSuccess: async (_, body) => {
      await refresh()
      onDone(`Department “${body.code ?? department?.code}” ${department ? 'updated' : 'created'}.`)
    },
  })
  const fields = useFieldErrors(m.error, formCopy)
  useFocusFirstError(formRef, m.error, bannerRef)

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const n = name.trim()
    const c = code.trim()
    if (!department) return m.mutate({ name: n, code: c })
    const body = {
      ...(n !== department.name && { name: n }),
      ...(c !== department.code && { code: c }),
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
      <FormField label="Code" name="code" hint="2–10 capital letters or digits, e.g. CS" error={fields.code}>
        <input
          type="text"
          autoCapitalize="characters"
          spellCheck={false}
          className={inputClass}
          value={code}
          onChange={(e) => setCode(e.target.value.toUpperCase())}
        />
      </FormField>
      <DialogFooter>
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={m.isPending}>
          {m.isPending ? 'Saving…' : department ? 'Save changes' : 'Create department'}
        </Button>
      </DialogFooter>
    </form>
  )
}

function DeleteForm(props: {
  slug: string
  department: Department
  onDone: (notice?: string) => void
  onCancel: () => void
}) {
  const { slug, department, onDone, onCancel } = props
  const refresh = useRefresh(slug, 'departments')
  const m = useMutation({
    mutationFn: () =>
      unwrap(
        api.DELETE('/universities/{slug}/departments/{id}', {
          params: { path: { slug, id: department.id } },
        }),
      ),
    onSuccess: async () => {
      await refresh()
      onDone(`Department “${department.code}” deleted.`)
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

export function DepartmentsPage({ slug }: { slug: string }) {
  const list = useCursorList<Department>([slug, 'departments'], (cursor) =>
    unwrap(
      api.GET('/universities/{slug}/departments', {
        params: { path: { slug }, query: { cursor, limit: 50 } },
      }),
    ),
  )
  const [editing, setEditing] = useState<Department | 'new' | null>(null)
  const [deleting, setDeleting] = useState<Department | null>(null)
  const [notice, setNotice] = useState('')
  const done = (n?: string) => {
    setEditing(null)
    setDeleting(null)
    if (n) setNotice(n)
  }
  const create = (
    <Button disabled={list.isPending} onClick={() => setEditing('new')}>
      <Plus size={16} />
      New department
    </Button>
  )
  const header = (
    <PageHeader title="Departments" description="Departments group courses and department admins." action={create} />
  )

  let body: React.ReactNode
  if (list.isPending) body = <Loading variant="table" columns={['Name', 'Code', '']} />
  else if (list.error) body = <ErrorBanner title="Couldn't load departments" onRetry={list.refetch}>{errorMessage(list.error, listCopy)}</ErrorBanner>
  else if (list.items.length === 0)
    body = (
      <EmptyState
        icon={Building2}
        title="No departments yet"
        body="Create a department before adding courses or department admins."
        action={create}
      />
    )
  else
    body = (
      <>
        <ListCount shown={list.items.length} />
        <Table caption="Departments" stack>
          <thead>
            <tr>
              <th scope="col">Name</th>
              <th scope="col">Code</th>
              <th scope="col">
                <span className="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((d) => (
              <tr key={d.id}>
                <td data-label="Name">{d.name}</td>
                <td data-label="Code">{d.code}</td>
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
        title={editing === 'new' || editing === null ? 'New department' : 'Edit department'}
      >
        {editing !== null && (
          <DepartmentForm
            key={editing === 'new' ? 'new' : editing.id}
            slug={slug}
            department={editing === 'new' ? null : editing}
            onDone={done}
            onCancel={() => setEditing(null)}
          />
        )}
      </Dialog>
      <Dialog
        open={deleting !== null}
        onClose={() => setDeleting(null)}
        title="Delete department?"
        description={
          deleting ? `${deleting.name} (${deleting.code}) will be removed. This can't be undone.` : undefined
        }
      >
        {deleting && (
          <DeleteForm
            key={deleting.id}
            slug={slug}
            department={deleting}
            onDone={done}
            onCancel={() => setDeleting(null)}
          />
        )}
      </Dialog>
    </>
  )
}
