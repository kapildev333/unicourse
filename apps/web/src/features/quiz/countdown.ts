import { formatDateTime } from '../../lib/format'

const MIN = 60_000

/**
 * Visible countdown line. `ms` is the time left until `deadlineAt`. Timed quizzes read "Time left: ..." at any length;
 * untimed ones read "Closes in ..." and print `deadlineAt` (the close) when more than a day remains.
 */
export function countdownText(ms: number, deadlineAt: string, timed = false): string {
  if (ms <= 0) return 'Time is up'
  if (timed) {
    if (ms >= 60 * MIN) {
      const total = Math.ceil(ms / MIN)
      const h = Math.floor(total / 60)
      const m = total % 60
      return m === 0 ? `Time left: ${h} h` : `Time left: ${h} h ${m} min`
    }
    return ms >= MIN ? `Time left: ${Math.ceil(ms / MIN)} min` : `Time left: ${Math.ceil(ms / 1000)} s`
  }
  if (ms >= 24 * 60 * MIN) return `Closes ${formatDateTime(deadlineAt)}`
  if (ms >= 60 * MIN) {
    const total = Math.ceil(ms / MIN)
    const h = Math.floor(total / 60)
    const m = total % 60
    return m === 0 ? `Closes in ${h} h` : `Closes in ${h} h ${m} min`
  }
  if (ms >= MIN) return `Closes in ${Math.ceil(ms / MIN)} min`
  return `Closes in ${Math.ceil(ms / 1000)} s`
}

const thresholds: [number, string][] = [
  [10 * MIN, '10 minutes left.'],
  [5 * MIN, '5 minutes left.'],
  [MIN, '1 minute left.'],
  [30_000, '30 seconds left.'],
]

/** Screen-reader text for the moment a threshold is crossed between two ticks; '' otherwise. */
export function announcement(prevMs: number, ms: number): string {
  if (prevMs > 0 && ms <= 0) return 'Time is up. Submitting your answers.'
  let out = ''
  for (const [t, text] of thresholds) if (prevMs > t && ms <= t && ms > 0) out = text
  return out
}
