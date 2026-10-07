import { useMutation, useQueryClient } from '@tanstack/react-query'
import { UserRound } from 'lucide-react'
import { useRef, useState } from 'react'
import { api, unwrap } from '../api/client'
import type { Me } from '../features/auth/use-me'
import { Button } from './button'
import { Dialog, DialogFooter } from './dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from './form-errors'
import { FormField, inputClass } from './form-field'

const copy = { 'validation_failed:displayName': 'Enter a name of up to 100 characters.' }

function NameForm({ user, onClose }: { user: Me['user']; onClose: () => void }) {
  const queryClient = useQueryClient()
  const [name, setName] = useState(user.displayName ?? '')
  const [empty, setEmpty] = useState(false)
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const m = useMutation({
    mutationFn: (displayName: string) => unwrap(api.PATCH('/me', { body: { displayName } })),
    onSuccess: (me) => {
      queryClient.setQueryData(['me'], me)
      onClose()
    },
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
      {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
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
      <DialogFooter>
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

/** Header control: opens a dialog with the signed-in email and the user's own name (PATCH /me). Sign out stays its own header button. */
export function AccountMenu({ user }: { user: Me['user'] }) {
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
        <span className="sr-only">Account </span>
        <span className="hidden max-w-32 truncate md:inline">{label}</span>
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} title="Your account">
        {open && <NameForm user={user} onClose={() => setOpen(false)} />}
      </Dialog>
    </>
  )
}
