import { useMutation } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { FormErrors, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass, inputSizedClass, textareaClass } from '../../components/form-field'
import { fmtNum, parseDecimal } from '../grades/grade'
import { countHint } from '../section/plain-text'
import { patchAssessment, useAssessmentRefresh, useAssessments, type Assessment } from './queries'
import { fromLocalInput, timeZone, toLocalInput } from './window'

type Kind = 'quiz' | 'assignment'

const copy = {
  'validation_failed:name': 'Enter a title of at most 100 characters.',
  'conflict:name': 'This section already has an assessment with this title.',
  'validation_failed:description': 'Use at most 10,000 characters of plain text.',
  'validation_failed:opensAt': 'Enter the date and time it opens.',
  'validation_failed:closesAt': 'Enter a time after Opens.',
  'validation_failed:maxPoints': 'Enter maximum points greater than 0.',
  'validation_failed:weight': 'Enter a weight from 0 to 100.',
  forbidden: "Only this section's staff and administrators can change quizzes and assignments.",
}

/** Next full hour; closes an hour later (quiz) or a week later at 23:59 (assignment). */
function defaults(kind: Kind) {
  const opens = new Date()
  opens.setMinutes(0, 0, 0)
  opens.setHours(opens.getHours() + 1)
  const closes = new Date(opens)
  if (kind === 'quiz') closes.setHours(closes.getHours() + 1)
  else {
    closes.setDate(closes.getDate() + 7)
    closes.setHours(23, 59, 0, 0)
  }
  return { opens: toLocalInput(opens.toISOString()), closes: toLocalInput(closes.toISOString()) }
}

/** One dialog for quizzes and assignments. Mount only while open so the form starts fresh each time. */
export function WindowDialog(props: {
  slug: string
  sectionId: string
  kind: Kind
  assessment?: Assessment
  onClose: () => void
  onSaved: (message: string) => void
}) {
  const { slug, sectionId, kind, assessment, onClose, onSaved } = props
  const refresh = useAssessmentRefresh(slug, sectionId)
  const all = useAssessments(slug, sectionId).data
  const otherWeight = (all?.items ?? []).filter((a) => a.id !== assessment?.id).reduce((s, a) => s + a.weight, 0)
  const d = assessment ? undefined : defaults(kind)
  const [name, setName] = useState(assessment?.name ?? '')
  const [description, setDescription] = useState(assessment?.description ?? '')
  const [opens, setOpens] = useState(assessment?.opensAt ? toLocalInput(assessment.opensAt) : (d?.opens ?? ''))
  const [closes, setCloses] = useState(assessment?.closesAt ? toLocalInput(assessment.closesAt) : (d?.closes ?? ''))
  const [maxPoints, setMax] = useState(assessment ? fmtNum(assessment.maxPoints) : '')
  const [weight, setWeight] = useState(assessment ? fmtNum(assessment.weight) : '')
  const [local, setLocal] = useState<Record<string, string>>({})
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const closesLabel = kind === 'quiz' ? 'Closes' : 'Due'

  const m = useMutation({
    mutationFn: (body: components['schemas']['AssessmentCreate'] | components['schemas']['AssessmentUpdate']) =>
      assessment
        ? patchAssessment(slug, sectionId, assessment.id, body)
        : unwrap(
            api.POST('/universities/{slug}/sections/{id}/assessments', {
              params: { path: { slug, id: sectionId } },
              body: body as components['schemas']['AssessmentCreate'],
            }),
          ),
    onSuccess: async (a) => {
      await refresh()
      onSaved(`${a.name} ${assessment ? 'saved' : 'created'}.`)
    },
  })
  const server = useFieldErrors(m.error, copy)
  const fields: Record<string, string> = { ...server, ...local }
  const hasLocal = Object.keys(local).length > 0
  useFocusFirstError(formRef, hasLocal ? local : m.error, bannerRef)

  function submit(e: React.FormEvent) {
    e.preventDefault()
    const errs: Record<string, string> = {}
    const n = name.trim()
    const o = fromLocalInput(opens)
    const c = fromLocalInput(closes)
    const max = kind === 'assignment' ? parseDecimal(maxPoints) : undefined
    const w = parseDecimal(weight)
    if (!n || [...n].length > 100) errs.name = copy['validation_failed:name']
    if (!o) errs.opensAt = copy['validation_failed:opensAt']
    if (!c) errs.closesAt = `Enter the date and time it ${kind === 'quiz' ? 'closes' : 'is due'}.`
    else if (o && Date.parse(c) <= Date.parse(o)) errs.closesAt = `${closesLabel} must be after Opens.`
    if (kind === 'assignment' && (max === undefined || max <= 0)) errs.maxPoints = copy['validation_failed:maxPoints']
    if (w === undefined || w > 100) errs.weight = copy['validation_failed:weight']
    setLocal(errs)
    if (Object.keys(errs).length > 0 || !o || !c || w === undefined) return
    if (!assessment)
      return m.mutate({ kind, name: n, description, opensAt: o, closesAt: c, weight: w, ...(max !== undefined && { maxPoints: max }) })
    const body = {
      ...(n !== assessment.name && { name: n }),
      ...(description !== assessment.description && { description }),
      ...(opens !== toLocalInput(assessment.opensAt ?? '') && { opensAt: o }),
      ...(closes !== toLocalInput(assessment.closesAt ?? '') && { closesAt: c }),
      ...(max !== undefined && max !== assessment.maxPoints && { maxPoints: max }),
      ...(w !== assessment.weight && { weight: w }),
    }
    if (Object.keys(body).length === 0) return onClose()
    m.mutate(body)
  }

  const noun = kind === 'quiz' ? 'quiz' : 'assignment'
  return (
    <Dialog open size="lg" onClose={onClose} title={assessment ? `Edit ${assessment.name}` : `New ${noun}`}>
      <form ref={formRef} noValidate onSubmit={submit} className="flex flex-col gap-4">
        {Object.keys(fields).length === 0 && <FormErrors ref={bannerRef} error={m.error} copy={copy} />}
        <FormField label="Title" name="name" error={fields.name}>
          <input type="text" maxLength={100} className={inputClass} value={name} onChange={(e) => setName(e.target.value)} />
        </FormField>
        <FormField
          label="Description"
          name="description"
          error={fields.description}
          hint={`${countHint(description, 10000)} ${kind === 'quiz' ? 'Optional. Students see it before they start.' : 'Optional. Students see it above the submission form.'}`}
        >
          <textarea rows={4} className={textareaClass} value={description} onChange={(e) => setDescription(e.target.value)} />
        </FormField>
        <FormField label="Opens" name="opensAt" error={fields.opensAt} hint={`Your time zone: ${timeZone()}.`}>
          <input type="datetime-local" className={`${inputSizedClass} w-60`} value={opens} onChange={(e) => setOpens(e.target.value)} />
        </FormField>
        <FormField
          label={closesLabel}
          name="closesAt"
          error={fields.closesAt}
          hint={
            kind === 'quiz'
              ? 'Answers not yet submitted are submitted automatically at this time.'
              : 'Students can submit and replace their work until this time.'
          }
        >
          <input type="datetime-local" className={`${inputSizedClass} w-60`} value={closes} onChange={(e) => setCloses(e.target.value)} />
        </FormField>
        {kind === 'assignment' && (
          <FormField
            label="Max points"
            name="maxPoints"
            error={fields.maxPoints}
            hint="Points a full score is worth. Scores may go up to twice this as bonus."
          >
            <input type="text" inputMode="decimal" className={`${inputSizedClass} w-32`} value={maxPoints} onChange={(e) => setMax(e.target.value)} />
          </FormField>
        )}
        <FormField
          label="Weight (%)"
          name="weight"
          error={fields.weight}
          hint={`Share of the section grade. All weights must total 100 to finalize. Other assessments total ${fmtNum(otherWeight)}%.`}
        >
          <input type="text" inputMode="decimal" className={`${inputSizedClass} w-32`} value={weight} onChange={(e) => setWeight(e.target.value)} />
        </FormField>
        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={m.isPending}>
            {m.isPending ? 'Saving…' : assessment ? 'Save changes' : `Create ${noun}`}
          </Button>
        </DialogFooter>
      </form>
    </Dialog>
  )
}
