import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import type { Assessment } from '../quiz/queries'
import { mine, renderSection, sectionFixture } from '../section/test-helpers'
import { AssignmentsPage } from './assignment-list'

afterEach(() => vi.unstubAllGlobals())

const S = '/universities/a/sections/s1'
const asg = (id: string, name: string, over: Partial<Assessment> = {}): Assessment => ({
  id,
  sectionId: 's1',
  kind: 'assignment',
  name,
  description: '',
  maxPoints: 20,
  weight: 20,
  opensAt: '2020-01-01T09:00:00Z',
  closesAt: '2099-01-01T09:00:00Z',
  published: false,
  position: 1,
  ...over,
})
const quiz = { ...asg('q1', 'Week 3 check'), kind: 'quiz' as const }
const err = (status: number, code: string) => ({ status, body: { error: { code, message: 'm', details: [] } } })
const list = (...items: Assessment[]) => ({ [`GET ${S}/assessments`]: () => ({ body: { items } }) })
const sub = (over: object = {}) => ({ userId: 'u0', body: 'x', submittedAt: '2026-10-09T15:12:00Z', files: [], ...over })
const subMe = (id: string, s: object | null) => ({ [`GET ${S}/assessments/${id}/submissions/me`]: () => (s ? { body: s } : err(404, 'not_found')) })
const sent = (fn: ReturnType<typeof mockApi>, method: string) => fn.mock.calls.filter(([r]) => r.method === method)

function setup(routes: Routes, o: { staff?: boolean } = {}) {
  const staff = o.staff ?? true
  const fn = mockApi({
    [`GET ${S}`]: () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: mine(staff ? undefined : 'student') }),
    ...routes,
  })
  renderSection({ path: '/a/sections/s1/assignments', ui: <AssignmentsPage slug="a" sectionId="s1" />, userId: staff ? 'u-ins' : 'u0' })
  return fn
}

it('staff rows have named actions, window text and no quizzes', async () => {
  setup({ ...list(quiz, asg('a1', 'Essay 1', { published: true })) })
  const art = await screen.findByRole('article')
  expect(screen.queryByText('Week 3 check')).toBeNull()
  expect(within(art).getByText(/^Opens .* · Due /)).toBeInTheDocument()
  expect(within(art).getByText('20 points. Weight 20%.')).toBeInTheDocument()
  expect(within(art).getByText('Scores released')).toBeInTheDocument()
  expect(within(art).getByText('Open')).toBeInTheDocument()
  expect(within(art).getByRole('link', { name: 'Submissions for Essay 1' })).toHaveAttribute('href', '/a/sections/s1/assignments/a1/submissions')
  for (const n of ['Hide scores for', 'Edit', 'Delete']) expect(within(art).getByRole('button', { name: `${n} Essay 1` })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'New assignment' })).toBeInTheDocument()
})

it('student: View with state badges, Submitted, score only when published', async () => {
  setup(
    {
      ...list(
        asg('a1', 'Open one'),
        asg('a2', 'Later', { opensAt: '2098-01-01T00:00:00Z' }),
        asg('a3', 'Past', { closesAt: '2021-01-01T00:00:00Z' }),
        asg('a4', 'Graded', { published: true }),
        asg('a5', 'Graded hidden'),
      ),
      ...subMe('a1', null),
      ...subMe('a2', null),
      ...subMe('a3', null),
      ...subMe('a4', sub({ score: 17 })),
      ...subMe('a5', sub({ score: 17 })),
    },
    { staff: false },
  )
  expect(await screen.findByRole('link', { name: 'View Open one' })).toHaveAttribute('href', '/a/sections/s1/assignments/a1')
  const arts = screen.getAllByRole('article')
  expect(within(arts[1]).getByText('Upcoming')).toBeInTheDocument()
  expect(within(arts[2]).getByText('Closed')).toBeInTheDocument()
  expect(within(arts[3]).getByText('Submitted')).toBeInTheDocument()
  expect(within(arts[3]).getByText(/Score 17 \/ 20\./)).toBeInTheDocument()
  expect(within(arts[4]).queryByText(/Score/)).toBeNull()
  expect(screen.queryByRole('button', { name: /Edit|Delete|New/ })).toBeNull()
})

it('delete names the submission count, deletes and focuses the heading', async () => {
  const fn = setup({
    ...list(asg('a1', 'Essay 1')),
    [`GET ${S}/assessments/a1/submissions`]: () => ({
      body: { items: ['u1', 'u2'].map((u) => ({ userId: u, email: `${u}@a.edu`, displayName: u, submittedAt: '2026-10-09T15:12:00Z', fileCount: 0 })) },
    }),
    [`DELETE ${S}/assessments/a1`]: () => ({ status: 204 }),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Essay 1' }))
  expect(await screen.findByText(/2 submissions and their files and scores are deleted too/)).toBeInTheDocument()
  fireEvent.click(await screen.findByRole('button', { name: 'Delete assignment' }))
  expect(await screen.findByText('Essay 1 deleted.')).toBeInTheDocument()
  expect(sent(fn, 'DELETE')).toHaveLength(1)
  await waitFor(() => expect(screen.getByRole('heading', { level: 2, name: 'Assignments' })).toHaveFocus())
})

it('delete with no submissions says so', async () => {
  setup({ ...list(asg('a1', 'Essay 1')), [`GET ${S}/assessments/a1/submissions`]: () => ({ body: { items: [] } }) })
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Essay 1' }))
  expect(await screen.findByText("No student has submitted. This can't be undone.")).toBeInTheDocument()
})

it('Release asks first and PATCHes published true', async () => {
  const fn = setup({
    ...list(asg('a1', 'Essay 1')),
    [`PATCH ${S}/assessments/a1`]: (r) => r.json().then((b) => ({ body: asg('a1', 'Essay 1', b) })),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Release scores for Essay 1' }))
  expect(await screen.findByRole('dialog', { name: 'Release scores for Essay 1?' })).toBeInTheDocument()
  fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Release scores' }))
  expect(await screen.findByText('Essay 1: scores released to students.')).toBeInTheDocument()
  expect(await sent(fn, 'PATCH')[0][0].json()).toEqual({ published: true })
})

it('empty states per role and the 403 copy', async () => {
  setup({ ...list(quiz) })
  expect(await screen.findByText('No assignments yet')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'New assignment' })).toBeInTheDocument()
})

it('empty state for students has no action', async () => {
  setup({ ...list() }, { staff: false })
  expect(await screen.findByText("Staff haven't added an assignment to this section yet.")).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'New assignment' })).toBeNull()
})

it('shows the 403 copy', async () => {
  setup({ [`GET ${S}/assessments`]: () => err(403, 'forbidden') }, { staff: false })
  expect(await screen.findByText("Only this section's staff and enrolled students can open its assignments.")).toBeInTheDocument()
})
