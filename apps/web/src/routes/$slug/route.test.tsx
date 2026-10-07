import {
  RouterProvider,
  createMemoryHistory,
  createRoute,
  createRootRoute,
  createRouter,
} from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { expect, it } from 'vitest'
import { mockApi } from '../../test-utils'
import { Route } from './route'

async function noAccess(universities: string[]) {
  mockApi({
    'GET /me': () => ({
      body: {
        user: { id: 'u', email: 'dean@harbor.edu', displayName: 'Dean' },
        memberships: universities.map((name) => ({
          university: {
            id: name,
            slug: name.toLowerCase().replace(/\W+/g, '-'),
            name,
          },
          role: 'admin',
          departmentIds: [],
        })),
      },
    }),
  })
  const root = createRootRoute()
  const layout = (Route as never as { update: (o: object) => typeof Route }).update({
    id: '/$slug',
    path: '/$slug',
    getParentRoute: () => root,
  })
  const me = createRoute({
    getParentRoute: () => layout,
    path: 'me',
    component: () => null,
  })
  const router = createRouter({
    routeTree: root.addChildren([layout.addChildren([me])]),
    history: createMemoryHistory({ initialEntries: ['/northfield/me'] }),
  })
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  await screen.findByRole('heading', {
    name: "You don't have access to this university",
  })
  return router
}

it('offers one primary button for a single membership', async () => {
  const router = await noAccess(['Harbor College'])
  expect(screen.getByText(/this university's address/)).toBeInTheDocument()
  expect(screen.queryByText('Go to your university:')).toBeNull()
  fireEvent.click(screen.getByRole('button', { name: 'Go to Harbor College' }))
  await waitFor(() => expect(router.state.location.pathname).toBe('/harbor-college/me'))
  expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument()
})

it('lists every university under the lead-in when there are several', async () => {
  await noAccess(['Harbor College', 'Bay Institute'])
  expect(screen.getByText('Go to your university:')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Bay Institute' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /^Go to / })).toBeNull()
})
