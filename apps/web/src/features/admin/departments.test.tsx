import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { mockApi } from '../../test-utils'
import { DepartmentsPage } from './departments'

afterEach(() => vi.unstubAllGlobals())

function renderAs(role = 'admin') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'x', slug: 'x', name: 'X U' }, role, departmentIds: [] }],
  })
  return render(
    <QueryClientProvider client={client}>
      <DepartmentsPage slug="x" />
    </QueryClientProvider>,
  )
}

const cs = { id: '1', name: 'Computer Science', code: 'CS' }
const math = { id: '2', name: 'Mathematics', code: 'MATH' }
const list = (items: unknown[]) => () => ({ body: { items } })
const L = 'GET /universities/x/departments'

it('lists departments from the API', async () => {
  mockApi({ [L]: list([cs, math]) })
  renderAs()
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
  renderAs()
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
  renderAs()
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
  renderAs()
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Computer Science' }))
  const dialog = await screen.findByRole('dialog', {
    name: 'Delete department?',
  })
  expect(within(dialog).getByText(/Computer Science \(CS\) will be removed/)).toBeInTheDocument()
  fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
  expect(await within(dialog).findByText(/still has courses or department admins/)).toBeInTheDocument()
  expect(
    fn.mock.calls.some(
      (c) =>
        (c[0] as Request).method === 'DELETE' && new URL((c[0] as Request).url).pathname.endsWith('/departments/1'),
    ),
  ).toBe(true)
  expect(within(dialog).getByRole('button', { name: 'Delete' })).toBeEnabled()
})

it('shows the empty state', async () => {
  mockApi({ [L]: list([]) })
  renderAs()
  expect(await screen.findByText('No departments yet')).toBeInTheDocument()
})

it('shows an error banner with retry when the list fails', async () => {
  mockApi({
    [L]: () => ({
      status: 500,
      body: { error: { code: 'internal', message: 'x', details: [] } },
    }),
  })
  renderAs()
  expect(await screen.findByRole('button', { name: 'Try again' })).toBeInTheDocument()
})

it('edits sending only changed fields and sends nothing when unchanged', async () => {
  const fn = mockApi({
    [L]: list([cs]),
    'PATCH /universities/x/departments/1': () => ({ body: { ...cs, name: 'X' } }),
  })
  const patches = () => fn.mock.calls.map((c) => c[0] as Request).filter((r) => r.method === 'PATCH')
  renderAs()
  fireEvent.click(await screen.findByRole('button', { name: 'Edit Computer Science' }))
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  expect(patches()).toHaveLength(0)
  fireEvent.click(screen.getByRole('button', { name: 'Edit Computer Science' }))
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'X' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(patches()).toHaveLength(1))
  expect(await patches()[0].clone().json()).toEqual({ name: 'X' })
})

it('stacks the table with labelled cells', async () => {
  mockApi({
    'GET /universities/x/departments': () => ({ body: { items: [{ id: 'd1', name: 'Computer Science', code: 'CS' }] } }),
  })
  renderAs()
  expect(await screen.findByText('CS')).toHaveAttribute('data-label', 'Code')
  expect(screen.getByRole('table')).toHaveClass('ui-table-stack')
})

it('shows no New, Edit or Delete to a department admin', async () => {
  mockApi({ [L]: list([cs]) })
  renderAs('dept_admin')
  expect(await screen.findByText('Computer Science')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /^New / })).toBeNull()
  expect(screen.queryByRole('button', { name: /^Edit / })).toBeNull()
  expect(screen.queryByRole('button', { name: /^Delete / })).toBeNull()
})
