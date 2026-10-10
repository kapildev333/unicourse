import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { signOut } from '../features/auth/sign-out'
import type { Me, MyMembership } from '../features/auth/use-me'
import { mockApi, renderWithQuery } from '../test-utils'
import { AccountMenu } from './account-menu'

vi.mock('../features/auth/sign-out', () => ({ signOut: vi.fn() }))

afterEach(() => vi.unstubAllGlobals())

const membership = { university: { id: 'a', slug: 'a', name: 'A U' }, role: 'member', departmentIds: [] } as MyMembership
const S = 'GET /universities/a/me/notification-settings'
const settings = (announcementMail: boolean) => ({ [S]: () => ({ body: { announcementMail } }) })
const me = { id: 'u1', email: 'a@a.edu', displayName: 'Alice' } as Me['user']

it('opens a dialog with the email and the current name', () => {
  mockApi(settings(true))
  renderWithQuery(<AccountMenu user={me} membership={membership} />)
  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  expect(screen.getByRole('dialog', { name: 'Your account' })).toBeInTheDocument()
  expect(screen.getByText('a@a.edu')).toBeInTheDocument()
  expect(screen.getByLabelText('Name')).toHaveValue('Alice')
})

it('saves the trimmed name with PATCH /me and updates the cached me', async () => {
  const fn = mockApi({
    ...settings(true),
    'PATCH /me': () => ({ body: { user: { ...me, displayName: 'Alice Doe' }, memberships: [] } }),
  })
  const { client } = renderWithQuery(<AccountMenu user={me} membership={membership} />)
  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: '  Alice Doe ' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(client.getQueryData<Me>(['me'])?.user.displayName).toBe('Alice Doe'))
  const req = fn.mock.calls.find(([r]) => r.method === 'PATCH')![0] as Request
  expect(req.method).toBe('PATCH')
  expect(await req.json()).toEqual({ displayName: 'Alice Doe' })
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
})

it('rejects an empty name without calling the API', () => {
  const fn = mockApi(settings(true))
  renderWithQuery(<AccountMenu user={me} membership={membership} />)
  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: '   ' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(screen.getByRole('alert')).toHaveTextContent('Enter a name of up to 100 characters.')
  expect(fn.mock.calls.every(([r]) => r.method === 'GET')).toBe(true)
})

it('shows the server error and keeps the dialog open', async () => {
  mockApi({
    ...settings(true),
    'PATCH /me': () => ({ status: 500, body: { error: { code: 'internal', message: 'x', details: [] } } }),
  })
  renderWithQuery(<AccountMenu user={me} membership={membership} />)
  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(await screen.findByText(/Something went wrong on our side/)).toBeInTheDocument()
  expect(screen.getByRole('dialog')).toBeInTheDocument()
})

it('has a Sign out in the dialog footer (the phone way out) that calls the shared sign-out', () => {
  mockApi(settings(true))
  const { client } = renderWithQuery(<AccountMenu user={me} membership={membership} />)
  expect(screen.queryByRole('button', { name: 'Sign out' })).toBeNull() // closed
  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  const btn = screen.getByRole('button', { name: 'Sign out' })
  expect(btn).toHaveClass('md:hidden')
  fireEvent.click(btn)
  expect(signOut).toHaveBeenCalledWith(client)
})

const label = 'Email me new announcements'
const patches = (fn: ReturnType<typeof mockApi>) => fn.mock.calls.filter(([r]) => r.method === 'PATCH').map(([r]) => new URL(r.url).pathname)

it('the checkbox reflects the saved setting and the hint names the university', async () => {
  mockApi(settings(false))
  renderWithQuery(<AccountMenu user={me} membership={membership} />)
  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  const box = screen.getByLabelText(label)
  expect(box).toBeDisabled()
  await waitFor(() => expect(box).toBeEnabled())
  expect(box).not.toBeChecked()
  expect(screen.getByText('In A U. They still appear under Notifications.')).toBeInTheDocument()
})

it('saves the setting only when the box changed', async () => {
  const fn = mockApi({
    ...settings(true),
    'PATCH /me': () => ({ body: { user: me, memberships: [] } }),
    'PATCH /universities/a/me/notification-settings': () => ({ body: { announcementMail: false } }),
  })
  renderWithQuery(<AccountMenu user={me} membership={membership} />)
  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  await waitFor(() => expect(screen.getByLabelText(label)).toBeEnabled())
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  expect(patches(fn)).toEqual(['/api/v1/me'])

  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  await waitFor(() => expect(screen.getByLabelText(label)).toBeEnabled())
  fireEvent.click(screen.getByLabelText(label))
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  const call = fn.mock.calls.find(([r]) => r.url.endsWith('/notification-settings') && r.method === 'PATCH')!
  expect(await call[0].json()).toEqual({ announcementMail: false })
})

it('shows the error and stays open when the setting fails to save', async () => {
  mockApi({
    ...settings(true),
    'PATCH /me': () => ({ body: { user: me, memberships: [] } }),
    'PATCH /universities/a/me/notification-settings': () => ({ status: 500, body: { error: { code: 'internal', message: 'x', details: [] } } }),
  })
  renderWithQuery(<AccountMenu user={me} membership={membership} />)
  fireEvent.click(screen.getByRole('button', { name: /^Account/ }))
  await waitFor(() => expect(screen.getByLabelText(label)).toBeEnabled())
  fireEvent.click(screen.getByLabelText(label))
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(await screen.findByText(/Something went wrong on our side/)).toBeInTheDocument()
  expect(screen.getByRole('dialog')).toBeInTheDocument()
})
