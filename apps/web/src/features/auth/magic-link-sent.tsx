import { useEffect, useRef, useState } from 'react'
import { Button } from '../../components/button'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { StatusMessage } from '../../components/status-message'

const cooldown = 30

export function MagicLinkSent(props: { email: string; onBack: () => void; onResend: () => Promise<unknown> }) {
  const { email, onBack, onResend } = props
  const heading = useRef<HTMLHeadingElement>(null)
  const [wait, setWait] = useState(cooldown)
  const [again, setAgain] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const waiting = wait > 0
  useEffect(() => heading.current?.focus(), [])
  useEffect(() => {
    if (!waiting) return
    const t = setInterval(() => setWait((w) => w - 1), 1000)
    return () => clearInterval(t)
  }, [waiting])

  async function resend() {
    setError(null)
    setAgain(false)
    setWait(cooldown)
    try {
      await onResend()
      setAgain(true)
    } catch (e) {
      setError(e)
    }
  }

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
        <button
          type="button"
          onClick={onBack}
          className="text-accent underline"
        >
          use a different email
        </button>
        .
      </p>
      <div className="mt-4 flex flex-col items-start gap-2">
        {error ? <ErrorBanner title="">{errorMessage(error)}</ErrorBanner> : null}
        <StatusMessage message={again ? 'Sent again.' : ''} />
        <Button variant="secondary" disabled={wait > 0} aria-describedby="resend-wait" onClick={() => void resend()}>
          Send the link again
        </Button>
        <p id="resend-wait" className="text-xs text-ink-muted">
          {wait > 0 ? `You can send it again in ${wait} s.` : ''}
        </p>
      </div>
    </div>
  )
}
