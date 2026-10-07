import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { TextLink, textLink } from './text-link'

it('is a router link with the shared text-link class and passes params through', async () => {
  const root = createRootRoute()
  const page = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: () => (
      <TextLink to="/$slug/me" params={{ slug: 'a' }}>
        Go
      </TextLink>
    ),
  })
  const stub = createRoute({ getParentRoute: () => root, path: '$slug/me', component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([page, stub]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(<RouterProvider router={router} />)
  const link = await screen.findByRole('link', { name: 'Go' })
  expect(link).toHaveAttribute('href', '/a/me')
  expect(link).toHaveClass(...textLink.split(' '))
})
