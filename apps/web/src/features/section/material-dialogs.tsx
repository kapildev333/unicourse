import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { useContentRefresh, weeksKey, type Material, type Week } from './queries'
import { materialsForbidden } from './week-dialog'

const notFound = 'This file was deleted. Reload the page.'

function usePatch(slug: string, sectionId: string, m: Material, onSaved: (message: string) => void) {
  const refresh = useContentRefresh(slug)
  return useMutation({
    mutationFn: (body: { title?: string; weekId?: string }) =>
      unwrap(
        api.PATCH('/universities/{slug}/sections/{id}/materials/{materialId}', {
          params: { path: { slug, id: sectionId, materialId: m.id } },
          body,
        }),
      ),
    onSuccess: async (_saved, body) => {
      await refresh(weeksKey(slug, sectionId))
      onSaved(body.title !== undefined ? `Renamed to ${body.title}.` : '')
    },
  })
}

type Props = {
  slug: string
  sectionId: string
  material: Material
  onClose: () => void
  onSaved: (message: string) => void
}

export function RenameDialog({ slug, sectionId, material, onClose, onSaved }: Props) {
  const copy = {
    'validation_failed:title': 'Enter a title of at most 200 characters, on one line.',
    forbidden: materialsForbidden,
    not_found: notFound,
  }
  const [title, setTitle] = useState(material.title)
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const m = usePatch(slug, sectionId, material, onSaved)
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)
  return (
    <Dialog open title="Rename file" onClose={onClose}>
      <form
        ref={formRef}
        noValidate
        className="flex flex-col gap-4"
        onSubmit={(e) => {
          e.preventDefault()
          const t = title.trim()
          if (t === material.title) return onClose()
          m.mutate({ title: t })
        }}
      >
        {!fields.title && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
        <FormField
          label="Title"
          name="title"
          error={fields.title}
          hint={`File name: ${material.fileName}. Downloads keep the file name.`}
        >
          <input type="text" maxLength={200} className={inputClass} value={title} onChange={(e) => setTitle(e.target.value)} />
        </FormField>
        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? 'Saving…' : 'Save title'}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  )
}

export function MoveDialog({ weeks, ...p }: Props & { weeks: Week[] }) {
  const { slug, sectionId, material, onClose, onSaved } = p
  const copy = { forbidden: materialsForbidden, not_found: notFound }
  const targets = weeks.filter((w) => w.id !== material.weekId)
  const [weekId, setWeekId] = useState(targets[0]?.id ?? '')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const target = targets.find((w) => w.id === weekId)
  const m = usePatch(slug, sectionId, material, () => onSaved(`${material.title} moved to ${target?.title}.`))
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)
  return (
    <Dialog open title="Move file" description={material.title} onClose={onClose}>
      <form
        ref={formRef}
        noValidate
        className="flex flex-col gap-4"
        onSubmit={(e) => {
          e.preventDefault()
          m.mutate({ weekId })
        }}
      >
        {!fields.weekId && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
        <FormField label="Week" name="weekId" error={fields.weekId}>
          <select className={inputClass} value={weekId} onChange={(e) => setWeekId(e.target.value)}>
            {targets.map((w) => (
              <option key={w.id} value={w.id}>
                {w.title}
              </option>
            ))}
          </select>
        </FormField>
        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? 'Moving…' : 'Move file'}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
