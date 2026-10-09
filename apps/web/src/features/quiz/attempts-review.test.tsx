import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import { mine, renderSection, sectionFixture } from '../section/test-helpers'
import { AttemptsPage } from './attempts-review'
import type { Assessment, Question } from './queries'

afterEach(() => vi.unstubAllGlobals())

const S = '/universities/a/sections/s1'
const A = `${S}/assessments/q1`
const quiz: Assessment = {
  id: 'q1',
  sectionId: 's1',
  kind: 'quiz',
  name: 'Week 3 check',
  description: '',
  maxPoints: 4,
  weight: 40,
  opensAt: '2020-01-01T09:00:00Z',
  closesAt: '2021-01-01T09:00:00Z',
  published: false,
  position: 1,
  questionCount: 2,
}
const questions: Question[] = [
  { id: 'k1', kind: 'choice', prompt: 'Capital of France?', points: 2, position: 1, multiple: false, choices: [{ text: 'Lyon', correct: false }, { text: 'Paris', correct: true }] },
  { id: 'k2', kind: 'blank', prompt: 'Largest planet?', points: 2, position: 2, multiple: false, choices: [], answers: ['Jupiter'] },
]
const row = (userId: string, name: string, over: object = {}) => ({
  userId,
  email: `${userId}@a.edu`,
  displayName: name,
  startedAt: '2026-10-12T13:05:00Z',
  autoSubmitted: false,
  ...over,
})
const rows = [
  row('u1', 'Dana Reyes', { submittedAt: '2026-10-12T13:40:00Z', score: 2, reviewedAt: '2026-10-13T09:00:00Z' }),
  row('u2', 'Tomás Ibarra', { submittedAt: '2026-10-12T14:00:00Z', autoSubmitted: true, score: 1 }),
  row('u3', 'Lee Park'),
]
const attempt = {
  userId: 'u2',
  startedAt: '2026-10-12T13:05:00Z',
  submittedAt: '2026-10-12T14:00:00Z',
  autoSubmitted: false,
  answers: { k1: { choices: [0] }, k2: { text: 'Saturn' } },
  grading: { k1: { points: 0, correct: false, feedback: '' }, k2: { points: 1, correct: false, feedback: '' } },
  score: 1,
}

function setup(path: string, routes: Routes = {}) {
  const fn = mockApi({
    [`GET ${S}`]: () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: mine('instructor') }),
    [`GET ${S}/assessments`]: () => ({ body: { items: [quiz] } }),
    [`GET ${A}/attempts`]: () => ({ body: { items: rows } }),
    [`GET ${A}/questions`]: () => ({ body: { items: questions } }),
    [`GET ${A}/attempts/u2`]: () => ({ body: attempt }),
    ...routes,
  })
  renderSection({ path, ui: <AttemptsPage slug="a" sectionId="s1" assessmentId="q1" />, userId: 'u-ins' })
  return fn
}
const panel = '/a/sections/s1/quizzes/q1/attempts?user=u2'
const sent = (fn: ReturnType<typeof mockApi>, method: string) => fn.mock.calls.filter(([r]) => r.method === method)

it('lists attempts with Auto, In progress, Not reviewed and the unreviewed count', async () => {
  setup('/a/sections/s1/quizzes/q1/attempts')
  const table = await screen.findByRole('table', { name: 'Attempts for Week 3 check' })
  const [, dana, tomas, lee] = within(table).getAllByRole('row')
  expect(within(dana).getByText('2 / 4')).toBeInTheDocument()
  expect(within(tomas).getByText('Auto')).toBeInTheDocument()
  expect(within(tomas).getByText('Not reviewed')).toBeInTheDocument()
  expect(within(lee).getByText('In progress')).toBeInTheDocument()
  expect(within(lee).getByText('Not submitted')).toBeInTheDocument()
  expect(screen.getByText("1 of 2 submitted attempts isn't reviewed yet.")).toBeInTheDocument()
  expect(within(tomas).getByRole('link', { name: 'Tomás Ibarra' })).toHaveAttribute('href', expect.stringContaining('user=u2'))
})

it('empty state when nobody started', async () => {
  setup('/a/sections/s1/quizzes/q1/attempts', { [`GET ${A}/attempts`]: () => ({ body: { items: [] } }) })
  expect(await screen.findByText('No attempts yet')).toBeInTheDocument()
})

it('Release dialog notes the unreviewed count', async () => {
  setup('/a/sections/s1/quizzes/q1/attempts')
  fireEvent.click(await screen.findByRole('button', { name: 'Release scores for Week 3 check' }))
  expect(await screen.findByText(/1 submitted attempt isn't reviewed yet\./)).toBeInTheDocument()
})

it('opens one student in place with labelled inputs', async () => {
  setup(panel)
  expect(await screen.findByRole('heading', { level: 3, name: 'Tomás Ibarra' })).toBeInTheDocument()
  expect(screen.getByText('Lyon').previousElementSibling).toHaveTextContent('Answer:')
  expect(screen.getByText('Paris')).toBeInTheDocument()
  expect(screen.getByText('Jupiter')).toBeInTheDocument()
  expect(screen.getByLabelText('Points for question 2')).toHaveValue('1')
  expect(screen.getByLabelText('Feedback on question 1')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'All attempts' })).toBeInTheDocument()
  expect(screen.queryByRole('table')).toBeNull()
})

it('points above the maximum fail locally; PATCH sends only changed entries; score recomputes', async () => {
  const fn = setup(panel, {
    [`PATCH ${A}/attempts/u2`]: () => ({
      body: { ...attempt, grading: { ...attempt.grading, k1: { points: 2, correct: true, feedback: 'Good' } }, feedback: 'Well done', score: 3, reviewedAt: '2026-10-13T09:00:00Z' },
    }),
  })
  const p1 = await screen.findByLabelText('Points for question 1')
  fireEvent.change(p1, { target: { value: '3' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save review' }))
  expect(await screen.findByText('Enter 0 to 2 points.')).toBeInTheDocument()
  expect(sent(fn, 'PATCH')).toHaveLength(0)
  fireEvent.change(p1, { target: { value: '2' } })
  expect(screen.getByText(/Score 3 \/ 4/)).toHaveTextContent('(unsaved)')
  fireEvent.change(screen.getByLabelText('Feedback on question 1'), { target: { value: 'Good' } })
  fireEvent.change(screen.getByLabelText('Feedback for Tomás Ibarra'), { target: { value: 'Well done' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save review' }))
  expect(await screen.findByText('Review saved. Tomás Ibarra: 3 / 4.')).toBeInTheDocument()
  const [req] = sent(fn, 'PATCH')[0]
  expect(await req.json()).toEqual({ grading: { k1: { points: 2, feedback: 'Good' } }, feedback: 'Well done' })
  expect(screen.getByRole('button', { name: 'Save review' })).toBeEnabled()
  expect(screen.getByText(/Score 3 \/ 4/)).not.toHaveTextContent('unsaved')
})

it('nothing changed says so without a request', async () => {
  const fn = setup(panel)
  fireEvent.click(await screen.findByRole('button', { name: 'Save review' }))
  expect(await screen.findByText('Nothing to save.')).toBeInTheDocument()
  expect(sent(fn, 'PATCH')).toHaveLength(0)
})

it('maps server field errors onto the inputs', async () => {
  setup(panel, {
    [`PATCH ${A}/attempts/u2`]: () => ({
      status: 400,
      body: { error: { code: 'validation_failed', message: 'x', details: [{ field: 'grading.k2.points', message: 'must have at most 2 decimals' }, { field: 'feedback', message: 'must be at most 5000 characters' }] } },
    }),
  })
  fireEvent.change(await screen.findByLabelText('Points for question 2'), { target: { value: '1.5' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save review' }))
  await waitFor(() => expect(screen.getByLabelText('Points for question 2')).toHaveAttribute('aria-invalid', 'true'))
  expect(screen.getByText('Must have at most 2 decimals.')).toBeInTheDocument()
  expect(screen.getByLabelText('Feedback for Tomás Ibarra')).toHaveAttribute('aria-invalid', 'true')
})

it('409 shows the not-submitted copy', async () => {
  setup(panel, {
    [`PATCH ${A}/attempts/u2`]: () => ({ status: 409, body: { error: { code: 'conflict', message: 'the attempt is not submitted', details: [] } } }),
  })
  fireEvent.change(await screen.findByLabelText('Feedback for Tomás Ibarra'), { target: { value: 'Hi' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save review' }))
  expect(await screen.findByText("This attempt isn't submitted yet, so it can't be reviewed.")).toBeInTheDocument()
})

it('an unsubmitted attempt has a notice and no inputs', async () => {
  setup('/a/sections/s1/quizzes/q1/attempts?user=u3', {
    [`GET ${A}/attempts/u3`]: () => ({ body: { ...attempt, userId: 'u3', submittedAt: undefined, grading: undefined, score: undefined } }),
  })
  expect(await screen.findByText(/This attempt isn't submitted yet\. You can review it/)).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Save review' })).toBeNull()
  expect(screen.queryByLabelText('Points for question 1')).toBeNull()
})
