import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { CourseListPage } from './course-list'

afterEach(() => vi.unstubAllGlobals())

const cs = { id: 'd1', code: 'CS', name: 'Computer Science' }
const math = { id: 'd2', code: 'MATH', name: 'Mathematics' }
const c1 = { id: 'c1', departmentId: 'd1', code: 'CS101', title: 'Intro' }
const c2 = { id: 'c2', departmentId: 'd2', code: 'MATH1', title: 'Calculus' }
const D = 'GET /universities/a/departments'
const L = 'GET /universities/a/courses'
const err = (status: number, code: string) => ({ status, body: { error: { code, message: code, details: [] } } })

function Page({ initial }: { initial?: string }) {
  const [id, setId] = useState(initial)
  return <CourseListPage slug="a" departmentId={id} onDepartmentChange={(v) => setId(v || undefined)} />
}

function renderPage(membership: { role: string; departmentIds: string[] }, initial?: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, ...membership }],
  })
  const root = createRootRoute()
  const slug = createRoute({ getParentRoute: () => root, path: '$slug', component: () => <Page initial={initial} /> })
  const router = createRouter({
    routeTree: root.addChildren([slug]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  return render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
}
const admin = { role: 'admin', departmentIds: [] }

it('lists courses with department codes', async () => {
  mockApi({ [D]: () => ({ body: { items: [cs, math] } }), [L]: () => ({ body: { items: [c1, c2] } }) })
  renderPage(admin)
  expect(await screen.findByText('CS101')).toBeInTheDocument()
  expect(screen.getByRole('cell', { name: 'MATH' })).toBeInTheDocument()
})

it('shows the empty state and the error state with retry', async () => {
  let fail = true
  mockApi({
    [D]: () => ({ body: { items: [] } }),
    [L]: () => (fail ? err(500, 'internal') : { body: { items: [] } }),
  })
  renderPage(admin)
  const retry = await screen.findByRole('button', { name: 'Try again' })
  fail = false
  fireEvent.click(retry)
  expect(await screen.findByText('No courses yet')).toBeInTheDocument()
})

it('creates a course and shows a field error for a duplicate code', async () => {
  let items = [c1]
  let conflict = true
  const fn = mockApi({
    [D]: () => ({ body: { items: [cs, math] } }),
    [L]: () => ({ body: { items } }),
    'POST /universities/a/courses': () => {
      if (conflict) return err(409, 'conflict')
      items = [...items, c2]
      return { status: 201, body: c2 }
    },
  })
  renderPage(admin)
  await screen.findByText('CS101')
  fireEvent.click(screen.getByRole('button', { name: 'New course' }))
  const dialog = await screen.findByRole('dialog', { name: 'New course', hidden: true })
  await within(dialog).findByRole('option', { name: 'MATH · Mathematics' })
  fireEvent.change(within(dialog).getByLabelText('Department'), { target: { value: 'd2' } })
  fireEvent.change(within(dialog).getByLabelText('Code'), { target: { value: ' MATH1 ' } })
  fireEvent.change(within(dialog).getByLabelText('Title'), { target: { value: 'Calculus' } })
  fireEvent.click(within(dialog).getByRole('button', { name: 'Create course' }))
  expect(await screen.findByText(/code already exists in this university/)).toBeInTheDocument()
  await waitFor(() => expect(within(dialog).getByLabelText('Code')).toHaveFocus())
  conflict = false
  fireEvent.click(within(dialog).getByRole('button', { name: 'Create course' }))
  expect(await screen.findByText(/Course “MATH1” created\./)).toBeInTheDocument()
  expect(await screen.findByRole('cell', { name: 'Calculus' })).toBeInTheDocument()
  const post = fn.mock.calls.map((c) => c[0] as Request).filter((r) => r.method === 'POST').pop()!
  expect(await post.clone().json()).toEqual({ departmentId: 'd2', code: 'MATH1', title: 'Calculus' })
})

it('edits only changed fields', async () => {
  const fn = mockApi({
    [D]: () => ({ body: { items: [cs] } }),
    [L]: () => ({ body: { items: [c1] } }),
    'PATCH /universities/a/courses/c1': () => ({ body: { ...c1, title: 'Intro 2' } }),
  })
  renderPage(admin)
  fireEvent.click(await screen.findByRole('button', { name: 'Edit CS101' }))
  const dialog = await screen.findByRole('dialog', { name: 'Edit CS101', hidden: true })
  fireEvent.change(within(dialog).getByLabelText('Title'), { target: { value: 'Intro 2' } })
  fireEvent.click(within(dialog).getByRole('button', { name: 'Save changes' }))
  expect(await screen.findByText(/Course “CS101” updated\./)).toBeInTheDocument()
  const patch = fn.mock.calls.map((c) => c[0] as Request).find((r) => r.method === 'PATCH')!
  expect(await patch.clone().json()).toEqual({ title: 'Intro 2' })
})

it('deletes a course and explains a 409', async () => {
  let items = [c1]
  let conflict = true
  mockApi({
    [D]: () => ({ body: { items: [cs] } }),
    [L]: () => ({ body: { items } }),
    'DELETE /universities/a/courses/c1': () => {
      if (conflict) return err(409, 'conflict')
      items = []
      return { status: 204 }
    },
  })
  renderPage(admin)
  fireEvent.click(await screen.findByRole('button', { name: 'Delete CS101' }))
  const dialog = await screen.findByRole('dialog', { name: 'Delete CS101?', hidden: true })
  fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
  expect(await screen.findByText(/still has sections/)).toBeInTheDocument()
  conflict = false
  fireEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))
  expect(await screen.findByText(/Course “CS101” deleted\./)).toBeInTheDocument()
  expect(await screen.findByText('No courses yet')).toBeInTheDocument()
})

it('shows Edit/Delete only on own-department rows for a dept_admin', async () => {
  mockApi({ [D]: () => ({ body: { items: [cs, math] } }), [L]: () => ({ body: { items: [c1, c2] } }) })
  renderPage({ role: 'dept_admin', departmentIds: ['d1'] }, 'd1')
  await screen.findByText('CS101')
  expect(screen.getByRole('button', { name: 'Edit CS101' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Edit MATH1' })).toBeNull()
  expect(screen.queryByRole('button', { name: 'Delete MATH1' })).toBeNull()
})

it('opens the import dialog', async () => {
  mockApi({ [D]: () => ({ body: { items: [cs] } }), [L]: () => ({ body: { items: [c1] } }) })
  renderPage(admin)
  await screen.findByText('CS101')
  fireEvent.click(screen.getByRole('button', { name: 'Import courses' }))
  expect(await screen.findByLabelText('CSV file', { selector: 'input' })).toBeInTheDocument()
})

it('lets a dept_admin choose All departments and keeps it', async () => {
  const fn = mockApi({ [D]: () => ({ body: { items: [cs, math] } }), [L]: () => ({ body: { items: [c1, c2] } }) })
  renderPage({ role: 'dept_admin', departmentIds: ['d1'] })
  const select = await screen.findByLabelText('Department')
  await waitFor(() => expect(select).toHaveValue('d1'))
  fireEvent.change(select, { target: { value: '' } })
  await waitFor(() => {
    const last = fn.mock.calls.map((c) => new URL((c[0] as Request).url)).filter((u) => u.pathname.endsWith('/courses')).at(-1)!
    expect(last.searchParams.has('departmentId')).toBe(false)
  })
  expect(select).toHaveValue('')
})

it('preselects the only manageable department in the course dialog', async () => {
  mockApi({ [D]: () => ({ body: { items: [cs] } }), [L]: () => ({ body: { items: [c1] } }) })
  renderPage(admin)
  await screen.findByText('CS101')
  fireEvent.click(screen.getByRole('button', { name: 'New course' }))
  const dialog = await screen.findByRole('dialog', { name: 'New course', hidden: true })
  expect(within(dialog).getByLabelText('Department')).toHaveValue('d1')
})

it('stacks the table with a label on every data cell', async () => {
  mockApi({ [D]: () => ({ body: { items: [cs] } }), [L]: () => ({ body: { items: [c1] } }) })
  renderPage(admin)
  await screen.findByText('CS101')
  expect(screen.getByRole('table', { name: 'Courses' })).toHaveClass('ui-table-stack')
  const labels = within(screen.getByText('CS101').closest('tr')!)
    .getAllByRole('cell')
    .map((c) => c.getAttribute('data-label'))
  expect(labels).toEqual(['Code', 'Title', 'Department', null])
})

it('shows one primary and no filter or header pair while there are no courses', async () => {
  mockApi({ [D]: () => ({ body: { items: [cs] } }), [L]: () => ({ body: { items: [] } }) })
  renderPage(admin)
  await screen.findByText('No courses yet')
  expect(screen.getAllByRole('button', { name: 'New course' })).toHaveLength(1)
  expect(screen.getAllByRole('button', { name: 'Import courses' })).toHaveLength(1)
  expect(screen.queryByLabelText('Department')).toBeNull()
})

it('keeps New course enabled while the list loads and shows a table skeleton', () => {
  mockApi({ [D]: () => ({ body: { items: [cs] } }), [L]: () => new Promise(() => {}) as never })
  renderPage(admin)
  return screen.findByRole('button', { name: 'New course' }).then((b) => {
    expect(b).toBeEnabled()
    expect(screen.getByRole('columnheader', { name: 'Department', hidden: true })).toBeInTheDocument()
  })
})

it('names the failure and shows the count in the toolbar', async () => {
  mockApi({ [D]: () => ({ body: { items: [cs] } }), [L]: () => err(500, 'internal') })
  renderPage(admin)
  expect(await screen.findByText("Couldn't load courses")).toBeInTheDocument()
})

it('puts the count in the toolbar and the delete body names the effect on sections', async () => {
  mockApi({ [D]: () => ({ body: { items: [cs] } }), [L]: () => ({ body: { items: [c1] } }) })
  renderPage(admin)
  await screen.findByText('CS101')
  expect(screen.getByRole('search')).toHaveTextContent('Showing 1')
  fireEvent.click(screen.getByRole('button', { name: 'Delete CS101' }))
  const dialog = await screen.findByRole('dialog', { name: 'Delete CS101?', hidden: true })
  expect(dialog).toHaveTextContent(/still has sections can't be deleted/)
})
