import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRoute, createRootRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import { NotificationsPage } from './notifications-page'

afterEach(() => vi.unstubAllGlobals())

const base = { section: { id: 's1', number: '001' }, course: { code: 'CS101', title: 'Intro to CS' }, createdAt: '2026-10-09T09:00:00Z' }
const items = [
  { ...base, id: 'n1', kind: 'announcement', subject: 'Midterm moved', announcementId: 'a1' },
  { ...base, id: 'n2', kind: 'assessment_created', subject: 'Quiz 1', assessment: { id: 'q1', kind: 'quiz' } },
  { ...base, id: 'n3', kind: 'assessment_created', subject: 'Essay', assessment: { id: 'as1', kind: 'assignment' } },
  { ...base, id: 'n4', kind: 'scores_published', subject: 'Quiz 1', assessment: { id: 'q1', kind: 'quiz' } },
  { ...base, id: 'n5', kind: 'scores_published', subject: 'Lab', assessment: { id: 'm1', kind: 'manual' } },
  { ...base, id: 'n6', kind: 'grades_finalized', readAt: '2026-10-09T10:00:00Z' },
]
const L = 'GET /universities/a/me/notifications'
const C = 'GET /universities/a/me/notifications/unread-count'

function setup(routes: Routes, count = 5) {
  const fn = mockApi({ [C]: () => ({ body: { count } }), ...routes })
  const root = createRootRoute()
  const mk = (path: string) => createRoute({ getParentRoute: () => root, path, component: () => <p>at {path}</p> })
  const router = createRouter({
    routeTree: root.addChildren([
      createRoute({ getParentRoute: () => root, path: '$slug/notifications', component: () => <NotificationsPage slug="a" /> }),
      ...['$slug/sections/$id/announcements', '$slug/sections/$id/quizzes/$assessmentId', '$slug/sections/$id/assignments/$assessmentId', '$slug/grades'].map(mk),
    ]),
    history: createMemoryHistory({ initialEntries: ['/a/notifications'] }),
  })
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  return { fn, router }
}
const err = (status: number) => ({ status, body: { error: { code: 'internal', message: 'm', details: [] } } })

it('lists each kind with its copy and target', async () => {
  setup({ [L]: () => ({ body: { items } }) })
  const list = await screen.findByRole('list', { name: 'Notifications' })
  const link = (name: string) => within(list).getByRole('link', { name })
  expect(link('New announcement: Midterm moved')).toHaveAttribute('href', '/a/sections/s1/announcements')
  expect(link('New quiz: Quiz 1')).toHaveAttribute('href', '/a/sections/s1/quizzes/q1')
  expect(link('New assignment: Essay')).toHaveAttribute('href', '/a/sections/s1/assignments/as1')
  expect(link('Grades released: Quiz 1')).toHaveAttribute('href', '/a/sections/s1/quizzes/q1')
  expect(link('Grades released: Lab')).toHaveAttribute('href', '/a/grades')
  expect(link('Final grades posted for Intro to CS')).toHaveAttribute('href', '/a/grades')
  expect(screen.getAllByText(/CS101-001/)).toHaveLength(6)
  expect(screen.getAllByText('New')).toHaveLength(5) // n6 is read
  expect(screen.getAllByRole('button', { name: /^Mark as read/ })).toHaveLength(5)
})

it('Mark as read sends PATCH read:true', async () => {
  const { fn } = setup({
    [L]: () => ({ body: { items: [items[0]] } }),
    'PATCH /universities/a/me/notifications/n1': () => ({ body: { ...items[0], readAt: '2026-10-09T10:00:00Z' } }),
  })
  fireEvent.click(await screen.findByRole('button', { name: /^Mark as read/ }))
  await waitFor(() => expect(fn.mock.calls.some(([r]) => r.method === 'PATCH')).toBe(true))
  const req = fn.mock.calls.find(([r]) => r.method === 'PATCH')![0]
  expect(await req.json()).toEqual({ read: true })
})

it('following the link marks it read and navigates to the source', async () => {
  const { fn, router } = setup({
    [L]: () => ({ body: { items: [items[0]] } }),
    'PATCH /universities/a/me/notifications/n1': () => ({ body: items[0] }),
  })
  fireEvent.click(await screen.findByRole('link', { name: 'New announcement: Midterm moved' }))
  await waitFor(() => expect(router.state.location.pathname).toBe('/a/sections/s1/announcements'))
  await waitFor(() => expect(fn.mock.calls.some(([r]) => r.method === 'PATCH')).toBe(true))
})

it('Mark all as read sends PATCH and shows the status', async () => {
  const { fn } = setup({
    [L]: () => ({ body: { items } }),
    'PATCH /universities/a/me/notifications': () => ({ status: 204 }),
  })
  await waitFor(() => expect(screen.getByRole('button', { name: 'Mark all as read' })).toBeEnabled())
  fireEvent.click(screen.getByRole('button', { name: 'Mark all as read' }))
  expect(await screen.findByText('All notifications marked as read.')).toBeInTheDocument()
  const req = fn.mock.calls.find(([r]) => r.method === 'PATCH')![0]
  expect(await req.json()).toEqual({ read: true })
})

it('disables Mark all as read at zero unread', async () => {
  setup({ [L]: () => ({ body: { items: [items[5]] } }) }, 0)
  await screen.findByRole('list', { name: 'Notifications' })
  expect(screen.getByRole('button', { name: 'Mark all as read' })).toBeDisabled()
})

it('shows the empty state', async () => {
  setup({ [L]: () => ({ body: { items: [] } }) }, 0)
  expect(await screen.findByText('No notifications yet')).toBeInTheDocument()
  expect(screen.getByText(/appear here/)).toBeInTheDocument()
})

it('loads more with the cursor', async () => {
  const seen: (string | null)[] = []
  setup({
    [L]: (req) => {
      const c = new URL(req.url).searchParams.get('cursor')
      seen.push(c)
      return { body: c ? { items: [items[1]] } : { items: [items[0]], nextCursor: 'c2' } }
    },
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Load more' }))
  expect(await screen.findByText('New quiz: Quiz 1')).toBeInTheDocument()
  expect(seen).toEqual([null, 'c2'])
})

it('shows an error banner when the list fails', async () => {
  setup({ [L]: () => err(403) })
  expect(await screen.findByText("Couldn't load your notifications")).toBeInTheDocument()
})

it('shows a failure to mark one as read', async () => {
  setup({
    [L]: () => ({ body: { items: [items[0]] } }),
    'PATCH /universities/a/me/notifications/n1': () => err(500),
  })
  fireEvent.click(await screen.findByRole('button', { name: /^Mark as read/ }))
  expect(await screen.findByText("Couldn't mark notifications as read")).toBeInTheDocument()
})
