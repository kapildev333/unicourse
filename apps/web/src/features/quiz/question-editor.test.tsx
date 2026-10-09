import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import { mine, renderSection, sectionFixture } from '../section/test-helpers'
import { QuestionsPage, mapItemErrors, toBody, toDrafts } from './question-editor'
import type { Assessment, Question } from './queries'

afterEach(() => vi.unstubAllGlobals())

const S = '/universities/a/sections/s1'
const quiz: Assessment = {
  id: 'q1',
  sectionId: 's1',
  kind: 'quiz',
  name: 'Week 3 check',
  description: '',
  maxPoints: 3,
  weight: 40,
  opensAt: '2099-01-01T09:00:00Z',
  closesAt: '2099-01-01T10:00:00Z',
  published: false,
  position: 1,
  questionCount: 2,
}
const q1: Question = {
  id: 'k1',
  kind: 'choice',
  prompt: 'Which layer routes packets?',
  points: 2,
  position: 1,
  multiple: false,
  choices: [
    { text: 'Transport', correct: false },
    { text: 'Network', correct: true },
  ],
}
const q2: Question = { id: 'k2', kind: 'blank', prompt: 'Capital of France?', points: 1, position: 2, multiple: false, choices: [], answers: ['Paris', 'paris france'] }
const err = (status: number, code: string, details: unknown[] = []) => ({ status, body: { error: { code, message: 'm', details } } })
const items = [q1, q2]

let router: ReturnType<typeof renderSection>['router']
function setup(routes: Routes, o: { questions?: Question[]; attempts?: number; submitted?: number; student?: boolean } = {}) {
  const fn = mockApi({
    [`GET ${S}`]: () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: mine(o.student ? 'student' : undefined) }),
    [`GET ${S}/assessments`]: () => ({ body: { items: [quiz] } }),
    [`GET ${S}/assessments/q1/questions`]: () => ({ body: { items: o.questions ?? items } }),
    [`GET ${S}/assessments/q1/attempts`]: () => ({
      body: { items: Array.from({ length: o.attempts ?? 0 }, (_, i) => ({ userId: `u${i}`, email: 'a@a.edu', displayName: 'Dana Reyes', startedAt: '2026-10-12T13:05:00Z', autoSubmitted: false, ...(i < (o.submitted ?? 0) && { submittedAt: '2026-10-12T13:30:00Z' }) })) },
    }),
    ...routes,
  })
  ;({ router } = renderSection({ path: '/a/sections/s1/quizzes/q1/questions', ui: <QuestionsPage slug="a" sectionId="s1" assessmentId="q1" />, userId: o.student ? 'u0' : 'u-ins' }))
  return fn
}
const PUT = `PUT ${S}/assessments/q1/questions`
const put = (fn: ReturnType<typeof mockApi>) => fn.mock.calls.filter(([r]) => r.method === 'PUT')
const body = async (fn: ReturnType<typeof mockApi>) => put(fn)[0][0].json()
const save = () => fireEvent.click(screen.getByRole('button', { name: 'Save questions' }))
const cards = () => screen.getAllByRole('article')

it('loads the questions into cards with a total line', async () => {
  setup({})
  expect(await screen.findByRole('heading', { level: 2, name: 'Questions: Week 3 check' })).toBeInTheDocument()
  expect(cards()).toHaveLength(2)
  expect(screen.getByDisplayValue('Which layer routes packets?')).toBeInTheDocument()
  expect(screen.getByDisplayValue('Network')).toBeInTheDocument()
  expect((screen.getByLabelText('Accepted answers') as HTMLTextAreaElement).value).toBe('Paris\nparis france')
  expect(screen.getByText('2 questions, 3 points')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Save questions' })).toBeDisabled()
})

it('adds both kinds, moves, removes, and PUTs the body in order with trimmed lines', async () => {
  const fn = setup({ [PUT]: (r) => r.json().then((b: { items: unknown[] }) => ({ body: { items: b.items.map((it, i) => ({ id: `n${i}`, position: i, multiple: false, choices: [], ...(it as object) })) } })) }, { questions: [q1] })
  await screen.findByText('Question 1')
  fireEvent.click(screen.getByRole('button', { name: 'Add fill in the blank' }))
  await waitFor(() => expect(within(cards()[1]).getByLabelText('Prompt')).toHaveFocus())
  fireEvent.change(within(cards()[1]).getByLabelText('Prompt'), { target: { value: '  Capital of France?  ' } })
  fireEvent.change(within(cards()[1]).getByLabelText('Accepted answers'), { target: { value: ' Paris \n\n paris france ' } })
  fireEvent.click(screen.getByRole('button', { name: 'Add multiple choice' }))
  const third = cards()[2]
  fireEvent.change(within(third).getByLabelText('Prompt'), { target: { value: 'Pick' } })
  fireEvent.change(within(third).getByLabelText('Choice 1'), { target: { value: ' A ' } })
  fireEvent.change(within(third).getByLabelText('Choice 2'), { target: { value: 'B' } })
  fireEvent.click(within(third).getByRole('checkbox', { name: 'Correct (choice 2)' }))
  fireEvent.click(screen.getByRole('button', { name: 'Move question 3 up' }))
  await waitFor(() => expect(screen.getByRole('button', { name: 'Move question 2 up' })).toHaveFocus())
  fireEvent.click(screen.getAllByRole('button', { name: /^Remove question 1$/ })[0]) // drops the first question
  expect(screen.getByText('2 questions, 2 points')).toBeInTheDocument()
  save()
  await screen.findByText('Questions saved: 2 questions, 2 points.')
  expect(await body(fn)).toEqual({
    items: [
      { kind: 'choice', prompt: 'Pick', points: 1, choices: [{ text: 'A', correct: false }, { text: 'B', correct: true }] },
      { kind: 'blank', prompt: 'Capital of France?', points: 1, answers: ['Paris', 'paris france'] },
    ],
  })
})

it('remove focuses the previous card heading, or the first Add button', async () => {
  setup({})
  await screen.findByText('Question 2')
  fireEvent.click(screen.getByRole('button', { name: 'Remove question 2' }))
  await waitFor(() => expect(screen.getByRole('heading', { name: 'Question 1' })).toHaveFocus())
  fireEvent.click(screen.getByRole('button', { name: 'Remove question 1' }))
  expect(await screen.findByText('No questions yet')).toBeInTheDocument()
  await waitFor(() => expect(screen.getByRole('button', { name: 'Add multiple choice' })).toHaveFocus())
})

it('client errors per field without a request', async () => {
  const fn = setup({}, { questions: [] })
  fireEvent.click(await screen.findByRole('button', { name: 'Add multiple choice' }))
  fireEvent.click(screen.getByRole('button', { name: 'Add fill in the blank' }))
  fireEvent.change(within(cards()[0]).getByLabelText('Points'), { target: { value: '0' } })
  save()
  expect(await screen.findAllByText('Write a prompt of at most 2,000 characters.')).toHaveLength(2)
  expect(screen.getByText('Enter points greater than 0 and at most 1,000, like 2 or 2.5.')).toBeInTheDocument()
  expect(screen.getByText(/^Give 2 to 10 choices/)).toBeInTheDocument()
  expect(screen.getByText(/^Give 1 to 20 accepted answers/)).toBeInTheDocument()
  expect(put(fn)).toHaveLength(0)
  await waitFor(() => expect(within(cards()[0]).getByLabelText('Prompt')).toHaveFocus())
})

it('maps a server items[0].choices error to that card and focuses its first invalid control', async () => {
  setup({
    [PUT]: () =>
      err(400, 'validation_failed', [
        { field: 'items[0].choices', message: 'must have at least one correct choice' },
        { field: 'items[0].choices[0].text', message: 'is required' },
      ]),
  })
  await screen.findByText('Question 2')
  fireEvent.change(screen.getByLabelText('Choice 1'), { target: { value: 'Link' } }) // dirty, so Save is enabled
  save()
  expect(await within(cards()[0]).findByText(/^Give 2 to 10 choices/)).toBeInTheDocument()
  expect(within(cards()[1]).queryByText(/^Give 2 to 10 choices/)).toBeNull()
  await waitFor(() => expect(within(cards()[0]).getByLabelText('Choice 1')).toHaveFocus())
  expect(within(cards()[0]).getByLabelText('Choice 1')).toHaveAttribute('aria-invalid', 'true')
})

it('mapItemErrors collapses sub-paths and keeps the rest', () => {
  const { errors, rest } = mapItemErrors([
    { field: 'items[2].choices[3].text', message: 'is required' },
    { field: 'items', message: 'must have at most 100 questions' },
  ])
  expect(Object.keys(errors)).toEqual(['2'])
  expect(errors[2].choices).toMatch(/^Give 2 to 10/)
  expect(rest).toHaveLength(1)
})

it('toDrafts and toBody round-trip', () => {
  const body = toBody(toDrafts(items))
  expect(body.items[0]).toEqual({ id: 'k1', kind: 'choice', prompt: q1.prompt, points: 2, choices: q1.choices.map((c) => ({ text: c.text, correct: !!c.correct })) })
  expect(body.items[1]).toEqual({ id: 'k2', kind: 'blank', prompt: q2.prompt, points: 1, answers: ['Paris', 'paris france'] })
})

const NOTE = 'Students have started: you can fix prompts, points and answer keys; every submitted attempt is regraded when you save.'
const echo = (r: Request) => r.json().then((b: { items: object[] }) => ({ body: { items: b.items.map((it, i) => ({ position: i, multiple: false, choices: [], ...it })) } }))

it('with attempts: note shown, structure disabled, inputs enabled, ids sent, confirm names the count, status names the regrade', async () => {
  const fn = setup({ [PUT]: echo }, { attempts: 3, submitted: 2 })
  expect(await screen.findByText(NOTE)).toBeInTheDocument()
  expect(cards()).toHaveLength(2)
  for (const name of ['Add multiple choice', 'Add fill in the blank', 'Move question 1 down', 'Move question 2 up', 'Remove question 1', 'Remove choice 1', 'Remove choice 2']) {
    const b = screen.getAllByRole('button', { name })[0]
    expect(b).toBeDisabled()
    expect(b).toHaveAttribute('title', NOTE)
  }
  expect(screen.getByRole('button', { name: 'Add choice' })).toBeDisabled()
  expect(screen.queryByRole('button', { name: 'Save questions' })).toBeNull()
  const save = screen.getByRole('button', { name: 'Save and regrade' })
  expect(save).toBeDisabled()
  fireEvent.change(within(cards()[0]).getByLabelText('Points'), { target: { value: '3' } })
  fireEvent.change(screen.getAllByLabelText('Choice 2')[0], { target: { value: 'Net' } })
  fireEvent.click(screen.getByLabelText('Correct (choice 1)'))
  fireEvent.click(save)
  const dialog = await screen.findByRole('dialog', { name: 'Save and regrade?' })
  expect(dialog).toHaveTextContent('2 submitted attempts will be regraded with these answer keys and points. Points changed in a review are kept.')
  expect(put(fn)).toHaveLength(0)
  fireEvent.click(within(dialog).getByRole('button', { name: 'Save and regrade' }))
  expect(await screen.findByText('Saved. 2 attempts regraded.')).toBeInTheDocument()
  const sent = (await body(fn)) as { items: { id: string; points: number }[] }
  expect(sent.items.map((i) => i.id)).toEqual(['k1', 'k2'])
  expect(sent.items[0].points).toBe(3)
})

it('the confirm dialog with no submitted attempt says nothing is regraded; one attempt is singular', async () => {
  setup({ [PUT]: echo }, { attempts: 1 })
  await screen.findByText(NOTE)
  fireEvent.change(screen.getAllByLabelText('Choice 2')[0], { target: { value: 'Net' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save and regrade' }))
  const dialog = await screen.findByRole('dialog', { name: 'Save and regrade?' })
  expect(dialog).toHaveTextContent('No attempt is submitted yet, so nothing is regraded. Attempts in progress are graded with these keys when they are submitted.')
  fireEvent.click(within(dialog).getByRole('button', { name: 'Save and regrade' }))
  expect(await screen.findByText('Saved. 0 attempts regraded.')).toBeInTheDocument()
})

it('Cancel in the confirm dialog sends nothing', async () => {
  const fn = setup({ [PUT]: echo }, { attempts: 1, submitted: 1 })
  await screen.findByText(NOTE)
  fireEvent.change(screen.getAllByLabelText('Choice 2')[0], { target: { value: 'Net' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save and regrade' }))
  const dialog = await screen.findByRole('dialog', { name: 'Save and regrade?' })
  expect(dialog).toHaveTextContent('1 submitted attempt will be regraded')
  fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }))
  await waitFor(() => expect(screen.queryByRole('dialog', { name: 'Save and regrade?' })).toBeNull())
  expect(put(fn)).toHaveLength(0)
})

it('a 409 while editing shows the note in a banner, reloads the questions and locks the structure', async () => {
  let started = false
  const fn = setup({
    [PUT]: () => {
      started = true
      return err(409, 'conflict')
    },
    [`GET ${S}/assessments/q1/questions`]: () => ({ body: { items: started ? [{ ...q1, prompt: 'Server prompt' }, q2] : items } }),
    [`GET ${S}/assessments/q1/attempts`]: () => ({
      body: { items: started ? [{ userId: 'u0', email: 'a@a.edu', displayName: 'Dana Reyes', startedAt: '2026-10-12T13:05:00Z', autoSubmitted: false }] : [] },
    }),
  })
  await screen.findByText('Question 2')
  fireEvent.change(screen.getByLabelText('Choice 1'), { target: { value: 'Link' } })
  save()
  const banner = await screen.findByRole('alert')
  expect(banner).toHaveTextContent('Questions not saved')
  expect(banner).toHaveTextContent(NOTE)
  expect(await screen.findByDisplayValue('Server prompt')).toBeInTheDocument()
  expect(screen.queryByDisplayValue('Link')).toBeNull()
  expect(screen.getByRole('button', { name: 'Add multiple choice' })).toBeDisabled()
  expect(put(fn)).toHaveLength(1)
})

it('a 400 on items[i].id (a question added before a student started) switches to regrade mode', async () => {
  let started = false
  setup({
    [PUT]: () => {
      started = true
      return err(400, 'validation_failed', [{ field: 'items[2].id', message: 'is required once a student has started' }])
    },
    [`GET ${S}/assessments/q1/attempts`]: () => ({
      body: { items: started ? [{ userId: 'u0', email: 'a@a.edu', displayName: 'Dana Reyes', startedAt: '2026-10-12T13:05:00Z', autoSubmitted: false }] : [] },
    }),
  })
  await screen.findByText('Question 2')
  fireEvent.click(screen.getByRole('button', { name: 'Add fill in the blank' }))
  fireEvent.change(within(cards()[2]).getByLabelText('Prompt'), { target: { value: 'New' } })
  fireEvent.change(within(cards()[2]).getByLabelText('Accepted answers'), { target: { value: 'x' } })
  save()
  const banner = await screen.findByRole('alert')
  expect(banner).toHaveTextContent('Questions not saved')
  expect(banner).toHaveTextContent(NOTE)
  await waitFor(() => expect(cards()).toHaveLength(2))
  expect(screen.getByRole('button', { name: 'Add multiple choice' })).toBeDisabled()
  expect(screen.getByRole('button', { name: 'Save and regrade' })).toBeInTheDocument()
})

it('reports a quiz that is not in the list', async () => {
  setup({ [`GET ${S}/assessments`]: () => ({ body: { items: [] } }) })
  expect(await screen.findByText('This quiz doesn\'t exist or was deleted.')).toBeInTheDocument()
})

it('does not ask before leaving when clean', async () => {
  setup({})
  await screen.findByText('Question 2')
  fireEvent.click(screen.getByRole('link', { name: 'Participants' }))
  await waitFor(() => expect(router.state.location.pathname).toBe('/a/sections/s1/participants'))
  expect(screen.queryByRole('dialog', { name: 'Discard unsaved changes?' })).toBeNull()
})

it('releases the unsaved-changes guard after a successful save', async () => {
  setup({ [PUT]: (r) => r.json().then((b: { items: unknown[] }) => ({ body: { items: b.items.map((it, i) => ({ id: `n${i}`, position: i, multiple: false, choices: [], ...(it as object) })) } })) })
  await screen.findByText('Question 2')
  fireEvent.change(screen.getByLabelText('Choice 1'), { target: { value: 'Link' } })
  save()
  await screen.findByText(/^Questions saved/)
  fireEvent.click(screen.getByRole('link', { name: 'Participants' }))
  await waitFor(() => expect(router.state.location.pathname).toBe('/a/sections/s1/participants'))
  expect(screen.queryByRole('dialog', { name: 'Discard unsaved changes?' })).toBeNull()
})

it('a server field error clears when a question moves', async () => {
  setup({ [PUT]: () => err(400, 'validation_failed', [{ field: 'items[1].prompt', message: 'is required' }]) })
  await screen.findByText('Question 2')
  fireEvent.change(screen.getByLabelText('Choice 1'), { target: { value: 'Link' } })
  save()
  expect(await within(cards()[1]).findByText('Write a prompt of at most 2,000 characters.')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Move question 2 up' }))
  await waitFor(() => expect(screen.queryByText('Write a prompt of at most 2,000 characters.')).toBeNull())
})

it('a student sees No access, not the editor', async () => {
  setup({}, { student: true })
  expect(await screen.findByText("Only this section's staff can change its quizzes.")).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Save questions' })).toBeNull()
  expect(screen.queryByLabelText('Prompt')).toBeNull()
})

it('guards navigation when dirty', async () => {
  setup({})
  await screen.findByText('Question 2')
  fireEvent.change(screen.getByLabelText('Choice 1'), { target: { value: 'Link' } })
  fireEvent.click(screen.getByRole('link', { name: 'Participants' }))
  expect(await screen.findByRole('dialog', { name: 'Discard unsaved changes?' })).toBeInTheDocument()
})
