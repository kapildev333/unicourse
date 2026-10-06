import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { Roster } from './roster'

afterEach(() => vi.unstubAllGlobals())

const erin = { userId: 'u-erin', email: 'erin@a.edu', displayName: 'Erin Moe', createdAt: '2026-09-01T10:00:00Z' }
const sam = { userId: 'u-sam', email: 'sam@a.edu', displayName: '', createdAt: '2026-09-02T10:00:00Z' }
const E = 'GET /universities/a/sections/s1/enrollments'

it('lists the roster and removes a student after confirmation', async () => {
  let items = [erin, sam]
  const fn = mockApi({
    [E]: () => ({ body: { items } }),
    'DELETE /universities/a/sections/s1/enrollments/u-erin': () => {
      items = [sam]
      return { status: 204 }
    },
  })
  renderWithQuery(<Roster slug="a" sectionId="s1" canEdit />)
  expect(await screen.findByText('Erin Moe', { selector: 'td' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Import roster' })).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Remove Erin Moe from the roster' }))
  const dlg = await screen.findByRole('dialog', { name: 'Remove Erin Moe from this section?' })
  // jsdom has no dialog autofocus; Cancel is the first focusable, which a real browser focuses
  expect(within(dlg).getAllByRole('button')[0]).toHaveTextContent('Cancel')
  fireEvent.click(within(dlg).getByRole('button', { name: 'Remove' }))
  expect(await screen.findByText(/Erin Moe removed from the roster\./)).toBeInTheDocument()
  await waitFor(() => expect(screen.queryByText('Erin Moe', { selector: 'td' })).toBeNull())
  expect(fn.mock.calls.some((c) => (c[0] as Request).method === 'DELETE')).toBe(true)
})

it('is read-only without canEdit', async () => {
  mockApi({ [E]: () => ({ body: { items: [erin] } }) })
  renderWithQuery(<Roster slug="a" sectionId="s1" canEdit={false} />)
  expect(await screen.findByText('Erin Moe')).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /Remove/ })).toBeNull()
  expect(screen.queryByRole('button', { name: 'Import roster' })).toBeNull()
})

it('shows the empty state', async () => {
  mockApi({ [E]: () => ({ body: { items: [] } }) })
  renderWithQuery(<Roster slug="a" sectionId="s1" canEdit={false} />)
  expect(await screen.findByText('No students yet')).toBeInTheDocument()
})
