import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import { mine, renderSection, sectionFixture } from '../section/test-helpers'
import { QuizzesPage } from './quiz-list'
import type { Assessment } from './queries'

afterEach(() => vi.unstubAllGlobals())

const S = '/universities/a/sections/s1'
const quiz = (id: string, name: string, over: Partial<Assessment> = {}): Assessment => ({
  id,
  sectionId: 's1',
  kind: 'quiz',
  name,
  description: '',
  maxPoints: 10,
  weight: 40,
  opensAt: '2020-01-01T09:00:00Z',
  closesAt: '2099-01-01T09:00:00Z',
  published: false,
  position: 1,
  questionCount: 3,
  ...over,
})
const manual: Assessment = { ...quiz('m1', 'Midterm'), kind: 'manual', opensAt: undefined, closesAt: undefined }
const err = (status: number, code: string) => ({ status, body: { error: { code, message: 'm', details: [] } } })

function setup(routes: Routes, o: { staff?: boolean } = {}) {
  const staff = o.staff ?? true
  const fn = mockApi({
    [`GET ${S}`]: () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: mine(staff ? undefined : 'student') }),
    ...routes,
  })
  renderSection({ path: '/a/sections/s1/quizzes', ui: <QuizzesPage slug="a" sectionId="s1" />, userId: staff ? 'u-ins' : 'u0' })
  return fn
}
const list = (...items: Assessment[]) => ({ [`GET ${S}/assessments`]: () => ({ body: { items } }) })
const attempt = (over: object = {}) => ({ userId: 'u0', startedAt: '2026-10-12T13:05:00Z', deadlineAt: '2099-01-01T09:00:00Z', autoSubmitted: false, answers: {}, ...over })
const attemptMe = (id: string, a: object | null) => ({
  [`GET ${S}/assessments/${id}/attempts/me`]: () => (a ? { body: a } : err(404, 'not_found')),
})
const sent = (fn: ReturnType<typeof mockApi>, method: string) => fn.mock.calls.filter(([r]) => r.method === method)

it('staff rows have named actions and manual rows are absent', async () => {
  setup({ ...list(manual, quiz('q1', 'Week 3 check')) })
  await screen.findByRole('heading', { level: 3, name: 'Week 3 check' })
  for (const n of ['Questions for', 'Attempts for', 'Release scores for', 'Edit', 'Delete'])
    expect(screen.getByRole(n.includes('Questions') || n.includes('Attempts') ? 'link' : 'button', { name: `${n} Week 3 check` })).toBeInTheDocument()
  expect(screen.queryByText('Midterm')).toBeNull()
  expect(screen.getByRole('button', { name: 'New quiz' })).toBeInTheDocument()
})

it('shows window text, counts, and staff badges', async () => {
  setup({ ...list(quiz('q1', 'Week 3 check', { published: true }), quiz('q2', 'Empty one', { questionCount: 0, weight: 10 })) })
  const [first, second] = await screen.findAllByRole('article')
  expect(within(first).getByText(/^Opens .* · Closes /)).toBeInTheDocument()
  expect(within(first).getByText('3 questions, 10 points. Weight 40%.')).toBeInTheDocument()
  expect(within(first).getByText('Scores released')).toBeInTheDocument()
  expect(within(first).getByText('Open')).toBeInTheDocument()
  expect(within(second).getByText('No questions yet. Weight 10%.')).toBeInTheDocument()
  expect(within(second).getByText('No questions')).toBeInTheDocument()
})

it('student: Start when open and not started; no Start without questions', async () => {
  setup(
    {
      ...list(quiz('q1', 'Week 3 check'), quiz('q2', 'Empty one', { questionCount: 0 })),
      ...attemptMe('q1', null),
      ...attemptMe('q2', null),
    },
    { staff: false },
  )
  expect(await screen.findByRole('link', { name: 'Start Week 3 check' })).toHaveTextContent('Start')
  expect(screen.getByRole('link', { name: 'View Empty one' })).toBeInTheDocument()
  expect(screen.queryByText('No questions')).toBeNull() // staff only
  expect(screen.queryByRole('button', { name: /Edit|Delete/ })).toBeNull()
})

it('student: a failed attempts/me lookup shows View, never Start', async () => {
  setup({ ...list(quiz('q1', 'Week 3 check')), [`GET ${S}/assessments/q1/attempts/me`]: () => err(500, 'internal') }, { staff: false })
  expect(await screen.findByRole('link', { name: 'View Week 3 check' })).toBeInTheDocument()
  expect(screen.queryByRole('link', { name: 'Start Week 3 check' })).toBeNull()
})

it('student: Continue, View for upcoming, closed and submitted; score only when published', async () => {
  setup(
    {
      ...list(
        quiz('q1', 'In progress'),
        quiz('q2', 'Later', { opensAt: '2098-01-01T00:00:00Z' }),
        quiz('q3', 'Past', { closesAt: '2021-01-01T00:00:00Z' }),
        quiz('q4', 'Done', { published: true }),
        quiz('q5', 'Done hidden'),
      ),
      ...attemptMe('q1', attempt()),
      ...attemptMe('q2', null),
      ...attemptMe('q3', null),
      ...attemptMe('q4', attempt({ submittedAt: '2026-10-12T13:30:00Z', score: 8 })),
      ...attemptMe('q5', attempt({ submittedAt: '2026-10-12T13:30:00Z' })),
    },
    { staff: false },
  )
  expect(await screen.findByRole('link', { name: 'Continue In progress' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'View Later' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'View Past' })).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'View Done' })).toBeInTheDocument()
  const arts = screen.getAllByRole('article')
  expect(within(arts[1]).getByText('Upcoming')).toBeInTheDocument()
  expect(within(arts[2]).getByText('Closed')).toBeInTheDocument()
  expect(within(arts[3]).getByText('Submitted')).toBeInTheDocument()
  expect(within(arts[3]).getByText(/Score 8 \/ 10\./)).toBeInTheDocument()
  expect(within(arts[4]).queryByText(/Score/)).toBeNull()
})

it('delete names the attempt count, deletes, announces and focuses the heading', async () => {
  const fn = setup({
    ...list(quiz('q1', 'Week 3 check')),
    [`GET ${S}/assessments/q1/attempts`]: () => ({
      body: { items: [{ userId: 'u1', email: 'a@a.edu', displayName: 'Dana Reyes', startedAt: '2026-10-12T13:05:00Z', autoSubmitted: false }, { userId: 'u2', email: 'b@a.edu', displayName: 'Tomás Ibarra', startedAt: '2026-10-12T13:06:00Z', autoSubmitted: false }] },
    }),
    [`DELETE ${S}/assessments/q1`]: () => ({ status: 204 }),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Week 3 check' }))
  expect(await screen.findByText(/2 students have started it\. Their attempts and scores are deleted too/)).toBeInTheDocument()
  fireEvent.click(await screen.findByRole('button', { name: 'Delete quiz' }))
  expect(await screen.findByText('Week 3 check deleted.')).toBeInTheDocument()
  expect(sent(fn, 'DELETE')).toHaveLength(1)
  await waitFor(() => expect(screen.getByRole('heading', { level: 2, name: 'Quizzes' })).toHaveFocus())
})

it('delete falls back to neutral text when the attempts count fails', async () => {
  setup({ ...list(quiz('q1', 'Week 3 check')), [`GET ${S}/assessments/q1/attempts`]: () => err(500, 'internal') })
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Week 3 check' }))
  expect(await screen.findByText("Their attempts and scores are deleted too. This can't be undone.")).toBeInTheDocument()
  expect(screen.queryByText(/No student has started it/)).toBeNull()
})

it('staff never request attempts/me', async () => {
  const fn = setup({ ...list(quiz('q1', 'Week 3 check')) })
  await screen.findByRole('heading', { level: 3, name: 'Week 3 check' })
  expect(fn.mock.calls.filter(([r]) => r.url.endsWith('/attempts/me'))).toHaveLength(0)
})

it('delete with no attempts says no student has started it', async () => {
  setup({ ...list(quiz('q1', 'Week 3 check')), [`GET ${S}/assessments/q1/attempts`]: () => ({ body: { items: [] } }) })
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Week 3 check' }))
  expect(await screen.findByText(/No student has started it\. Its questions are deleted too/)).toBeInTheDocument()
})

it('Release asks first, PATCHes published true and announces; Hide acts at once', async () => {
  const fn = setup({
    ...list(quiz('q1', 'Week 3 check')),
    [`PATCH ${S}/assessments/q1`]: (r) => r.json().then((b) => ({ body: quiz('q1', 'Week 3 check', b) })),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Release scores for Week 3 check' }))
  expect(await screen.findByRole('dialog', { name: 'Release scores for Week 3 check?' })).toBeInTheDocument()
  expect(sent(fn, 'PATCH')).toHaveLength(0)
  fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: 'Release scores' }))
  expect(await screen.findByText('Week 3 check: scores released to students.')).toBeInTheDocument()
  expect(await sent(fn, 'PATCH')[0][0].json()).toEqual({ published: true })
})

it('Hide scores PATCHes immediately', async () => {
  const fn = setup({
    ...list(quiz('q1', 'Week 3 check', { published: true })),
    [`PATCH ${S}/assessments/q1`]: (r) => r.json().then((b) => ({ body: quiz('q1', 'Week 3 check', b) })),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Hide scores for Week 3 check' }))
  expect(await screen.findByText('Week 3 check: scores hidden from students.')).toBeInTheDocument()
  expect(await sent(fn, 'PATCH')[0][0].json()).toEqual({ published: false })
})

it('empty states per role', async () => {
  setup({ ...list(manual) })
  expect(await screen.findByText('No quizzes yet')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'New quiz' })).toBeInTheDocument()
})

it('empty state for students has no action', async () => {
  setup({ ...list() }, { staff: false })
  expect(await screen.findByText("Staff haven't added a quiz to this section yet.")).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'New quiz' })).toBeNull()
})

it('shows the 403 copy', async () => {
  setup({ [`GET ${S}/assessments`]: () => err(403, 'forbidden') }, { staff: false })
  expect(await screen.findByText("Only this section's staff and enrolled students can open its quizzes.")).toBeInTheDocument()
})

it('names the time limit in the window line', async () => {
  setup({ ...list(quiz('q1', 'Week 3 check', { timeLimitMinutes: 10 })) })
  expect(await screen.findByText(/^Opens .* · Closes .*, 10 min each$/)).toBeInTheDocument()
})
