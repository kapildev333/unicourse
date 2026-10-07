import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { ApiError } from '../../api/errors'
import { Button } from '../../components/button'
import { Dialog } from '../../components/dialog'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage, FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { Table } from '../../components/table'
import { personName } from '../courses/access'
import { fmt2, finalPercent, letterFor, sentence } from './grade'
import { useGradesRefresh, type Gradebook, type GradeScale } from './queries'

type Student = Gradebook['students'][number]

export const finalizeCopy = {
  'validation_failed:weights': 'Assessment weights must total 100% before you finalize.',
  'validation_failed:reason': 'Enter a reason for this change.',
  'validation_failed:letter': "Choose a letter from the university's grade scale.",
  conflict: 'Grades for this section were already finalized, perhaps in another tab. Close this dialog to see them.',
  forbidden: "Only this section's instructors and administrators can finalize grades.",
}
const needReason = 'Enter a reason for this change.'

/** Select values: 'letter:B+', 'status:incomplete', or '' (computed, finalize dialog only). */
const toGrade = (v: string) =>
  v.startsWith('letter:')
    ? { letter: v.slice(7) }
    : { status: v.slice(7) as 'incomplete' | 'withdrawn' }

function GradeOptions({ scale }: { scale: GradeScale }) {
  return (
    <>
      {scale.rows.map((r) => (
        <option key={r.letter} value={`letter:${r.letter}`}>
          {r.letter}
        </option>
      ))}
      <option value="status:incomplete">Incomplete</option>
      <option value="status:withdrawn">Withdrawn</option>
    </>
  )
}

/** Mount only while open (parent renders it conditionally). */
export function FinalizeDialog(props: {
  slug: string
  sectionId: string
  code: string
  gradebook: Gradebook
  scale: GradeScale
  onClose: () => void
  onDone: (message: string) => void
}) {
  const { slug, sectionId, code, gradebook, scale, onClose, onDone } = props
  const { assessments, students } = gradebook
  const refresh = useGradesRefresh(slug)
  const [picks, setPicks] = useState<Record<string, { value: string; reason: string }>>({})
  const [local, setLocal] = useState<Record<string, string>>({})
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)

  const m = useMutation({
    mutationFn: (overrides: { userId: string; reason: string }[]) =>
      unwrap(
        api.POST('/universities/{slug}/sections/{id}/final-grades', {
          params: { path: { slug, id: sectionId } },
          body: overrides.length > 0 ? { overrides: overrides as never } : {},
        }),
      ),
    onSuccess: async () => {
      await refresh()
      onDone(`Final grades recorded for ${students.length} students.`)
    },
  })

  const rows = students.map((s) => {
    const { percent, missing } = finalPercent(assessments, s.scores)
    return { s, percent, missing, letter: letterFor(scale.rows, percent)?.letter ?? '—' }
  })
  const withMissing = rows.filter((r) => r.missing > 0).length
  const tally = scale.rows
    .map((r) => [r.letter, rows.filter((x) => x.letter === r.letter).length] as const)
    .filter(([, n]) => n > 0)

  // server errors: overrides[i].field -> row; anything else -> the banner
  const fieldCopy = useFieldErrors(m.error, finalizeCopy)
  const serverRows: Record<string, string> = {}
  const bannerMessages: string[] = []
  if (m.error instanceof ApiError && m.error.details.length > 0)
    for (const d of m.error.details) {
      const match = /^overrides\[(\d+)\]\.(\w+)$/.exec(d.field)
      const userId = match && m.variables?.[Number(match[1])]?.userId
      if (userId) serverRows[userId] ||= finalizeCopy[`validation_failed:${match[2]}` as keyof typeof finalizeCopy] ?? sentence(d.message)
      else bannerMessages.push(fieldCopy[d.field] ?? sentence(d.message))
    }
  else if (m.error) bannerMessages.push(errorMessage(m.error, finalizeCopy))
  const rowErrors = { ...serverRows, ...local }
  const conflict = m.error instanceof ApiError && m.error.code === 'conflict'
  const hasLocal = Object.keys(local).length > 0
  useFocusFirstError(formRef, hasLocal ? local : m.error, bannerRef)

  const close = () => {
    if (conflict) void refresh()
    onClose()
  }

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const chosen = students.filter((s) => picks[s.userId]?.value)
    const errs: Record<string, string> = {}
    for (const s of chosen) if (!picks[s.userId].reason.trim()) errs[s.userId] = needReason
    setLocal(errs)
    if (Object.keys(errs).length > 0) return
    m.mutate(chosen.map((s) => ({ userId: s.userId, ...toGrade(picks[s.userId].value), reason: picks[s.userId].reason.trim() })))
  }

  const setPick = (s: Student, patch: Partial<{ value: string; reason: string }>) =>
    setPicks((p) => ({ ...p, [s.userId]: { ...{ value: '', reason: '' }, ...p[s.userId], ...patch } }))

  return (
    <Dialog
      open
      onClose={close}
      wide
      title={`Finalize grades for ${code}`}
      description="Every enrolled student gets a final grade from the current scores. Missing scores count as 0 and every assessment is published to students."
    >
      <form ref={formRef} noValidate onSubmit={submit} className="flex flex-col gap-4">
        {bannerMessages.length > 0 && (
          <ErrorBanner ref={bannerRef}>
            {bannerMessages.map((t) => (
              <p key={t}>{t}</p>
            ))}
          </ErrorBanner>
        )}
        <p className="text-sm">
          {students.length} students · {withMissing} with missing scores · Preview:{' '}
          {tally.map(([l, n]) => `${l} ${n}`).join(' · ') || '—'}
        </p>
        <div className="max-h-[55vh] overflow-y-auto">
          <Table caption="Preview of final grades">
            <thead>
              <tr>
                <th scope="col">Student</th>
                <th scope="col" className="text-right">
                  Percent
                </th>
                <th scope="col">Letter</th>
                <th scope="col">Grade</th>
                <th scope="col">Reason</th>
              </tr>
            </thead>
            <tbody>
              {rows.map(({ s, percent, missing, letter }) => {
                const pick = picks[s.userId]
                const err = rowErrors[s.userId]
                const name = personName(s)
                return (
                  <tr key={s.userId}>
                    <th scope="row" className="font-bold">
                      {name}
                    </th>
                    <td className="text-right">
                      {fmt2(percent)}%
                      {missing > 0 && <span className="text-ink-muted"> ({missing} missing)</span>}
                    </td>
                    <td>{letter}</td>
                    <td>
                      <select
                        aria-label={`Grade for ${name}`}
                        className={inputClass}
                        value={pick?.value ?? ''}
                        onChange={(e) => setPick(s, { value: e.target.value })}
                      >
                        <option value="">Computed: {letter}</option>
                        <GradeOptions scale={scale} />
                      </select>
                    </td>
                    <td>
                      {pick?.value && (
                        <>
                          <input
                            type="text"
                            aria-label={`Reason for ${name}`}
                            className={inputClass}
                            value={pick.reason}
                            aria-invalid={err ? true : undefined}
                            aria-describedby={err ? `reason-err-${s.userId}` : undefined}
                            onChange={(e) => setPick(s, { reason: e.target.value })}
                          />
                          {err && (
                            <p id={`reason-err-${s.userId}`} role="alert" className="text-sm text-danger">
                              <span className="sr-only">Error: </span>
                              {err}
                            </p>
                          )}
                        </>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </Table>
        </div>
        <p className="text-xs text-ink-muted">Preview only. The recorded grades come from the server.</p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending || conflict}>
            {m.isPending ? 'Finalizing…' : `Finalize ${students.length} grades`}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}

/** Mount only while open. Change one student's final grade after finalizing. */
export function OverrideDialog(props: {
  slug: string
  sectionId: string
  student: Student
  scale: GradeScale
  onClose: () => void
  onDone: (message: string) => void
}) {
  const { slug, sectionId, student, scale, onClose, onDone } = props
  const f = student.final
  const refresh = useGradesRefresh(slug)
  const [value, setValue] = useState(f?.status === 'graded' ? `letter:${f.letter}` : `status:${f?.status ?? 'incomplete'}`)
  const [reason, setReason] = useState('')
  const [local, setLocal] = useState<Record<string, string>>({})
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const name = personName(student)

  const m = useMutation({
    mutationFn: (body: { reason: string }) =>
      unwrap(
        api.PATCH('/universities/{slug}/sections/{id}/final-grades/{userId}', {
          params: { path: { slug, id: sectionId, userId: student.userId } },
          body: body as never,
        }),
      ),
    onSuccess: async () => {
      await refresh()
      const label = value.startsWith('letter:') ? value.slice(7) : value === 'status:incomplete' ? 'Incomplete' : 'Withdrawn'
      onDone(`Final grade for ${name} changed to ${label}.`)
    },
  })
  const server = useFieldErrors(m.error, finalizeCopy)
  const fields = { ...server, ...local }
  const hasLocal = Object.keys(local).length > 0
  useFocusFirstError(formRef, hasLocal ? local : m.error, bannerRef)

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const r = reason.trim()
    setLocal(r ? {} : { reason: needReason })
    if (r) m.mutate({ ...toGrade(value), reason: r })
  }

  return (
    <Dialog open onClose={onClose} title={`Change final grade for ${name}`}>
      <form ref={formRef} noValidate onSubmit={submit} className="flex flex-col gap-4">
        {!fields.letter && !fields.reason && <FormErrors ref={bannerRef} error={m.error} copy={finalizeCopy} />}
        <FormField label="Grade" name="letter" error={fields.letter}>
          <select className={inputClass} value={value} onChange={(e) => setValue(e.target.value)}>
            <GradeOptions scale={scale} />
          </select>
        </FormField>
        <FormField label="Reason" name="reason" error={fields.reason}>
          <input type="text" className={inputClass} value={reason} onChange={(e) => setReason(e.target.value)} />
        </FormField>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? 'Saving…' : 'Change grade'}
          </Button>
        </div>
      </form>
    </Dialog>
  )
}
