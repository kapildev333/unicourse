import { useEffect, useRef } from 'react'
import { Button } from '../../components/button'

export function MagicLinkSent({ email, onBack }: { email: string; onBack: () => void }) {
  const heading = useRef<HTMLHeadingElement>(null)
  useEffect(() => heading.current?.focus(), [])
  return (
    <div>
      <h1 ref={heading} tabIndex={-1} className="text-2xl text-ink">
        Check your email
      </h1>
      <p className="mt-2 text-sm">
        If {email} can sign in to UniCourse, we've sent it a link. The link works once and expires in 15 minutes.
      </p>
      <p className="mt-4 text-sm text-ink-muted">
        Didn't get it? Check spam, or{' '}
        <Button variant="ghost" size="sm" onClick={onBack}>
          use a different email
        </Button>
      </p>
    </div>
  )
}
