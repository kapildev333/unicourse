import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { SectionDialog } from './section-dialog'

afterEach(() => vi.unstubAllGlobals())

const term = { id: 't1', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }
const course = { id: 'c1', departmentId: 'd1', code: 'CS101', title: 'Intro' }
const meeting = { id: 'm1', weekday: 1, startsAt: '10:00', endsAt: '11:15', location: 'Room 101' }
const existing = {
  id: 's1',
  courseId: 'c1',
  termId: 't1',
  number: '001',
  course,
  term,
  staff: [],
  pendingStaff: [],
  meetings: [meeting],
}

function setup(routes: Parameters<typeof mockApi>[0], section?: typeof existing) {
  const fetchFn = mockApi({
    'GET /universities/a/courses': () => ({ body: { items: [course] } }),
    'GET /universities/a/me/sections': () => ({ body: { terms: [] } }),
    ...routes,
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role: 'admin', departmentIds: [] }],
  })
  const onClose = vi.fn()
  const onSaved = vi.fn()
  const root = createRootRoute()
  const r = createRoute({
    getParentRoute: () => root,
    path: '$slug',
    component: () => <SectionDialog slug="a" term={term} section={section} onClose={onClose} onSaved={onSaved} />,
  })
  const router = createRouter({
    routeTree: root.addChildren([r]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  return { fetchFn, onClose, onSaved }
}

const bodyOf = async (fetchFn: ReturnType<typeof mockApi>, method: string) => {
  const req = fetchFn.mock.calls.map((c) => c[0]).find((r) => r.method === method)
  return req ? JSON.parse(await req.text()) : undefined
}
const group = (n: number) => within(screen.getByRole('group', { name: `Meeting ${n}` }))
const add = () => fireEvent.click(screen.getByRole('button', { name: 'Add meeting time' }))
const fill = (g: ReturnType<typeof group>, label: string, v: string) =>
  fireEvent.change(g.getByLabelText(label), { target: { value: v } })

it('create: sends meetings in order; none without rows; next row defaults to next weekday', async () => {
  const created = { ...existing, meetings: [] }
  const { fetchFn } = setup({ 'POST /universities/a/sections': () => ({ status: 201, body: created }) })
  await screen.findByRole('option', { name: /CS101/ })
  fireEvent.change(screen.getByLabelText('Course'), { target: { value: 'c1' } })
  fireEvent.change(screen.getByLabelText('Section number'), { target: { value: '001' } })
  fireEvent.click(screen.getByRole('button', { name: 'Create section' }))
  await waitFor(() => expect(fetchFn.mock.calls.some((c) => c[0].method === 'POST')).toBe(true))
  expect(await bodyOf(fetchFn, 'POST')).toEqual({ courseId: 'c1', termId: 't1', number: '001' })
})

it('create: two rows are posted in order', async () => {
  const { fetchFn } = setup({ 'POST /universities/a/sections': () => ({ status: 201, body: existing }) })
  await screen.findByRole('option', { name: /CS101/ })
  fireEvent.change(screen.getByLabelText('Course'), { target: { value: 'c1' } })
  fireEvent.change(screen.getByLabelText('Section number'), { target: { value: '001' } })
  add()
  fill(group(1), 'Starts', '10:00')
  fill(group(1), 'Ends', '11:15')
  fill(group(1), 'Location', ' Room 101 ')
  add()
  expect(group(2).getByLabelText('Day')).toHaveValue('2')
  fill(group(2), 'Starts', '09:00')
  fill(group(2), 'Ends', '09:50')
  fireEvent.click(screen.getByRole('button', { name: 'Create section' }))
  await waitFor(() => expect(fetchFn.mock.calls.some((c) => c[0].method === 'POST')).toBe(true))
  expect((await bodyOf(fetchFn, 'POST')).meetings).toEqual([
    { weekday: 1, startsAt: '10:00', endsAt: '11:15', location: 'Room 101' },
    { weekday: 2, startsAt: '09:00', endsAt: '09:50', location: 'Room 101' },
  ])
})

const invalid = (field: string) => ({
  status: 422,
  body: { error: { code: 'validation_failed', message: 'bad', details: [{ field, message: 'x' }] } },
})

it('maps a meetings[1].endsAt error to row 2 and keeps it after removing row 1', async () => {
  setup({ 'POST /universities/a/sections': () => invalid('meetings[1].endsAt') })
  await screen.findByRole('option', { name: /CS101/ })
  add()
  add()
  fireEvent.click(screen.getByRole('button', { name: 'Create section' }))
  const ends = await within(await screen.findByRole('group', { name: 'Meeting 2' })).findByLabelText('Ends')
  expect(ends).toHaveAttribute('aria-invalid', 'true')
  await waitFor(() => expect(ends).toHaveFocus())
  expect(screen.getByText('Enter an end time after the start time.')).toBeInTheDocument()
  expect(group(1).getByLabelText('Ends')).not.toHaveAttribute('aria-invalid')
  expect(screen.queryByText('bad')).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Remove meeting 1' }))
  expect(group(1).getByLabelText('Ends')).toHaveAttribute('aria-invalid', 'true')
})

it('shows the 20-row error and disables Add at 20 rows', async () => {
  setup({ 'POST /universities/a/sections': () => invalid('meetings') })
  await screen.findByRole('option', { name: /CS101/ })
  for (let i = 0; i < 20; i++) add()
  expect(screen.getByRole('button', { name: 'Add meeting time' })).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: 'Create section' }))
  expect(await screen.findByText('Add at most 20 meeting times.')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByText('Add at most 20 meeting times.')).toHaveFocus())
})

it('edit: removing every row sends meetings []', async () => {
  const { fetchFn, onSaved } = setup(
    { 'PATCH /universities/a/sections/s1': () => ({ body: { ...existing, meetings: [] } }) },
    existing,
  )
  expect(await screen.findByText('Edit CS101-001')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Remove meeting 1' }))
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(await bodyOf(fetchFn, 'PATCH')).toEqual({ meetings: [] })
})

it('edit: changing only the number omits meetings', async () => {
  const { fetchFn, onSaved } = setup(
    { 'PATCH /universities/a/sections/s1': () => ({ body: { ...existing, number: '002' } }) },
    existing,
  )
  fireEvent.change(await screen.findByLabelText('Section number'), { target: { value: '002' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(await bodyOf(fetchFn, 'PATCH')).toEqual({ number: '002' })
})

it('edit: no change closes without a request', async () => {
  const { fetchFn, onClose } = setup({}, existing)
  fireEvent.click(await screen.findByRole('button', { name: 'Save changes' }))
  expect(onClose).toHaveBeenCalled()
  expect(fetchFn.mock.calls.some((c) => c[0].method === 'PATCH')).toBe(false)
})

it('edit: never fetches /courses', async () => {
  const { fetchFn } = setup({}, existing)
  expect(await screen.findByText('Edit CS101-001')).toBeInTheDocument()
  await new Promise((r) => setTimeout(r, 50))
  expect(fetchFn.mock.calls.some((c) => new URL(c[0].url).pathname.endsWith('/courses'))).toBe(false)
})
