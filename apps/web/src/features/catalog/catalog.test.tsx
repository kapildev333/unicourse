import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { CatalogPage } from './catalog'

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date(2026, 9, 6))
})
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

const fall = { id: 't2', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }
const mt = (weekday: number, startsAt: string, endsAt: string, location = '') => ({
  id: `m-${weekday}-${startsAt}`,
  weekday,
  startsAt,
  endsAt,
  location,
})
const sec = (id: string, code: string, staff: unknown[] = [], meetings: unknown[] = []) => ({
  id,
  courseId: `c-${code}`,
  termId: fall.id,
  number: '001',
  course: { id: `c-${code}`, departmentId: 'd1', code, title: `${code} title` },
  term: fall,
  staff,
  pendingStaff: [],
  meetings,
})
const dana = { userId: 'u1', email: 'dana@a.edu', displayName: 'Dana Poe', role: 'instructor' }
const items = [
  sec('s1', 'CS201', [dana], [mt(1, '10:30', '11:45', 'Room 101'), mt(3, '10:30', '11:45')]),
  sec('s2', 'CS202', [], [mt(2, '10:00', '11:00')]),
  sec('s3', 'CS203', [], [mt(2, '11:00', '12:00')]),
  sec('s4', 'CS300'),
]
const mine = [{ ...sec('m1', 'CS101', [], [mt(1, '10:00', '11:00'), mt(2, '09:00', '11:00')]), myRole: 'student' }]
const S = 'GET /universities/a/sections'

function Wrapper({ initial }: { initial: { termId?: string; departmentId?: string; q?: string } }) {
  const [p, setP] = useState(initial)
  return (
    <>
      <button onClick={() => setP({})}>outside-clear</button>
      <CatalogPage slug="a" {...p} onSearchChange={(n) => setP((o) => ({ ...o, ...n }))} />
    </>
  )
}

function setup(sections: unknown[] = items, terms: unknown[] = [fall], initial = {}) {
  const fn = mockApi({
    'GET /universities/a/terms': () => ({ body: { items: terms } }),
    'GET /universities/a/departments': () => ({ body: { items: [{ id: 'd1', code: 'CS', name: 'Computer Science' }] } }),
    [S]: () => ({ body: { items: sections } }),
    'GET /universities/a/me/sections': () => ({ body: { terms: [{ term: fall, sections: mine }] } }),
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute()
  const r = createRoute({ getParentRoute: () => root, path: '$slug', component: () => <Wrapper initial={initial} /> })
  const stub = createRoute({ getParentRoute: () => root, path: '$slug/sections/$id', component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([r, stub]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  return fn
}

const sectionUrls = (fn: ReturnType<typeof mockApi>) =>
  fn.mock.calls.map((c) => new URL((c[0] as Request).url)).filter((u) => u.pathname.endsWith('/a/sections'))

it('shows rows with link, instructor, meetings and no-meeting text', async () => {
  setup()
  const link = await screen.findByRole('link', { name: /^CS201-001/ })
  expect(link).toHaveAttribute('href', '/a/sections/s1')
  const row = link.closest('li')!
  expect(within(row).getByText('Dana Poe')).toBeInTheDocument()
  expect(within(row).getByText('Mon 10:30–11:45 · Room 101')).toBeInTheDocument()
  expect(within(screen.getByText('CS300 title').closest('li')!).getByText('No meeting times')).toBeInTheDocument()
})

it('notes clashes with my sections, not touching ones', async () => {
  setup()
  await screen.findByText('CS201 title')
  await waitFor(() => expect(screen.getAllByText('Clashes with CS101-001').length).toBe(2))
  expect(within(screen.getByText('CS201 title').closest('li')!).getByText('Clashes with CS101-001')).toBeInTheDocument()
  expect(within(screen.getByText('CS202 title').closest('li')!).getByText('Clashes with CS101-001')).toBeInTheDocument()
  expect(within(screen.getByText('CS203 title').closest('li')!).queryByText(/Clashes/)).toBeNull()
})

it('debounces search into one request with q and the term', async () => {
  const fn = setup()
  await screen.findByText('CS201 title')
  const before = sectionUrls(fn).length
  const input = screen.getByLabelText('Search')
  for (const v of ['c', 'cs', 'cs2']) fireEvent.change(input, { target: { value: v } })
  await waitFor(() => expect(sectionUrls(fn).length).toBe(before + 1))
  const u = sectionUrls(fn).at(-1)!
  expect(u.searchParams.get('q')).toBe('cs2')
  expect(u.searchParams.get('termId')).toBe('t2')
})

it('clears the input when q is removed from outside and does not write it back', async () => {
  const fn = setup(items, [fall], { q: 'cs2' })
  const input = await screen.findByLabelText('Search')
  expect(input).toHaveValue('cs2')
  fireEvent.click(screen.getByText('outside-clear'))
  await waitFor(() => expect(input).toHaveValue(''))
  await new Promise((r) => setTimeout(r, 450))
  expect(input).toHaveValue('')
  expect(sectionUrls(fn).at(-1)!.searchParams.has('q')).toBe(false)
})

it("typing 'intro ' then waiting 450 ms keeps 'intro ' in the input", async () => {
  setup()
  const input = await screen.findByLabelText('Search')
  fireEvent.change(input, { target: { value: 'intro ' } })
  await new Promise((r) => setTimeout(r, 450))
  expect(input).toHaveValue('intro ')
})

it('choosing a department sends departmentId', async () => {
  const fn = setup()
  await screen.findByText('CS201 title')
  fireEvent.change(screen.getByLabelText('Department'), { target: { value: 'd1' } })
  await waitFor(() => expect(sectionUrls(fn).at(-1)!.searchParams.get('departmentId')).toBe('d1'))
})

it('empty with a search says no match', async () => {
  setup([], [fall], { q: 'zzz' })
  expect(await screen.findByText('No sections match')).toBeInTheDocument()
})

it('no terms', async () => {
  setup([], [])
  expect(await screen.findByText('No terms yet')).toBeInTheDocument()
})
