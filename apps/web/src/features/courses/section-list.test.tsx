import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { defaultTermId } from './queries'
import { SectionListPage } from './section-list'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

const terms = [
  { id: 't1', name: 'Spring 2026', startsOn: '2026-01-10', endsOn: '2026-05-20' },
  { id: 't2', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' },
]

it.each([
  ['inside a term', '2026-03-01', 't1'],
  ['between terms', '2026-06-30', 't2'],
  ['after all terms', '2027-01-01', 't2'],
  ['before all terms', '2025-01-01', 't1'],
])('defaultTermId %s', (_, today, want) => {
  expect(defaultTermId(terms, today)).toBe(want)
})

it('defaultTermId with no terms is undefined', () => {
  expect(defaultTermId([], '2026-01-01')).toBeUndefined()
})

const course = (code: string, departmentId: string) => ({ id: `c-${code}`, departmentId, code, title: `${code} title` })
const section = (id: string, code: string, departmentId: string, extra = {}) => ({
  id,
  courseId: `c-${code}`,
  termId: 't1',
  number: '001',
  course: course(code, departmentId),
  term: terms[0],
  staff: [],
  pendingStaff: [],
  meetings: [],
  ...extra,
})
const dana = { userId: 'u1', email: 'dana@a.edu', displayName: 'Dana Poe', role: 'instructor' }
const s1 = section('s1', 'CS101', 'd1', { staff: [dana], pendingStaff: [{ email: 'x@a.edu', role: 'ta' }] })
const s2 = section('s2', 'MATH1', 'd2')
const T = 'GET /universities/a/terms'
const L = 'GET /universities/a/sections'

function Page({ initial }: { initial?: string }) {
  const [id, setId] = useState<string | undefined>(initial)
  return <SectionListPage slug="a" termId={id} onTermChange={setId} />
}
function renderPage(membership: { role: string; departmentIds: string[] }, initialTerm?: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, ...membership }],
  })
  const root = createRootRoute()
  const slug = createRoute({ getParentRoute: () => root, path: '$slug', component: () => <Page initial={initialTerm} /> })
  const sec = createRoute({ getParentRoute: () => root, path: 'sections/$id', component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([slug, sec]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  return render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}
const admin = { role: 'admin', departmentIds: [] }

it('lists sections with staff chips and invited chips', async () => {
  vi.setSystemTime(new Date('2026-03-01T12:00:00'))
  mockApi({ [T]: () => ({ body: { items: terms } }), [L]: () => ({ body: { items: [s1, s2] } }) })
  renderPage(admin)
  const row = (await screen.findByText('CS101')).closest('tr')!
  expect(within(row).getByRole('link', { name: 'CS101' })).toHaveAttribute('href', '/a/sections/s1')
  const cells = within(row).getAllByRole('cell')
  expect(within(cells[2]).getByText('Dana Poe')).toBeInTheDocument()
  expect(within(cells[3]).getByText('x@a.edu Invited')).toBeInTheDocument()
  expect(screen.getByRole('table', { name: 'Sections in Spring 2026' })).toBeInTheDocument()
  const none = screen.getByText('MATH1').closest('tr')!
  expect(within(none).getAllByText('none')).toHaveLength(2)
})

it('shows the empty state when there are no terms', async () => {
  mockApi({ [T]: () => ({ body: { items: [] } }) })
  renderPage(admin)
  expect(await screen.findByText('No terms yet')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'New section' })).toBeDisabled()
})

it('only shows Delete for the dept_admin own departments', async () => {
  mockApi({ [T]: () => ({ body: { items: terms } }), [L]: () => ({ body: { items: [s1, s2] } }) })
  renderPage({ role: 'dept_admin', departmentIds: ['d1'] })
  await screen.findByText('CS101')
  expect(screen.getByRole('button', { name: 'Delete CS101 section 001' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Delete MATH1 section 001' })).toBeNull()
})

it('Edit opens the section dialog in edit mode only for managed departments', async () => {
  vi.setSystemTime(new Date('2026-03-01T12:00:00'))
  mockApi({ [T]: () => ({ body: { items: terms } }), [L]: () => ({ body: { items: [s1, s2] } }) })
  renderPage({ role: 'dept_admin', departmentIds: ['d1'] })
  await screen.findByText('CS101')
  expect(screen.queryByRole('button', { name: 'Edit MATH1 section 001' })).toBeNull()
  fireEvent.click(screen.getByRole('button', { name: 'Edit CS101 section 001' }))
  expect(await screen.findByRole('dialog', { name: 'Edit CS101-001' })).toBeInTheDocument()
})

it('creates and deletes a section', async () => {
  vi.setSystemTime(new Date('2026-03-01T12:00:00'))
  let items = [s1]
  const fn = mockApi({
    [T]: () => ({ body: { items: terms } }),
    [L]: () => ({ body: { items } }),
    'GET /universities/a/courses': () => ({ body: { items: [course('CS102', 'd1')] } }),
    'POST /universities/a/sections': async (req) => {
      const b = await req.json()
      items = [...items, section('s3', 'CS102', 'd1', { number: b.number })]
      return { status: 201, body: items[1] }
    },
    'DELETE /universities/a/sections/s1': () => {
      items = items.filter((s) => s.id !== 's1')
      return { status: 204 }
    },
  })
  renderPage(admin)
  await screen.findByText('CS101')
  fireEvent.click(screen.getByRole('button', { name: 'New section' }))
  const dlg = await screen.findByRole('dialog', { name: 'New section' })
  await waitFor(() => expect(within(dlg).getByRole('option', { name: 'CS102 — CS102 title' })).toBeInTheDocument())
  fireEvent.change(within(dlg).getByLabelText('Course'), { target: { value: 'c-CS102' } })
  fireEvent.change(within(dlg).getByLabelText('Section number'), { target: { value: ' 002 ' } })
  fireEvent.click(within(dlg).getByRole('button', { name: 'Create section' }))
  expect(await screen.findByText(/Section CS102 002 created\./)).toBeInTheDocument()
  const post = fn.mock.calls.map((c) => c[0] as Request).find((r) => r.method === 'POST')!
  expect(await post.clone().json()).toEqual({ courseId: 'c-CS102', termId: 't1', number: '002' })

  fireEvent.click(screen.getByRole('button', { name: 'Delete CS101 section 001' }))
  const del = await screen.findByRole('dialog', { name: 'Delete CS101 section 001?' })
  fireEvent.click(within(del).getByRole('button', { name: 'Delete' }))
  expect(await screen.findByText(/Section CS101 001 deleted\./)).toBeInTheDocument()
  await waitFor(() => expect(screen.queryByRole('link', { name: 'CS101' })).toBeNull())
})

it('falls back to the default term for an unknown termId', async () => {
  vi.setSystemTime(new Date('2026-03-01T12:00:00'))
  mockApi({ [T]: () => ({ body: { items: terms } }), [L]: () => ({ body: { items: [s1] } }) })
  renderPage(admin, 'nope')
  expect(await screen.findByText('CS101')).toBeInTheDocument()
  expect(screen.queryByText('No terms yet')).toBeNull()
})

it('does not request sections when there are no terms', async () => {
  const fn = mockApi({ [T]: () => ({ body: { items: [] } }) })
  renderPage(admin)
  await screen.findByText('No terms yet')
  expect(fn.mock.calls.some((c) => new URL((c[0] as Request).url).pathname.endsWith('/sections'))).toBe(false)
})

it('hints when the course picker is truncated', async () => {
  vi.setSystemTime(new Date('2026-03-01T12:00:00'))
  mockApi({
    [T]: () => ({ body: { items: terms } }),
    [L]: () => ({ body: { items: [s1] } }),
    'GET /universities/a/courses': () => ({ body: { items: [course('CS102', 'd1')], nextCursor: 'n' } }),
  })
  renderPage(admin)
  await screen.findByText('CS101')
  fireEvent.click(screen.getByRole('button', { name: 'New section' }))
  expect(await screen.findByText('Showing the first 200 courses. Use Import sections for the others.')).toBeInTheDocument()
})

it('requests only the department courses for a dept_admin with one department', async () => {
  vi.setSystemTime(new Date('2026-03-01T12:00:00'))
  const fn = mockApi({
    [T]: () => ({ body: { items: terms } }),
    [L]: () => ({ body: { items: [s1] } }),
    'GET /universities/a/courses': () => ({ body: { items: [course('CS102', 'd1')] } }),
  })
  renderPage({ role: 'dept_admin', departmentIds: ['d1'] })
  await screen.findByText('CS101')
  fireEvent.click(screen.getByRole('button', { name: 'New section' }))
  await screen.findByRole('option', { name: 'CS102 — CS102 title' })
  const u = fn.mock.calls.map((c) => new URL((c[0] as Request).url)).find((u) => u.pathname.endsWith('/courses'))!
  expect(u.searchParams.get('departmentId')).toBe('d1')
  expect(screen.queryByText(/Showing the first 200/)).toBeNull()
})

const jo = { userId: 'u2', email: 'jo@a.edu', displayName: '', role: 'instructor' }
const many = section('s9', 'BIO1', 'd1', {
  staff: [dana, jo, { ...dana, userId: 'u3', displayName: 'Eve Ray' }],
  pendingStaff: [],
})

it('stacks the table with labels and shows staff as plain text, +N past two names', async () => {
  vi.setSystemTime(new Date('2026-03-01T12:00:00'))
  mockApi({ [T]: () => ({ body: { items: terms } }), [L]: () => ({ body: { items: [s1, many] } }) })
  renderPage(admin)
  const row = (await screen.findByText('BIO1')).closest('tr')!
  expect(screen.getByRole('table', { name: 'Sections in Spring 2026' })).toHaveClass('ui-table-stack')
  expect(within(row).getAllByRole('cell').map((c) => c.getAttribute('data-label'))).toEqual([
    'Course',
    'Section',
    'Instructors',
    'TAs',
    null,
  ])
  expect(within(row).getByText('Dana Poe +2').parentElement).toHaveAttribute('title', 'Dana Poe, jo@a.edu, Eve Ray')
  expect(row.querySelector('.border')).toBeNull()
  expect(screen.getByRole('search')).toHaveTextContent('Showing 2')
})

it('without terms: New section disabled with its reason, no Import or Term, Set up terms for admins only', async () => {
  mockApi({ [T]: () => ({ body: { items: [] } }) })
  renderPage(admin)
  await screen.findByText('No terms yet')
  expect(screen.getByRole('button', { name: 'New section' })).toBeDisabled()
  expect(screen.getByText('Add a term first')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Import sections' })).toBeNull()
  expect(screen.queryByLabelText('Term')).toBeNull()
  expect(screen.getByRole('link', { name: 'Set up terms' })).toHaveAttribute('href', '/a/admin/terms')
})

it('without terms a dept_admin gets no Set up terms action', async () => {
  mockApi({ [T]: () => ({ body: { items: [] } }) })
  renderPage({ role: 'dept_admin', departmentIds: ['d1'] })
  await screen.findByText('No terms yet')
  expect(screen.queryByRole('link', { name: 'Set up terms' })).toBeNull()
})

it('names the failure', async () => {
  vi.setSystemTime(new Date('2026-03-01T12:00:00'))
  mockApi({
    [T]: () => ({ body: { items: terms } }),
    [L]: () => ({ status: 500, body: { error: { code: 'internal', message: 'x', details: [] } } }),
  })
  renderPage(admin)
  expect(await screen.findByText("Couldn't load sections")).toBeInTheDocument()
})
