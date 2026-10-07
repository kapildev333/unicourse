import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { ParticipantsPage } from '../section/participants'

afterEach(() => vi.unstubAllGlobals())

const ins = { userId: 'u-ins', email: 'ins@a.edu', displayName: 'Ida Ins', role: 'instructor' }
const ta = { userId: 'u-ta', email: 'ta@a.edu', displayName: 'Tia Ta', role: 'ta' }
const tom = { userId: 'u-tom', email: 'tom@a.edu', displayName: 'Tom', role: 'ta' }
const course = { id: 'c1', departmentId: 'd1', code: 'CS101', title: 'Intro' }
const term = { id: 't1', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }
const mk = (staff: unknown[], pendingStaff: unknown[] = []) => ({
  id: 's1',
  courseId: 'c1',
  termId: 't1',
  number: '001',
  course,
  term,
  staff,
  pendingStaff,
  meetings: [],
})
const err = (status: number, code: string) => ({ status, body: { error: { code, message: code, details: [] } } })
const S = 'GET /universities/a/sections/s1'
const E = 'GET /universities/a/sections/s1/enrollments'

function renderPage(userId: string, role = 'member', departmentIds: string[] = []) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: userId, email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role, departmentIds }],
  })
  const root = createRootRoute()
  const slug = createRoute({
    getParentRoute: () => root,
    path: '$slug',
    component: () => <ParticipantsPage slug="a" sectionId="s1" />,
  })
  const others = ['$slug/courses/sections', '$slug/me'].map((path) =>
    createRoute({ getParentRoute: () => root, path, component: () => null }),
  )
  const router = createRouter({
    routeTree: root.addChildren([slug, ...others]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  return render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}

it('lets an instructor add and remove TAs only', async () => {
  mockApi({ [S]: () => ({ body: mk([ins, ta]) }), [E]: () => ({ body: { items: [] } }) })
  renderPage('u-ins')
  expect(await screen.findByRole('button', { name: 'Add TA' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Add instructor' })).toBeNull()
  expect(screen.getByRole('button', { name: 'Remove Tia Ta as TA' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /Remove Ida Ins/ })).toBeNull()
  // title band instructor line, outside the Staff region
  const band = screen.getByRole('heading', { name: /CS101/ }).parentElement!.parentElement!
  expect(within(band).getByText('Ida Ins')).toBeInTheDocument()
})

it('adds a TA and shows the chip after refetch', async () => {
  let staff = [ins]
  const fn = mockApi({
    [S]: () => ({ body: mk(staff) }),
    [E]: () => ({ body: { items: [] } }),
    'POST /universities/a/sections/s1/staff': () => {
      staff = [ins, tom]
      return { status: 201, body: tom }
    },
  })
  renderPage('u-ins')
  fireEvent.change(await screen.findByLabelText('Add by email'), { target: { value: ' tom@a.edu ' } })
  fireEvent.click(screen.getByRole('button', { name: 'Add TA' }))
  expect(await screen.findByText(/Tom added as TA\./)).toBeInTheDocument()
  const group = screen.getByRole('heading', { name: 'Teaching assistants' }).parentElement!
  expect(await within(group).findByText('Tom')).toBeInTheDocument()
  const post = fn.mock.calls.map((c) => c[0] as Request).find((r) => r.method === 'POST')!
  expect(await post.json()).toEqual({ email: 'tom@a.edu', role: 'ta' })
})

it('shows the invitation notice and the pending row on 202', async () => {
  let pending: unknown[] = []
  mockApi({
    [S]: () => ({ body: mk([ins], pending) }),
    [E]: () => ({ body: { items: [] } }),
    'POST /universities/a/sections/s1/staff': () => {
      pending = [{ email: 'new@a.edu', role: 'ta' }]
      return { status: 202, body: { invited: true, invitation: { email: 'new@a.edu' } } }
    },
  })
  renderPage('u-ins')
  fireEvent.change(await screen.findByLabelText('Add by email'), { target: { value: 'new@a.edu' } })
  fireEvent.click(screen.getByRole('button', { name: 'Add TA' }))
  expect((await screen.findByText(/invitation sent to new@a.edu/i)).closest('[role=status]')).not.toBeNull()
  const row = (await screen.findByText('new@a.edu', { selector: 'p' })).closest('li')!
  expect(within(row).getByText('Invited')).toBeInTheDocument()
  expect(within(row).getByRole('button', { name: 'Cancel invitation for new@a.edu' })).toBeInTheDocument()
})

it('removes staff through the dialog', async () => {
  let staff = [ins, ta]
  mockApi({
    [S]: () => ({ body: mk(staff) }),
    [E]: () => ({ body: { items: [] } }),
    'DELETE /universities/a/sections/s1/staff/u-ta': () => {
      staff = [ins]
      return { status: 204 }
    },
  })
  renderPage('u-ins')
  fireEvent.click(await screen.findByRole('button', { name: 'Remove Tia Ta as TA' }))
  const dlg = await screen.findByRole('dialog', { name: 'Remove Tia Ta?' })
  expect(dlg).toHaveTextContent('Tia Ta will no longer be a TA on CS101 section 001.')
  fireEvent.click(within(dlg).getByRole('button', { name: 'Remove' }))
  expect(await screen.findByText(/Tia Ta removed from the staff\./)).toBeInTheDocument()
  await waitFor(() => expect(screen.queryByRole('button', { name: 'Remove Tia Ta as TA' })).toBeNull())
})

it('shows both add buttons to an admin', async () => {
  mockApi({ [S]: () => ({ body: mk([ins]) }), [E]: () => ({ body: { items: [] } }) })
  renderPage('u-admin', 'admin')
  expect(await screen.findByRole('button', { name: 'Add instructor' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Add TA' })).toBeInTheDocument()
})

it('explains a 409 and focuses the email input', async () => {
  mockApi({
    [S]: () => ({ body: mk([ins]) }),
    [E]: () => ({ body: { items: [] } }),
    'POST /universities/a/sections/s1/staff': () => err(409, 'conflict'),
  })
  renderPage('u-ins')
  fireEvent.change(await screen.findByLabelText('Add by email'), { target: { value: 'dup@a.edu' } })
  fireEvent.click(screen.getByRole('button', { name: 'Add TA' }))
  expect(await screen.findByText(/can't be added/)).toBeInTheDocument()
  await waitFor(() => expect(screen.getByLabelText('Add by email')).toHaveFocus())
})

it('gives a student a read-only view and never fetches the roster', async () => {
  const fn = mockApi({
    [S]: () => ({ body: mk([ins, ta]) }),
    [E]: () => ({ body: { items: [] } }),
    'GET /universities/a/me/sections': () => ({
      body: { terms: [{ term: {}, sections: [{ id: 's1', myRole: 'student' }] }] },
    }),
  })
  renderPage('u-stu')
  await screen.findAllByText('Ida Ins') // the title band and the Staff section both name the instructor
  expect(within(screen.getByRole('region', { name: /^Staff/ })).getByText('Ida Ins')).toBeInTheDocument()
  expect(screen.queryByLabelText('Add by email')).toBeNull()
  expect(screen.queryByRole('button', { name: /Remove/ })).toBeNull()
  expect(screen.queryByRole('heading', { name: 'Roster' })).toBeNull()
  expect(fn.mock.calls.some((c) => (c[0] as Request).url.includes('/enrollments'))).toBe(false)
  expect(await screen.findByText('Student')).toBeInTheDocument()
})

it('shows the 403 state', async () => {
  mockApi({ [S]: () => err(403, 'forbidden') })
  renderPage('u-x')
  expect(await screen.findByText('No access')).toBeInTheDocument()
  expect(screen.getByText("You aren't staff on or enrolled in this section.")).toBeInTheDocument()
})

it('shows a dept_admin of another department a read-only view with no role badge and no roster', async () => {
  const fn = mockApi({ [S]: () => ({ body: mk([ins, ta]) }), [E]: () => ({ body: { items: [] } }) })
  renderPage('u-da', 'dept_admin', ['d2'])
  await screen.findAllByText('Ida Ins') // the title band and the Staff section both name the instructor
  expect(within(screen.getByRole('region', { name: /^Staff/ })).getByText('Ida Ins')).toBeInTheDocument()
  expect(screen.queryByText('Student')).toBeNull()
  expect(screen.queryByText(/You are:/)).toBeNull()
  expect(screen.queryByLabelText('Add by email')).toBeNull()
  expect(screen.queryByRole('button', { name: /Remove/ })).toBeNull()
  expect(screen.queryByRole('heading', { name: 'Roster' })).toBeNull()
  expect(fn.mock.calls.some((c) => (c[0] as Request).url.includes('/enrollments'))).toBe(false)
})

it('ignores Enter in the email field when both add buttons are shown', async () => {
  const fn = mockApi({ [S]: () => ({ body: mk([ins]) }), [E]: () => ({ body: { items: [] } }) })
  renderPage('u-admin', 'admin')
  const input = await screen.findByLabelText('Add by email')
  fireEvent.change(input, { target: { value: 'tom@a.edu' } })
  expect(fireEvent.keyDown(input, { key: 'Enter' })).toBe(false)
  expect(fn.mock.calls.some((c) => (c[0] as Request).method === 'POST')).toBe(false)
})

it('submits with the only available role on Enter', async () => {
  const fn = mockApi({
    [S]: () => ({ body: mk([ins]) }),
    [E]: () => ({ body: { items: [] } }),
    'POST /universities/a/sections/s1/staff': () => ({ status: 201, body: tom }),
  })
  renderPage('u-ins')
  const input = await screen.findByLabelText('Add by email')
  fireEvent.change(input, { target: { value: 'tom@a.edu' } })
  expect(fireEvent.keyDown(input, { key: 'Enter' })).toBe(true)
  fireEvent.submit(input.closest('form')!)
  await waitFor(() => expect(fn.mock.calls.some((c) => (c[0] as Request).method === 'POST')).toBe(true))
  const post = fn.mock.calls.map((c) => c[0] as Request).find((r) => r.method === 'POST')!
  expect(await post.json()).toEqual({ email: 'tom@a.edu', role: 'ta' })
})

const pend = (email: string, role: string, invitationId: string) => ({ email, role, invitationId })
const P = 'DELETE /universities/a/sections/s1/pending-staff/inv-1'

it('shows Cancel on pending rows only to those who may remove them', async () => {
  const pending = [pend('i@a.edu', 'instructor', 'inv-0'), pend('t@a.edu', 'ta', 'inv-1')]
  const routes = { [S]: () => ({ body: mk([ins], pending) }), [E]: () => ({ body: { items: [] } }) }
  const cancelBtn = (e: string) => screen.queryByRole('button', { name: `Cancel invitation for ${e}` })

  mockApi(routes)
  const a = renderPage('u-ins')
  await screen.findByText('t@a.edu')
  expect(cancelBtn('t@a.edu')).toBeInTheDocument()
  expect(cancelBtn('i@a.edu')).toBeNull()
  a.unmount()

  mockApi(routes)
  const b = renderPage('u-admin', 'admin')
  await screen.findByText('t@a.edu')
  expect(cancelBtn('t@a.edu')).toBeInTheDocument()
  expect(cancelBtn('i@a.edu')).toBeInTheDocument()
  b.unmount()

  mockApi(routes)
  const c = renderPage('u-da', 'dept_admin', ['d1'])
  await screen.findByText('t@a.edu')
  expect(cancelBtn('i@a.edu')).toBeInTheDocument()
  c.unmount()

  mockApi(routes)
  const d = renderPage('u-da', 'dept_admin', ['d2'])
  await screen.findByText('t@a.edu')
  expect(cancelBtn('t@a.edu')).toBeNull()
  d.unmount()

  mockApi({ ...routes, [S]: () => ({ body: mk([ins, ta], pending) }) })
  renderPage('u-ta')
  await screen.findByText('t@a.edu')
  expect(cancelBtn('t@a.edu')).toBeNull()
})

it('cancels a pending invitation and refetches', async () => {
  let pending = [pend('t@a.edu', 'ta', 'inv-1')]
  const fn = mockApi({
    [S]: () => ({ body: mk([ins], pending) }),
    [E]: () => ({ body: { items: [] } }),
    [P]: () => {
      pending = []
      return { status: 204 }
    },
  })
  renderPage('u-ins')
  fireEvent.click(await screen.findByRole('button', { name: 'Cancel invitation for t@a.edu' }))
  expect(await screen.findByText(/Invitation for t@a.edu cancelled for this section/)).toBeInTheDocument()
  await waitFor(() => expect(screen.queryByText('t@a.edu')).toBeNull())
  const del = fn.mock.calls.map((c) => c[0] as Request).find((r) => r.method === 'DELETE')!
  expect(new URL(del.url).pathname).toBe('/api/v1/universities/a/sections/s1/pending-staff/inv-1')
})

it('says Already cancelled on 404', async () => {
  mockApi({
    [S]: () => ({ body: mk([ins], [pend('t@a.edu', 'ta', 'inv-1')]) }),
    [E]: () => ({ body: { items: [] } }),
    [P]: () => err(404, 'not_found'),
  })
  renderPage('u-ins')
  fireEvent.click(await screen.findByRole('button', { name: 'Cancel invitation for t@a.edu' }))
  expect(await screen.findByText('Already cancelled.')).toBeInTheDocument()
})

it('wraps the add row and keeps the helper outside the flex row', async () => {
  mockApi({ [S]: () => ({ body: mk([ins]) }), [E]: () => ({ body: { items: [] } }) })
  renderPage('u-ins')
  const input = await screen.findByLabelText('Add by email')
  const row = screen.getByRole('button', { name: 'Add TA' }).parentElement!
  expect(row).toHaveClass('flex-wrap', 'items-start')
  expect(row).toContainElement(input)
  const hint = screen.getByText(/we'll email them an invitation/)
  expect(row).not.toContainElement(hint)
  expect(input).toHaveAccessibleDescription(/invitation/)
})

it('headings carry counts', async () => {
  mockApi({ [S]: () => ({ body: mk([ins], [{ invitationId: 'i1', email: 'p@a.edu', role: 'ta' }]) }), [E]: () => ({ body: { items: [] } }) })
  renderPage('u-x', 'admin')
  expect(await screen.findByRole('heading', { name: 'Staff 2' })).toBeInTheDocument()
  expect(await screen.findByRole('heading', { name: 'Roster 0' })).toBeInTheDocument()
})

it('names the failure', async () => {
  mockApi({ [S]: () => err(500, 'internal') })
  renderPage('u-x')
  expect(await screen.findByText("Couldn't load section")).toBeInTheDocument()
})

it('clears the email box before the refetch resolves', async () => {
  let posted = false
  let release!: () => void
  const gate = new Promise<void>((r) => (release = r))
  mockApi({
    [S]: async () => {
      if (posted) await gate
      return { body: mk([ins]) }
    },
    [E]: () => ({ body: { items: [] } }),
    'POST /universities/a/sections/s1/staff': () => {
      posted = true
      return { status: 201, body: tom }
    },
  })
  renderPage('u-ins')
  const input = await screen.findByLabelText('Add by email')
  fireEvent.change(input, { target: { value: 'tom@a.edu' } })
  fireEvent.click(screen.getByRole('button', { name: 'Add TA' }))
  await waitFor(() => expect(input).toHaveValue(''))
  release()
})
