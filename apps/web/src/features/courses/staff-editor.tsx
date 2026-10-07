import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { ErrorBanner } from '../../components/error-banner'
import { FormErrors, errorMessage, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { StatusMessage } from '../../components/status-message'
import { useRefresh } from '../admin/queries'
import { personName } from './access'
import { RemoveDialog } from './remove-dialog'

type SectionStaff = components['schemas']['SectionStaff']
type PendingStaff = components['schemas']['PendingStaff']
type Role = 'instructor' | 'ta'

const copy = {
  'conflict:email':
    "This person can't be added: they are already on the staff, already invited to this section, or their invitation to the university is still pending.",
  'validation_failed:email': 'Enter a valid email address.',
  forbidden: 'You can only add TAs to this section.',
}
const roleName = (r: Role) => (r === 'ta' ? 'TA' : 'instructor')

export function StaffEditor(props: {
  slug: string
  sectionId: string
  sectionLabel?: string
  staff: SectionStaff[]
  pendingStaff: PendingStaff[]
  canManageInstructors: boolean
  canManageTAs: boolean
}) {
  const { slug, sectionId, sectionLabel = 'this section', staff, pendingStaff, canManageInstructors, canManageTAs } = props
  const refresh = useRefresh(slug, 'sections')
  const [email, setEmail] = useState('')
  const [notice, setNotice] = useState('')
  const [removing, setRemoving] = useState<SectionStaff | null>(null)
  const roleRef = useRef<Role>(canManageInstructors ? 'instructor' : 'ta')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)

  const add = useMutation({
    mutationFn: (body: { email: string; role: Role }) =>
      unwrap(api.POST('/universities/{slug}/sections/{id}/staff', { params: { path: { slug, id: sectionId } }, body })),
    onSuccess: async (data, body) => {
      if ('invited' in data) {
        await refresh([slug, 'my-sections'], [slug, 'invitations'])
        setNotice(`Invitation sent to ${body.email}. They will appear on the staff once they sign in.`)
      } else {
        await refresh([slug, 'my-sections'])
        setNotice(`${personName(data)} added as ${body.role === 'ta' ? 'TA' : 'Instructor'}.`)
      }
      setEmail('')
    },
  })
  const remove = useMutation({
    mutationFn: (s: SectionStaff) =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/staff/{userId}', {
          params: { path: { slug, id: sectionId, userId: s.userId } },
        }),
      ),
    onSuccess: async (_, s) => {
      await refresh([slug, 'my-sections'])
      setRemoving(null)
      setNotice(`${personName(s)} removed from the staff.`)
    },
  })
  const cancel = useMutation({
    mutationFn: (s: PendingStaff) =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/pending-staff/{invitationId}', {
          params: { path: { slug, id: sectionId, invitationId: s.invitationId } },
        }),
      ),
    onSuccess: async (_, s) => {
      await refresh([slug, 'my-sections'], [slug, 'invitations'])
      setNotice(`Invitation for ${s.email} cancelled for this section`)
    },
    // a 404 means the row is already gone: refresh so it disappears
    onError: () => void refresh([slug, 'my-sections'], [slug, 'invitations']),
  })
  const fields = useFieldErrors(add.error, copy)
  useFocusFirstError(formRef, add.error, bannerRef)

  function group(title: string, role: Role, canRemove: boolean, empty: string) {
    const rows = staff.filter((s) => s.role === role)
    const pending = pendingStaff.filter((s) => s.role === role)
    return (
      <div>
        <h3 className="text-sm font-semibold">{title}</h3>
        <ul className="mt-2 flex flex-col divide-y divide-line rounded-lg border border-line bg-surface">
          {rows.length + pending.length === 0 && <li className="px-4 py-3 text-sm text-ink-muted">{empty}</li>}
          {rows.map((s) => (
            <li key={s.userId} className="flex items-center justify-between gap-2 px-4 py-2">
              <div>
                <p className="text-sm font-medium">{personName(s)}</p>
                {s.displayName && <p className="text-xs text-ink-muted">{s.email}</p>}
              </div>
              {canRemove && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-danger hover:bg-danger-soft"
                  aria-label={`Remove ${personName(s)} as ${roleName(role)}`}
                  onClick={() => {
                    remove.reset()
                    setRemoving(s)
                  }}
                >
                  Remove
                </Button>
              )}
            </li>
          ))}
          {pending.map((s) => (
            <li key={s.invitationId} className="flex items-center justify-between gap-2 px-4 py-2">
              <div>
                <p className="text-sm font-medium">
                  {s.email} <Badge tone="warning">Invited</Badge>
                </p>
                <p className="text-xs text-ink-muted">Joins when they sign in.</p>
              </div>
              {canRemove && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-danger hover:bg-danger-soft"
                  disabled={cancel.isPending}
                  aria-label={`Cancel invitation for ${s.email}`}
                  onClick={() => {
                    setNotice('')
                    cancel.mutate(s)
                  }}
                >
                  Cancel
                </Button>
              )}
            </li>
          ))}
        </ul>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <StatusMessage message={notice} />
      {cancel.error && <ErrorBanner>{errorMessage(cancel.error, { not_found: 'Already cancelled' })}</ErrorBanner>}
      {group('Instructors', 'instructor', canManageInstructors, 'No instructors yet.')}
      {group('Teaching assistants', 'ta', canManageTAs, 'No TAs yet.')}
      {(canManageInstructors || canManageTAs) && (
        <form
          ref={formRef}
          noValidate
          className="flex items-end gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            setNotice('')
            add.mutate({ email: email.trim(), role: roleRef.current })
          }}
        >
          <FormField
            label="Add by email"
            name="email"
            hint="If they don't have an account yet, we'll email them an invitation."
            error={fields.email}
          >
            <input
              type="email"
              autoComplete="off"
              className={`${inputClass} w-72`}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              onKeyDown={(e) => {
                // with two submit buttons the role is ambiguous: require a button click
                if (e.key === 'Enter' && canManageInstructors && canManageTAs) e.preventDefault()
              }}
            />
          </FormField>
          {canManageInstructors && (
            <Button type="submit" variant="secondary" disabled={add.isPending} onClick={() => (roleRef.current = 'instructor')}>
              Add instructor
            </Button>
          )}
          {canManageTAs && (
            <Button type="submit" variant="secondary" disabled={add.isPending} onClick={() => (roleRef.current = 'ta')}>
              Add TA
            </Button>
          )}
        </form>
      )}
      {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={add.error} copy={copy} />}
      <RemoveDialog
        open={removing !== null}
        title={`Remove ${removing ? personName(removing) : ''}?`}
        description={`${removing ? personName(removing) : ''} will no longer be ${
          removing?.role === 'ta' ? 'a TA' : 'an instructor'
        } on ${sectionLabel}.`}
        confirmLabel="Remove"
        pendingLabel="Removing…"
        pending={remove.isPending}
        error={remove.error}
        onConfirm={() => removing && remove.mutate(removing)}
        onClose={() => setRemoving(null)}
      />
    </div>
  )
}
