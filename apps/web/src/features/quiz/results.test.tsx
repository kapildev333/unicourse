import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import type { Assessment, Attempt, Question } from './queries'
import { Results } from './results'

const assessment = (published: boolean): Assessment => ({
  id: 'q1', sectionId: 's1', kind: 'quiz', name: 'Week 3 check', description: '', maxPoints: 3, weight: 10, published, position: 1,
})
const questions: Question[] = [
  { id: 'a', kind: 'choice', prompt: 'Which layer routes packets?', points: 2, position: 1, multiple: false, choices: [{ text: 'Transport', correct: false }, { text: 'Network', correct: true }] },
  { id: 'b', kind: 'blank', prompt: 'Capital of France?', points: 1, position: 2, multiple: false, choices: [], answers: ['paris', 'paris france'] },
]
const attempt = (over: Partial<Attempt> = {}): Attempt => ({
  userId: 'u0', startedAt: '2026-10-12T13:05:00Z', deadlineAt: '2099-01-01T09:00:00Z', submittedAt: '2026-10-12T13:40:00Z', autoSubmitted: false,
  answers: { a: { choices: [0] }, b: { text: 'Lyon' } },
  grading: { a: { points: 0, correct: false, feedback: 'Review chapter 2' }, b: { points: 1, correct: true, feedback: '' } },
  score: 1, feedback: 'Good effort', ...over,
})

it('hides score, keys and grading until published, even when the fields are present', () => {
  render(<Results assessment={assessment(false)} attempt={attempt()} questions={questions} />)
  expect(screen.getByText(/^Submitted /)).toBeInTheDocument()
  expect(screen.getByText('Your answer: Transport')).toBeInTheDocument()
  expect(screen.getByText(/appear here once staff release/)).toBeInTheDocument()
  for (const t of [/^Score/, /Correct answer/, /Accepted answers/, 'Correct', 'Incorrect', /Feedback/]) expect(screen.queryByText(t)).toBeNull()
})

it('published: score, per-question points, keys, feedback', () => {
  render(<Results assessment={assessment(true)} attempt={attempt()} questions={questions} />)
  expect(screen.getByText('Score 1 / 3')).toBeInTheDocument()
  expect(screen.getByText('Question 1, 0 / 2 points')).toBeInTheDocument()
  expect(screen.getByText('Question 2, 1 / 1 point')).toBeInTheDocument()
  expect(screen.getByText('Incorrect')).toBeInTheDocument()
  expect(screen.getByText('Correct')).toBeInTheDocument()
  expect(screen.getByText('Correct answer: Network')).toBeInTheDocument()
  expect(screen.getByText('Accepted answers: paris, paris france')).toBeInTheDocument()
  expect(screen.getByText('Feedback: Review chapter 2')).toBeInTheDocument()
  expect(screen.getByRole('heading', { level: 4, name: 'Feedback from staff' })).toBeInTheDocument()
  expect(screen.getByText('Good effort')).toBeInTheDocument()
})

it('automatic submit wording, no-answer wording and the pending line', () => {
  const { rerender } = render(<Results assessment={assessment(false)} attempt={attempt({ autoSubmitted: true, answers: {} })} questions={questions} />)
  expect(screen.getByText(/^Submitted automatically at /)).toBeInTheDocument()
  expect(screen.getAllByText('Your answer: No answer')).toHaveLength(2)
  rerender(<Results assessment={assessment(false)} attempt={attempt({ submittedAt: undefined })} questions={questions} />)
  expect(screen.getByRole('status')).toHaveTextContent('Submitting your answers…')
})

it('focuses the heading when asked', () => {
  render(<Results assessment={assessment(false)} attempt={attempt()} questions={questions} focusHeading />)
  expect(screen.getByRole('heading', { name: 'Your results' })).toHaveFocus()
})
