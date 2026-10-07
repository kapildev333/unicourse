import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, textareaClass } from '../../components/form-field'
import { countHint } from './plain-text'
import { sectionKey, useContentRefresh } from './queries'

const copy = {
  'validation_failed:description': 'Use at most 10,000 characters of plain text.',
  forbidden: "Only this section's staff and administrators can edit its description.",
}

/** Mount only while open so every open starts from the saved text. */
export function DescriptionDialog(props: {
  slug: string
  sectionId: string
  initial: string
  onClose: () => void
  onSaved: (message: string) => void
}) {
  const { slug, sectionId, initial, onClose, onSaved } = props
  const refresh = useContentRefresh(slug)
  const [value, setValue] = useState(initial)
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const m = useMutation({
    mutationFn: (description: string) =>
      unwrap(
        api.PUT('/universities/{slug}/sections/{id}/description', {
          params: { path: { slug, id: sectionId } },
          body: { description },
        }),
      ),
    onSuccess: async (_, description) => {
      await refresh(sectionKey(slug, sectionId))
      onSaved(description.trim() ? 'Description saved.' : 'Description removed.')
    },
  })
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)

  function submit(e: React.FormEvent) {
    e.preventDefault()
    if (value === initial) return onClose()
    m.mutate(value)
  }

  return (
    <Dialog open size="lg" title={initial ? 'Edit description' : 'Add description'} onClose={onClose}>
      <form ref={formRef} noValidate onSubmit={submit} className="flex flex-col gap-4">
        {!fields.description && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
        <FormField
          label="Description"
          name="description"
          error={fields.description}
          hint={`Plain text; line breaks are kept and web links work. ${countHint(value, 10000)}`}
        >
          <textarea rows={12} className={textareaClass} value={value} onChange={(e) => setValue(e.target.value)} />
        </FormField>
        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? 'Saving…' : 'Save description'}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
