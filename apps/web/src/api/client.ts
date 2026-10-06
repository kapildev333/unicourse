import createClient from 'openapi-fetch'
import type { paths } from './types'
import { ApiError, type ErrorCode } from './errors'

export const api = createClient<paths>({
  baseUrl: `${window.location.origin}/api/v1`,
  credentials: 'include',
  fetch: (r) => fetch(r), // late-bound so tests can stub global fetch
})

const codeByStatus: Record<number, ErrorCode> = {
  400: 'validation_failed',
  401: 'unauthenticated',
  403: 'forbidden',
  404: 'not_found',
  409: 'conflict',
  429: 'rate_limited',
}

type Envelope = { error?: { code?: ErrorCode; message?: string; details?: { field: string; message: string }[] } }

export async function unwrap<T>(
  p: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T> {
  const { data, error, response } = await p
  if (!response.ok) {
    const e = (error as Envelope | undefined)?.error
    if (response.status === 401) window.dispatchEvent(new Event('unicourse:unauthenticated'))
    throw new ApiError(
      response.status,
      e?.code ?? codeByStatus[response.status] ?? 'internal',
      e?.message || response.statusText || `Request failed (${response.status})`,
      e?.details ?? [],
      response.headers.get('X-Request-Id') ?? undefined,
    )
  }
  return data as T
}
