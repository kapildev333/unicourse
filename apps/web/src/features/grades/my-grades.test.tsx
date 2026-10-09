import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider, createMemoryHistory, createRootRoute, createRoute, createRouter } from '@tanstack/react-router'
import { render, screen, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { usPreset } from './presets'
import { MyGradesPage } from './my-grades'

afterEach(() => vi.unstubAllGlobals())

const fall = { id: 't2', name: 'Fall 2026', startsOn: '2026-08-25', endsOn: '2026-12-15' }
const spring = { id: 't1', name: 'Spring 2026', startsOn: '2026-01-10', endsOn: '2026-05-15' }
const course = (code: string, title: string) => ({ id: `c-${code}`, departmentId: 'd1', code, title, credits: 3 })
const mid = { id: 'a1', name: 'Midterm', weight: 40, maxPoints: 40, dueOn: '2026-10-12', points: 34 }
const quiz = { id: 'q1', name: 'Week 3 check', kind: 'quiz', weight: 20, maxPoints: 10, points: 8 }
const fin = { id: 'a2', name: 'Final', weight: 60, maxPoints: 100 }

const grades = (over: Record<string, unknown> = {}) => ({
  cumulativeGpa: 3.47,
  cumulativeCredits: 24,
  // oldest first on purpose: the page sorts
  terms: [
    { term: spring, gpa: 3.1, credits: 6, sections: [] },
    {
      term: fall,
      gpa: 3.5,
      credits: 9,
      sections: [
        {
          section: { id: 's1', number: '001' },
          course: course('CS101', 'Intro to CS'),
          credits: 3,
          final: { status: 'graded', letter: 'B+', points: 3.3, percent: 88.4 },
          unpublishedCount: 0,
          assessments: [mid, fin],
        },
        {
          section: { id: 's3', number: '001' },
          course: course('MA201', 'Linear Algebra'),
          credits: 4,
          final: { status: 'incomplete' },
          unpublishedCount: 0,
          assessments: [],
        },
        {
          section: { id: 's4', number: '001' },
          course: course('PH110', 'Physics I'),
          credits: 3,
          running: { percent: 84.1, letter: 'B' },
          unpublishedCount: 2,
          assessments: [
            { ...mid, id: 'a3', name: 'Quiz' },
            { ...fin, id: 'a4', name: 'Lab', points: 9 },
          ],
        },
        {
          section: { id: 's5', number: '002' },
          course: course('HI101', 'History'),
          credits: 3,
          unpublishedCount: 0,
          assessments: [{ ...mid, id: 'a5', name: 'Essay', weight: 100 }],
        },
      ],
    },
  ],
  ...over,
})

const mine = (sections: unknown[] = []) => ({ terms: [{ term: fall, sections }] })

function renderPage(g: unknown, sections: unknown[] = []) {
  const fn = mockApi({
    'GET /universities/a/me/grades': () => ({ body: g }),
    'GET /universities/a/me/sections': () => ({ body: mine(sections) }),
    'GET /universities/a/grade-scale': () => ({ body: { rows: usPreset } }),
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const root = createRootRoute()
  const r = createRoute({ getParentRoute: () => root, path: '$slug', component: () => <MyGradesPage slug="a" /> })
  const router = createRouter({
    routeTree: root.addChildren([r]),
    history: createMemoryHistory({ initialEntries: ['/a'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  )
  return fn
}
const card = async (code: string) => (await screen.findByRole('link', { name: code })).closest('li') as HTMLElement

it('shows cumulative GPA and credits, terms newest first', async () => {
  renderPage(grades())
  const gpa = await screen.findByText('3.47')
  expect(gpa).toBeInTheDocument()
  expect(screen.getByText('24.0')).toBeInTheDocument()
  expect(screen.getByText('Term GPA 3.50 · 9.0 credits')).toBeInTheDocument()
  const heads = screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)
  expect(heads.indexOf('Fall 2026')).toBeLessThan(heads.indexOf('Spring 2026'))
})

it('graded card shows the final letter, points and percent, and links to the section', async () => {
  renderPage(grades())
  const c = await card('CS101-001')
  expect(within(c).getByText('B+')).toBeInTheDocument()
  expect(within(c).getByText('3.30 grade points · 88.40%')).toBeInTheDocument()
  expect(within(c).getByRole('link', { name: 'CS101-001' })).toHaveAttribute('href', '/a/sections/s1')
})

it('incomplete and running cards', async () => {
  renderPage(grades())
  const inc = await card('MA201-001')
  expect(within(inc).getByText('Incomplete')).toBeInTheDocument()
  expect(within(inc).getByText('Not counted in GPA')).toBeInTheDocument()
  const run = await card('PH110-001')
  expect(within(run).getByText('Current B')).toBeInTheDocument()
  expect(within(run).getByText('84.10% so far')).toBeInTheDocument()
  expect(within(run).getByText('2 assessments not yet published')).toBeInTheDocument()
  expect(within(run).queryByText('Counts published scores only.')).not.toBeInTheDocument()
})

it.each([
  [1, '1 assessment not yet published'],
  [0, 'Counts published scores only.'],
])('running card with unpublishedCount %i says %s', async (n, text) => {
  const g = grades()
  g.terms[1].sections![2].unpublishedCount = n
  renderPage(g)
  expect(within(await card('PH110-001')).getByText(text)).toBeInTheDocument()
})

it('computes nothing itself: no final and no running means No grade yet, and only the three endpoints are called', async () => {
  const fn = renderPage(grades())
  const c = await card('HI101-002')
  expect(within(c).getByText('No grade yet')).toBeInTheDocument()
  expect(c.textContent).not.toMatch(/%\s*so far/)
  // the only % in the card is the assessment weight in the closed details
  expect(c.textContent?.replace(/\d+%/g, '')).not.toContain('%')
  await screen.findByText('How letters and grade points are assigned')
  const paths = new Set(fn.mock.calls.map(([r]) => new URL(r.url).pathname.replace('/api/v1', '')))
  expect([...paths].sort()).toEqual(
    ['/universities/a/grade-scale', '/universities/a/me/grades', '/universities/a/me/sections'].sort(),
  )
})

it('lists published scores in a details and handles none', async () => {
  renderPage(grades())
  const c = await card('CS101-001')
  expect(within(c).getByText('Published scores (2)')).toBeInTheDocument()
  expect(within(c).getByText(/^Midterm · 34 \/ 40 · 40% · Due /)).toBeInTheDocument()
  expect(within(c).getByText('Final · Not graded yet · 60%')).toBeInTheDocument()
  expect(within(await card('MA201-001')).getByText('No published scores yet.')).toBeInTheDocument()
})

it('empty terms shows No grades yet with a catalog link', async () => {
  renderPage(grades({ terms: [], cumulativeGpa: undefined, cumulativeCredits: 0 }))
  expect(await screen.findByText('No grades yet')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Browse the catalog' })).toBeInTheDocument()
})

it('no cumulative GPA shows a dash and the note', async () => {
  renderPage(grades({ cumulativeGpa: undefined }))
  expect(await screen.findByText('Your GPA appears once a section has a final grade.')).toBeInTheDocument()
  expect(screen.getByText('—')).toBeInTheDocument()
})

it('Gradebooks lists sections where the viewer is instructor or TA', async () => {
  renderPage(grades(), [
    { id: 's2', number: '001', myRole: 'ta', course: course('MA201', 'Linear Algebra') },
    { id: 's1', number: '001', myRole: 'student', course: course('CS101', 'Intro') },
  ])
  const l = await screen.findByRole('link', { name: 'MA201-001' })
  expect(screen.getByRole('heading', { name: 'Gradebooks' })).toBeInTheDocument()
  expect(l.getAttribute('href')).toBe('/a/sections/s2/grades')
})

it('a student-only viewer sees no Gradebooks heading', async () => {
  renderPage(grades(), [{ id: 's1', number: '001', myRole: 'student', course: course('CS101', 'Intro') }])
  await card('CS101-001')
  expect(screen.queryByRole('heading', { name: 'Gradebooks' })).not.toBeInTheDocument()
})

it('explains Graded credits with the in-progress credits from the cards, and uses Disclosure chevrons', async () => {
  renderPage(grades())
  expect(await screen.findByText('Graded credits')).toBeInTheDocument()
  expect(screen.getByText(/Counts finalized sections only\. 6\.0 credits in progress\./)).toBeInTheDocument()
  const c = await card('CS101-001')
  const summary = within(c).getByText('Published scores (2)')
  expect(summary.closest('summary')).toHaveClass('list-none')
  expect(summary.closest('summary')?.querySelector('svg')).toHaveAttribute('aria-hidden', 'true')
  expect(screen.getByText('How letters and grade points are assigned').closest('summary')).toHaveClass('list-none')
})

it('gives every card the same fixed-height grade block', async () => {
  renderPage(grades())
  const c = await card('PH110-001')
  expect(within(c).getByText('Current B').parentElement).toHaveClass('min-h-14')
})

it('links a quiz row to its take page and leaves manual rows as plain text', async () => {
  const g = grades()
  g.terms[1].sections![0].assessments = [{ ...mid, kind: 'manual' }, quiz]
  renderPage(g)
  const c = await card('CS101-001')
  expect(within(c).getByRole('link', { name: 'Week 3 check' })).toHaveAttribute('href', '/a/sections/s1/quizzes/q1')
  expect(within(c).queryByRole('link', { name: 'Midterm' })).toBeNull()
  expect(within(c).getByText(/^Midterm · 34 \/ 40/)).toBeInTheDocument()
})
