import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { CalendarFeedDialog } from './calendar-feed-dialog'

afterEach(() => vi.unstubAllGlobals())

const P = '/universities/a/me/calendar-feed'
const none = { status: 404, body: { error: { code: 'not_found', message: 'none', details: [] } } }
const url = 'https://x.test/api/v1/universities/a/calendar-feeds/tok.ics'
const calls = (fn: ReturnType<typeof mockApi>, m: string) => fn.mock.calls.filter(([r]) => r.method === m).length

it('none -> create shows the URL once with a copy button', async () => {
  const write = vi.fn().mockResolvedValue(undefined)
  vi.stubGlobal('navigator', { clipboard: { writeText: write } })
  const fn = mockApi({ [`GET ${P}`]: () => none, [`POST ${P}`]: () => ({ status: 201, body: { url, createdAt: '2026-10-09T12:00:00Z' } }) })
  renderWithQuery(<CalendarFeedDialog slug="a" onClose={() => {}} />)
  fireEvent.click(await screen.findByRole('button', { name: 'Create link' }))
  expect(await screen.findByLabelText('Calendar link')).toHaveValue(url)
  fireEvent.click(screen.getByRole('button', { name: 'Copy link' }))
  expect(await screen.findByText('Link copied.')).toBeInTheDocument()
  expect(write).toHaveBeenCalledWith(url)
  expect(screen.getByText(/shown only once/)).toBeInTheDocument()
  expect(calls(fn, 'POST')).toBe(1)
})

it('copy falls back to selecting the link', async () => {
  vi.stubGlobal('navigator', { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('no')) } })
  mockApi({ [`GET ${P}`]: () => none, [`POST ${P}`]: () => ({ status: 201, body: { url, createdAt: '2026-10-09T12:00:00Z' } }) })
  renderWithQuery(<CalendarFeedDialog slug="a" onClose={() => {}} />)
  fireEvent.click(await screen.findByRole('button', { name: 'Create link' }))
  fireEvent.click(await screen.findByRole('button', { name: 'Copy link' }))
  expect(await screen.findByText('Select the link and copy it.')).toBeInTheDocument()
})

it('exists -> new link asks first, then POSTs', async () => {
  const fn = mockApi({
    [`GET ${P}`]: () => ({ body: { createdAt: '2026-10-09T12:00:00Z' } }),
    [`POST ${P}`]: () => ({ status: 201, body: { url, createdAt: '2026-10-10T12:00:00Z' } }),
  })
  renderWithQuery(<CalendarFeedDialog slug="a" onClose={() => {}} />)
  expect(await screen.findByText(/Your private link was created on Oct 9, 2026/)).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Create new link' }))
  expect(screen.getByText('The old link stops working in every calendar that uses it.')).toBeInTheDocument()
  expect(calls(fn, 'POST')).toBe(0)
  fireEvent.click(screen.getByRole('button', { name: 'Create new link' }))
  expect(await screen.findByLabelText('Calendar link')).toHaveValue(url)
})

it('turn off sends DELETE and returns to the none state', async () => {
  const fn = mockApi({ [`GET ${P}`]: () => ({ body: { createdAt: '2026-10-09T12:00:00Z' } }), [`DELETE ${P}`]: () => ({ status: 204 }) })
  renderWithQuery(<CalendarFeedDialog slug="a" onClose={() => {}} />)
  fireEvent.click(await screen.findByRole('button', { name: 'Turn off' }))
  expect(await screen.findByRole('button', { name: 'Create link' })).toBeInTheDocument()
  await waitFor(() => expect(calls(fn, 'DELETE')).toBe(1))
})

it('a failed create shows the error', async () => {
  mockApi({ [`GET ${P}`]: () => none, [`POST ${P}`]: () => ({ status: 500, body: { error: { code: 'internal', message: 'boom', details: [] } } }) })
  renderWithQuery(<CalendarFeedDialog slug="a" onClose={() => {}} />)
  fireEvent.click(await screen.findByRole('button', { name: 'Create link' }))
  expect(await screen.findByRole('alert')).toBeInTheDocument()
})
