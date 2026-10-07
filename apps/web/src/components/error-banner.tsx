import { CircleAlert } from 'lucide-react'
import { forwardRef } from 'react'
import { Button } from './button'

export const ErrorBanner = forwardRef<
  HTMLDivElement,
  {
    title?: string
    children: React.ReactNode
    onRetry?: () => void
    id?: string
  }
>(function ErrorBanner({ title = 'Something went wrong', children, onRetry, id }, ref) {
  return (
    <div
      ref={ref}
      id={id}
      tabIndex={-1}
      role="alert"
      className="flex gap-3 rounded border border-danger/30 bg-danger-soft px-4 py-3 text-sm text-danger"
    >
      <CircleAlert className="mt-0.5 shrink-0" />
      <div className="min-w-0 flex-1">
        {title && (
          <p className="font-bold">
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
    </div>
  )
})
