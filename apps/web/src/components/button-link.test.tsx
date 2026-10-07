import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { Button, base, sizes, variants } from './button'
import { ButtonLink } from './button-link'

it('is a link with the Button class recipe and passes params through', async () => {
  const root = createRootRoute()
  const page = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: () => (
      <>
        <ButtonLink to="/$slug/me" params={{ slug: 'a' }} variant="secondary" size="sm">Go</ButtonLink>
        <Button variant="secondary" size="sm">Plain</Button>
      </>
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
  const recipe = `${base} ${sizes.sm} ${variants.secondary}`.split(' ')
  expect(link).toHaveClass(...recipe)
  expect(screen.getByRole('button', { name: 'Plain' })).toHaveClass(...recipe)
})
