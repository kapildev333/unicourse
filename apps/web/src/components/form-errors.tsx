import { useEffect, useMemo } from 'react'
import { ApiError } from '../api/errors'
import { ErrorBanner } from './error-banner'

/** Keys are '<code>' (banner) or '<code>:<field>' (field), e.g. 'conflict:code'. */
export type FieldCopy = Record<string, string>

const defaults: Record<string, string> = {
  forbidden: "You don't have permission to do this.",
  not_found: 'This item no longer exists. Reload the page.',
  internal: 'Something went wrong on our side. Try again in a moment.',
}

const sentence = (s: string) => {
  const t = s.trim()
  if (!t) return t
  const c = t[0].toUpperCase() + t.slice(1)
  return /[.!?]$/.test(c) ? c : c + '.'
}

export function errorMessage(error: unknown, copy: FieldCopy = {}): string {
  if (!(error instanceof ApiError)) return 'Something went wrong. Check your connection and try again.'
  const base =
    copy[error.code] ??
    (error.code === 'validation_failed' && error.details.length > 0 ? 'Please fix the highlighted fields.' : undefined) ??
    defaults[error.code] ??
    error.message
  return error.requestId ? `${base} Reference: ${error.requestId}.` : base
}

export function useFieldErrors(error: unknown, copy: FieldCopy = {}): Record<string, string> {
  return useMemo(() => {
    const out: Record<string, string> = {}
    if (!(error instanceof ApiError)) return out
    if (error.details.length === 0) {
      const prefix = `${error.code}:`
      for (const key of Object.keys(copy)) {
        const field = key.slice(prefix.length)
        if (key.startsWith(prefix) && field && !(field in out)) out[field] = copy[key]
      }
      return out
    }
    for (const d of error.details) {
      if (!d.field || d.field in out) continue
      out[d.field] = copy[`${error.code}:${d.field}`] ?? sentence(d.message)
    }
    return out
  }, [error, copy])
}

export function FormErrors({ error, copy }: { error: unknown; copy?: FieldCopy }) {
  if (!error) return null
  const general = error instanceof ApiError ? error.details.filter((d) => d.field === '' || d.field === 'body') : []
  return (
    <ErrorBanner>
      {errorMessage(error, copy)}
      {general.length > 0 && (
        <ul className="mt-1 list-disc pl-5">
          {general.map((d, i) => (
            <li key={i}>{sentence(d.message)}</li>
          ))}
        </ul>
      )}
    </ErrorBanner>
  )
}

export function useFocusFirstError(
  formRef: React.RefObject<HTMLElement | null>,
  error: unknown,
  bannerRef?: React.RefObject<HTMLElement | null>,
) {
  useEffect(() => {
    if (!error) return
    const el = formRef.current?.querySelector<HTMLElement>('[aria-invalid="true"]')
    ;(el ?? bannerRef?.current)?.focus()
  }, [error, formRef, bannerRef])
}
