import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Button } from '../../components/button'
import { Dialog } from '../../components/dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { useAllDepartments, useRefresh } from '../admin/queries'
import { useUniversity } from '../auth/use-me'

type Course = components['schemas']['Course']

const copy = {
  'conflict:code': 'A course with this code already exists in this department.',
  forbidden: 'You can only manage courses in your own departments.',
  'validation_failed:departmentId': 'Choose a department.',
  'validation_failed:code': 'Enter a course code of at most 20 characters.',
  'validation_failed:title': 'Enter a title of at most 200 characters.',
}

/** Mount only while open (parent renders it conditionally) so the form starts fresh each time. */
export function CourseDialog(props: {
  slug: string
  mode: 'create' | 'edit'
  course?: Course
  defaultDepartmentId?: string
  onClose: () => void
  onSaved: (code: string) => void
}) {
  const { slug, mode, course, defaultDepartmentId, onClose, onSaved } = props
  const { membership } = useUniversity()
  const departments = useAllDepartments(slug).data?.items ?? []
  const managed = membership.role === 'admin' ? departments : departments.filter((d) => membership.departmentIds.includes(d.id))
  const refresh = useRefresh(slug, 'courses')
  const [departmentId, setDepartmentId] = useState(
    course?.departmentId ?? (managed.some((d) => d.id === defaultDepartmentId) ? defaultDepartmentId! : ''),
  )
  const [code, setCode] = useState(course?.code ?? '')
  const [title, setTitle] = useState(course?.title ?? '')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)

  const m = useMutation({
    mutationFn: (body: components['schemas']['CourseUpdate']) =>
      course
        ? unwrap(api.PATCH('/universities/{slug}/courses/{id}', { params: { path: { slug, id: course.id } }, body }))
        : unwrap(
            api.POST('/universities/{slug}/courses', {
              params: { path: { slug } },
              body: body as components['schemas']['CourseCreate'],
            }),
          ),
    onSuccess: async (saved) => {
      await refresh(...(course ? [[slug, 'sections'], [slug, 'my-sections']] : []))
      onSaved(saved.code)
    },
  })
  const fields = useFieldErrors(m.error, copy)
  useFocusFirstError(formRef, m.error, bannerRef)

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const c = code.trim()
    const t = title.trim()
    if (!course) return m.mutate({ departmentId, code: c, title: t })
    const body = {
      ...(departmentId !== course.departmentId && { departmentId }),
      ...(c !== course.code && { code: c }),
      ...(t !== course.title && { title: t }),
    }
    if (Object.keys(body).length === 0) return onClose()
    m.mutate(body)
  }

  // a dept_admin editing a course keeps its department option even if listing is partial
  return (
    <Dialog open onClose={onClose} title={mode === 'create' ? 'New course' : `Edit ${course?.code ?? ''}`}>
      <form ref={formRef} noValidate onSubmit={submit} className="flex flex-col gap-4">
        {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
        <FormField label="Department" name="departmentId" error={fields.departmentId}>
          <select className={inputClass} value={departmentId} onChange={(e) => setDepartmentId(e.target.value)}>
            <option value="">Choose a department</option>
            {managed.map((d) => (
              <option key={d.id} value={d.id}>
                {d.code} — {d.name}
              </option>
            ))}
          </select>
        </FormField>
        <FormField label="Code" name="code" hint="Up to 20 characters, e.g. CS101" error={fields.code}>
          <input type="text" className={`${inputClass} font-mono`} value={code} onChange={(e) => setCode(e.target.value)} />
        </FormField>
        <FormField label="Title" name="title" error={fields.title}>
          <input type="text" className={inputClass} value={title} onChange={(e) => setTitle(e.target.value)} />
        </FormField>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? 'Saving…' : mode === 'create' ? 'Create course' : 'Save changes'}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}
