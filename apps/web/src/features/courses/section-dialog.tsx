import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { useRefresh } from '../admin/queries'
import { useUniversity } from '../auth/use-me'
import { classCode } from '../../lib/schedule'
import { canManageDepartment } from './access'
import { MeetingsEditor, nextKey, toInput, type MeetingRow } from './meetings-editor'
import { useAllCourses } from './queries'

type Section = components['schemas']['Section']

const copy = {
  'conflict:number': 'This course already has a section with this number in this term.',
  forbidden: 'You can only manage sections in your own departments.',
  'validation_failed:courseId': 'Choose a course.',
  'validation_failed:number': 'Enter a section number of at most 20 characters.',
  'validation_failed:meetings': 'Add at most 20 meeting times.',
}

/** Mount only while open (parent renders it conditionally) so the form starts fresh each time. */
export function SectionDialog(props: {
  slug: string
  term: { id: string; name: string }
  /** edit mode: number and meetings */
  section?: Section
  onClose: () => void
  onSaved: (code: string, number: string) => void
}) {
  const { slug, term, section, onClose, onSaved } = props
  const { membership } = useUniversity()
  const onlyDept = membership.role === 'dept_admin' && membership.departmentIds.length === 1 ? membership.departmentIds[0] : undefined
  const courseData = useAllCourses(slug, onlyDept, !section).data
  const courses = (courseData?.items ?? []).filter((c) => canManageDepartment(membership, c.departmentId))
  const refresh = useRefresh(slug, 'sections')
  const [courseId, setCourseId] = useState('')
  const [number, setNumber] = useState(section?.number ?? '')
  const [rows, setRows] = useState<MeetingRow[]>(() => section?.meetings.map((m) => ({ ...m, key: nextKey() })) ?? [])
  const [sentKeys, setSentKeys] = useState<string[]>([])
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)

  const m = useMutation({
    mutationFn: (body: components['schemas']['SectionCreate'] | components['schemas']['SectionUpdate']) =>
      section
        ? unwrap(
            api.PATCH('/universities/{slug}/sections/{id}', {
              params: { path: { slug, id: section.id } },
              body: body as components['schemas']['SectionUpdate'],
            }),
          )
        : unwrap(
            api.POST('/universities/{slug}/sections', {
              params: { path: { slug } },
              body: body as components['schemas']['SectionCreate'],
            }),
          ),
    onSuccess: async (s) => {
      await refresh([slug, 'my-sections'])
      onSaved(s.course.code, s.number)
    },
  })
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)

  return (
    <Dialog open wide onClose={onClose} title={section ? `Edit ${classCode(section)}` : 'New section'}>
      <form
        ref={formRef}
        noValidate
        onSubmit={(e) => {
          e.preventDefault()
          // on-screen order, blank rows included: details[i] is the i-th sent row
          setSentKeys(rows.map((r) => r.key))
          const meetings = rows.map(toInput)
          if (!section)
            return m.mutate({
              courseId,
              termId: term.id,
              number: number.trim(),
              ...(rows.length ? { meetings } : {}),
            })
          const n = number.trim()
          const body = {
            ...(n !== section.number && { number: n }),
            ...(JSON.stringify(meetings) !== JSON.stringify(section.meetings.map(toInput)) && { meetings }),
          }
          if (Object.keys(body).length === 0) return onClose()
          m.mutate(body)
        }}
        className="flex flex-col gap-4"
      >
        {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
        <p className="text-sm text-ink-muted">Term: {term.name}</p>
        {section ? (
          <p className="text-sm text-ink-muted">
            Course: {section.course.code} — {section.course.title}
          </p>
        ) : (
          <FormField
            label="Course"
            name="courseId"
            hint={courseData?.nextCursor ? 'Showing the first 200 courses. Use Import sections for the others.' : undefined}
            error={fields.courseId}
          >
            <select className={inputClass} value={courseId} onChange={(e) => setCourseId(e.target.value)}>
              <option value="">Choose a course</option>
              {courses.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.code} — {c.title}
                </option>
              ))}
            </select>
          </FormField>
        )}
        <FormField label="Section number" name="number" hint="For example 001." error={fields.number}>
          <input type="text" className={inputClass} value={number} onChange={(e) => setNumber(e.target.value)} />
        </FormField>
        <MeetingsEditor rows={rows} onChange={setRows} fields={fields} sentKeys={sentKeys} />
        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? 'Saving…' : section ? 'Save changes' : 'Create section'}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
