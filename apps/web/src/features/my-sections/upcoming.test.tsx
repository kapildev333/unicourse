import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { cleanup, render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { setCampus } from '../../lib/format'
import { mockApi } from '../../test-utils'
import { Upcoming } from './upcoming'

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['Date'] })
  vi.setSystemTime(new Date('2026-10-12T12:00:00Z'))
  setCampus({ timezone: 'UTC', locale: 'en-US' })
})
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
  setCampus()
  cleanup()
})

const U = 'GET /universities/a/me/upcoming'
const base = {
  section: { id: 's1', number: '001' },
  course: { code: 'CS101', title: 'Intro' },
  started: false,
  opensAt: '2026-10-01T00:00:00Z',
}
const item = (o: object) => ({ assessmentId: 'q1', kind: 'quiz', name: 'Quiz 1', myRole: 'student', dueAt: '2026-10-14T15:00:00Z', ...base, ...o })

function renderIt() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute()
  const r = createRoute({ getParentRoute: () => root, path: '$slug', component: () => <Upcoming slug="a" /> })
  const stub = (path: string) => createRoute({ getParentRoute: () => root, path, component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([
      r,
      stub('$slug/sections/$id/quizzes/$assessmentId'),
      stub('$slug/sections/$id/assignments/$assessmentId'),
      stub('$slug/sections/$id/quizzes/$assessmentId/attempts'),
      stub('$slug/sections/$id/assignments/$assessmentId/submissions'),
    ]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}

it('student rows: copy per kind, badges, links', async () => {
  mockApi({
    [U]: () => ({
      body: {
        items: [
          item({ started: true }),
          item({ assessmentId: 'a1', kind: 'assignment', name: 'Essay', dueAt: '2026-10-12T11:00:00Z', lateUntil: '2026-10-13T11:00:00Z' }),
          item({ assessmentId: 'q2', name: 'Quiz 2', opensAt: '2026-10-13T09:00:00Z' }),
        ],
      },
    }),
  })
  renderIt()
  const list = await screen.findByRole('list', { name: 'Upcoming work' })
  const [quiz, essay, later] = within(list).getAllByRole('listitem')
  expect(quiz).toHaveTextContent('CS101-001 · Quiz · Closes Oct 14, 2026, 3:00 PM')
  expect(within(quiz).getByText('In progress')).toBeInTheDocument()
  expect(within(quiz).getByRole('link', { name: 'Quiz 1' })).toHaveAttribute('href', '/a/sections/s1/quizzes/q1')
  expect(essay).toHaveTextContent('Assignment · Late until Oct 13, 2026, 11:00 AM')
  expect(within(essay).getByText('Past due')).toBeInTheDocument()
  expect(within(essay).getByRole('link', { name: 'Essay' })).toHaveAttribute('href', '/a/sections/s1/assignments/a1')
  expect(later).toHaveTextContent('Opens Oct 13, 2026, 9:00 AM')
})

it('staff rows: counts and review links', async () => {
  mockApi({
    [U]: () => ({
      body: {
        items: [
          item({ myRole: 'instructor', submittedCount: 3, enrolledCount: 10 }),
          item({ assessmentId: 'a1', kind: 'assignment', name: 'Essay', myRole: 'ta', submittedCount: 0, enrolledCount: 10 }),
        ],
      },
    }),
  })
  renderIt()
  const [quiz, essay] = await screen.findAllByRole('listitem')
  expect(within(quiz).getByText('3 of 10 submitted')).toBeInTheDocument()
  expect(within(quiz).getByRole('link', { name: 'Quiz 1' })).toHaveAttribute('href', '/a/sections/s1/quizzes/q1/attempts')
  expect(within(essay).getByRole('link', { name: 'Essay' })).toHaveAttribute('href', '/a/sections/s1/assignments/a1/submissions')
})

it('empty state and error banner', async () => {
  mockApi({ [U]: () => ({ body: { items: [] } }) })
  renderIt()
  expect(await screen.findByText('Nothing due in the next 14 days.')).toBeInTheDocument()
  cleanup()
  mockApi({ [U]: () => ({ status: 500, body: { error: { code: 'internal', message: 'boom', details: [] } } }) })
  renderIt()
  expect(await screen.findByText("Couldn't load what's due")).toBeInTheDocument()
})
