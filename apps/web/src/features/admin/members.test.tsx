import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { MembersPage } from './members'

const navigate = vi.fn()
vi.mock('@tanstack/react-router', async (orig) => ({
  ...(await orig<typeof import('@tanstack/react-router')>()),
  useNavigate: () => navigate,
}))

afterEach(() => {
  vi.unstubAllGlobals()
  navigate.mockClear()
})

const bob = {
  id: 'm1',
  userId: 'u1',
  email: 'bob@a.edu',
  displayName: 'Bob Roe',
  role: 'dept_admin',
  status: 'active',
  departmentIds: ['d1'],
}
const L = 'GET /universities/x/memberships'
const base = {
  [L]: () => ({ body: { items: [bob] } }),
  'GET /universities/x/departments': () => ({
    body: { items: [{ id: 'd1', name: 'Computer Science', code: 'CS' }] },
  }),
  'GET /me': () => ({
    body: { user: { id: 'u0', email: 'a@a.edu', displayName: 'A' }, memberships: [] },
  }),
}
const conflict = {
  status: 409,
  body: { error: { code: 'conflict', message: 'last', details: [] } },
}

async function openRoleDialog() {
  renderWithQuery(<MembersPage slug="x" />)
  fireEvent.click(await screen.findByRole('button', { name: 'Change role for bob@a.edu' }))
  return screen.findByRole('dialog', { name: 'Change role for bob@a.edu' })
}

it('lists role badge and department code', async () => {
  mockApi(base)
  renderWithQuery(<MembersPage slug="x" />)
  expect(await screen.findByText('Department admin')).toBeInTheDocument()
  expect(await screen.findByText('CS')).toBeInTheDocument()
})

it('requires a department for a department admin and focuses the first checkbox', async () => {
  mockApi({
    ...base,
    [L]: () => ({ body: { items: [{ ...bob, role: 'member', departmentIds: [] }] } }),
    'PATCH /universities/x/memberships/m1': () => ({
      status: 400,
      body: {
        error: {
          code: 'validation_failed',
          message: 'bad',
          details: [{ field: 'departmentIds', message: 'required' }],
        },
      },
    }),
  })
  await openRoleDialog()
  fireEvent.change(screen.getByLabelText('Role'), { target: { value: 'dept_admin' } })
  expect(await screen.findByText('Computer Science')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Save role' }))
  expect(await screen.findByText('Choose at least one department for a department admin.')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByRole('checkbox')).toHaveFocus())
})

it('sends an empty department list for member and shows the last-admin 409', async () => {
  const fn = mockApi({
    ...base,
    'PATCH /universities/x/memberships/m1': () => conflict,
  })
  await openRoleDialog()
  fireEvent.change(screen.getByLabelText('Role'), { target: { value: 'member' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save role' }))
  expect(await screen.findByText(/only admin/)).toBeInTheDocument()
  expect(screen.getByRole('dialog')).toHaveAttribute('open')
  const patch = fn.mock.calls.map((c) => c[0] as Request).find((r) => r.method === 'PATCH')!
  expect(await patch.clone().json()).toEqual({ role: 'member', departmentIds: [] })
})

it('sends a demoted caller to the admin index and refetches me', async () => {
  const fn = mockApi({
    ...base,
    [L]: () => ({ body: { items: [{ ...bob, userId: 'u0', role: 'admin', departmentIds: [] }] } }),
    'PATCH /universities/x/memberships/m1': () => ({ body: { ...bob, userId: 'u0' } }),
  })
  await openRoleDialog()
  fireEvent.change(screen.getByLabelText('Role'), { target: { value: 'member' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save role' }))
  await waitFor(() =>
    expect(navigate).toHaveBeenCalledWith({ to: '/$slug/admin', params: { slug: 'x' }, replace: true }),
  )
  const meGets = () => fn.mock.calls.filter((c) => new URL((c[0] as Request).url).pathname.endsWith('/me')).length
  await waitFor(() => expect(meGets()).toBe(2))
})

it("changing someone else's role neither navigates nor refetches me", async () => {
  const fn = mockApi({
    ...base,
    'PATCH /universities/x/memberships/m1': () => ({ body: bob }),
  })
  await openRoleDialog()
  fireEvent.change(screen.getByLabelText('Role'), { target: { value: 'member' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save role' }))
  expect(await screen.findByText(/bob@a.edu is now/)).toBeInTheDocument()
  expect(navigate).not.toHaveBeenCalled()
  expect(fn.mock.calls.filter((c) => new URL((c[0] as Request).url).pathname.endsWith('/me'))).toHaveLength(1)
})
