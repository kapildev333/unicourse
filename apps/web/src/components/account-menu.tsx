import { useMutation, useQueryClient } from '@tanstack/react-query'
import { LogOut, UserRound } from 'lucide-react'
import { useRef, useState } from 'react'
import { api, unwrap } from '../api/client'
import { signOut } from '../features/auth/sign-out'
import type { Me, MyMembership } from '../features/auth/use-me'
import { useNotificationSettings, useSaveNotificationSettings } from '../features/notifications/queries'
import { Button } from './button'
import { Dialog, DialogFooter } from './dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from './form-errors'
import { FormField, inputClass } from './form-field'

const copy = { 'validation_failed:displayName': 'Enter a name of up to 100 characters.' }

function NameForm({ user, membership, onClose }: { user: Me['user']; membership: MyMembership; onClose: () => void }) {
  const queryClient = useQueryClient()
  const slug = membership.university.slug
  const settings = useNotificationSettings(slug)
  const saveSettings = useSaveNotificationSettings(slug)
  const [mailChoice, setMailChoice] = useState<boolean>()
  const mail = mailChoice ?? settings.data?.announcementMail ?? true
  const [name, setName] = useState(user.displayName ?? '')
  const [empty, setEmpty] = useState(false)
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const m = useMutation({
    mutationFn: async (displayName: string) => {
      const me = await unwrap(api.PATCH('/me', { body: { displayName } }))
      queryClient.setQueryData(['me'], me)
      // the settings PATCH only when the box changed
      if (settings.data && mail !== settings.data.announcementMail) await saveSettings.mutateAsync(mail)
      return me
    },
    onSuccess: onClose,
  })
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)
  const error = empty ? copy['validation_failed:displayName'] : fields.displayName

  return (
    <form
      ref={formRef}
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault()
        const n = name.trim()
        setEmpty(!n)
        if (n) m.mutate(n)
      }}
    >
      {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={m.error ?? settings.error} copy={copy} />}
      <p className="text-sm text-ink-muted">
        Signed in as <span className="font-medium text-ink">{user.email}</span>
      </p>
      <FormField label="Name" name="displayName" error={error} hint="Shown to other people instead of your email, in every university.">
        <input
          type="text"
          autoComplete="name"
          maxLength={100}
          className={inputClass}
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </FormField>
      <FormField
        label="Email me new announcements"
        name="announcementMail"
        hint={`In ${membership.university.name}. They still appear under Notifications.`}
      >
        <input
          type="checkbox"
          className="size-5 self-start accent-accent focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-60"
          checked={mail}
          disabled={!settings.data}
          onChange={(e) => setMailChoice(e.target.checked)}
        />
      </FormField>
      <DialogFooter>
        {/* phones only: the header's Sign out is hidden below md to keep the bar on one row */}
        <Button variant="ghost-danger" className="mr-auto md:hidden" onClick={() => void signOut(queryClient)}>
          <LogOut />
          Sign out
        </Button>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button type="submit" disabled={m.isPending}>
          {m.isPending ? 'Saving…' : 'Save'}
        </Button>
      </DialogFooter>
    </form>
  )
}

/** Header control: opens a dialog with the signed-in email and the user's own name (PATCH /me) and the announcement-mail opt-out for the current university. Sign out stays its own header button. */
export function AccountMenu({ user, membership }: { user: Me['user']; membership: MyMembership }) {
  const [open, setOpen] = useState(false)
  const label = user.displayName || user.email
  return (
    <>
      <Button
        variant="ghost-neutral"
        title={`Account (${user.email})`}
        className="max-md:w-10 max-md:px-0"
        onClick={() => setOpen(true)}
      >
        <UserRound />
        <span className="sr-only">Account</span>{' '}
        <span className="hidden max-w-32 truncate md:inline">{label}</span>
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} title="Your account">
        {open && <NameForm user={user} membership={membership} onClose={() => setOpen(false)} />}
      </Dialog>
    </>
  )
}
