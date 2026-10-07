import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import type { Me } from '../features/auth/use-me'
import { mockApi, renderWithQuery } from '../test-utils'
import { AccountMenu } from './account-menu'

afterEach(() => vi.unstubAllGlobals())

const me = { id: 'u1', email: 'a@a.edu', displayName: 'Alice' } as Me['user']

it('opens a dialog with the email and the current name', () => {
  renderWithQuery(<AccountMenu user={me} />)
  fireEvent.click(screen.getByRole('button', { name: 'Account' }))
  expect(screen.getByRole('dialog', { name: 'Your account' })).toBeInTheDocument()
  expect(screen.getByText('a@a.edu')).toBeInTheDocument()
  expect(screen.getByLabelText('Name')).toHaveValue('Alice')
})

it('saves the trimmed name with PATCH /me and updates the cached me', async () => {
  const fn = mockApi({
    'PATCH /me': () => ({ body: { user: { ...me, displayName: 'Alice Doe' }, memberships: [] } }),
  })
  const { client } = renderWithQuery(<AccountMenu user={me} />)
  fireEvent.click(screen.getByRole('button', { name: 'Account' }))
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: '  Alice Doe ' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  await waitFor(() => expect(client.getQueryData<Me>(['me'])?.user.displayName).toBe('Alice Doe'))
  const req = fn.mock.calls[0][0] as Request
  expect(req.method).toBe('PATCH')
  expect(await req.json()).toEqual({ displayName: 'Alice Doe' })
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
})

it('rejects an empty name without calling the API', () => {
  const fn = mockApi({})
  renderWithQuery(<AccountMenu user={me} />)
  fireEvent.click(screen.getByRole('button', { name: 'Account' }))
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: '   ' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(screen.getByRole('alert')).toHaveTextContent('Enter a name of up to 100 characters.')
  expect(fn).not.toHaveBeenCalled()
})

it('shows the server error and keeps the dialog open', async () => {
  mockApi({
    'PATCH /me': () => ({ status: 500, body: { error: { code: 'internal', message: 'x', details: [] } } }),
  })
  renderWithQuery(<AccountMenu user={me} />)
  fireEvent.click(screen.getByRole('button', { name: 'Account' }))
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(await screen.findByText(/Something went wrong on our side/)).toBeInTheDocument()
  expect(screen.getByRole('dialog')).toBeInTheDocument()
})
