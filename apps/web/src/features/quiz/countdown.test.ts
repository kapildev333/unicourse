import { expect, it } from 'vitest'
import { announcement, countdownText } from './countdown'
import { formatDateTime } from '../../lib/format'

const iso = '2026-10-12T13:00:00Z'
const m = (min: number, s = 0) => (min * 60 + s) * 1000

it('countdownText by range', () => {
  expect(countdownText(m(25 * 60), iso)).toBe(`Closes ${formatDateTime(iso)}`)
  expect(countdownText(m(3 * 60 + 12), iso)).toBe('Closes in 3 h 12 min')
  expect(countdownText(m(2 * 60), iso)).toBe('Closes in 2 h')
  expect(countdownText(m(12, 1), iso)).toBe('Closes in 13 min')
  expect(countdownText(m(1), iso)).toBe('Closes in 1 min')
  expect(countdownText(59_000, iso)).toBe('Closes in 59 s')
  expect(countdownText(0, iso)).toBe('Time is up')
  expect(countdownText(-5, iso)).toBe('Time is up')
})

it('announcement speaks only when a threshold is crossed', () => {
  expect(announcement(m(10, 1), m(9, 59))).toBe('10 minutes left.')
  expect(announcement(m(5, 1), m(4, 59))).toBe('5 minutes left.')
  expect(announcement(m(1, 1), m(0, 59))).toBe('1 minute left.')
  expect(announcement(31_000, 29_000)).toBe('30 seconds left.')
  expect(announcement(1000, 0)).toBe('Time is up. Submitting your answers.')
  expect(announcement(m(9), m(8, 59))).toBe('')
  expect(announcement(m(4), m(3, 59))).toBe('')
  expect(announcement(m(10, 1), m(10))).toBe('10 minutes left.') // landing exactly on it counts
  expect(announcement(m(10), m(9, 59))).toBe('') // already announced on the previous tick
})
