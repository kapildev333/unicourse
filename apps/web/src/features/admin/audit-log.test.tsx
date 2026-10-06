import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { AuditLogPage } from './audit-log'

afterEach(() => vi.unstubAllGlobals())

const L = 'GET /universities/x/audit-log'
const entry = (id: string, extra = {}) => ({
  id,
  action: 'term.update',
  entityType: 'term',
  entityId: 't1',
  at: '2026-10-05T14:03:22Z',
  actor: { id: 'u1', email: 'alice@a.edu', displayName: 'Alice' },
  ...extra,
})
const members = {
  'GET /universities/x/memberships': () => ({ body: { items: [] } }),
}
const gets = (fn: ReturnType<typeof mockApi>) =>
  fn.mock.calls
    .map((c) => c[0] as Request)
    .filter((r) => new URL(r.url).pathname.endsWith('/audit-log'))
    .map((r) => new URL(r.url).searchParams)

it('loads more pages by cursor', async () => {
  const fn = mockApi({
    ...members,
    [L]: (req) =>
      new URL(req.url).searchParams.get('cursor') === 'c2'
        ? { body: { items: [entry('e2', { actor: undefined })] } }
        : { body: { items: [entry('e1')], nextCursor: 'c2' } },
  })
  renderWithQuery(<AuditLogPage slug="x" />)
  await screen.findByText('Alice')
  fireEvent.click(screen.getByRole('button', { name: 'Load more' }))
  expect(await screen.findByText('System')).toBeInTheDocument()
  expect(screen.getByText('Alice')).toBeInTheDocument()
  expect(gets(fn)[1].get('cursor')).toBe('c2')
  expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument()
})

it('filters by entity type', async () => {
  const fn = mockApi({ ...members, [L]: () => ({ body: { items: [entry('e1')] } }) })
  renderWithQuery(<AuditLogPage slug="x" />)
  await screen.findByText('Alice')
  fireEvent.change(screen.getByLabelText('Entity'), { target: { value: 'term' } })
  await screen.findByRole('button', { name: 'Clear filters' })
  expect(gets(fn).some((p) => p.get('entityType') === 'term')).toBe(true)
})

it('shows before and after with the changed key marked', async () => {
  mockApi({
    ...members,
    [L]: () => ({
      body: {
        items: [entry('e1', { before: { name: 'Fall 26', code: 'F' }, after: { name: 'Fall 2026', code: 'F' } })],
      },
    }),
  })
  renderWithQuery(<AuditLogPage slug="x" />)
  await screen.findByText('Term updated')
  expect(screen.getByText('Before')).toBeInTheDocument()
  expect(screen.getByText('After')).toBeInTheDocument()
  expect(screen.getAllByText(/\(changed\)/)).toHaveLength(2)
  expect(screen.getByText('Fall 2026 (changed)')).toBeInTheDocument()
})

it('shows the empty state', async () => {
  mockApi({ ...members, [L]: () => ({ body: { items: [] } }) })
  renderWithQuery(<AuditLogPage slug="x" />)
  expect(await screen.findByText('No changes recorded yet')).toBeInTheDocument()
})

it('sends local-day bounds for the date filters', async () => {
  const fn = mockApi({ ...members, [L]: () => ({ body: { items: [entry('e1')] } }) })
  renderWithQuery(<AuditLogPage slug="x" />)
  await screen.findByText('Alice')
  fireEvent.change(screen.getByLabelText('From'), { target: { value: '2026-10-01' } })
  fireEvent.change(screen.getByLabelText('To'), { target: { value: '2026-10-01' } })
  await screen.findByRole('button', { name: 'Clear filters' })
  await waitFor(() => {
    const last = gets(fn).at(-1)!
    expect(last.get('from')).toBe(new Date(2026, 9, 1).toISOString())
    expect(last.get('to')).toBe(new Date(2026, 9, 2).toISOString())
  })
})
