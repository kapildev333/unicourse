import { fireEvent, screen, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { InvitationsPage } from './invitations'

afterEach(() => vi.unstubAllGlobals())

const L = 'GET /universities/x/invitations'
const future = new Date(Date.now() + 86_400_000).toISOString()
const past = new Date(Date.now() - 86_400_000).toISOString()
const inv = (id: string, extra = {}) => ({
  id,
  email: `${id}@a.edu`,
  role: 'member',
  departmentIds: [],
  expiresAt: future,
  ...extra,
})
const depts = {
  'GET /universities/x/departments': () => ({
    body: { items: [{ id: 'd1', name: 'Computer Science', code: 'CS' }] },
  }),
}
const body = async (fn: ReturnType<typeof mockApi>, method: string) =>
  fn.mock.calls
    .map((c) => c[0] as Request)
    .find((r) => r.method === method)!
    .clone()
    .json()

it('derives status words and only offers Revoke on pending', async () => {
  mockApi({
    ...depts,
    [L]: () => ({
      body: {
        items: [inv('acc', { acceptedAt: past }), inv('exp', { expiresAt: past }), inv('pen')],
      },
    }),
  })
  renderWithQuery(<InvitationsPage slug="x" userRole="admin" />)
  expect(await screen.findByText('Accepted')).toBeInTheDocument()
  expect(screen.getByText('Expired')).toBeInTheDocument()
  expect(screen.getByText('Pending')).toBeInTheDocument()
  expect(screen.getAllByRole('button', { name: /^Revoke invitation for/ })).toHaveLength(1)
})

it('fixes the role to member for a department admin', async () => {
  const fn = mockApi({
    ...depts,
    [L]: () => ({ body: { items: [] } }),
    'POST /universities/x/invitations': () => ({ status: 201, body: inv('new') }),
  })
  renderWithQuery(<InvitationsPage slug="x" userRole="dept_admin" />)
  await screen.findByText('No invitations')
  fireEvent.click(screen.getAllByRole('button', { name: 'Invite someone' })[0])
  expect(screen.queryByLabelText('Role')).not.toBeInTheDocument()
  expect(screen.getByText('Role: Member')).toBeInTheDocument()
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'c@a.edu' } })
  fireEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
  expect(await screen.findByText(/Invitation sent to c@a\.edu\./)).toBeInTheDocument()
  expect(await body(fn, 'POST')).toEqual({ email: 'c@a.edu', role: 'member', departmentIds: [] })
})

it('creates a department admin invitation with departments', async () => {
  const fn = mockApi({
    ...depts,
    [L]: () => ({ body: { items: [] } }),
    'POST /universities/x/invitations': () => ({ status: 201, body: inv('new') }),
  })
  renderWithQuery(<InvitationsPage slug="x" userRole="admin" />)
  await screen.findByText('No invitations')
  fireEvent.click(screen.getAllByRole('button', { name: 'Invite someone' })[0])
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'c@a.edu' } })
  fireEvent.change(screen.getByLabelText('Role'), { target: { value: 'dept_admin' } })
  fireEvent.click(await screen.findByRole('checkbox', { name: /Computer Science/ }))
  fireEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
  await screen.findByText(/Invitation sent to c@a\.edu\./)
  expect(await body(fn, 'POST')).toEqual({ email: 'c@a.edu', role: 'dept_admin', departmentIds: ['d1'] })
})

it('shows the duplicate 409 copy on the email field', async () => {
  mockApi({
    ...depts,
    [L]: () => ({ body: { items: [] } }),
    'POST /universities/x/invitations': () => ({
      status: 409,
      body: { error: { code: 'conflict', message: 'dup', details: [] } },
    }),
  })
  renderWithQuery(<InvitationsPage slug="x" userRole="admin" />)
  await screen.findByText('No invitations')
  fireEvent.click(screen.getAllByRole('button', { name: 'Invite someone' })[0])
  fireEvent.change(screen.getByLabelText('Email'), { target: { value: 'c@a.edu' } })
  fireEvent.click(screen.getByRole('button', { name: 'Send invitation' }))
  expect(await screen.findByText(/already has a pending invitation/)).toBeInTheDocument()
})

it('revokes through a confirmation dialog', async () => {
  const fn = mockApi({
    ...depts,
    [L]: () => ({ body: { items: [inv('pen')] } }),
    'DELETE /universities/x/invitations/pen': () => ({ status: 204 }),
  })
  renderWithQuery(<InvitationsPage slug="x" userRole="admin" />)
  fireEvent.click(await screen.findByRole('button', { name: 'Revoke invitation for pen@a.edu' }))
  const dialog = await screen.findByRole('dialog', { name: 'Revoke invitation?' })
  fireEvent.click(within(dialog).getByRole('button', { name: 'Revoke' }))
  expect(await screen.findByText(/Invitation for pen@a\.edu revoked\./)).toBeInTheDocument()
  expect(
    fn.mock.calls.some(
      (c) =>
        (c[0] as Request).method === 'DELETE' && new URL((c[0] as Request).url).pathname.endsWith('/invitations/pen'),
    ),
  ).toBe(true)
})
