import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { ErrorBanner } from '../../components/error-banner'
import { FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass, textareaClass } from '../../components/form-field'
import { countHint } from './plain-text'
import { useContentRefresh, weeksKey, type Week } from './queries'

export const materialsForbidden = "Only this section's staff and administrators can change its materials."

const copy = {
  'validation_failed:title': 'Enter a title of at most 100 characters, on one line.',
  'validation_failed:notes': 'Use at most 5,000 characters for notes.',
  'validation_failed:weeks': 'A section can have at most 100 weeks.',
  forbidden: materialsForbidden,
  not_found: 'This week was deleted. Reload the page.',
}

/** Mount only while open so the form starts fresh each time. */
export function WeekDialog(props: {
  slug: string
  sectionId: string
  week?: Week
  defaultTitle: string
  onClose: () => void
  onSaved: (message: string) => void
}) {
  const { slug, sectionId, week, defaultTitle, onClose, onSaved } = props
  const refresh = useContentRefresh(slug)
  const [title, setTitle] = useState(week?.title ?? defaultTitle)
  const [notes, setNotes] = useState(week?.notes ?? '')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const path = { slug, id: sectionId }

  const m = useMutation({
    mutationFn: (patch: { title?: string; notes?: string }) =>
      week
        ? unwrap(
            api.PATCH('/universities/{slug}/sections/{id}/weeks/{weekId}', {
              params: { path: { ...path, weekId: week.id } },
              body: patch,
            }),
          )
        : unwrap(
            api.POST('/universities/{slug}/sections/{id}/weeks', {
              params: { path },
              body: { title: patch.title!, notes: patch.notes! },
            }),
          ),
    onSuccess: async (saved) => {
      await refresh(weeksKey(slug, sectionId))
      onSaved(week ? `${saved.title} saved.` : `${saved.title} added.`)
    },
  })
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const t = title.trim()
    if (!week) return m.mutate({ title: t, notes })
    const patch = {
      ...(t !== week.title && { title: t }),
      ...(notes !== week.notes && { notes }),
    }
    if (Object.keys(patch).length === 0) return onClose()
    m.mutate(patch)
  }

  return (
    <Dialog open size="lg" title={week ? 'Edit week' : 'Add week'} onClose={onClose}>
      <form ref={formRef} noValidate onSubmit={submit} className="flex flex-col gap-4">
        {fields.weeks ? (
          <ErrorBanner ref={bannerRef}>{fields.weeks}</ErrorBanner>
        ) : (
          !(fields.title || fields.notes) && <FormErrors ref={bannerRef} error={m.error} copy={copy} />
        )}
        <FormField label="Title" name="title" error={fields.title}>
          <input type="text" maxLength={100} className={inputClass} value={title} onChange={(e) => setTitle(e.target.value)} />
        </FormField>
        <FormField
          label="Notes"
          name="notes"
          error={fields.notes}
          hint={`Optional. Shown above this week's files; web links work. ${countHint(notes, 5000)}`}
        >
          <textarea rows={4} className={textareaClass} value={notes} onChange={(e) => setNotes(e.target.value)} />
        </FormField>
        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? (week ? 'Saving…' : 'Adding…') : week ? 'Save changes' : 'Add week'}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
