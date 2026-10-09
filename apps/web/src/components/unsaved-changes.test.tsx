import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter, Link } from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { expect, it } from 'vitest'
import { UnsavedChangesGuard } from './unsaved-changes'

function setup(when: boolean) {
  const root = createRootRoute()
  const router = createRouter({
    routeTree: root.addChildren([
      createRoute({
        getParentRoute: () => root,
        path: '/',
        component: () => (
          <>
            <UnsavedChangesGuard when={when} />
            <Link to={'/other' as never}>Leave</Link>
          </>
        ),
      }),
      createRoute({ getParentRoute: () => root, path: '/other', component: () => <p>Other page</p> }),
    ]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(<RouterProvider router={router} />)
}

it('lets navigation pass when there is nothing unsaved', async () => {
  setup(false)
  fireEvent.click(await screen.findByRole('link', { name: 'Leave' }))
  expect(await screen.findByText('Other page')).toBeInTheDocument()
})

it('asks first; Keep editing stays, Discard changes leaves', async () => {
  setup(true)
  fireEvent.click(await screen.findByRole('link', { name: 'Leave' }))
  expect(await screen.findByRole('dialog', { name: 'Discard unsaved changes?' })).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Keep editing' }))
  await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
  expect(screen.queryByText('Other page')).toBeNull()
  fireEvent.click(screen.getByRole('link', { name: 'Leave' }))
  fireEvent.click(await screen.findByRole('button', { name: 'Discard changes' }))
  expect(await screen.findByText('Other page')).toBeInTheDocument()
})
