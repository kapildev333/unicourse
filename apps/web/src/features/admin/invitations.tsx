import { useMutation } from '@tanstack/react-query'
import { Mail, UserPlus } from 'lucide-react'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Badge, RoleBadge, roleLabel } from '../../components/badge'
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
import { formatDate } from '../../lib/format'
import { useCursorList } from '../../lib/use-cursor-list'
import { DepartmentPicker } from './department-picker'
import { useAllDepartments, useRefresh } from './queries'
import { PageHeader } from '../../components/page-header'

type Invitation = components['schemas']['Invitation']
type Role = Invitation['role']

const formCopy = {
  'validation_failed:email': 'Enter a valid email address.',
  'validation_failed:departmentIds': 'Choose at least one department for a department admin.',
  'conflict:email': 'This email already has access or a pending invitation.',
  forbidden: 'You can only invite members.',
  not_found: 'That department no longer exists. Reload and try again.',
}
const listCopy = { forbidden: "You don't have permission to view this." }
const revokeCopy = { conflict: 'This invitation was already accepted.' }

function InviteForm(props: { slug: string; role: string; onDone: (notice?: string) => void; onCancel: () => void }) {
  const { slug, role: callerRole, onDone, onCancel } = props
  const refresh = useRefresh(slug, 'invitations')
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<Role>('member')
  const [departmentIds, setDepartmentIds] = useState<string[]>([])
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const fixed = callerRole === 'dept_admin'
  const m = useMutation({
    mutationFn: (body: components['schemas']['InvitationCreate']) =>
      unwrap(
        api.POST('/universities/{slug}/invitations', {
          params: { path: { slug } },
          body,
        }),
      ),
    onSuccess: async (_, body) => {
      await refresh()
      onDone(`Invitation sent to ${body.email}.`)
    },
  })
  const fields = useFieldErrors(m.error, formCopy)
  useFocusFirstError(formRef, m.error, bannerRef)

  return (
    <form
      ref={formRef}
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault()
        const r = fixed ? 'member' : role
        m.mutate({
          email: email.trim(),
          role: r,
          departmentIds: r === 'dept_admin' ? departmentIds : [],
        })
      }}
    >
      {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={m.error} copy={formCopy} />}
      <FormField label="Email" name="email" error={fields.email}>
        <input
          type="email"
          autoComplete="off"
          className={inputClass}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
        />
      </FormField>
      {fixed ? (
        <p className="text-sm">Role: {roleLabel('member')}</p>
      ) : (
        <FormField label="Role" name="role" error={fields.role}>
          <select className={inputClass} value={role} onChange={(e) => setRole(e.target.value as Role)}>
            <option value="admin">{roleLabel('admin')}</option>
            <option value="dept_admin">{roleLabel('dept_admin')}</option>
            <option value="member">{roleLabel('member')}</option>
          </select>
        </FormField>
      )}
      {!fixed && role === 'dept_admin' && (
        <DepartmentPicker slug={slug} value={departmentIds} onChange={setDepartmentIds} error={fields.departmentIds} />
      )}
      <DialogFooter>
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={m.isPending}>
          {m.isPending ? 'Sending…' : 'Send invitation'}
        </Button>
      </DialogFooter>
    </form>
  )
}

function RevokeForm(props: {
  slug: string
  invitation: Invitation
  onDone: (notice?: string) => void
  onCancel: () => void
}) {
  const { slug, invitation, onDone, onCancel } = props
  const refresh = useRefresh(slug, 'invitations')
  const m = useMutation({
    mutationFn: () =>
      unwrap(
        api.DELETE('/universities/{slug}/invitations/{id}', {
          params: { path: { slug, id: invitation.id } },
        }),
      ),
    onSuccess: async () => {
      await refresh()
      onDone(`Invitation for ${invitation.email} revoked.`)
    },
  })
  return (
    <div className="flex flex-col gap-4">
      {m.error != null && <ErrorBanner title="">{errorMessage(m.error, revokeCopy)}</ErrorBanner>}
      <DialogFooter>
        <Button variant="secondary" onClick={onCancel}>
          Cancel
        </Button>
        <Button variant="danger" disabled={m.isPending} onClick={() => m.mutate()}>
          {m.isPending ? 'Revoking…' : 'Revoke'}
        </Button>
      </DialogFooter>
    </div>
  )
}

type Status = 'accepted' | 'expired' | 'pending'
const statusOf = (i: Invitation): Status =>
  i.acceptedAt ? 'accepted' : new Date(i.expiresAt).getTime() < Date.now() ? 'expired' : 'pending'

// The caller's role is `userRole`, not `role`: jsx-a11y/aria-role flags a `role` prop on components.
export function InvitationsPage({ slug, userRole }: { slug: string; userRole: string }) {
  const list = useCursorList<Invitation>([slug, 'invitations'], (cursor) =>
    unwrap(
      api.GET('/universities/{slug}/invitations', {
        params: { path: { slug }, query: { cursor, limit: 50 } },
      }),
    ),
  )
  const depts = useAllDepartments(slug)
  const deptCode = new Map(depts.data?.items.map((d) => [d.id, d.code]))
  const [inviting, setInviting] = useState(false)
  const [revoking, setRevoking] = useState<Invitation | null>(null)
  const [notice, setNotice] = useState('')
  const done = (n?: string) => {
    setInviting(false)
    setRevoking(null)
    if (n) setNotice(n)
  }
  const create = (
    <Button disabled={list.isPending} onClick={() => setInviting(true)}>
      <UserPlus size={16} />
      Invite someone
    </Button>
  )

  let body: React.ReactNode
  if (list.isPending) body = <Loading variant="table" columns={['Email', 'Role', 'Departments', 'Status', 'Expires', '']} />
  else if (list.error) body = <ErrorBanner title="Couldn't load invitations" onRetry={list.refetch}>{errorMessage(list.error, listCopy)}</ErrorBanner>
  else if (list.items.length === 0)
    body = (
      <EmptyState
        icon={Mail}
        title="No invitations"
        body="Invite colleagues by email. They'll join when they sign in."
        action={create}
      />
    )
  else
    body = (
      <>
        <ListCount shown={list.items.length} className="mb-2 text-right" />
        <Table caption="Invitations" stack>
          <thead>
            <tr>
              <th scope="col">Email</th>
              <th scope="col">Role</th>
              <th scope="col">Departments</th>
              <th scope="col">Status</th>
              <th scope="col">Expires</th>
              <th scope="col">
                <span className="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((i) => {
              const status = statusOf(i)
              return (
                <tr key={i.id}>
                  <td data-label="Email">{i.email}</td>
                  <td data-label="Role">
                    <RoleBadge role={i.role} />
                  </td>
                  <td data-label="Departments" data-empty={i.departmentIds.length === 0 || undefined}>
                    {i.departmentIds.length === 0
                      ? '—'
                      : i.departmentIds.map((id) => deptCode.get(id) ?? 'Unknown').join(', ')}
                  </td>
                  <td data-label="Status">
                    <Badge tone={status === 'pending' ? 'warning' : 'neutral'}>
                      {status === 'accepted' ? 'Accepted' : status === 'expired' ? 'Expired' : 'Pending'}
                    </Badge>
                  </td>
                  <td data-label="Expires" data-empty={status === 'accepted' || undefined}>
                    {status === 'accepted' ? <span className="text-ink-muted">—</span> : formatDate(i.expiresAt)}
                  </td>
                  <td className="text-right whitespace-nowrap">
                    {status === 'pending' && (
                      <Button
                        variant="ghost-danger"
                        size="sm"
                        aria-label={`Revoke invitation for ${i.email}`}
                        onClick={() => setRevoking(i)}
                      >
                        Revoke
                      </Button>
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
          showCount={false}
        />
      </>
    )

  return (
    <>
      <PageHeader as="h2"
        title="Invitations"
        description="People you invite get access when they first sign in with that email."
        action={create}
      />
      <StatusMessage message={notice} />
      {body}
      <Dialog open={inviting} onClose={() => setInviting(false)} title="Invite someone">
        {inviting && <InviteForm slug={slug} role={userRole} onDone={done} onCancel={() => setInviting(false)} />}
      </Dialog>
      <Dialog
        open={revoking !== null}
        onClose={() => setRevoking(null)}
        title="Revoke invitation?"
        description={revoking ? `${revoking.email} won't be able to join with this invitation.` : undefined}
      >
        {revoking && (
          <RevokeForm
            key={revoking.id}
            slug={slug}
            invitation={revoking}
            onDone={done}
            onCancel={() => setRevoking(null)}
          />
        )}
      </Dialog>
    </>
  )
}
