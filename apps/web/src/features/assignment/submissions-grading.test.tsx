import { fireEvent, screen, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import type { Assessment, Submission } from '../quiz/queries'
import { mine, renderSection, sectionFixture } from '../section/test-helpers'
import { SubmissionsPage } from './submissions-grading'

afterEach(() => vi.unstubAllGlobals())

const S = '/universities/a/sections/s1'
const a: Assessment = {
  id: 'q1',
  sectionId: 's1',
  kind: 'assignment',
  name: 'Essay 1',
  description: '',
  maxPoints: 20,
  weight: 20,
  opensAt: '2020-01-01T09:00:00Z',
  closesAt: '2021-01-01T09:00:00Z',
  published: false,
  position: 1,
}
const row = (userId: string, displayName: string, over: object = {}) => ({
  userId,
  email: `${userId}@a.edu`,
  displayName,
  submittedAt: '2026-10-09T15:12:00Z',
  fileCount: 2,
  ...over,
})
const rows = [row('u1', 'Dana Reyes', { score: 17, gradedAt: '2026-10-10T10:00:00Z' }), row('u2', 'Tomás Ibarra')]
const full = (over: Partial<Submission> = {}): Submission => ({
  userId: 'u1',
  body: 'Line one\nLine two',
  submittedAt: '2026-10-09T15:12:00Z',
  files: [{ id: 'f1', fileName: 'essay.pdf', contentType: 'application/pdf', sizeBytes: 2048, position: 1, uploadedAt: '2026-10-09T15:12:00Z' }],
  feedback: '',
  ...over,
})
const sent = (fn: ReturnType<typeof mockApi>, method: string) => fn.mock.calls.filter(([r]) => r.method === method)

function setup(routes: Routes = {}, userId?: string) {
  const fn = mockApi({
    [`GET ${S}`]: () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: mine() }),
    [`GET ${S}/assessments`]: () => ({ body: { items: [a] } }),
    [`GET ${S}/assessments/q1/submissions`]: () => ({ body: { items: rows } }),
    ...routes,
  })
  renderSection({
    path: '/a/sections/s1/assignments/q1/submissions',
    ui: <SubmissionsPage slug="a" sectionId="s1" assessmentId="q1" userId={userId} />,
    userId: 'u-ins',
  })
  return fn
}

it('lists submissions with links, score and graded columns', async () => {
  setup()
  const table = await screen.findByRole('table', { name: 'Submissions for Essay 1' })
  const [dana, tomas] = within(table).getAllByRole('row').slice(1)
  expect(within(dana).getByRole('link', { name: 'Dana Reyes' })).toHaveAttribute('href', expect.stringContaining('user=u1'))
  expect(within(dana).getByText('u1@a.edu')).toBeInTheDocument()
  expect(within(dana).getByText('17 / 20')).toBeInTheDocument()
  expect(within(tomas).getAllByText('Not graded')).toHaveLength(2)
  expect(within(tomas).getByText('2')).toBeInTheDocument()
})

it('empty state', async () => {
  setup({ [`GET ${S}/assessments/q1/submissions`]: () => ({ body: { items: [] } }) })
  expect(await screen.findByText('No submissions yet')).toBeInTheDocument()
})

it('Release dialog notes how many are ungraded', async () => {
  setup()
  fireEvent.click(await screen.findByRole('button', { name: 'Release scores for Essay 1' }))
  expect(await screen.findByText(/1 submission isn't graded yet\./)).toBeInTheDocument()
})

it('panel: body, download links, Score above 2x max errors without a request', async () => {
  const fn = setup({ [`GET ${S}/assessments/q1/submissions/u1`]: () => ({ body: full() }) }, 'u1')
  expect(await screen.findByRole('heading', { name: 'Dana Reyes' })).toBeInTheDocument()
  expect(screen.getByText(/Line one/)).toHaveClass('whitespace-pre-wrap')
  expect(screen.getByRole('link', { name: 'Download essay.pdf' })).toHaveAttribute('href', `/api/v1${S}/assessments/q1/submissions/u1/files/f1/download`)
  fireEvent.change(screen.getByLabelText('Score for Dana Reyes'), { target: { value: '41' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save grade' }))
  expect(await screen.findByText('Enter at most 40 (twice the maximum).')).toBeInTheDocument()
  fireEvent.change(screen.getByLabelText('Score for Dana Reyes'), { target: { value: '' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save grade' }))
  expect(await screen.findByText('Enter a score.')).toBeInTheDocument()
  expect(sent(fn, 'PATCH')).toHaveLength(0)
})

it('panel: Save grade PATCHes score and feedback and announces', async () => {
  const fn = setup(
    { [`GET ${S}/assessments/q1/submissions/u1`]: () => ({ body: full() }), [`PATCH ${S}/assessments/q1/submissions/u1`]: (r) => r.json().then((b) => ({ body: full(b) })) },
    'u1',
  )
  fireEvent.change(await screen.findByLabelText('Score for Dana Reyes'), { target: { value: '17.5' } })
  fireEvent.change(screen.getByLabelText('Feedback for Dana Reyes'), { target: { value: 'Good use of sources.' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save grade' }))
  expect(await screen.findByText('Grade saved. Dana Reyes: 17.5 / 20.')).toBeInTheDocument()
  expect(await sent(fn, 'PATCH')[0][0].json()).toEqual({ score: 17.5, feedback: 'Good use of sources.' })
})

it('panel: server field error lands under Score', async () => {
  setup(
    {
      [`GET ${S}/assessments/q1/submissions/u1`]: () => ({ body: full() }),
      [`PATCH ${S}/assessments/q1/submissions/u1`]: () => ({ status: 400, body: { error: { code: 'validation_failed', message: 'm', details: [{ field: 'score', message: 'must be at most 40' }] } } }),
    },
    'u1',
  )
  fireEvent.change(await screen.findByLabelText('Score for Dana Reyes'), { target: { value: '5' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save grade' }))
  expect(await screen.findByText('Must be at most 40.')).toBeInTheDocument()
  expect(screen.getByLabelText('Score for Dana Reyes')).toHaveAttribute('aria-invalid', 'true')
})
