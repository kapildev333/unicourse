import { useDeferredValue, useState } from 'react'
import type { components } from '../../api/types'
import { FormField, inputClass } from '../../components/form-field'
import { defaultTermId, todayIso, useAllTerms } from '../courses/queries'

type Term = components['schemas']['Term']

/**
 * All terms (newest first), the chosen term (unknown or missing id falls back to the default), and whether to show the not-in-session note.
 * A term picked by search may lie beyond the first 200: `pick` remembers it so the URL id still resolves.
 */
export function useTermChoice(slug: string, termId?: string) {
  const query = useAllTerms(slug)
  const [picked, pick] = useState<Term>()
  const terms: Term[] = [...(query.data?.items ?? [])].sort((a, b) => b.startsOn.localeCompare(a.startsOn))
  const today = todayIso()
  const term =
    terms.find((t) => t.id === termId) ??
    (picked && picked.id === termId ? picked : undefined) ??
    terms.find((t) => t.id === defaultTermId(terms, today))
  const notInSession = !!term && term.id !== termId && !(term.startsOn <= today && today <= term.endsOn)
  return { query, terms, term, picked, pick, notInSession }
}

/** `more`: the unfiltered list had a next page, so a search field is shown (and `slug` is searched). */
export function TermPicker(props: {
  slug: string
  terms: Term[]
  more?: boolean
  value?: string
  /** a term picked by search that may not be in `terms` */
  picked?: Term
  onChange: (id: string, term?: Term) => void
  hint?: string
  className?: string
}) {
  const { slug, terms, more, value, picked, onChange, hint, className = 'w-full md:w-64' } = props
  const [search, setSearch] = useState('')
  const deferred = useDeferredValue(search.trim())
  const found = useAllTerms(slug, deferred, !!deferred)
  const searching = search !== ''
  const shown = searching ? found.data?.items ?? [] : terms
  const current = shown.some((t) => t.id === value) ? undefined : [picked, ...terms].find((t) => t?.id === value)
  const options = current ? [current, ...shown] : shown
  const pick = (id: string) => onChange(id, options.find((t) => t.id === id))
  return (
    <>
      {(more || searching) && (
        <FormField
          label="Search terms"
          name="termSearch"
          hint={
            searching && found.data?.nextCursor ? 'Showing the first 200 matches. Type more to narrow them.' : undefined
          }
        >
          <input
            type="search"
            className={`${inputClass} ${className}`}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </FormField>
      )}
      <FormField label="Term" name="termId" hint={hint}>
        <select className={`${inputClass} ${className}`} value={value ?? ''} onChange={(e) => pick(e.target.value)}>
          {options.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </select>
      </FormField>
    </>
  )
}
