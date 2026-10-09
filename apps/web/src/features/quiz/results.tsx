import { Check, X } from 'lucide-react'
import { useEffect, useRef } from 'react'
import { Badge } from '../../components/badge'
import { formatDateTime } from '../../lib/format'
import { fmtNum } from '../grades/grade'
import { PlainText } from '../section/plain-text'
import type { Assessment, Attempt, Question } from './queries'

const pts = (n: number) => `${fmtNum(n)} ${n === 1 ? 'point' : 'points'}`

function yourAnswer(q: Question, a: Attempt): string {
  const ans = a.answers[q.id]
  if (q.kind === 'blank') return ans?.text?.trim() ? ans.text : 'No answer'
  const texts = (ans?.choices ?? []).map((i) => q.choices[i]?.text).filter((t): t is string => !!t)
  return texts.length ? texts.join(', ') : 'No answer'
}

function keyLine(q: Question): string {
  if (q.kind === 'blank') return q.answers?.length ? `Accepted answers: ${q.answers.join(', ')}` : ''
  const right = q.choices.filter((c) => c.correct).map((c) => c.text)
  if (!right.length) return ''
  return `${right.length === 1 ? 'Correct answer' : 'Correct answers'}: ${right.join(', ')}`
}

/** The student's own attempt after it was (or is about to be) submitted. Grading and keys render only once published. */
export function Results(props: { assessment: Assessment; attempt: Attempt; questions: Question[]; focusHeading?: boolean }) {
  const { assessment, attempt, questions, focusHeading } = props
  const heading = useRef<HTMLHeadingElement>(null)
  useEffect(() => {
    if (focusHeading) heading.current?.focus()
  }, [focusHeading])

  const submitted = !!attempt.submittedAt
  const shown = submitted && assessment.published
  const when = attempt.submittedAt ? formatDateTime(attempt.submittedAt) : ''

  return (
    <section aria-labelledby="results-h" className="flex flex-col gap-4">
      <div>
        <h3 id="results-h" ref={heading} tabIndex={-1} className="text-xl font-bold focus:outline-none">
          Your results
        </h3>
        {submitted ? (
          <p className="mt-1 text-sm text-ink-muted">{attempt.autoSubmitted ? `Submitted automatically at ${when}` : `Submitted ${when}`}</p>
        ) : (
          <p role="status" className="mt-1 text-sm text-ink-muted">
            Submitting your answers…
          </p>
        )}
      </div>
      {submitted &&
        (shown && attempt.score !== undefined ? (
          <p className="text-xl font-bold">
            Score {fmtNum(attempt.score)} / {fmtNum(assessment.maxPoints)}
          </p>
        ) : (
          <p className="text-sm text-ink-muted">Your score and the correct answers appear here once staff release the scores.</p>
        ))}
      {shown && attempt.feedback && (
        <div>
          <h4 className="font-bold">Feedback from staff</h4>
          <PlainText text={attempt.feedback} className="mt-1 max-w-prose" />
        </div>
      )}
      <ol className="flex flex-col gap-4">
        {questions.map((q, i) => {
          const mark = shown ? attempt.grading?.[q.id] : undefined
          const key = shown ? keyLine(q) : ''
          return (
            <li key={q.id}>
              <article aria-label={`Question ${i + 1}`} className="rounded border border-line bg-surface p-4 sm:p-6">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <p className="text-sm text-ink-muted">
                    {mark ? `Question ${i + 1}, ${fmtNum(mark.points)} / ${pts(q.points)}` : `Question ${i + 1}, ${pts(q.points)}`}
                  </p>
                  {mark && (
                    <Badge tone="neutral">
                      {mark.correct ? <Check aria-hidden="true" /> : <X aria-hidden="true" />}
                      {mark.correct ? 'Correct' : 'Incorrect'}
                    </Badge>
                  )}
                </div>
                <p className="mt-1 font-bold whitespace-pre-wrap wrap-break-word">{q.prompt}</p>
                <p className="mt-2 wrap-break-word">Your answer: {yourAnswer(q, attempt)}</p>
                {key && <p className="mt-1 wrap-break-word">{key}</p>}
                {mark?.feedback && <p className="mt-1 whitespace-pre-wrap wrap-break-word">Feedback: {mark.feedback}</p>}
              </article>
            </li>
          )
        })}
      </ol>
    </section>
  )
}
