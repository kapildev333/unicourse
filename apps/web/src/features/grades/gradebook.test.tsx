import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { GradebookPage } from './gradebook'
import { usPreset } from './presets'

afterEach(() => vi.unstubAllGlobals())

const term = { id: 't1', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }
const section = (staff: unknown[] = []) => ({
  id: 's1',
  courseId: 'c1',
  termId: 't1',
  number: '001',
  course: { id: 'c1', departmentId: 'd1', code: 'CS101', title: 'Intro to CS', credits: 3 },
  term,
  staff,
  pendingStaff: [],
  meetings: [],
})
const asm = (id: string, name: string, maxPoints: number, weight: number, published = false) => ({
  id,
  sectionId: 's1',
  name,
  maxPoints,
  weight,
  published,
  position: id === 'a1' ? 1 : 2,
  ...(id === 'a1' && { dueOn: '2026-10-12' }),
})
const stu = (id: string, name: string, scores: Record<string, number> = {}, extra = {}) => ({
  userId: `u-${id}`,
  email: `${id}@a.edu`,
  displayName: name,
  scores,
  ...extra,
})
const book = (
  students = [
    stu('erin', 'Erin Moe', { a1: 30 }, { percent: 85, letter: 'B' }),
    stu('sam', 'Sam Lee'),
    stu('ann', 'Ann Ode'),
    stu('bo', 'Bo Kim'),
  ],
) => ({
  assessments: [asm('a1', 'Midterm', 40, 40), asm('a2', 'Final exam', 100, 50)],
  students,
  stats: { mean: 81.4, median: 83, letters: { B: 1, A: 1 } },
})

type Role = 'instructor' | 'ta' | 'admin' | 'member'
const GB = 'GET /universities/a/sections/s1/gradebook'
const PUT1 = 'PUT /universities/a/sections/s1/assessments/a1/scores'

function renderPage(role: Role, routes: Parameters<typeof mockApi>[0] = {}, gb: unknown = book()) {
  const staff = role === 'instructor' || role === 'ta' ? [{ userId: 'u0', role, email: 'x@a.edu' }] : []
  const fn = mockApi({
    'GET /universities/a/sections/s1': () => ({ body: section(staff) }),
    [GB]: () => ({ body: gb }),
    'GET /universities/a/me/sections': () => ({ body: { terms: [] } }),
    'GET /universities/a/grade-scale': () => ({ body: { rows: usPreset } }),
    ...routes,
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [
      {
        university: { id: 'a', slug: 'a', name: 'A U' },
        role: role === 'admin' ? 'admin' : 'member',
        departmentIds: [],
      },
    ],
  })
  const root = createRootRoute()
  const r = createRoute({
    getParentRoute: () => root,
    path: '$slug/sections/$id/grades',
    component: () => <GradebookPage slug="a" sectionId="s1" />,
  })
  const router = createRouter({
    routeTree: root.addChildren([r]),
    history: createMemoryHistory({ initialEntries: ['/a/sections/s1/grades'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  return fn
}
const calls = (fn: ReturnType<typeof mockApi>, key: string) =>
  fn.mock.calls.filter(([r]) => `${r.method} ${new URL(r.url).pathname.replace('/api/v1', '')}` === key)
const input = (n: string) => screen.findByLabelText(n) as Promise<HTMLInputElement>
const type = (el: HTMLElement, v: string) => fireEvent.change(el, { target: { value: v } })

it('instructor sees headers, weights warning, rows and stats', async () => {
  renderPage('instructor')
  expect(await screen.findByText('40 pts · 40%')).toBeInTheDocument()
  expect(screen.getByText('Due Oct 12, 2026')).toBeInTheDocument()
  expect(screen.getByText('Weights total 90%')).toBeInTheDocument()
  expect(screen.getByText('Must total 100% to finalize')).toBeInTheDocument()
  expect(screen.getByRole('rowheader', { name: /Erin Moe/ })).toHaveTextContent('erin@a.edu')
  expect(screen.getByText('85.00%')).toBeInTheDocument()
  expect(screen.getByText('81.40%')).toBeInTheDocument()
  expect(screen.getByText('Letters: A 1 · B 1')).toBeInTheDocument()
})

it('renders under the section band and tab strip with one h1', async () => {
  renderPage('instructor')
  await screen.findByText('40 pts · 40%')
  expect(screen.getAllByRole('heading', { level: 1 }).map((h) => h.textContent)).toEqual(['CS101 · Section 001'])
  expect(screen.getByRole('heading', { level: 2, name: 'Gradebook' })).toBeInTheDocument()
  const tabs = screen.getByRole('navigation', { name: 'Section pages' })
  expect(within(tabs).getByRole('link', { name: 'Grades' })).toHaveAttribute('href', '/a/sections/s1/grades')
  expect(within(tabs).getByRole('link', { name: 'Grades' })).toHaveAttribute('aria-current', 'page')
})

it('says plain "(1 missing)" when every unscored assessment is published', async () => {
  const b = book([stu('sam', 'Sam Lee', { a1: 30 })])
  b.assessments = [asm('a1', 'Midterm', 40, 40, true), asm('a2', 'Final exam', 100, 60, true)]
  renderPage('instructor', {}, b)
  expect(await screen.findByText('(1 missing)')).toBeInTheDocument()
})

it('labels published-only missing against the count Finalize uses', async () => {
  const b = book([stu('sam', 'Sam Lee', { a1: 30 }), stu('ann', 'Ann Ode')])
  b.assessments[0] = asm('a1', 'Midterm', 40, 40, true)
  renderPage('instructor', {}, b)
  expect(await screen.findByText('(1 missing, 2 at finalize)')).toBeInTheDocument()
  expect(screen.getByText('(1 at finalize)')).toBeInTheDocument()
})

it('saves on blur with one PUT, shows Saved, announces and refetches', async () => {
  const fn = renderPage('instructor', { [PUT1]: () => ({ body: { items: [] } }) })
  const cell = await input('Score for Sam Lee, Midterm')
  type(cell, '34')
  fireEvent.blur(cell)
  expect(await screen.findByText('Saved')).toBeInTheDocument()
  expect(calls(fn, PUT1)).toHaveLength(1)
  expect(JSON.parse(await calls(fn, PUT1)[0][0].text())).toEqual({ scores: [{ userId: 'u-sam', points: 34 }] })
  expect(screen.getByText('Saved: Sam Lee, Midterm.')).toBeInTheDocument()
  await waitFor(() => expect(calls(fn, GB).length).toBeGreaterThan(1))
})

it('no PUT without a change, and Escape restores the server value', async () => {
  const fn = renderPage('instructor', { [PUT1]: () => ({ body: {} }) })
  const cell = await input('Score for Erin Moe, Midterm')
  fireEvent.focus(cell)
  fireEvent.blur(cell)
  type(cell, '39')
  fireEvent.keyDown(cell, { key: 'Escape' })
  expect(cell.value).toBe('30')
  fireEvent.blur(cell)
  expect(calls(fn, PUT1)).toHaveLength(0)
})

it('invalid text shows an inline error and sends nothing', async () => {
  const fn = renderPage('instructor', { [PUT1]: () => ({ body: {} }) })
  const cell = await input('Score for Sam Lee, Midterm')
  type(cell, '3a')
  fireEvent.blur(cell)
  expect(await screen.findByText('Enter a number like 42 or 42.5.')).toBeInTheDocument()
  expect(cell).toHaveAttribute('aria-invalid', 'true')
  expect(calls(fn, PUT1)).toHaveLength(0)
})

it('maps a server validation error under the cell and retries on the next blur', async () => {
  let n = 0
  const fn = renderPage('instructor', {
    [PUT1]: () =>
      ++n === 1
        ? {
            status: 400,
            body: {
              error: {
                code: 'validation_failed',
                message: 'bad',
                details: [{ field: 'scores[0].points', message: 'bad' }],
              },
            },
          }
        : { body: {} },
  })
  const cell = await input('Score for Sam Lee, Midterm')
  type(cell, '80')
  fireEvent.blur(cell)
  expect(await screen.findByText('Enter a number from 0 to twice the maximum points.')).toBeInTheDocument()
  type(cell, '70')
  fireEvent.blur(cell)
  await screen.findByText('Saved')
  expect(calls(fn, PUT1)).toHaveLength(2)
})

it('a server failure marks the cell and shows the page banner', async () => {
  renderPage('instructor', {
    [PUT1]: () => ({ status: 500, body: { error: { code: 'internal', message: 'boom', details: [] } } }),
  })
  const cell = await input('Score for Sam Lee, Midterm')
  type(cell, '10')
  fireEvent.blur(cell)
  expect(await screen.findByText('Not saved.')).toBeInTheDocument()
  expect(screen.getByRole('alert')).toHaveTextContent('Something went wrong on our side')
})

it('pastes a column downward in one PUT, skipping blanks and dropping the overflow', async () => {
  const fn = renderPage('instructor', { [PUT1]: () => ({ body: {} }) })
  const cell = await input('Score for Erin Moe, Midterm')
  fireEvent.paste(cell, { clipboardData: { getData: () => '30\n31\n\n33\n40' } })
  await screen.findByText(/^Saved 3 scores for Midterm\./)
  expect(screen.getByText(/1 pasted values didn't fit below the last student and were ignored\./)).toBeInTheDocument()
  expect(JSON.parse(await calls(fn, PUT1)[0][0].text())).toEqual({
    scores: [
      { userId: 'u-erin', points: 30 },
      { userId: 'u-sam', points: 31 },
      { userId: 'u-bo', points: 33 },
    ],
  })
})

it('Enter and ArrowRight move focus', async () => {
  renderPage('instructor')
  const erin = await input('Score for Erin Moe, Midterm')
  erin.focus()
  fireEvent.keyDown(erin, { key: 'Enter' })
  expect(await input('Score for Sam Lee, Midterm')).toHaveFocus()
  const sam = (await input('Score for Sam Lee, Midterm')) as HTMLInputElement
  sam.setSelectionRange(sam.value.length, sam.value.length)
  fireEvent.keyDown(sam, { key: 'ArrowRight' })
  expect(await input('Score for Sam Lee, Final exam')).toHaveFocus()
})

it('publish checkbox sends PATCH and reports', async () => {
  const fn = renderPage('instructor', {
    'PATCH /universities/a/sections/s1/assessments/a1': () => ({ body: asm('a1', 'Midterm', 40, 40, true) }),
  })
  fireEvent.click(await screen.findByRole('checkbox', { name: 'Published Midterm' }))
  expect(await screen.findByText('Midterm published. Students can now see its scores.')).toBeInTheDocument()
  expect(JSON.parse(await calls(fn, 'PATCH /universities/a/sections/s1/assessments/a1')[0][0].text())).toEqual({
    published: true,
  })
})

it('TA has inputs but no assessment controls', async () => {
  renderPage('ta')
  await input('Score for Erin Moe, Midterm')
  expect(screen.queryByRole('button', { name: 'New assessment' })).not.toBeInTheDocument()
  expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
  expect(screen.getAllByText('Draft').length).toBe(2)
  expect(screen.queryByRole('button', { name: /Finalize/ })).not.toBeInTheDocument()
})

it('admin has instructor powers (D65): inputs, New assessment, Grade scale link', async () => {
  renderPage('admin')
  await input('Score for Erin Moe, Midterm')
  expect(screen.getByRole('button', { name: 'New assessment' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Grade scale' })).toHaveAttribute('href', '/a/grade-scale')
})

it('a non-staff member gets No access and no gradebook request', async () => {
  const fn = renderPage('member')
  expect(await screen.findByText('No access')).toBeInTheDocument()
  expect(calls(fn, GB)).toHaveLength(0)
})

it('empty states', async () => {
  renderPage('instructor', {}, { ...book([]) })
  expect(await screen.findByText('No students yet')).toBeInTheDocument()
})

it('no assessments: instructor gets the action, TA does not', async () => {
  renderPage('instructor', {}, { ...book(), assessments: [] })
  const t = await screen.findByText('No assessments yet')
  expect(
    within(t.closest('div')!.parentElement!).getAllByRole('button', { name: 'New assessment' }).length,
  ).toBeGreaterThan(0)
})

it('TA sees the no-assessments copy without an action', async () => {
  renderPage('ta', {}, { ...book(), assessments: [] })
  expect(await screen.findByText("The instructor hasn't added assessments yet.")).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'New assessment' })).not.toBeInTheDocument()
})

it('Export CSV requests text/csv and downloads a named file', async () => {
  const create = vi.fn(() => 'blob:x')
  const revoke = vi.fn()
  vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: create, revokeObjectURL: revoke }))
  let name = ''
  const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (this: HTMLAnchorElement) {
    name = this.download
  })
  const fn = renderPage('instructor', {
    [GB]: (req) =>
      req.headers.get('Accept') === 'text/csv' ? { body: undefined } : { body: book() },
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Export CSV' }))
  await waitFor(() => expect(create).toHaveBeenCalled())
  await waitFor(() => expect(revoke).toHaveBeenCalledWith('blob:x'))
  expect(calls(fn, GB).some(([r]) => r.headers.get('Accept') === 'text/csv')).toBe(true)
  expect(name).toBe('CS101-001-gradebook.csv')
  click.mockRestore()
})

it('Import scores opens the dialog for staff and admins, not for plain members', async () => {
  renderPage('ta')
  fireEvent.click(await screen.findByRole('button', { name: 'Import scores' }))
  expect(await screen.findByRole('heading', { name: 'Import scores' })).toBeInTheDocument()
})

it('admin has the Import scores button', async () => {
  renderPage('admin')
  expect(await screen.findByRole('button', { name: 'Import scores' })).toBeInTheDocument()
})

it('a newer draft survives the response of an older save and is saved on its own blur', async () => {
  let release!: () => void
  const fn = renderPage('instructor', {
    [PUT1]: () => new Promise((r) => { release = () => r({ body: {} }) }),
  })
  const cell = await input('Score for Sam Lee, Midterm')
  fireEvent.change(cell, { target: { value: '34' } })
  fireEvent.blur(cell)
  await screen.findByText('Saving…')
  fireEvent.change(cell, { target: { value: '35' } })
  await waitFor(() => expect(release).toBeTypeOf('function'))
  await act(async () => release())
  await waitFor(() => expect(screen.queryByText('Saving…')).not.toBeInTheDocument())
  expect(cell.value).toBe('35')
  expect(screen.queryByText('Saved')).not.toBeInTheDocument()
  fireEvent.blur(cell)
  await waitFor(() => expect(calls(fn, PUT1)).toHaveLength(2))
  expect(JSON.parse(await calls(fn, PUT1)[1][0].text())).toEqual({ scores: [{ userId: 'u-sam', points: 35 }] })
})

it('the band and table caption keep a hyphenated course code', async () => {
  const sec = section([{ userId: 'u0', role: 'instructor', email: 'x@a.edu' }])
  renderPage('instructor', {
    'GET /universities/a/sections/s1': () => ({ body: { ...sec, course: { ...sec.course, code: 'CS-101' } } }),
  })
  expect(await screen.findByRole('heading', { level: 1, name: 'CS-101 · Section 001' })).toBeInTheDocument()
  expect(await screen.findByRole('table', { name: 'Gradebook for CS-101-001' })).toBeInTheDocument()
})

const finalBook = () => ({
  ...book([
    stu('erin', 'Erin Moe', { a1: 30 }, { final: { status: 'graded', letter: 'B+', points: 3.3, percent: 88 } }),
    stu('sam', 'Sam Lee', {}, { final: { status: 'incomplete', overrideReason: 'Medical leave' } }),
  ]),
})

it('finalized gradebook shows the Final column, Change buttons and reopens', async () => {
  const DEL = 'DELETE /universities/a/sections/s1/final-grades'
  const fn = renderPage('instructor', { [DEL]: () => ({ status: 204 }) }, finalBook())
  expect(await screen.findByText('B+')).toBeInTheDocument()
  expect(screen.getByText('Incomplete')).toBeInTheDocument()
  expect(screen.getByText('Changed')).toBeInTheDocument()
  expect(screen.getByText(/Final grades are recorded/)).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Change final grade for Erin Moe' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Finalize grades' })).not.toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Reopen grades' }))
  expect(await screen.findByText(/This deletes the 2 final grades/)).toBeInTheDocument()
  fireEvent.click(screen.getAllByRole('button', { name: 'Reopen grades' })[1])
  expect(await screen.findByText('Grades reopened. Finalize again when scores are ready.')).toBeInTheDocument()
  expect(calls(fn, DEL)).toHaveLength(1)
})

it('Finalize grades is disabled with its reason until weights total 100', async () => {
  renderPage('instructor')
  const btn = await screen.findByRole('button', { name: 'Finalize grades' })
  expect(btn).toBeDisabled()
  expect(screen.getByText('Weights must total 100% to finalize. They total 90%.')).toBeInTheDocument()
  expect(btn).toHaveAccessibleDescription('Weights must total 100% to finalize. They total 90%.')
})

it('Finalize grades says there are no students to grade', async () => {
  renderPage('instructor', {}, { ...book([]), stats: { letters: {} } })
  await screen.findByRole('button', { name: 'Finalize grades' })
  expect(screen.getByText('No students to grade.')).toBeInTheDocument()
})

it('the sticky Student cells keep their surface on row hover', async () => {
  renderPage('instructor')
  expect(await screen.findByRole('rowheader', { name: /Erin Moe/ })).toHaveClass('bg-surface!')
})

it('Finalize grades says to add assessments when there are none', async () => {
  renderPage('instructor', {}, { ...book(), assessments: [] })
  await screen.findByRole('button', { name: 'Finalize grades' })
  expect(screen.getByText('Add assessments before you finalize.')).toBeInTheDocument()
})

it('Finalize grades opens the dialog when weights total 100', async () => {
  const b = book()
  b.assessments[1] = asm('a2', 'Final exam', 100, 60)
  renderPage('admin', {}, b)
  fireEvent.click(await screen.findByRole('button', { name: 'Finalize grades' }))
  expect(await screen.findByRole('heading', { name: 'Finalize grades for CS101-001' })).toBeInTheDocument()
})

const pub = (students: Record<string, unknown>[], final = false) => ({
  assessments: [asm('a1', 'Midterm', 40, 40, true), asm('a2', 'Final exam', 100, 60, false)],
  students: final
    ? students.map((s) => ({ ...s, final: { status: 'graded', letter: 'B', points: 3, percent: 85 } }))
    : students,
  stats: { mean: 81.4, median: 83, letters: { B: 1, A: 1, C: 0 } },
})

it('caps the Student cells and keeps the full name in the title', async () => {
  renderPage('instructor')
  const th = await screen.findByRole('rowheader', { name: /Erin Moe/ })
  expect(th).toHaveAttribute('title', 'Erin Moe')
  expect(th).toHaveClass('max-w-48', 'truncate')
  expect(screen.getByRole('columnheader', { name: 'Student' })).toHaveClass('max-w-48', 'truncate')
})

it('flags missing scores of published assessments and shows one email line for unnamed students', async () => {
  const bare = { ...stu('zed', ''), displayName: undefined }
  renderPage('instructor', {}, pub([stu('erin', 'Erin Moe', { a1: 30 }, { percent: 75 }), stu('sam', 'Sam Lee', {}, { percent: 10 }), bare]))
  await screen.findByText('Letters: A 1 · B 1')
  const sam = screen.getByRole('row', { name: /Sam Lee/ })
  expect(within(sam).getByText('Missing')).toBeInTheDocument()
  expect(within(sam).getByText('(1 missing, 2 at finalize)')).toBeInTheDocument()
  expect(within(screen.getByRole('row', { name: /Erin Moe/ })).queryByText('Missing')).not.toBeInTheDocument()
  const zed = screen.getByRole('rowheader', { name: 'zed@a.edu' })
  expect(zed.querySelectorAll('span')).toHaveLength(0)
  expect(screen.getAllByText('zed@a.edu')).toHaveLength(1)
})

it('with final grades shows Finalized, Running headers and the instructor note', async () => {
  renderPage('instructor', {}, pub([stu('erin', 'Erin Moe', { a1: 30 })], true))
  expect(await screen.findByText('Finalized')).toBeInTheDocument()
  expect(screen.getByRole('columnheader', { name: 'Running %' })).toBeInTheDocument()
  expect(screen.getByRole('columnheader', { name: 'Running letter' })).toBeInTheDocument()
  expect(screen.getByRole('columnheader', { name: 'Final' })).toBeInTheDocument()
  expect(screen.getByText(/reopen grades to recompute/)).toBeInTheDocument()
})

it('TA sees the TA note when finalized', async () => {
  renderPage('ta', {}, pub([stu('erin', 'Erin Moe', { a1: 30 })], true))
  expect(
    await screen.findByText('Final grades are recorded. Score changes do not change them until an instructor reopens grades.'),
  ).toBeInTheDocument()
})

it('omits letters with a zero count and gives the footer headers the surface background', async () => {
  renderPage('instructor', {}, pub([stu('erin', 'Erin Moe', { a1: 30 })]))
  expect(await screen.findByText('Letters: A 1 · B 1')).toBeInTheDocument()
  expect(screen.getByRole('rowheader', { name: 'Mean' })).toHaveClass('bg-surface')
})

it('names what failed to load', async () => {
  renderPage('instructor', { [GB]: () => ({ status: 500, body: { error: { code: 'internal', message: 'x', details: [] } } }) })
  expect(await screen.findByText("Couldn't load the gradebook")).toBeInTheDocument()
})

it('an explicit 0 score on a published assessment is not Missing', async () => {
  renderPage('instructor', {}, pub([stu('erin', 'Erin Moe', { a1: 0 }, { percent: 0 })]))
  await screen.findByText('Letters: A 1 · B 1')
  expect(screen.queryByText('Missing')).not.toBeInTheDocument()
})
