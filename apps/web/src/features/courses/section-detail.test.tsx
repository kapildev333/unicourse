import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { SectionDetailPage } from './section-detail'

afterEach(() => vi.unstubAllGlobals())

const term = { id: 't1', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }
const section = (meetings: unknown[]) => ({
  id: 's1',
  courseId: 'c1',
  termId: 't1',
  number: '001',
  course: { id: 'c1', departmentId: 'd1', code: 'CS101', title: 'Intro' },
  term,
  staff: [],
  pendingStaff: [],
  meetings,
})

function renderPage(meetings: unknown[], mine: unknown[]) {
  mockApi({
    'GET /universities/a/sections/s1': () => ({ body: section(meetings) }),
    'GET /universities/a/me/sections': () => ({ body: { terms: [{ term, sections: mine }] } }),
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role: 'member', departmentIds: [] }],
  })
  const root = createRootRoute()
  const r = createRoute({
    getParentRoute: () => root,
    path: '$slug',
    component: () => <SectionDetailPage slug="a" sectionId="s1" />,
  })
  const me = createRoute({ getParentRoute: () => r, path: 'me', component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([r.addChildren([me])]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}

it('lists meetings and shows the Student badge for an enrolled member', async () => {
  renderPage(
    [{ id: 'm1', weekday: 1, startsAt: '10:00', endsAt: '11:15', location: 'Room 101' }],
    [{ id: 's1', myRole: 'student' }],
  )
  expect(await screen.findByText('Monday 10:00–11:15 · Room 101')).toBeInTheDocument()
  expect(await screen.findByText('Student')).toBeInTheDocument()
})

it('says so without meetings, and a non-enrolled member gets no badge', async () => {
  renderPage([], [])
  expect(await screen.findByText('No meeting times yet.')).toBeInTheDocument()
  expect(screen.queryByText('Student')).not.toBeInTheDocument()
})
