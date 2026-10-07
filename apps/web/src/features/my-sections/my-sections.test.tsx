import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, within } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { MySectionsPage } from './my-sections'

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date(2026, 9, 6))
})
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

const U = 'GET /universities/a/me/sections'
const T = 'GET /universities/a/terms'
const fall = { id: 't2', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }
const spring = { id: 't1', name: 'Spring 2026', startsOn: '2026-01-10', endsOn: '2026-05-20' }
const winter = { id: 't3', name: 'Winter 2027', startsOn: '2027-01-05', endsOn: '2027-02-01' }
const mt = (weekday: number, startsAt: string, endsAt: string, location = '') => ({
  id: `m-${weekday}-${startsAt}`,
  weekday,
  startsAt,
  endsAt,
  location,
})
const sec = (id: string, code: string, term: typeof fall, myRole: string, staff: unknown[] = [], meetings: unknown[] = []) => ({
  id,
  courseId: `c-${code}`,
  termId: term.id,
  number: '001',
  course: { id: `c-${code}`, departmentId: 'd1', code, title: `${code} title` },
  term,
  staff,
  pendingStaff: [],
  meetings,
  myRole,
})
const dana = { userId: 'u1', email: 'dana@a.edu', displayName: 'Dana Poe', role: 'instructor' }
const data = {
  terms: [
    {
      term: fall,
      sections: [
        sec('s1', 'CS101', fall, 'ta', [dana], [mt(1, '10:00', '11:15', 'Room 101'), mt(3, '10:00', '11:15')]),
        sec('s2', 'CS102', fall, 'instructor', [], [mt(1, '10:30', '11:30')]),
        sec('s4', 'CS300', fall, 'student'),
      ],
    },
    { term: spring, sections: [sec('s3', 'MATH1', spring, 'student')] },
  ],
}
const termsBody = (items: unknown[]) => ({ body: { items } })

function Wrapper({ initial }: { initial: { termId?: string; view?: 'schedule' } }) {
  const [p, setP] = useState(initial)
  return <MySectionsPage slug="a" {...p} onSearchChange={(n) => setP((o) => ({ ...o, ...n }))} />
}

function renderPage(role = 'member', initial: { termId?: string; view?: 'schedule' } = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role, departmentIds: [] }],
  })
  const root = createRootRoute()
  const me = createRoute({ getParentRoute: () => root, path: '$slug', component: () => <Wrapper initial={initial} /> })
  const stub = (path: string) => createRoute({ getParentRoute: () => root, path, component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([me, stub('$slug/sections/$id'), stub('$slug/courses'), stub('$slug/catalog')]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  return render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}

const ok = () => mockApi({ [U]: () => ({ body: data }), [T]: () => termsBody([spring, winter, fall]) })

it('defaults to the term containing today and lists only its sections', async () => {
  ok()
  renderPage()
  await screen.findByText('CS101-001')
  expect(screen.queryByText('MATH1-001')).toBeNull()
  expect(screen.queryByText(/No term is in session/)).toBeNull()
  const ta = screen.getByText('CS101-001').closest('li')!
  expect(within(ta).getByText('TA')).toBeInTheDocument()
  expect(within(ta).getByText('Dana Poe')).toBeInTheDocument()
  expect(within(screen.getByText('CS102-001').closest('li')!).getByText('Instructor')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: /^CS101-001/ })).toHaveAttribute('href', '/a/sections/s1')
})

it('picks the next term when none is in session and says so', async () => {
  vi.setSystemTime(new Date(2026, 11, 20))
  mockApi({ [U]: () => ({ body: { terms: [{ term: winter, sections: [sec('s5', 'ART1', winter, 'student')] }] } }), [T]: () => termsBody([spring, winter, fall]) })
  renderPage()
  expect(await screen.findByText('ART1-001')).toBeInTheDocument()
  expect(screen.getByText('No term is in session today. Showing Winter 2027.')).toBeInTheDocument()
})

it('shows "No terms yet" without crashing when there are no terms', async () => {
  mockApi({ [U]: () => ({ body: { terms: [] } }), [T]: () => termsBody([]) })
  renderPage()
  expect(await screen.findByText('No terms yet')).toBeInTheDocument()
})

it('shows the chosen term', async () => {
  ok()
  renderPage()
  await screen.findByText('CS101-001')
  fireEvent.change(screen.getByLabelText('Term'), { target: { value: 't1' } })
  expect(await screen.findByText('MATH1-001')).toBeInTheDocument()
  expect(screen.queryByText('CS101-001')).toBeNull()
})

it.each([
  ['admin', true],
  ['dept_admin', true],
  ['member', false],
])('%s: empty term offers the catalog, Courses link is %s', async (role, courses) => {
  ok()
  renderPage(role, { termId: 't3' })
  expect(await screen.findByText('No sections in Winter 2027')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Browse the catalog' })).toBeInTheDocument()
  const link = screen.queryByRole('link', { name: 'Go to Courses' })
  if (courses) expect(link).toHaveAttribute('href', '/a/courses')
  else expect(link).toBeNull()
})

it('schedule view: blocks, clashes, untimed list, empty days', async () => {
  ok()
  renderPage('member', { view: 'schedule' })
  const mon = await screen.findByRole('region', { name: 'Monday' })
  const cs101 = within(mon).getByRole('link', { name: /^CS101-001/ })
  const cs102 = within(mon).getByRole('link', { name: /^CS102-001/ })
  expect(cs101).toHaveTextContent('Clashes with CS102-001')
  expect(cs102).toHaveTextContent('Clashes with CS101-001')
  expect(cs101).toHaveAttribute('data-clash')
  expect(cs102).toHaveAttribute('data-clash')
  expect(cs101).toHaveTextContent('CS101 title')
  expect(cs101).toHaveTextContent('Dana Poe')
  expect(cs101).toHaveTextContent('Room 101')
  const wed = within(screen.getByRole('region', { name: 'Wednesday' })).getByRole('link', { name: /^CS101-001/ })
  expect(wed).not.toHaveTextContent('Clashes')
  expect(wed).not.toHaveAttribute('data-clash')
  const none = screen.getByRole('region', { name: 'No meeting times' })
  expect(within(none).getByRole('link', { name: 'CS300-001' })).toBeInTheDocument()
  expect(within(screen.getByRole('region', { name: 'Tuesday' })).getByText('No meetings')).toBeInTheDocument()
})

it('schedule view with no timed sections says so', async () => {
  ok()
  renderPage('member', { termId: 't1', view: 'schedule' })
  expect(await screen.findByText(/has meeting times yet/)).toBeInTheDocument()
  expect(screen.getByRole('region', { name: 'No meeting times' })).toBeInTheDocument()
})

it('List and Schedule buttons are pressed toggles', async () => {
  ok()
  renderPage()
  await screen.findByText('CS101-001')
  const list = screen.getByRole('button', { name: 'List' })
  const sched = screen.getByRole('button', { name: 'Schedule' })
  expect(list).toHaveAttribute('aria-pressed', 'true')
  expect(sched).toHaveAttribute('aria-pressed', 'false')
  fireEvent.click(sched)
  expect(await screen.findByRole('region', { name: 'Monday' })).toBeInTheDocument()
  expect(sched).toHaveAttribute('aria-pressed', 'true')
  fireEvent.click(list)
  expect(await screen.findByRole('list', { name: /^My sections in/ })).toBeInTheDocument()
})

it('shows an error with retry', async () => {
  let fail = true
  mockApi({
    [T]: () => termsBody([fall]),
    [U]: () =>
      fail ? { status: 500, body: { error: { code: 'internal', message: 'boom', details: [] } } } : { body: data },
  })
  renderPage()
  const retry = await screen.findByRole('button', { name: 'Try again' })
  fail = false
  fireEvent.click(retry)
  expect(await screen.findByText('CS101-001')).toBeInTheDocument()
})

const media = (wide: boolean) =>
  vi.stubGlobal('matchMedia', () => ({ matches: wide, addEventListener: () => {}, removeEventListener: () => {} }))

it('phone: the schedule is an agenda of weekday lists, one link per meeting, no grid', async () => {
  media(false)
  ok()
  renderPage('member', { view: 'schedule' })
  const mon = await screen.findByRole('region', { name: 'Monday' })
  expect(within(mon).getAllByRole('link')).toHaveLength(2)
  const cs101 = within(mon).getByRole('link', { name: /^CS101-001/ })
  expect(cs101).toHaveAttribute('href', '/a/sections/s1')
  expect(cs101).toHaveTextContent('Clashes with CS102-001')
  expect(cs101).not.toHaveTextContent('Dana Poe')
  expect(within(mon).getAllByRole('listitem')).toHaveLength(2)
  expect(screen.getByRole('region', { name: 'Wednesday' })).toBeInTheDocument()
  expect(screen.queryByRole('region', { name: 'Tuesday' })).toBeNull()
  expect(screen.queryByRole('region', { name: 'Sunday' })).toBeNull()
  expect(document.querySelector('.overflow-x-auto')).toBeNull()
})

it("phone: the agenda marks today's heading", async () => {
  vi.setSystemTime(new Date(2026, 9, 7, 10, 30))
  media(false)
  ok()
  renderPage('member', { view: 'schedule' })
  const wed = await screen.findByRole('region', { name: 'Wednesday' })
  const head = within(wed).getByRole('heading')
  expect(head).toHaveAttribute('aria-current', 'date')
  expect(head).toHaveTextContent('Today')
  expect(document.querySelectorAll('[aria-current="date"]')).toHaveLength(1)
})

it('from 768 px the grid sits in a scroll wrapper, with a narrow track for an empty Saturday and Sunday', async () => {
  media(true)
  ok()
  renderPage('member', { view: 'schedule' })
  await screen.findByRole('region', { name: 'Monday' })
  const grid = document.querySelector('.overflow-x-auto > div') as HTMLElement
  expect(grid.style.gridTemplateColumns).toBe(
    '3.5rem minmax(0,1fr) minmax(0,1fr) minmax(0,1fr) minmax(0,1fr) minmax(0,1fr) minmax(0,0.5fr) minmax(0,0.5fr)',
  )
  expect(screen.getByRole('region', { name: 'Sunday' })).toBeInTheDocument()
})

it('grid block: instructor is screen-reader text only; untimed list separates with a middle dot', async () => {
  ok()
  renderPage('member', { view: 'schedule' })
  const mon = await screen.findByRole('region', { name: 'Monday' })
  const link = within(mon).getByRole('link', { name: /^CS101-001/ })
  expect(link).toHaveAccessibleName(/Dana Poe/)
  expect(within(link).getByText('Dana Poe')).toHaveClass('sr-only')
  const none = screen.getByRole('region', { name: 'No meeting times' })
  expect(none).toHaveTextContent('CS300-001 · CS300 title')
  expect(none).not.toHaveTextContent('—')
})

it("grid marks today's column and draws the time line only there", async () => {
  vi.setSystemTime(new Date(2026, 9, 6, 10, 30))
  ok()
  renderPage('member', { view: 'schedule' })
  const tue = await screen.findByRole('region', { name: 'Tuesday' })
  const head = within(tue).getByRole('heading')
  expect(head).toHaveAttribute('aria-current', 'date')
  expect(head).toHaveTextContent('Today')
  expect(document.querySelectorAll('[aria-current="date"]')).toHaveLength(1)
  expect(document.querySelectorAll('[data-now-line]')).toHaveLength(1)
  expect(document.querySelector('[data-now-label]')).toHaveTextContent('10:30')
})

it('own instructor entry reads You', async () => {
  const mine = { userId: 'u0', email: 'x@a.edu', displayName: 'X', role: 'instructor' }
  mockApi({
    [U]: () => ({ body: { terms: [{ term: fall, sections: [sec('s9', 'CS900', fall, 'instructor', [mine])] }] } }),
    [T]: () => termsBody([fall]),
  })
  renderPage()
  const card = (await screen.findByText('CS900-001')).closest('li')!
  expect(within(card).getByText('You')).toBeInTheDocument()
  expect(within(card).queryByText('X')).toBeNull()
})

it('instructor and TA cards link to the gradebook; students get none', async () => {
  ok()
  renderPage()
  const ta = (await screen.findByText('CS101-001')).closest('li')!
  expect(within(ta).getByRole('link', { name: 'Gradebook' })).toHaveAttribute('href', '/a/sections/s1/grades')
  expect(within(screen.getByText('CS102-001').closest('li')!).getByRole('link', { name: 'Gradebook' })).toBeInTheDocument()
  expect(within(screen.getByText('CS300-001').closest('li')!).queryByRole('link', { name: /Gradebook/ })).toBeNull()
})

it('term options show the name only; meetings with the same time and room merge', async () => {
  mockApi({
    [U]: () => ({
      body: { terms: [{ term: fall, sections: [sec('s7', 'CS700', fall, 'student', [], [mt(1, '10:00', '11:00', 'H1'), mt(3, '10:00', '11:00', 'H1')])] }] },
    }),
    [T]: () => termsBody([fall]),
  })
  renderPage()
  expect(await screen.findByText('Mon, Wed 10:00–11:00 · H1')).toBeInTheDocument()
  expect(screen.getByRole('option', { name: 'Fall 2026' })).toBeInTheDocument()
})

it.each([
  ['admin', true],
  ['member', false],
])('%s on "No terms yet": Set up terms link is %s', async (role, shown) => {
  mockApi({ [U]: () => ({ body: { terms: [] } }), [T]: () => termsBody([]) })
  renderPage(role)
  await screen.findByText('No terms yet')
  const link = screen.queryByRole('link', { name: 'Set up terms' })
  if (shown) expect(link).toHaveAttribute('href', '/a/admin/terms')
  else expect(link).toBeNull()
})

it('a failed load names what failed', async () => {
  mockApi({ [T]: () => termsBody([fall]), [U]: () => ({ status: 500, body: { error: { code: 'internal', message: 'boom', details: [] } } }) })
  renderPage()
  expect(await screen.findByText("Couldn't load your sections")).toBeInTheDocument()
})
