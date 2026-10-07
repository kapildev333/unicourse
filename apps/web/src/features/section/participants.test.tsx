import { screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { ParticipantsPage } from './participants'
import { mine, renderSection, sectionFixture } from './test-helpers'

afterEach(() => vi.unstubAllGlobals())

function setup(o: { userId?: string; role?: 'admin'; mine?: ReturnType<typeof mine> }) {
  mockApi({
    'GET /universities/a/sections/s1': () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: o.mine ?? mine() }),
    'GET /universities/a/sections/s1/enrollments': () => ({
      body: { items: [] },
    }),
  })
  renderSection({
    path: '/a/sections/s1/participants',
    ui: <ParticipantsPage slug="a" sectionId="s1" />,
    ...o,
  })
}

it('shows staff and roster to an instructor, with Participants current', async () => {
  setup({ userId: 'u-ins' })
  expect(await screen.findByRole('heading', { name: /^Staff/ })).toBeInTheDocument()
  expect(await screen.findByRole('heading', { name: 'Roster' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Participants' })).toHaveAttribute('aria-current', 'page')
})

it('shows a student the staff only', async () => {
  setup({ mine: mine('student') })
  expect(await screen.findByRole('heading', { name: /^Staff/ })).toBeInTheDocument()
  expect(screen.queryByRole('heading', { name: 'Roster' })).toBeNull()
})

it('shows a plain member the staff read-only, as before the split', async () => {
  setup({})
  expect(await screen.findByRole('heading', { name: /^Staff/ })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Add TA' })).toBeNull()
  expect(screen.queryByRole('heading', { name: 'Roster' })).toBeNull()
})
