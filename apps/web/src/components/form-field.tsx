import { cloneElement } from 'react'

export function FormField(props: { label: string; name: string; error?: string; children: React.ReactElement }) {
  const { label, name, error, children } = props
  const errorId = `${name}-error`
  return (
    <div className="mb-3 flex flex-col gap-1">
      <label htmlFor={name} className="text-sm font-medium">
        {label}
      </label>
      {cloneElement(children, {
        id: name,
        name,
        'aria-invalid': error ? true : undefined,
        'aria-describedby': error ? errorId : undefined,
      } as React.HTMLAttributes<HTMLElement>)}
      {error && (
        <p id={errorId} role="alert" className="text-sm text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}
