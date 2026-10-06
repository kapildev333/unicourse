import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { ApiError } from '../../api/errors'
import { Button } from '../../components/button'
import { ErrorBanner } from '../../components/error-banner'
import { FormErrors, errorMessage, useFieldErrors, useFocusFirstError } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
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
const providerCopy = { 'validation_failed:slug': identifierCopy, 'validation_failed:email': 'Enter a valid email address.' }
const magicCopy = { 'validation_failed:email': 'Enter a valid email address.' }

type Provider = 'google' | 'microsoft'

export function SignInPage(props: { next?: string; error?: string; email?: string }) {
  const next = safeNext(props.next)
  const [identifier, setIdentifier] = useState(props.email ?? '')
  const [email, setEmail] = useState(props.email ?? '')
  const [pending, setPending] = useState<Provider | 'magic' | null>(null)
  const [provError, setProvError] = useState<unknown>(null)
  const [provValue, setProvValue] = useState('')
  const [clientError, setClientError] = useState<{ identifier: string } | null>(null)
  const [magicError, setMagicError] = useState<unknown>(null)
  const [sentTo, setSentTo] = useState<string | null>(null)
  const formA = useRef<HTMLFormElement>(null)
  const formB = useRef<HTMLFormElement>(null)
  const provBannerRef = useRef<HTMLDivElement>(null)
  const magicBannerRef = useRef<HTMLDivElement>(null)

  const provFields = useFieldErrors(provError, providerCopy)
  const magicFields = useFieldErrors(magicError, magicCopy)
  const identifierError = clientError?.identifier ?? provFields.slug ?? provFields.email
  const emailError = magicFields.email
  useFocusFirstError(formA, clientError ?? provError, provBannerRef)
  useFocusFirstError(formB, magicError, magicBannerRef)

  async function start(provider: Provider) {
    const v = identifier.trim()
    setProvError(null)
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
      window.location.assign(url)
    } catch (e) {
      setProvError(e)
      setPending(null)
    }
  }

  async function sendLink(e: React.FormEvent) {
    e.preventDefault()
    const v = email.trim()
    setMagicError(null)
    setPending('magic')
    try {
      await unwrap(api.POST('/auth/magic-link', { body: { email: v, next } }))
      setSentTo(v)
    } catch (err) {
      setMagicError(err)
    } finally {
      setPending(null)
    }
  }

  const callback = props.error ? (callbackCopy[props.error] ?? fallbackCopy) : null
  const busy = pending !== null
  const provBanner =
    provError && !identifierError ? (
      <div className="mb-4">
        <ErrorBanner ref={provBannerRef} title="">
          {provError instanceof ApiError && provError.status === 404
            ? `We couldn't find a university for “${provValue}”, or this sign-in option isn't available for it. Check the spelling or use an email link below.`
            : errorMessage(provError, providerCopy)}
        </ErrorBanner>
      </div>
    ) : null

  return (
    <main id="main" tabIndex={-1} className="flex min-h-screen items-start justify-center bg-canvas px-4 pt-24">
      <div className="w-full max-w-sm rounded-lg border border-line bg-surface p-8 shadow-sm">
        {sentTo !== null ? (
          <MagicLinkSent email={sentTo} onBack={() => setSentTo(null)} />
        ) : (
          <>
            <p className="text-sm font-semibold text-accent">UniCourse</p>
            <h1 className="mt-6 text-xl font-semibold tracking-tight">Sign in</h1>
            <p className="mt-1 text-sm text-ink-muted">Use your university account.</p>
            {callback && (
              <div className="mt-4">
                <ErrorBanner title="">{callback}</ErrorBanner>
              </div>
            )}
            {props.email && (
              <p className="mt-4 text-sm text-ink-muted">
                You've been invited. Sign in as {props.email} to accept.
              </p>
            )}
            <form
              ref={formA}
              noValidate
              aria-label="Sign in with your university account"
              className="mt-6 flex flex-col gap-4"
              onSubmit={(e) => e.preventDefault()}
            >
              {provBanner}
              <FormField
                label="University email or university ID"
                name="identifier"
                hint="For example alice@example.edu or example"
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
              {(['google', 'microsoft'] as const).map((p) => (
                <Button key={p} variant="secondary" className="w-full" disabled={busy} onClick={() => void start(p)}>
                  {pending === p ? 'Redirecting…' : p === 'google' ? 'Continue with Google' : 'Continue with Microsoft'}
                </Button>
              ))}
            </form>
            <div className="my-6 flex items-center gap-3 text-xs text-ink-muted">
              <span aria-hidden="true" className="h-px flex-1 bg-line" />
              or
              <span aria-hidden="true" className="h-px flex-1 bg-line" />
            </div>
            <form
              ref={formB}
              noValidate
              aria-label="Email me a sign-in link"
              className="flex flex-col gap-4"
              onSubmit={(e) => void sendLink(e)}
            >
              {magicError && !emailError ? <FormErrors ref={magicBannerRef} error={magicError} copy={magicCopy} /> : null}
              <FormField label="Email" name="email" error={emailError}>
                <input
                  type="email"
                  autoComplete="email"
                  className={inputClass}
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </FormField>
              <Button type="submit" className="w-full" disabled={busy}>
                {pending === 'magic' ? 'Sending…' : 'Email me a sign-in link'}
              </Button>
            </form>
          </>
        )}
      </div>
    </main>
  )
}
