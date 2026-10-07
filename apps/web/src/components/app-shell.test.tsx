import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRoute, createRootRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { expect, it } from 'vitest'
import type { Me } from '../features/auth/use-me'
import { AppShell } from './app-shell'

const uni = (slug: string) => ({ id: slug, slug, name: `${slug} University` })
const mk = (roles: ('admin' | 'dept_admin' | 'member')[]): Me => ({
  user: { id: 'u1', email: 'a@a.edu', displayName: 'A' } as Me['user'],
  memberships: roles.map((role, i) => ({
    university: uni(`u${i}`),
    role,
    departmentIds: [],
  })),
})

async function shell(me: Me) {
  const root = createRootRoute({
    component: () => (
      <AppShell me={me} membership={me.memberships[0]}>
        <p>content</p>
      </AppShell>
    ),
  })
  const stub = createRoute({
    getParentRoute: () => root,
    path: '$slug/catalog',
    component: () => null,
  })
  const router = createRouter({
    routeTree: root.addChildren([stub]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  await screen.findByText('content')
  return router
}

it('hides Admin and Courses from members', async () => {
  await shell(mk(['member']))
  expect(screen.getByRole('link', { name: 'My sections' })).toBeInTheDocument()
  expect(screen.queryByRole('link', { name: 'Admin' })).toBeNull()
  expect(screen.queryByRole('link', { name: 'Courses' })).toBeNull()
})

it('shows Admin and Courses to admins', async () => {
  await shell(mk(['admin']))
  expect(screen.getByRole('link', { name: 'Admin' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Courses' })).toBeInTheDocument()
})

it('shows the university switcher only with several memberships', async () => {
  await shell(mk(['admin']))
  expect(screen.queryByLabelText('University')).toBeNull()
})

it('shows the university switcher with two memberships', async () => {
  await shell(mk(['admin', 'member']))
  expect(screen.getByLabelText('University')).toBeInTheDocument()
})

it('shows Catalog to everyone', async () => {
  await shell(mk(['member']))
  expect(screen.getByRole('link', { name: 'Catalog' })).toBeInTheDocument()
})

it('the header search opens the catalog filtered by the typed text', async () => {
  const router = await shell(mk(['member']))
  const box = screen.getByRole('searchbox', { name: 'Find a course' })
  fireEvent.change(box, { target: { value: '  MA201 ' } })
  fireEvent.submit(screen.getByRole('search', { name: 'Catalog' }))
  await waitFor(() => expect(router.state.location.pathname).toBe('/u0/catalog'))
  expect(router.state.location.search).toEqual({ q: 'MA201' })
})
