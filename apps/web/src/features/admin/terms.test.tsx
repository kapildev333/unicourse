import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { mockApi } from '../../test-utils'
import { TermsPage } from './terms'

afterEach(() => vi.unstubAllGlobals())

function renderAs(role = 'admin') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'x', slug: 'x', name: 'X U' }, role, departmentIds: [] }],
  })
  return render(
    <QueryClientProvider client={client}>
      <TermsPage slug="x" />
    </QueryClientProvider>,
  )
}

const fall = {
  id: '1',
  name: 'Fall 2026',
  startsOn: '2026-09-01',
  endsOn: '2026-12-15',
}
const L = 'GET /universities/x/terms'
const err = (status: number, code: string, details: unknown[] = []) => ({
  status,
  body: { error: { code, message: 'm', details } },
})

async function openCreate() {
  await screen.findByText('Fall 2026')
  fireEvent.click(screen.getByRole('button', { name: 'New term' }))
}

it('renders dates as local dates and uses date inputs in the dialog', async () => {
  mockApi({ [L]: () => ({ body: { items: [fall] } }) })
  renderAs()
  const fmt = new Intl.DateTimeFormat(undefined, { dateStyle: 'medium' })
  expect(await screen.findByText(fmt.format(new Date(2026, 8, 1)))).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'New term' }))
  expect(screen.getByLabelText('Starts on')).toHaveAttribute('type', 'date')
  expect(screen.getByLabelText('Ends on')).toHaveAttribute('type', 'date')
})

it('shows the endsOn field error and focuses the input', async () => {
  mockApi({
    [L]: () => ({ body: { items: [fall] } }),
    'POST /universities/x/terms': () =>
      err(400, 'validation_failed', [{ field: 'endsOn', message: 'must be after startsOn' }]),
  })
  renderAs()
  await openCreate()
  fireEvent.change(screen.getByLabelText('Name'), {
    target: { value: 'Spring' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Create term' }))
  const msg = await screen.findByText('End date must be after the start date.')
  const input = screen.getByLabelText('Ends on')
  expect(input).toHaveAttribute('aria-invalid', 'true')
  expect(input.getAttribute('aria-describedby')).toContain(msg.closest('p')!.id)
  await waitFor(() => expect(input).toHaveFocus())
})

it('shows the name conflict copy', async () => {
  mockApi({
    [L]: () => ({ body: { items: [fall] } }),
    'POST /universities/x/terms': () => err(409, 'conflict'),
  })
  renderAs()
  await openCreate()
  fireEvent.click(screen.getByRole('button', { name: 'Create term' }))
  expect(await screen.findByText('A term with this name already exists.')).toBeInTheDocument()
})

it('shows the delete 409 copy and the empty state', async () => {
  mockApi({
    [L]: () => ({ body: { items: [fall] } }),
    'DELETE /universities/x/terms/1': () => err(409, 'conflict'),
  })
  renderAs()
  await screen.findByText('Fall 2026')
  fireEvent.click(screen.getByRole('button', { name: 'Delete Fall 2026' }))
  fireEvent.click(await screen.findByRole('button', { name: 'Delete' }))
  expect(await screen.findByText('This term still has sections. Delete them first.')).toBeInTheDocument()
})

it('shows the empty state', async () => {
  mockApi({ [L]: () => ({ body: { items: [] } }) })
  renderAs()
  expect(await screen.findByText('No terms yet')).toBeInTheDocument()
})

it('stacks rows on phones and shows Current, Upcoming and Past from the dates', async () => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date(2026, 9, 7))
  try {
    mockApi({
      [L]: () => ({
        body: {
          items: [
            fall,
            { id: '2', name: 'Spring 2027', startsOn: '2027-01-10', endsOn: '2027-05-01' },
            { id: '4', name: 'Starts today', startsOn: '2026-10-07', endsOn: '2026-12-01' },
            { id: '5', name: 'Ends today', startsOn: '2026-08-01', endsOn: '2026-10-07' },
            { id: '3', name: 'Spring 2026', startsOn: '2026-01-10', endsOn: '2026-05-01' },
          ],
        },
      }),
    })
    renderAs()
    await screen.findByText('Fall 2026')
    const row = (n: string) => screen.getByText(n).closest('tr')!
    expect(within(row('Fall 2026')).getByText('Current')).toBeInTheDocument()
    expect(within(row('Spring 2027')).getByText('Upcoming')).toBeInTheDocument()
    expect(within(row('Spring 2026')).getByText('Past')).toBeInTheDocument()
    expect(within(row('Starts today')).getByText('Current')).toBeInTheDocument()
    expect(within(row('Ends today')).getByText('Current')).toBeInTheDocument()
    expect(screen.getByRole('table')).toHaveClass('ui-table-stack')
    expect(screen.getByText('Fall 2026')).toHaveAttribute('data-label', 'Name')
  } finally {
    vi.useRealTimers()
  }
})

it('names what failed to load', async () => {
  mockApi({ [L]: () => err(500, 'internal') })
  renderAs()
  expect(await screen.findByText("Couldn't load terms")).toBeInTheDocument()
})

it('shows no New, Edit or Delete to a department admin', async () => {
  mockApi({ 'GET /universities/x/terms': () => ({ body: { items: [fall] } }) })
  renderAs('dept_admin')
  expect(await screen.findByText('Fall 2026')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /^New / })).toBeNull()
  expect(screen.queryByRole('button', { name: /^Edit / })).toBeNull()
  expect(screen.queryByRole('button', { name: /^Delete / })).toBeNull()
})
