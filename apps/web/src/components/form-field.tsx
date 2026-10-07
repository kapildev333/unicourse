import { cloneElement, useId } from 'react'

export const inputSizedClass =
  'h-10 rounded border border-line-strong bg-surface px-3 text-sm text-ink placeholder:text-ink-muted transition-colors hover:not-aria-invalid:border-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus disabled:cursor-not-allowed disabled:opacity-60 aria-invalid:border-danger'
export const inputClass = inputSizedClass.replace('h-10 ', 'h-10 w-full ')
export const textareaClass = inputClass.replace('h-10 ', 'min-h-32 resize-y py-2 ')
export const fileInputClass =
  'text-sm text-ink file:mr-3 file:h-10 file:rounded file:border file:border-ink file:bg-surface file:px-4 file:text-sm file:font-bold file:text-ink hover:file:bg-line focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus disabled:opacity-60'

export function FormField(props: {
  label: string
  name: string
  error?: string
  hint?: string
  children: React.ReactElement<React.HTMLAttributes<HTMLElement>>
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
