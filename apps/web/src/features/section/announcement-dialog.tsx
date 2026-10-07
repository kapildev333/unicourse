import { useMutation } from '@tanstack/react-query'
import { useId, useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass, textareaClass } from '../../components/form-field'
import { countHint } from './plain-text'
import { announcementsKey, useContentRefresh, type Announcement } from './queries'

const copy = {
  'validation_failed:title': 'Enter a title of at most 200 characters, on one line.',
  'validation_failed:body': 'Write a message of at most 20,000 characters.',
  forbidden: "Only this section's staff and administrators can post announcements.",
  not_found: 'This announcement was deleted. Reload the page.',
}

/** Mount only while open so the form starts fresh each time. */
export function AnnouncementDialog(props: {
  slug: string
  sectionId: string
  announcement?: Announcement
  onClose: () => void
  onSaved: (message: string) => void
}) {
  const { slug, sectionId, announcement, onClose, onSaved } = props
  const refresh = useContentRefresh(slug)
  const [title, setTitle] = useState(announcement?.title ?? '')
  const [body, setBody] = useState(announcement?.body ?? '')
  const [email, setEmail] = useState(true)
  const emailHintId = useId()
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const path = { slug, id: sectionId }

  const m = useMutation({
    mutationFn: (patch: { title?: string; body?: string }) =>
      announcement
        ? unwrap(
            api.PATCH('/universities/{slug}/sections/{id}/announcements/{announcementId}', {
              params: { path: { ...path, announcementId: announcement.id } },
              body: patch,
            }),
          )
        : unwrap(
            api.POST('/universities/{slug}/sections/{id}/announcements', {
              params: { path },
              body: { title: patch.title!, body: patch.body!, email },
            }),
          ),
    onSuccess: async (saved) => {
      await refresh(announcementsKey(slug, sectionId))
      onSaved(
        announcement
          ? 'Announcement saved. Edits are not emailed.'
          : saved.emailed
            ? 'Announcement posted and emailed to the section.'
            : email
              ? 'Announcement posted. No email was sent: nobody else is in this section yet.'
              : 'Announcement posted. No email was sent.',
      )
    },
  })
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const t = title.trim()
    if (!announcement) return m.mutate({ title: t, body })
    const patch = { ...(t !== announcement.title && { title: t }), ...(body !== announcement.body && { body }) }
    if (Object.keys(patch).length === 0) return onClose()
    m.mutate(patch)
  }

  return (
    <Dialog
      open
      size="lg"
      title={announcement ? 'Edit announcement' : 'New announcement'}
      description={announcement ? 'Edits are not emailed again.' : undefined}
      onClose={onClose}
    >
      <form ref={formRef} noValidate onSubmit={submit} className="flex flex-col gap-4">
        {!(fields.title || fields.body) && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
        <FormField label="Title" name="title" error={fields.title}>
          <input type="text" maxLength={200} className={inputClass} value={title} onChange={(e) => setTitle(e.target.value)} />
        </FormField>
        <FormField label="Message" name="body" error={fields.body} hint={countHint(body, 20000)}>
          <textarea rows={8} className={textareaClass} value={body} onChange={(e) => setBody(e.target.value)} />
        </FormField>
        {!announcement && (
          <div>
            <label className="flex items-center gap-2 text-sm font-medium text-ink">
              <input
                type="checkbox"
                className="size-4 accent-accent"
                checked={email}
                onChange={(e) => setEmail(e.target.checked)}
                aria-describedby={emailHintId}
              />
              Also email everyone in this section
            </label>
            <p id={emailHintId} className="mt-1 pl-6 text-sm text-ink-muted">
              Students and staff get a copy by email. Edits are never emailed.
            </p>
          </div>
        )}
        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? (announcement ? 'Saving…' : 'Posting…') : announcement ? 'Save changes' : 'Post announcement'}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
