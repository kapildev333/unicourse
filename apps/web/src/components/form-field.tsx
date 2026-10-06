import { cloneElement, useId } from 'react'

export const inputClass =
  'h-9 w-full rounded-md border border-line-strong bg-surface px-3 text-sm text-ink placeholder:text-ink-muted focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus aria-[invalid=true]:border-danger'

export function FormField(props: {
  label: string
  name: string
  error?: string
  hint?: string
  children: React.ReactElement
}) {
  const { label, name, error, hint, children } = props
  const id = useId()
  const errorId = `${id}-error`
  const hintId = `${id}-hint`
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-medium text-ink">
        {label}
      </label>
      {cloneElement(children, {
        id,
        name,
        'aria-invalid': error ? true : undefined,
        'aria-describedby':
          [children.props['aria-describedby'], hint && hintId, error && errorId].filter(Boolean).join(' ') || undefined,
      } as React.HTMLAttributes<HTMLElement>)}
      {hint && (
        <p id={hintId} className="text-xs text-ink-muted">
          {hint}
        </p>
      )}
      {error && (
        <p id={errorId} role="alert" className="text-sm text-danger">
          <span className="sr-only">Error: </span>
          {error}
        </p>
      )}
    </div>
  )
}
