import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRoute, createRootRoute, createRouter } from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { Bell } from './bell'

afterEach(() => vi.unstubAllGlobals())

async function bell(count: number, at = '/a/me') {
  const fn = mockApi({ 'GET /universities/a/me/notifications/unread-count': () => ({ body: { count } }) })
  const root = createRootRoute({ component: () => <Bell slug="a" /> })
  const page = createRoute({ getParentRoute: () => root, path: '$slug/notifications', component: () => null })
  const me = createRoute({ getParentRoute: () => root, path: '$slug/me', component: () => null })
  const router = createRouter({ routeTree: root.addChildren([page, me]), history: createMemoryHistory({ initialEntries: [at] }) })
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  await screen.findByRole('link', { name: /^Notifications/ })
  return fn
}

it('names the unread count and shows it in the pill', async () => {
  await bell(3)
  const link = await screen.findByRole('link', { name: 'Notifications, 3 unread' })
  expect(link).toHaveAttribute('title', 'Notifications')
  expect(link).toHaveAttribute('href', '/a/notifications')
  expect(link).toHaveTextContent('3')
})

it('caps the pill at 99+', async () => {
  await bell(250)
  expect(await screen.findByText('99+')).toBeInTheDocument()
})

it('shows no pill at zero', async () => {
  const fn = await bell(0)
  await vi.waitFor(() => expect(fn).toHaveBeenCalled())
  const link = screen.getByRole('link', { name: 'Notifications' })
  expect(link).toHaveTextContent('')
})

it('is the current page on the notifications page', async () => {
  await bell(0, '/a/notifications')
  expect(screen.getByRole('link', { name: 'Notifications' })).toHaveAttribute('aria-current', 'page')
})
