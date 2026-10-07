import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { render } from '@testing-library/react'
import type { Section } from './queries'

export const course = { id: 'c1', departmentId: 'd1', code: 'CS101', title: 'Intro', credits: 3 }
export const term = { id: 't1', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }

export const sectionFixture = (over: Partial<Section> = {}): Section => ({
  id: 's1',
  courseId: 'c1',
  termId: 't1',
  number: '001',
  course,
  term,
  staff: [{ userId: 'u-ins', role: 'instructor', email: 'ins@a.edu', displayName: 'Ida Ins' }],
  pendingStaff: [],
  meetings: [],
  description: '',
  ...over,
})

export const mine = (role?: 'student' | 'ta' | 'instructor') => ({
  terms: role ? [{ term, sections: [{ id: 's1', myRole: role }] }] : [],
})

/** Memory router with the section routes; `ui` renders where `path` matches. */
export function renderSection(opts: {
  path: string
  ui: React.ReactElement
  userId?: string
  role?: 'admin' | 'dept_admin' | 'member'
  departmentIds?: string[]
}) {
  const { path, ui, userId = 'u0', role = 'member', departmentIds = [] } = opts
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: userId, email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role, departmentIds }],
  })
  const root = createRootRoute()
  const slug = createRoute({ getParentRoute: () => root, path: '$slug' })
  const at = (p: string) =>
    createRoute({ getParentRoute: () => slug, path: p, component: () => (path === `/a/${p.replace('$id', 's1')}` ? ui : null) })
  const router = createRouter({
    routeTree: root.addChildren([
      slug.addChildren([
        createRoute({ getParentRoute: () => slug, path: 'me', component: () => null }),
        createRoute({ getParentRoute: () => slug, path: 'courses/sections', component: () => null }),
        at('sections/$id'),
        at('sections/$id/announcements'),
        at('sections/$id/materials'),
        at('sections/$id/participants'),
        at('sections/$id/grades'),
      ]),
    ]),
    history: createMemoryHistory({ initialEntries: [path] }),
  })
  const r = render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  return { client, router, ...r }
}
