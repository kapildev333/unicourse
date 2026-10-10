import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { WindowDialog } from './quiz-dialog'
import type { Assessment } from './queries'
import { setCampus } from '../../lib/format'
import { fromLocalInput } from './window'

afterEach(() => {
  vi.unstubAllGlobals()
  setCampus()
})

const A = '/universities/a/sections/s1/assessments'
const quiz: Assessment = {
  id: 'q1',
  sectionId: 's1',
  kind: 'quiz',
  name: 'Week 3 check',
  description: '',
  maxPoints: 10,
  weight: 40,
  opensAt: '2026-10-12T13:00:00Z',
  closesAt: '2026-10-12T14:00:00Z',
  published: false,
  position: 1,
  questionCount: 3,
}

function setup(kind: 'quiz' | 'assignment', routes: Parameters<typeof mockApi>[0], assessment?: Assessment) {
  const fn = mockApi({
    [`GET ${A}`]: () => ({ body: { items: [{ ...quiz, id: 'other', weight: 25 }, ...(assessment ? [assessment] : [])] } }),
    ...routes,
  })
  const onClose = vi.fn()
  const onSaved = vi.fn()
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  render(
    <QueryClientProvider client={client}>
      <WindowDialog slug="a" sectionId="s1" kind={kind} assessment={assessment} onClose={onClose} onSaved={onSaved} />
    </QueryClientProvider>,
  )
  return { fn, onClose, onSaved }
}
const sent = (fn: ReturnType<typeof mockApi>, method: string) => fn.mock.calls.filter(([r]) => r.method === method)
const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } })
const created = (r: Request) => r.json().then((b) => ({ status: 201, body: { ...quiz, ...b } }))

it('creates a quiz without maxPoints; times go out as ISO', async () => {
  const { fn, onSaved } = setup('quiz', { [`POST ${A}`]: created })
  type('Title', ' Week 3 check ')
  type('Opens', '2026-10-12T09:00')
  type('Closes', '2026-10-12T10:00')
  type('Weight (%)', '40')
  expect(await screen.findByText(/Other assessments total 25%\./)).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Create quiz' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalledWith('Week 3 check created.'))
  expect(JSON.parse(await sent(fn, 'POST')[0][0].text())).toEqual({
    kind: 'quiz',
    name: 'Week 3 check',
    description: '',
    opensAt: fromLocalInput('2026-10-12T09:00'),
    closesAt: fromLocalInput('2026-10-12T10:00'),
    weight: 40,
  })
})

it('creates an assignment with maxPoints and the Due label', async () => {
  const { fn, onSaved } = setup('assignment', { [`POST ${A}`]: created })
  type('Title', 'Essay 1')
  type('Opens', '2026-10-12T09:00')
  type('Due', '2026-10-19T23:59')
  type('Max points', '20')
  type('Weight (%)', '20')
  fireEvent.click(screen.getByRole('button', { name: 'Create assignment' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(JSON.parse(await sent(fn, 'POST')[0][0].text())).toMatchObject({ kind: 'assignment', maxPoints: 20, weight: 20 })
})

it('starts a new dialog at the next full hour', () => {
  setup('quiz', {})
  const v = (screen.getByLabelText('Opens') as HTMLInputElement).value
  expect(v).toMatch(/T\d\d:00$/)
  expect(Date.parse(fromLocalInput(v)!)).toBeGreaterThan(Date.now())
  expect(Date.parse(fromLocalInput((screen.getByLabelText('Closes') as HTMLInputElement).value)!) - Date.parse(fromLocalInput(v)!)).toBe(3_600_000)
})

it('refuses Closes before Opens without a request', async () => {
  const { fn } = setup('quiz', {})
  type('Title', 'Week 3 check')
  type('Opens', '2026-10-12T10:00')
  type('Closes', '2026-10-12T09:00')
  type('Weight (%)', '40')
  fireEvent.click(screen.getByRole('button', { name: 'Create quiz' }))
  expect(await screen.findByText('Closes must be after Opens.')).toBeInTheDocument()
  expect(screen.getByLabelText('Closes')).toHaveAttribute('aria-invalid', 'true')
  expect(sent(fn, 'POST')).toHaveLength(0)
})

it('maps a server closesAt error under Closes', async () => {
  setup('quiz', {
    [`POST ${A}`]: () => ({
      status: 400,
      body: { error: { code: 'validation_failed', message: 'bad', details: [{ field: 'closesAt', message: 'must be after opensAt' }] } },
    }),
  })
  type('Title', 'Week 3 check')
  type('Weight (%)', '40')
  fireEvent.click(screen.getByRole('button', { name: 'Create quiz' }))
  expect(await screen.findByText('Enter a time after Opens.')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByLabelText('Closes')).toHaveAttribute('aria-invalid', 'true'))
})

it('edit sends only the changed fields; unchanged closes without a request', async () => {
  const { fn, onClose, onSaved } = setup('quiz', { [`PATCH ${A}/q1`]: (r) => r.json().then((b) => ({ body: { ...quiz, ...b } })) }, quiz)
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  expect(onClose).toHaveBeenCalled()
  expect(sent(fn, 'PATCH')).toHaveLength(0)
  type('Weight (%)', '35')
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalledWith('Week 3 check saved.'))
  expect(JSON.parse(await sent(fn, 'PATCH')[0][0].text())).toEqual({ weight: 35 })
})

const asg: Assessment = { ...quiz, id: 'a1', kind: 'assignment', maxPoints: 20, closesAt: '2026-10-19T20:00:00Z' }

it('quiz has Time limit, assignment has no Time limit', () => {
  setup('quiz', {})
  expect(screen.getByLabelText('Time limit (minutes)')).toBeInTheDocument()
  expect(screen.queryByLabelText('Late submissions until')).toBeNull()
})

it('assignment shows the penalty only once Late submissions until is filled', () => {
  setup('assignment', {})
  expect(screen.queryByLabelText('Time limit (minutes)')).toBeNull()
  expect(screen.queryByLabelText('Late penalty (%)')).toBeNull()
  type('Late submissions until', '2026-10-21T09:00')
  expect(screen.getByLabelText('Late penalty (%)')).toHaveValue('0')
})

it('creates a quiz with the time limit', async () => {
  const { fn, onSaved } = setup('quiz', { [`POST ${A}`]: created })
  type('Title', 'T')
  type('Weight (%)', '40')
  type('Time limit (minutes)', '10')
  fireEvent.click(screen.getByRole('button', { name: 'Create quiz' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(JSON.parse(await sent(fn, 'POST')[0][0].text())).toMatchObject({ timeLimitMinutes: 10 })
})

it('creates an assignment with a late window and penalty', async () => {
  const { fn, onSaved } = setup('assignment', { [`POST ${A}`]: created })
  type('Title', 'Essay')
  type('Opens', '2026-10-12T09:00')
  type('Due', '2026-10-19T23:59')
  type('Late submissions until', '2026-10-21T09:00')
  type('Late penalty (%)', '20')
  type('Max points', '20')
  type('Weight (%)', '20')
  fireEvent.click(screen.getByRole('button', { name: 'Create assignment' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(JSON.parse(await sent(fn, 'POST')[0][0].text())).toMatchObject({
    lateUntil: fromLocalInput('2026-10-21T09:00'),
    latePenaltyPercent: 20,
  })
})

it('an unfilled late time sends neither lateUntil nor a penalty', async () => {
  const { fn, onSaved } = setup('assignment', { [`POST ${A}`]: created })
  type('Title', 'Essay')
  type('Max points', '20')
  type('Weight (%)', '20')
  fireEvent.click(screen.getByRole('button', { name: 'Create assignment' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  const body = JSON.parse(await sent(fn, 'POST')[0][0].text())
  expect(body).not.toHaveProperty('lateUntil')
  expect(body).not.toHaveProperty('latePenaltyPercent')
  expect(body).not.toHaveProperty('timeLimitMinutes')
})

it.each([
  ['quiz', 'Time limit (minutes)', '0', 'Enter whole minutes from 1 to 1,440.'],
  ['quiz', 'Time limit (minutes)', '1441', 'Enter whole minutes from 1 to 1,440.'],
  ['quiz', 'Time limit (minutes)', '1.5', 'Enter whole minutes from 1 to 1,440.'],
] as const)('refuses %s %s=%s without a request', async (kind, label, value, message) => {
  const { fn } = setup(kind, {})
  type('Title', 'T')
  type('Weight (%)', '40')
  type(label, value)
  fireEvent.click(screen.getByRole('button', { name: 'Create quiz' }))
  expect(await screen.findByText(message)).toBeInTheDocument()
  expect(sent(fn, 'POST')).toHaveLength(0)
})

it('refuses a late time not after Due and a penalty over 100 without a request', async () => {
  const { fn } = setup('assignment', {})
  type('Title', 'Essay')
  type('Due', '2026-10-19T23:59')
  type('Late submissions until', '2026-10-19T23:59')
  type('Max points', '20')
  type('Weight (%)', '20')
  type('Late penalty (%)', '101')
  fireEvent.click(screen.getByRole('button', { name: 'Create assignment' }))
  expect(await screen.findByText('Late submissions until must be after Due.')).toBeInTheDocument()
  expect(screen.getByText('Enter a penalty from 0 to 100.')).toBeInTheDocument()
  expect(sent(fn, 'POST')).toHaveLength(0)
})

it('maps a server lateUntil error under its field', async () => {
  setup('assignment', {
    [`POST ${A}`]: () => ({
      status: 400,
      body: { error: { code: 'validation_failed', message: 'bad', details: [{ field: 'lateUntil', message: 'must be after closesAt' }] } },
    }),
  })
  type('Title', 'Essay')
  type('Max points', '20')
  type('Weight (%)', '20')
  fireEvent.click(screen.getByRole('button', { name: 'Create assignment' }))
  expect(await screen.findByText('Late submissions until must be after Due.')).toBeInTheDocument()
})

it('clearing the limit sends null', async () => {
  const timed = { ...quiz, timeLimitMinutes: 10 }
  const one = setup('quiz', { [`PATCH ${A}/q1`]: (r) => r.json().then((b) => ({ body: { ...timed, ...b } })) }, timed)
  type('Time limit (minutes)', '')
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(one.onSaved).toHaveBeenCalled())
  expect(JSON.parse(await sent(one.fn, 'PATCH')[0][0].text())).toEqual({ timeLimitMinutes: null })
})

it('clearing the late time sends null', async () => {
  const withLate = { ...asg, lateUntil: '2026-10-21T09:00:00Z', latePenaltyPercent: 20 }
  const { fn, onSaved } = setup('assignment', { [`PATCH ${A}/a1`]: (r) => r.json().then((b) => ({ body: { ...withLate, ...b } })) }, withLate)
  type('Late submissions until', '')
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(JSON.parse(await sent(fn, 'PATCH')[0][0].text())).toEqual({ lateUntil: null })
})

it('changing only the penalty sends only the penalty', async () => {
  const withLate = { ...asg, lateUntil: '2026-10-21T09:00:00Z', latePenaltyPercent: 20 }
  const { fn, onSaved } = setup('assignment', { [`PATCH ${A}/a1`]: (r) => r.json().then((b) => ({ body: { ...withLate, ...b } })) }, withLate)
  type('Late penalty (%)', '25')
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalled())
  expect(JSON.parse(await sent(fn, 'PATCH')[0][0].text())).toEqual({ latePenaltyPercent: 25 })
})

it('the hint names the campus zone and the fields show campus wall time', () => {
  setCampus({ timezone: 'America/New_York', locale: 'en-US' })
  setup('quiz', {}, quiz)
  expect(screen.getAllByText('Times are in America/New_York.').length).toBeGreaterThan(0)
  expect(screen.getByLabelText('Opens')).toHaveValue('2026-10-12T09:00')
})
