import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { DepartmentsPage } from './departments'

afterEach(() => vi.unstubAllGlobals())

const cs = { id: '1', name: 'Computer Science', code: 'CS' }
const math = { id: '2', name: 'Mathematics', code: 'MATH' }
const list = (items: unknown[]) => () => ({ body: { items } })
const L = 'GET /universities/x/departments'

it('lists departments from the API', async () => {
  mockApi({ [L]: list([cs, math]) })
  renderWithQuery(<DepartmentsPage slug="x" />)
  expect(await screen.findByText('Computer Science')).toBeInTheDocument()
  expect(screen.getByText('MATH')).toBeInTheDocument()
})

it('creates a department, uppercases the code, and refreshes the list', async () => {
  let items = [cs, math]
  const fn = mockApi({
    [L]: () => ({ body: { items } }),
    'POST /universities/x/departments': () => {
      items = [...items, { id: '3', name: 'Physics', code: 'PHYS' }]
      return { status: 201, body: items[2] }
    },
  })
  renderWithQuery(<DepartmentsPage slug="x" />)
  await screen.findByText('Computer Science')
  fireEvent.click(screen.getByRole('button', { name: 'New department' }))
  fireEvent.change(screen.getByLabelText('Name'), {
    target: { value: 'Physics' },
  })
  fireEvent.change(screen.getByLabelText('Code'), {
    target: { value: 'phys' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Create department' }))
  expect(await screen.findByRole('cell', { name: 'Physics' })).toBeInTheDocument()
  expect(screen.getByText(/Department “PHYS” created\./)).toBeInTheDocument()
  const post = fn.mock.calls.map((c) => c[0] as Request).find((r) => r.method === 'POST')!
  expect(await post.clone().json()).toEqual({ name: 'Physics', code: 'PHYS' })
})

it('shows the conflict copy on the code field and focuses it', async () => {
  mockApi({
    [L]: list([cs]),
    'POST /universities/x/departments': () => ({
      status: 409,
      body: { error: { code: 'conflict', message: 'dup', details: [] } },
    }),
  })
  renderWithQuery(<DepartmentsPage slug="x" />)
  await screen.findByText('Computer Science')
  fireEvent.click(screen.getByRole('button', { name: 'New department' }))
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'Cs2' } })
  fireEvent.change(screen.getByLabelText('Code'), { target: { value: 'CS' } })
  fireEvent.click(screen.getByRole('button', { name: 'Create department' }))
  expect(await screen.findByText(/code already exists/)).toBeInTheDocument()
  await waitFor(() => expect(screen.getByLabelText('Code')).toHaveFocus())
})

it('confirms before deleting and shows the 409 copy', async () => {
  const fn = mockApi({
    [L]: list([cs]),
    'DELETE /universities/x/departments/1': () => ({
      status: 409,
      body: { error: { code: 'conflict', message: 'in use', details: [] } },
    }),
  })
  renderWithQuery(<DepartmentsPage slug="x" />)
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Computer Science' }))
  const dialog = await screen.findByRole('dialog', {
    name: 'Delete department?',
  })
  expect(within(dialog).getByText(/Computer Science \(CS\) will be removed/)).toBeInTheDocument()
  fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
  expect(await within(dialog).findByText(/still has courses or department admins/)).toBeInTheDocument()
  expect(fn.mock.calls.some((c) => (c[0] as Request).method === 'DELETE')).toBe(true)
  expect(within(dialog).getByRole('button', { name: 'Delete' })).toBeEnabled()
})

it('shows the empty state', async () => {
  mockApi({ [L]: list([]) })
  renderWithQuery(<DepartmentsPage slug="x" />)
  expect(await screen.findByText('No departments yet')).toBeInTheDocument()
})

it('shows an error banner with retry when the list fails', async () => {
  mockApi({
    [L]: () => ({
      status: 500,
      body: { error: { code: 'internal', message: 'x', details: [] } },
    }),
  })
  renderWithQuery(<DepartmentsPage slug="x" />)
  expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument()
})
