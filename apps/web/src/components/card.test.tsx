import { Link, RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { render, screen, within } from '@testing-library/react'
import { expect, it } from 'vitest'
import { RoleBadge } from './badge'
import { Card } from './card'

it('is one link named by code and title, inside one list item, carrying meta, badge and note', async () => {
  const root = createRootRoute()
  const page = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: () => (
      <ul>
        <Card
          code="CS101-001"
          title="Intro to CS"
          meta={['Dana Poe', 'Mon 10:00–11:15 · Room 101']}
          // eslint-disable-next-line jsx-a11y/aria-role -- RoleBadge's existing prop is named role
          badge={<RoleBadge role="ta" />}
          note="Clashes with CS102-001"
          params={{ slug: 'a', id: 's1' }}
        />
      </ul>
    ),
  })
  const stub = createRoute({ getParentRoute: () => root, path: '$slug/sections/$id', component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([page, stub]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(<RouterProvider router={router} />)
  const link = await screen.findByRole('link', { name: 'CS101-001 Intro to CS' })
  expect(link).toHaveAttribute('href', '/a/sections/s1')
  expect(screen.getAllByRole('listitem')).toHaveLength(1)
  expect(screen.getByRole('listitem')).toContainElement(link)
  for (const t of ['Dana Poe', 'Mon 10:00–11:15 · Room 101', 'TA', 'Clashes with CS102-001'])
    expect(within(link).getByText(t)).toBeInTheDocument()
  expect(screen.getByRole('heading', { level: 3, name: 'Intro to CS' })).toBeInTheDocument()
})

it('badge row is pushed to the foot; a footer link sits outside the main link', async () => {
  const root = createRootRoute()
  const page = createRoute({
    getParentRoute: () => root,
    path: '/',
    component: () => (
      <ul>
        <Card
          code="CS101-001"
          title="Intro"
          meta={[]}
          badge={<span>Instructor</span>}
          footer={<Link to="/$slug/sections/$id" params={{ slug: 'a', id: 's1' }}>Gradebook</Link>}
          params={{ slug: 'a', id: 's1' }}
        />
      </ul>
    ),
  })
  const stub = createRoute({ getParentRoute: () => root, path: '$slug/sections/$id', component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([page, stub]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(<RouterProvider router={router} />)
  const main = await screen.findByRole('link', { name: 'CS101-001 Intro' })
  const foot = screen.getByRole('link', { name: 'Gradebook' })
  expect(screen.getAllByRole('link')).toHaveLength(2)
  expect(main).not.toContainElement(foot)
  expect(screen.getByRole('listitem')).toContainElement(foot)
  expect(screen.getByText('Instructor').parentElement).toHaveClass('mt-auto')
})
