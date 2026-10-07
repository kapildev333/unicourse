import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { CoursesLayout } from './courses-layout'

it('renders the page title above the tabs', async () => {
  const client = new QueryClient()
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'a@a.edu', displayName: 'A' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role: 'admin', departmentIds: [] }],
  })
  const root = createRootRoute()
  const slug = createRoute({ getParentRoute: () => root, path: '$slug' })
  const courses = createRoute({ getParentRoute: () => slug, path: 'courses', component: CoursesLayout })
  const index = createRoute({ getParentRoute: () => courses, path: '/', component: () => <p>list</p> })
  const router = createRouter({
    routeTree: root.addChildren([slug.addChildren([courses.addChildren([index])])]),
    history: createMemoryHistory({ initialEntries: ['/a/courses'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  const h1 = await screen.findByRole('heading', { level: 1, name: 'Courses' })
  const nav = screen.getByRole('navigation', { name: 'Courses sections' })
  expect(h1.compareDocumentPosition(nav) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
})
