import { useDeferredValue, useId, useState } from 'react'
import { FormField, inputClass } from '../../components/form-field'
import { useAllDepartments } from './queries'

export function DepartmentPicker(props: {
  slug: string
  value: string[]
  onChange: (ids: string[]) => void
  error?: string
}) {
  const { slug, value, onChange, error } = props
  const errId = useId()
  const q = useAllDepartments(slug)
  const [search, setSearch] = useState('')
  const deferred = useDeferredValue(search.trim())
  const found = useAllDepartments(slug, deferred, !!deferred)
  const searching = search.trim() !== ''
  const shownQuery = searching ? found : q
  const items = shownQuery.data?.items ?? []
  return (
    <fieldset aria-describedby={error ? errId : undefined}>
      <legend className="text-sm font-medium">Departments</legend>
      {(q.data?.nextCursor || searching) && (
        <div className="mt-1.5">
          <FormField
            label="Search departments"
            name="departmentSearch"
            hint={
              searching && found.data?.nextCursor
                ? 'Showing the first 200 matches. Type more to narrow them.'
                : undefined
            }
          >
            <input
              type="search"
              className={inputClass}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && e.preventDefault()}
            />
          </FormField>
        </div>
      )}
      <div className="mt-1.5 max-h-48 overflow-y-auto rounded border border-line-strong p-2">
        {shownQuery.isPending ? (
          <p role="status" className="text-sm text-ink-muted">
            Loading departments…
          </p>
        ) : shownQuery.error ? (
          <p role="alert" className="text-sm text-danger">
            Couldn't load departments. Close this dialog and try again.
          </p>
        ) : items.length === 0 ? (
          <p className="text-sm text-ink-muted">
            {searching ? 'No departments match.' : 'No departments yet. Create one under Departments.'}
          </p>
        ) : (
          items.map((d) => (
            <label key={d.id} className="flex items-center gap-2 py-1 text-sm">
              <input
                type="checkbox"
                className="size-4 accent-accent"
                aria-invalid={error ? true : undefined}
                checked={value.includes(d.id)}
                onChange={(e) => onChange(e.target.checked ? [...value, d.id] : value.filter((x) => x !== d.id))}
              />
              {d.name}
              <span className="text-xs text-ink-muted">{d.code}</span>
            </label>
          ))
        )}
      </div>
      {error && (
        <p id={errId} role="alert" className="mt-1.5 text-sm text-danger">
          <span className="sr-only">Error: </span>
          {error}
        </p>
      )}
    </fieldset>
  )
}
