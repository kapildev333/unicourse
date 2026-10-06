import { useId } from 'react'
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
  const items = q.data?.items ?? []
  return (
    <fieldset aria-describedby={error ? errId : undefined}>
      <legend className="text-sm font-medium">Departments</legend>
      <div className="mt-1.5 max-h-48 overflow-y-auto rounded-md border border-line-strong p-2">
        {q.isPending ? (
          <p role="status" className="text-sm text-ink-muted">
            Loading departments…
          </p>
        ) : q.error ? (
          <p role="alert" className="text-sm text-danger">
            Couldn't load departments. Close this dialog and try again.
          </p>
        ) : items.length === 0 ? (
          <p className="text-sm text-ink-muted">No departments yet. Create one under Departments.</p>
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
