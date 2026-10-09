import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import { mine, renderSection, sectionFixture } from '../section/test-helpers'
import type { Assessment, Attempt, Question } from './queries'
import { TakeQuizPage } from './take-quiz'

beforeEach(() => vi.useFakeTimers({ shouldAdvanceTime: true }))
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

const S = '/universities/a/sections/s1'
const A = `${S}/assessments/q1`
const err = (status: number, code: string, message = 'm') => ({ status, body: { error: { code, message, details: [] } } })
const quiz = (over: Partial<Assessment> = {}): Assessment => ({
  id: 'q1', sectionId: 's1', kind: 'quiz', name: 'Week 3 check', description: 'Chapters 1 to 3.', maxPoints: 4, weight: 40,
  opensAt: '2020-01-01T09:00:00Z', closesAt: '2099-01-01T09:00:00Z', published: false, position: 1, questionCount: 3, ...over,
})
const questions: Question[] = [
  { id: 'a', kind: 'choice', prompt: 'Which layer routes packets?', points: 2, position: 1, multiple: false, choices: [{ text: 'Transport' }, { text: 'Network' }, { text: 'Link' }] },
  { id: 'b', kind: 'choice', prompt: 'Pick the prime numbers', points: 1, position: 2, multiple: true, choices: [{ text: '2' }, { text: '4' }, { text: '5' }] },
  { id: 'c', kind: 'blank', prompt: 'Capital of France?', points: 1, position: 3, multiple: false, choices: [] },
]
const attempt = (over: Partial<Attempt> = {}): Attempt => ({ userId: 'u0', startedAt: '2026-10-12T13:05:00Z', deadlineAt: '2099-01-01T09:00:00Z', autoSubmitted: false, answers: {}, ...over })

type Setup = { assessment?: Assessment; attempt?: Attempt | null; routes?: Routes }
function setup(o: Setup = {}) {
  let current: Attempt | null = o.attempt === undefined ? null : o.attempt
  const fn = mockApi({
    [`GET ${S}`]: () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: mine('student') }),
    [`GET ${S}/assessments`]: () => ({ body: { items: [o.assessment ?? quiz()] } }),
    [`GET ${A}/questions`]: () => ({ body: { items: questions } }),
    [`GET ${A}/attempts/me`]: () => (current ? { body: current } : err(404, 'not_found')),
    [`POST ${A}/attempts`]: () => {
      current = attempt()
      return { status: 201, body: current }
    },
    [`PUT ${A}/attempts/me`]: async (req) => {
      const body = await req.json()
      current = attempt({ answers: body.answers, ...(body.submit ? { submittedAt: '2026-10-12T13:40:00Z' } : {}) })
      return { body: current }
    },
    ...o.routes,
  })
  renderSection({ path: '/a/sections/s1/quizzes/q1', ui: <TakeQuizPage slug="a" sectionId="s1" assessmentId="q1" />, userId: 'u0' })
  return fn
}
const puts = (fn: ReturnType<typeof mockApi>) => fn.mock.calls.filter(([r]) => r.method === 'PUT')
const body = async (call: unknown[]) => JSON.parse(await (call[0] as Request).text())
const tick = (ms: number) => act(() => vi.advanceTimersByTimeAsync(ms))

it('intro, explicit Start quiz, then the form with radios, checkboxes and a blank', async () => {
  const fn = setup()
  expect(await screen.findByRole('heading', { level: 2, name: 'Week 3 check' })).toBeInTheDocument()
  expect(screen.getByText('3 questions, 4 points.')).toBeInTheDocument()
  expect(screen.getByText('Chapters 1 to 3.')).toBeInTheDocument()
  expect(fn.mock.calls.some(([r]) => r.method === 'POST')).toBe(false)
  fireEvent.click(await screen.findByRole('button', { name: 'Start quiz' }))
  const g1 = await screen.findByRole('group', { name: /Which layer routes packets/ })
  expect(g1).toHaveTextContent('Choose one answer.')
  expect(screen.getAllByRole('radio')).toHaveLength(3)
  const g2 = screen.getByRole('group', { name: /Pick the prime numbers/ })
  expect(g2).toHaveTextContent('Choose all that apply.')
  expect(screen.getAllByRole('checkbox')).toHaveLength(3)
  expect(screen.getByRole('group', { name: /Capital of France/ })).toBeInTheDocument()
  expect(screen.getByLabelText('Your answer')).toBeInTheDocument()
  expect(screen.getByText('Answers save automatically.')).toBeInTheDocument()
})

it('start 409 texts map to page copy', async () => {
  setup({ routes: { [`POST ${A}/attempts`]: () => err(409, 'conflict', 'closed') } })
  fireEvent.click(await screen.findByRole('button', { name: 'Start quiz' }))
  expect(await screen.findByText(/This quiz closed on .* You didn't start it\./)).toBeInTheDocument()
})

it('upcoming and unstarted-closed states have no Start button', async () => {
  setup({ assessment: quiz({ opensAt: '2098-01-01T09:00:00Z' }) })
  expect(await screen.findByText(/^This quiz opens /)).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Start quiz' })).toBeNull()
})

it('autosave: two changes within a second give one PUT without submit, then Saved', async () => {
  const fn = setup({ attempt: attempt() })
  const radios = await screen.findAllByRole('radio')
  fireEvent.click(radios[1])
  await tick(500)
  fireEvent.change(screen.getByLabelText('Your answer'), { target: { value: 'Paris' } })
  await tick(900)
  expect(puts(fn)).toHaveLength(0)
  await tick(200)
  await waitFor(() => expect(puts(fn)).toHaveLength(1))
  expect(await body(puts(fn)[0])).toEqual({ answers: { a: { choices: [1] }, c: { text: 'Paris' } } })
  expect(await screen.findByText('Saved')).toBeInTheDocument()
})

it('a change during an in-flight save sends a second PUT after the first resolves', async () => {
  let release: () => void = () => {}
  const gate = new Promise<void>((r) => (release = r))
  let n = 0
  const fn = setup({
    attempt: attempt(),
    routes: {
      [`PUT ${A}/attempts/me`]: async () => {
        if (++n === 1) await gate
        return { body: attempt() }
      },
    },
  })
  const radios = await screen.findAllByRole('radio')
  fireEvent.click(radios[0])
  await tick(1100)
  await waitFor(() => expect(puts(fn)).toHaveLength(1))
  fireEvent.click(radios[2])
  await tick(1100)
  expect(puts(fn)).toHaveLength(1)
  release()
  await waitFor(() => expect(puts(fn)).toHaveLength(2))
  expect(await body(puts(fn)[1])).toEqual({ answers: { a: { choices: [2] } } })
})

it('a server error shows Not saved and retries after 5 s without losing typed text', async () => {
  let fail = true
  const fn = setup({
    attempt: attempt(),
    routes: { [`PUT ${A}/attempts/me`]: () => (fail ? err(500, 'internal') : { body: attempt() }) },
  })
  const input = await screen.findByLabelText('Your answer')
  fireEvent.change(input, { target: { value: 'Pari' } })
  await tick(1100)
  expect(await screen.findByText('Not saved. Retrying…')).toBeInTheDocument()
  expect(input).toHaveValue('Pari')
  fail = false
  await tick(5100)
  await waitFor(() => expect(puts(fn)).toHaveLength(2))
  expect(await screen.findByText('Saved')).toBeInTheDocument()
  expect(input).toHaveValue('Pari')
})

it('Submit quiz confirms, sends submit:true once and shows the results with focus', async () => {
  const fn = setup({ attempt: attempt({ answers: { a: { choices: [1] } } }) })
  await screen.findAllByRole('radio')
  fireEvent.click(screen.getByRole('button', { name: 'Submit quiz' }))
  expect(await screen.findByRole('dialog', { name: 'Submit quiz?' })).toHaveTextContent('You answered 1 of 3 questions.')
  const confirm = screen.getAllByRole('button', { name: 'Submit quiz' }).at(-1) as HTMLElement
  fireEvent.click(confirm)
  fireEvent.click(confirm)
  const heading = await screen.findByRole('heading', { name: 'Your results' })
  await waitFor(() => expect(heading).toHaveFocus())
  expect(screen.getByText(/^Submitted /)).toBeInTheDocument()
  const sub = puts(fn).filter((c) => (c[0] as Request).url.includes('attempts/me'))
  expect(sub).toHaveLength(1)
  expect(await body(sub[0])).toEqual({ answers: { a: { choices: [1] } }, submit: true })
})

it('at zero it saves once without submit, then polls the attempt until the server submitted it', async () => {
  let swept = false
  const deadlineAt = new Date(Date.now() + 2500).toISOString()
  const fn = setup({
    attempt: attempt({ deadlineAt, answers: { a: { choices: [1] } } }),
    routes: {
      [`GET ${A}/attempts/me`]: () => ({
        body: swept ? attempt({ deadlineAt, answers: { a: { choices: [1] } }, submittedAt: '2026-10-12T13:40:00Z', autoSubmitted: true }) : attempt({ deadlineAt, answers: { a: { choices: [1] } } }),
      }),
    },
  })
  await screen.findAllByRole('radio')
  await tick(2000)
  await tick(2000)
  await tick(2000)
  await waitFor(() => expect(puts(fn)).toHaveLength(1))
  expect(await body(puts(fn)[0])).toEqual({ answers: { a: { choices: [1] } } })
  expect(await screen.findByText('Submitting your answers…')).toBeInTheDocument()
  swept = true
  await tick(3100)
  expect(await screen.findByText(/^Submitted automatically at /)).toBeInTheDocument()
  expect(puts(fn)).toHaveLength(1)
})

it('a 409 at zero with an unsaved change shows the banner and the results', async () => {
  let closed = false
  const deadlineAt = new Date(Date.now() + 2500).toISOString()
  const fn = setup({
    attempt: attempt({ deadlineAt }),
    routes: {
      [`PUT ${A}/attempts/me`]: () => {
        closed = true
        return err(409, 'conflict', 'closed; your answers as of the closing time were submitted')
      },
      [`GET ${A}/attempts/me`]: () => ({ body: closed ? attempt({ deadlineAt, submittedAt: '2026-10-12T13:40:00Z', autoSubmitted: true }) : attempt({ deadlineAt }) }),
    },
  })
  fireEvent.click((await screen.findAllByRole('radio'))[0])
  await tick(3000)
  await tick(1000)
  expect(await screen.findByText("Your last change wasn't saved")).toBeInTheDocument()
  expect(await screen.findByText(/^Submitted automatically at /)).toBeInTheDocument()
  expect(puts(fn).length).toBeGreaterThan(0)
})

it('deadlineAt moved later on the refetch: no save at zero', async () => {
  let n = 0
  const soon = new Date(Date.now() + 2500).toISOString()
  const later = new Date(Date.now() + 3_600_000).toISOString()
  const fn = setup({
    attempt: attempt({ deadlineAt: soon }),
    routes: { [`GET ${A}/attempts/me`]: () => ({ body: attempt({ deadlineAt: ++n === 1 ? soon : later }) }) },
  })
  await screen.findAllByRole('radio')
  await tick(4000)
  await tick(1000)
  expect(puts(fn)).toHaveLength(0)
  expect(screen.getByText(/^Closes in /)).toBeInTheDocument()
})

it('countdown line and the screen-reader announcer speak at thresholds only', async () => {
  setup({ attempt: attempt({ deadlineAt: new Date(Date.now() + 5 * 60_000 + 2500).toISOString() }) })
  await screen.findAllByRole('radio')
  expect(screen.getByText(/^Closes in 6 min$/)).toBeInTheDocument()
  await tick(4000)
  expect(screen.getByText('5 minutes left.')).toBeInTheDocument()
})

const submitted = (answers: Attempt['answers'] = {}) => attempt({ answers, submittedAt: '2026-10-12T13:40:00Z', autoSubmitted: true })
const delay = (ms: number) => new Promise((r) => setTimeout(r, ms))

it('a submit that fails after the close runs the zero path: final save without submit, then polling', async () => {
  let swept = false
  const deadlineAt = new Date(Date.now() + 8000).toISOString()
  const fn = setup({
    attempt: attempt({ deadlineAt, answers: { a: { choices: [1] } } }),
    routes: {
      [`PUT ${A}/attempts/me`]: async (req) => {
        const b = await req.json()
        if (b.submit) {
          await delay(4000)
          return err(500, 'internal')
        }
        swept = true
        return { body: attempt({ deadlineAt, answers: b.answers }) }
      },
      [`GET ${A}/attempts/me`]: () => ({ body: swept ? submitted({ a: { choices: [1] } }) : attempt({ deadlineAt, answers: { a: { choices: [1] } } }) }),
    },
  })
  await screen.findAllByRole('radio')
  await tick(5000)
  fireEvent.click(screen.getByRole('button', { name: 'Submit quiz' }))
  fireEvent.click(screen.getAllByRole('button', { name: 'Submit quiz' }).at(-1) as HTMLElement)
  await tick(3500) // zero is reached with the submit still pending
  await tick(1000) // then the submit fails
  await waitFor(() => expect(puts(fn)).toHaveLength(2))
  expect(await body(puts(fn)[1])).toEqual({ answers: { a: { choices: [1] } } })
  await tick(3100)
  expect(await screen.findByText(/^Submitted automatically at /)).toBeInTheDocument()
})

it('zero reached while a submit is pending sends no extra PUT', async () => {
  const fn = setup({
    attempt: attempt({ deadlineAt: new Date(Date.now() + 5000).toISOString() }),
    routes: {
      [`PUT ${A}/attempts/me`]: async () => {
        await delay(3000)
        return { body: attempt({ submittedAt: '2026-10-12T13:40:00Z' }) }
      },
    },
  })
  await screen.findAllByRole('radio')
  await tick(3000)
  fireEvent.click(screen.getByRole('button', { name: 'Submit quiz' }))
  fireEvent.click(screen.getAllByRole('button', { name: 'Submit quiz' }).at(-1) as HTMLElement)
  await tick(4000)
  expect(await screen.findByRole('heading', { name: 'Your results' })).toBeInTheDocument()
  expect(puts(fn)).toHaveLength(1)
  expect(await body(puts(fn)[0])).toEqual({ answers: {}, submit: true })
})

it('an autosave 409 "already submitted" shows the results without a banner', async () => {
  let done = false
  setup({
    attempt: attempt(),
    routes: {
      [`PUT ${A}/attempts/me`]: () => {
        done = true
        return err(409, 'conflict', 'already submitted')
      },
      [`GET ${A}/attempts/me`]: () => ({ body: done ? submitted() : attempt() }),
    },
  })
  fireEvent.click((await screen.findAllByRole('radio'))[0])
  await tick(1100)
  expect(await screen.findByText(/^Submitted automatically at /)).toBeInTheDocument()
  expect(screen.queryByText("Your last change wasn't saved")).toBeNull()
})

it('an autosave 400 asks for a reload', async () => {
  setup({ attempt: attempt(), routes: { [`PUT ${A}/attempts/me`]: () => err(400, 'validation', 'bad') } })
  fireEvent.click((await screen.findAllByRole('radio'))[0])
  await tick(1100)
  expect(await screen.findByText("Your answers couldn't be saved. Reload the page.")).toBeInTheDocument()
})

it('at zero it waits for an autosave in flight before the final PUT', async () => {
  let release: () => void = () => {}
  const gate = new Promise<void>((r) => (release = r))
  let n = 0
  const deadlineAt = new Date(Date.now() + 3000).toISOString()
  const fn = setup({
    attempt: attempt({ deadlineAt }),
    routes: {
      [`PUT ${A}/attempts/me`]: async () => {
        if (++n === 1) await gate
        return { body: attempt({ deadlineAt }) }
      },
    },
  })
  fireEvent.click((await screen.findAllByRole('radio'))[0])
  await tick(1100)
  await waitFor(() => expect(puts(fn)).toHaveLength(1))
  await tick(3000)
  expect(puts(fn)).toHaveLength(1)
  release()
  await waitFor(() => expect(puts(fn)).toHaveLength(2))
})

it('an edit made while the zero check is pending is saved after the deadline moved later', async () => {
  let release: () => void = () => {}
  const gate = new Promise<void>((r) => (release = r))
  let n = 0
  const soon = new Date(Date.now() + 2500).toISOString()
  const fn = setup({
    attempt: attempt({ deadlineAt: soon }),
    routes: {
      [`GET ${A}/attempts/me`]: async () => {
        if (++n >= 2) {
          await gate
          return { body: attempt({ deadlineAt: new Date(Date.now() + 3_600_000).toISOString() }) }
        }
        return { body: attempt({ deadlineAt: soon }) }
      },
    },
  })
  const radios = await screen.findAllByRole('radio')
  await tick(3000)
  fireEvent.click(radios[0])
  release()
  await tick(500)
  expect(puts(fn)).toHaveLength(0)
  await tick(1500)
  await waitFor(() => expect(puts(fn)).toHaveLength(1))
  expect(await body(puts(fn)[0])).toEqual({ answers: { a: { choices: [0] } } })
})

it('a timed quiz counts down to deadlineAt, not closesAt', async () => {
  setup({
    assessment: quiz({ timeLimitMinutes: 10 }),
    attempt: attempt({ deadlineAt: new Date(Date.now() + 8 * 60_000 + 30_000).toISOString() }),
  })
  await screen.findAllByRole('radio')
  expect(screen.getByText('Time left: 9 min')).toBeInTheDocument()
  expect(screen.getByText(/, 10 min each$/)).toBeInTheDocument()
})

it('zero at deadlineAt (before closesAt) saves once without submit', async () => {
  const fn = setup({
    assessment: quiz({ timeLimitMinutes: 10 }),
    attempt: attempt({ deadlineAt: new Date(Date.now() + 2500).toISOString(), answers: { a: { choices: [1] } } }),
  })
  await screen.findAllByRole('radio')
  await tick(2000)
  await tick(2000)
  await waitFor(() => expect(puts(fn)).toHaveLength(1))
  expect(await body(puts(fn)[0])).toEqual({ answers: { a: { choices: [1] } } })
})

it('the Start intro of a timed quiz states the limit; it adds the clamp when the close comes first', async () => {
  setup({ assessment: quiz({ timeLimitMinutes: 30 }) })
  expect(
    await screen.findByText(
      'You have one attempt and 30 minutes from the moment you start. Your answers are saved as you go and submitted automatically when the time is up or the quiz closes, whichever is first.',
    ),
  ).toBeInTheDocument()
})

it('the timed intro names the close when it comes before the limit would run out', async () => {
  setup({ assessment: quiz({ timeLimitMinutes: 60, closesAt: new Date(Date.now() + 20 * 60_000).toISOString() }) })
  expect(await screen.findByText(/before your time would run out\.$/)).toBeInTheDocument()
  expect(screen.getByText(/^You have one attempt and 60 minutes/)).toBeInTheDocument()
})
