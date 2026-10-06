import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { MySectionsPage } from './my-sections'

afterEach(() => vi.unstubAllGlobals())

const U = 'GET /universities/a/me/sections'
const fall = { id: 't2', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }
const spring = { id: 't1', name: 'Spring 2026', startsOn: '2026-01-10', endsOn: '2026-05-20' }
const sec = (id: string, code: string, term: typeof fall, myRole: string, staff: unknown[] = []) => ({
  id,
  courseId: `c-${code}`,
  termId: term.id,
  number: '001',
  course: { id: `c-${code}`, departmentId: 'd1', code, title: `${code} title` },
  term,
  staff,
  pendingStaff: [],
  myRole,
})
const dana = { userId: 'u1', email: 'dana@a.edu', displayName: 'Dana Poe', role: 'instructor' }
const data = {
  terms: [
    { term: fall, sections: [sec('s1', 'CS101', fall, 'ta', [dana]), sec('s2', 'CS102', fall, 'instructor')] },
    { term: spring, sections: [sec('s3', 'MATH1', spring, 'student')] },
  ],
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute()
  const me = createRoute({ getParentRoute: () => root, path: '$slug', component: () => <MySectionsPage slug="a" /> })
  const detail = createRoute({ getParentRoute: () => root, path: '$slug/sections/$id', component: () => null })
  const router = createRouter({
    routeTree: root.addChildren([me, detail]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  return render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}

it('groups sections by term in the given order with text role badges and links', async () => {
  mockApi({ [U]: () => ({ body: data }) })
  renderPage()
  await screen.findByText('CS101')
  const heads = screen.getAllByRole('heading', { level: 2 })
  expect(heads.map((h) => h.firstChild?.textContent)).toEqual(['Fall 2026', 'Spring 2026'])
  const fallRegion = screen.getByRole('region', { name: /Fall 2026/ })
  expect(within(fallRegion).getByText('CS101')).toBeInTheDocument()
  expect(within(fallRegion).queryByText('MATH1')).toBeNull()
  const ta = screen.getByText('CS101').closest('tr')!
  expect(within(ta).getByText('TA')).toBeInTheDocument()
  expect(within(ta).getByText('Dana Poe')).toBeInTheDocument()
  expect(within(screen.getByText('MATH1').closest('tr')!).getByText('Student')).toBeInTheDocument()
  expect(within(screen.getByText('CS102').closest('tr')!).getByText('Instructor')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'MATH1' })).toHaveAttribute('href', '/a/sections/s3')
})

it('shows the empty state when there are no sections', async () => {
  mockApi({ [U]: () => ({ body: { terms: [] } }) })
  renderPage()
  expect(await screen.findByText('No sections yet')).toBeInTheDocument()
})

it('shows an error with retry', async () => {
  let fail = true
  mockApi({
    [U]: () =>
      fail
        ? { status: 500, body: { error: { code: 'internal', message: 'boom', details: [] } } }
        : { body: data },
  })
  renderPage()
  const retry = await screen.findByRole('button', { name: 'Try again' })
  fail = false
  fireEvent.click(retry)
  expect(await screen.findByText('CS101')).toBeInTheDocument()
})
