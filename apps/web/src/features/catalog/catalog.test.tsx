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

function setup(sections: unknown[] = items, terms: unknown[] = [fall], initial = {}, role = 'member') {
  const fn = mockApi({
    'GET /me': () => ({
      body: {
        user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
        memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role, departmentIds: [] }],
      },
    }),
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

it('has no Search input of its own, only Term and Department', async () => {
  setup()
  await screen.findByText('CS201 title')
  expect(screen.queryByLabelText('Search')).toBeNull()
  expect(screen.getByLabelText('Department')).toBeInTheDocument()
})

it('sends q from the page params to the section request', async () => {
  const fn = setup(items, [fall], { q: 'cs2' })
  await screen.findByText('CS201 title')
  await waitFor(() => expect(sectionUrls(fn).at(-1)!.searchParams.get('q')).toBe('cs2'))
})

it('choosing a department sends departmentId', async () => {
  const fn = setup()
  await screen.findByText('CS201 title')
  fireEvent.change(screen.getByLabelText('Department'), { target: { value: 'd1' } })
  await waitFor(() => expect(sectionUrls(fn).at(-1)!.searchParams.get('departmentId')).toBe('d1'))
})

it('empty with a search says no match', async () => {
  setup([], [fall], { q: 'zzz' })
  expect(await screen.findByText('No sections match "zzz"')).toBeInTheDocument()
})

it('Clear filters resets the department and keeps the term', async () => {
  const fn = setup(items, [fall], { termId: 't2', departmentId: 'd1' })
  await screen.findByText('CS201 title')
  fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }))
  await waitFor(() => expect(sectionUrls(fn).at(-1)!.searchParams.has('departmentId')).toBe(false))
  expect(sectionUrls(fn).at(-1)!.searchParams.get('termId')).toBe('t2')
  expect(screen.queryByRole('button', { name: 'Clear filters' })).toBeNull()
})

it('no match with only a department names it and offers Clear filters', async () => {
  setup([], [fall], { departmentId: 'd1' })
  expect(await screen.findByText('No sections match "Computer Science"')).toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: 'Clear filters' })).toHaveLength(2)
})

it('no match with an unknown department says plain No sections match', async () => {
  setup([], [fall], { departmentId: 'zz' })
  expect(await screen.findByText('No sections match')).toBeInTheDocument()
})

it('a card in my sections shows its role badge; others none', async () => {
  const w = [{ ...sec('s1', 'CS201'), id: 'm1' }, sec('s9', 'CS900')]
  setup(w)
  const card = (await screen.findByText('CS201 title')).closest('li')!
  expect(await within(card).findByText('Enrolled')).toBeInTheDocument()
  expect(within(screen.getByText('CS900 title').closest('li')!).queryByText('Enrolled')).toBeNull()
})

it('count reads Showing 5 sections', async () => {
  setup([...items, sec('s5', 'CS400')])
  expect(await screen.findByText('Showing 5 sections')).toBeInTheDocument()
})

it('no terms', async () => {
  setup([], [])
  expect(await screen.findByText('No terms yet')).toBeInTheDocument()
  expect(screen.queryByRole('link', { name: 'Set up terms' })).toBeNull()
})

it('no terms: an admin gets a Set up terms link to the Terms tab', async () => {
  setup([], [], {}, 'admin')
  expect(await screen.findByRole('link', { name: 'Set up terms' })).toHaveAttribute('href', '/a/admin/terms')
})
