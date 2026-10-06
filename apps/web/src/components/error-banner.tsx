import { forwardRef } from 'react'
import { Button } from './button'

export const ErrorBanner = forwardRef<
  HTMLDivElement,
  { title?: string; children: React.ReactNode; onRetry?: () => void; id?: string }
>(function ErrorBanner({ title = 'Something went wrong', children, onRetry, id }, ref) {
  return (
    <div
      ref={ref}
      id={id}
      tabIndex={-1}
      role="alert"
      className="rounded-lg border border-danger/30 bg-danger-soft px-4 py-3 text-sm text-danger"
    >
      {title && (
        <p className="font-semibold">
          <span className="sr-only">Error: </span>
          {title}
        </p>
      )}
      <div>{children}</div>
      {onRetry && (
        <Button variant="secondary" size="sm" className="mt-2" onClick={onRetry}>
          Try again
        </Button>
      )}
    </div>
  )
})
