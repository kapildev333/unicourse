import type { components } from '../../api/types'
import { FormField, inputClass } from '../../components/form-field'
import { formatDate } from '../../lib/format'
import { defaultTermId, todayIso, useAllTerms } from '../courses/queries'

type Term = components['schemas']['Term']

/** All terms (newest first), the chosen term (unknown or missing id falls back to the default), and whether to show the not-in-session note. */
export function useTermChoice(slug: string, termId?: string) {
  const query = useAllTerms(slug)
  const terms: Term[] = [...(query.data?.items ?? [])].sort((a, b) => b.startsOn.localeCompare(a.startsOn))
  const today = todayIso()
  const term = terms.find((t) => t.id === termId) ?? terms.find((t) => t.id === defaultTermId(terms, today))
  const notInSession = !!term && term.id !== termId && !(term.startsOn <= today && today <= term.endsOn)
  return { query, terms, term, notInSession }
}

export function TermPicker({ terms, value, onChange }: { terms: Term[]; value?: string; onChange: (id: string) => void }) {
  return (
    <FormField label="Term" name="termId">
      <select className={`${inputClass} w-64`} value={value ?? ''} onChange={(e) => onChange(e.target.value)}>
        {terms.map((t) => (
          <option key={t.id} value={t.id}>
            {t.name} ({formatDate(t.startsOn)} – {formatDate(t.endsOn)})
          </option>
        ))}
      </select>
    </FormField>
  )
}
