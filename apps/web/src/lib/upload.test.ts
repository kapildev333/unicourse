import { afterEach, expect, it, vi } from 'vitest'
import { ApiError } from '../api/errors'
import { sendWithProgress } from './upload'

class FakeXhr {
  static last: FakeXhr
  upload: { onprogress?: (e: { lengthComputable: boolean; loaded: number; total: number }) => void } = {}
  withCredentials = false
  status = 200
  statusText = ''
  responseText = ''
  headers: Record<string, string> = {}
  onload?: () => void
  onerror?: () => void
  onabort?: () => void
  ontimeout?: () => void
  open = vi.fn()
  setRequestHeader = vi.fn()
  send = vi.fn()
  getResponseHeader = (k: string) => this.headers[k] ?? null
  constructor() {
    FakeXhr.last = this
  }
}
vi.stubGlobal('XMLHttpRequest', FakeXhr)
afterEach(() => vi.clearAllMocks())

it('reports bytes and resolves with the JSON body', async () => {
  const seen: number[][] = []
  const p = sendWithProgress<{ id: string }>('/u', new FormData(), (s, t) => seen.push([s, t]))
  const x = FakeXhr.last
  expect(x.open).toHaveBeenCalledWith('POST', '/u')
  expect(x.withCredentials).toBe(true)
  x.upload.onprogress?.({ lengthComputable: true, loaded: 5, total: 10 })
  x.upload.onprogress?.({ lengthComputable: false, loaded: 7, total: 0 })
  x.responseText = '{"id":"m1"}'
  x.onload?.()
  await expect(p).resolves.toEqual({ id: 'm1' })
  expect(seen).toEqual([[5, 10]])
})

it('turns a 4xx envelope into an ApiError', async () => {
  const p = sendWithProgress('/u', new FormData(), () => {})
  const x = FakeXhr.last
  x.status = 400
  x.headers['X-Request-Id'] = 'r1'
  x.responseText = '{"error":{"code":"validation_failed","message":"bad","details":[{"field":"file","message":"is larger than 100 MB"}]}}'
  x.onload?.()
  const e = await p.catch((err) => err)
  expect(e).toBeInstanceOf(ApiError)
  expect(e).toMatchObject({ status: 400, code: 'validation_failed', details: [{ field: 'file', message: 'is larger than 100 MB' }] })
})

it('maps a non-JSON 413 by status', async () => {
  const p = sendWithProgress('/u', new FormData(), () => {})
  const x = FakeXhr.last
  x.status = 413
  x.statusText = 'Request Entity Too Large'
  x.responseText = '<html>'
  x.onload?.()
  await expect(p).rejects.toMatchObject({ status: 413 })
})

it('rejects on a network error', async () => {
  const p = sendWithProgress('/u', new FormData(), () => {})
  FakeXhr.last.onerror?.()
  const e = await p.catch((err) => err)
  expect(e).toBeInstanceOf(Error)
  expect(e).not.toBeInstanceOf(ApiError)
})
