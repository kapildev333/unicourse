import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { AdminLayout } from './admin-layout'

it('marks only the current admin tab as active', async () => {
  const client = new QueryClient()
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'a@a.edu', displayName: 'A' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role: 'admin', departmentIds: [] }],
  })
  const root = createRootRoute()
  const slug = createRoute({ getParentRoute: () => root, path: '$slug' })
  const admin = createRoute({ getParentRoute: () => slug, path: 'admin', component: AdminLayout })
  const tabs = ['departments', 'terms', 'members', 'invitations', 'audit'].map((p) =>
    createRoute({ getParentRoute: () => admin, path: p, component: () => <p>{p} page</p> }),
  )
  const router = createRouter({
    routeTree: root.addChildren([slug.addChildren([admin.addChildren(tabs)])]),
    history: createMemoryHistory({ initialEntries: ['/a/admin/members'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  await screen.findByText('members page')
  expect(screen.getByRole('link', { name: 'Members' })).toHaveAttribute('aria-current', 'page')
  expect(screen.getByRole('link', { name: 'Members' })).toHaveClass('border-accent', 'text-accent')
  for (const name of ['Departments', 'Terms', 'Invitations', 'Audit log']) {
    const link = screen.getByRole('link', { name })
    expect(link).not.toHaveAttribute('aria-current')
    expect(link).toHaveClass('border-transparent')
    expect(link).not.toHaveClass('border-accent')
  }
})
