import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { ApiError } from '../../api/errors'
import { Mail, MailOpen } from 'lucide-react'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { EntryLayout } from '../../components/entry-layout'
import { ErrorBanner } from '../../components/error-banner'
import { FormErrors, errorMessage, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { BrandMark } from './brand-mark'
import { MagicLinkSent } from './magic-link-sent'
import { safeNext } from './safe-next'

const callbackCopy: Record<string, string> = {
  unauthenticated: "Sign-in didn't complete. Please try again.",
  forbidden: "This email can't sign in to UniCourse. Ask your university administrator to invite you.",
  validation_failed: 'That sign-in link has expired or was already used. Request a new one below.',
  rate_limited: 'Too many sign-in attempts. Wait a minute, then try again.',
  no_membership:
    "You're signed in, but your account isn't a member of any university yet. Ask your administrator for an invitation.",
}
const fallbackCopy = 'Something went wrong while signing in. Please try again.'

const identifierCopy = 'Enter your university email or university ID.'
const magicEmailCopy = 'Enter your email to get a sign-in link'
const providerCopy = {
  'validation_failed:slug': identifierCopy,
  'validation_failed:email': 'Enter a valid email address.',
}
const magicCopy = { 'validation_failed:email': 'Enter a valid email address.' }

type Provider = 'google' | 'microsoft'
type Method = Provider | 'magic'

const lastKey = 'unicourse:last-sign-in'
const readLast = (): Method | null => {
  try {
    const v = localStorage.getItem(lastKey)
    return v === 'google' || v === 'microsoft' || v === 'magic' ? v : null
  } catch {
    return null
  }
}
const writeLast = (m: Method) => {
  try {
    localStorage.setItem(lastKey, m)
  } catch {
    // storage blocked: the badge is a convenience only
  }
}
const providerLabel = { google: 'Continue with Google', microsoft: 'Continue with Microsoft' }

export function SignInPage(props: { next?: string; error?: string; email?: string }) {
  const next = safeNext(props.next)
  const [identifier, setIdentifier] = useState(props.email ?? '')
  const [pending, setPending] = useState<Method | null>(null)
  const [provError, setProvError] = useState<unknown>(null)
  const [provValue, setProvValue] = useState('')
  const [clientError, setClientError] = useState<{ identifier: string } | null>(null)
  const [magicError, setMagicError] = useState<unknown>(null)
  const [sentTo, setSentTo] = useState<string | null>(null)
  const [last] = useState(readLast)
  const form = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)

  const provFields = useFieldErrors(provError, providerCopy)
  const magicFields = useFieldErrors(magicError, magicCopy)
  const identifierError = clientError?.identifier ?? provFields.slug ?? provFields.email ?? magicFields.email
  useFocusFirstError(form, clientError ?? provError ?? magicError, bannerRef)

  async function start(provider: Provider) {
    const v = identifier.trim()
    setProvError(null)
    setMagicError(null)
    if (!v) {
      setClientError({ identifier: identifierCopy })
      return
    }
    setClientError(null)
    setProvValue(v)
    setPending(provider)
    try {
      const body = v.includes('@') ? { email: v, next } : { slug: v.toLowerCase(), next }
      const { url } = await unwrap(api.POST('/auth/oidc/{provider}/start', { params: { path: { provider } }, body }))
      writeLast(provider)
      window.location.assign(url)
    } catch (e) {
      setProvError(e)
      setPending(null)
    }
  }

  const post = (email: string) => unwrap(api.POST('/auth/magic-link', { body: { email, next } }))

  async function sendLink(e: React.FormEvent) {
    e.preventDefault()
    const v = identifier.trim()
    setProvError(null)
    setMagicError(null)
    if (!v.includes('@')) {
      setClientError({ identifier: magicEmailCopy })
      return
    }
    setClientError(null)
    setPending('magic')
    try {
      await post(v)
      writeLast('magic')
      setSentTo(v)
    } catch (err) {
      setMagicError(err)
    } finally {
      setPending(null)
    }
  }

  const callback = props.error ? (callbackCopy[props.error] ?? fallbackCopy) : null
  const busy = pending !== null
  const lastBadge = (m: Method) =>
    last === m ? (
      <Badge tone="neutral" className="ml-1">
        Last used
      </Badge>
    ) : null
  const banner =
    provError && !identifierError ? (
      <ErrorBanner ref={bannerRef} title="">
        {provError instanceof ApiError && provError.status === 404
          ? `We couldn't find a university for “${provValue}”, or this sign-in option isn't available for it. Check the spelling or use an email link below.`
          : errorMessage(provError, providerCopy)}
      </ErrorBanner>
    ) : magicError && !identifierError ? (
      <FormErrors ref={bannerRef} error={magicError} copy={magicCopy} />
    ) : null

  return (
    <EntryLayout>
      {sentTo !== null ? (
        <MagicLinkSent email={sentTo} onBack={() => setSentTo(null)} onResend={() => post(sentTo)} />
      ) : (
        <>
          <h1 className="text-2xl text-ink">Sign in</h1>
          <p className="mt-1 text-sm text-ink-muted">Use your university account.</p>
          {callback && (
            <div className="mt-4">
              <ErrorBanner title="">{callback}</ErrorBanner>
            </div>
          )}
          {props.email && (
            <p className="mt-4 flex items-start gap-2 text-sm text-ink">
              <MailOpen className="mt-0.5 shrink-0" aria-hidden="true" />
              <span>
                You've been invited. Sign in as <span className="whitespace-nowrap">{props.email}</span> to accept.
              </span>
            </p>
          )}
          <form
            ref={form}
            noValidate
            aria-label="Sign in"
            className="mt-6 flex flex-col gap-4"
            onSubmit={(e) => void sendLink(e)}
          >
            {banner}
            <FormField
              label="University email or university ID"
              name="identifier"
              hint="For example alice@example.edu, or your university's short name such as northfield"
              error={identifierError}
            >
              <input
                type="text"
                autoComplete="email"
                inputMode="email"
                autoCapitalize="none"
                spellCheck={false}
                className={inputClass}
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
              />
            </FormField>
            <Button type="submit" size="lg" className="w-full" disabled={busy}>
              <Mail size={16} aria-hidden="true" />
              {pending === 'magic' ? 'Sending…' : 'Email me a sign-in link'}
              {lastBadge('magic')}
            </Button>
            {(['google', 'microsoft'] as const).map((p) => (
              <Button
                key={p}
                variant="secondary"
                size="lg"
                className="w-full"
                disabled={busy}
                onClick={() => void start(p)}
              >
                <BrandMark provider={p} />
                {pending === p ? 'Redirecting…' : providerLabel[p]}
                {lastBadge(p)}
              </Button>
            ))}
          </form>
        </>
      )}
    </EntryLayout>
  )
}
