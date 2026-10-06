import { afterEach, expect, it, vi } from 'vitest'
import { api, unwrap } from './client'
import { ApiError } from './errors'

function mockFetch(status: number, body: unknown, headers: Record<string, string> = {}) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json', ...headers } })),
  )
}

afterEach(() => vi.unstubAllGlobals())

it('returns data on 200', async () => {
  mockFetch(200, { id: 'u1', email: 'a@b.c' })
  expect(await unwrap(api.GET('/me'))).toEqual({ id: 'u1', email: 'a@b.c' })
})

it('throws ApiError with details on 400', async () => {
  mockFetch(400, { error: { code: 'validation_failed', message: '1 field is invalid', details: [{ field: 'endsOn', message: 'must be after startsOn' }] } })
  const err = await unwrap(api.GET('/me')).catch((e) => e)
  expect(err).toBeInstanceOf(ApiError)
  expect(err.code).toBe('validation_failed')
  expect(err.details[0].field).toBe('endsOn')
})

it('dispatches event on 401', async () => {
  mockFetch(401, { error: { code: 'unauthenticated', message: 'sign in', details: [] } })
  const handler = vi.fn()
  window.addEventListener('unicourse:unauthenticated', handler)
  await expect(unwrap(api.GET('/me'))).rejects.toBeInstanceOf(ApiError)
  window.removeEventListener('unicourse:unauthenticated', handler)
  expect(handler).toHaveBeenCalledTimes(1)
})

it('carries X-Request-Id on 500', async () => {
  mockFetch(500, { error: { code: 'internal', message: 'oops', details: [] } }, { 'X-Request-Id': 'abc' })
  const err = await unwrap(api.GET('/me')).catch((e) => e)
  expect(err.requestId).toBe('abc')
})

it('derives code and message from status when the body is not an envelope', async () => {
  vi.stubGlobal('fetch', vi.fn(async () => new Response('404 page not found', { status: 404, headers: { 'Content-Type': 'text/plain' } })))
  const err = await unwrap(api.GET('/me')).catch((e) => e)
  expect(err).toBeInstanceOf(ApiError)
  expect(err.code).toBe('not_found')
  expect(err.message).not.toBe('')
})
