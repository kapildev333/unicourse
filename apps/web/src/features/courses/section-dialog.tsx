import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { Button } from '../../components/button'
import { Dialog } from '../../components/dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { useRefresh } from '../admin/queries'
import { useUniversity } from '../auth/use-me'
import { canManageDepartment } from './access'
import { useAllCourses } from './queries'

const copy = {
  'conflict:number': 'This course already has a section with this number in this term.',
  forbidden: 'You can only manage sections in your own departments.',
  'validation_failed:courseId': 'Choose a course.',
  'validation_failed:number': 'Enter a section number of at most 20 characters.',
}

/** Mount only while open (parent renders it conditionally) so the form starts fresh each time. */
export function SectionDialog(props: {
  slug: string
  term: { id: string; name: string }
  onClose: () => void
  onSaved: (code: string, number: string) => void
}) {
  const { slug, term, onClose, onSaved } = props
  const { membership } = useUniversity()
  const courses = (useAllCourses(slug).data?.items ?? []).filter((c) => canManageDepartment(membership, c.departmentId))
  const refresh = useRefresh(slug, 'sections')
  const [courseId, setCourseId] = useState('')
  const [number, setNumber] = useState('')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)

  const m = useMutation({
    mutationFn: (body: { courseId: string; termId: string; number: string }) =>
      unwrap(api.POST('/universities/{slug}/sections', { params: { path: { slug } }, body })),
    onSuccess: async (s) => {
      await refresh([slug, 'my-sections'])
      onSaved(s.course.code, s.number)
    },
  })
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)

  return (
    <Dialog open onClose={onClose} title="New section">
      <form
        ref={formRef}
        noValidate
        onSubmit={(e) => {
          e.preventDefault()
          m.mutate({ courseId, termId: term.id, number: number.trim() })
        }}
        className="flex flex-col gap-4"
      >
        {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
        <p className="text-sm text-ink-muted">Term: {term.name}</p>
        <FormField label="Course" name="courseId" error={fields.courseId}>
          <select className={inputClass} value={courseId} onChange={(e) => setCourseId(e.target.value)}>
            <option value="">Choose a course</option>
            {courses.map((c) => (
              <option key={c.id} value={c.id}>
                {c.code} — {c.title}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label="Section number" name="number" hint="For example 001." error={fields.number}>
          <input type="text" className={`${inputClass} font-mono`} value={number} onChange={(e) => setNumber(e.target.value)} />
        </FormField>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? 'Saving…' : 'Create section'}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}
