import { expect, it } from 'vitest'
import { setCampus } from '../../lib/format'
import { assignmentState, fromLocalInput, minutes, timeZone, toLocalInput, windowLine, windowState } from './window'

const a = { opensAt: '2026-10-12T13:00:00Z', closesAt: '2026-10-12T14:00:00Z' }
const t = (s: string) => Date.parse(s)

it('windowState around both bounds', () => {
  expect(windowState(a, t('2026-10-12T12:59:59Z'))).toBe('upcoming')
  expect(windowState(a, t('2026-10-12T13:00:00Z'))).toBe('open')
  expect(windowState(a, t('2026-10-12T13:59:59Z'))).toBe('open')
  expect(windowState(a, t('2026-10-12T14:00:00Z'))).toBe('closed')
})

it('windowLine names Closes for quizzes and Due for assignments', () => {
  expect(windowLine(a)).toMatch(/^Opens .* · Closes /)
  expect(windowLine(a, 'assignment')).toMatch(/ · Due /)
})

it('local input round-trips and rejects empty or invalid', () => {
  const iso = '2026-10-12T13:30:00.000Z'
  expect(fromLocalInput(toLocalInput(iso))).toBe(iso)
  expect(fromLocalInput('')).toBeUndefined()
  expect(fromLocalInput('nonsense')).toBeUndefined()
})

it('assignmentState has a late window between closesAt and lateUntil', () => {
  const b = { ...a, lateUntil: '2026-10-13T14:00:00Z' }
  expect(assignmentState(b, t('2026-10-12T12:00:00Z'))).toBe('upcoming')
  expect(assignmentState(b, t('2026-10-12T13:30:00Z'))).toBe('open')
  expect(assignmentState(b, t('2026-10-12T14:00:00Z'))).toBe('late')
  expect(assignmentState(b, t('2026-10-13T13:59:59Z'))).toBe('late')
  expect(assignmentState(b, t('2026-10-13T14:00:00Z'))).toBe('closed')
  expect(assignmentState(a, t('2026-10-12T14:00:00Z'))).toBe('closed')
})

it('windowLine appends the limit and the late window', () => {
  expect(windowLine({ ...a, timeLimitMinutes: 10 })).toMatch(/Closes .*, 10 min each$/)
  expect(windowLine({ ...a, lateUntil: '2026-10-13T14:00:00Z' }, 'assignment')).toMatch(/ · Due .*, late until /)
  expect(windowLine(a)).not.toMatch(/each|late/)
})

it('minutes pluralises', () => {
  expect(minutes(1)).toBe('1 minute')
  expect(minutes(30)).toBe('30 minutes')
})

it('local input is campus wall time', () => {
  setCampus({ timezone: 'America/New_York', locale: 'en-US' })
  expect(toLocalInput('2026-10-26T19:00:00Z')).toBe('2026-10-26T15:00')
  expect(fromLocalInput('2026-10-26T15:00')).toBe('2026-10-26T19:00:00.000Z')
  expect(timeZone()).toBe('America/New_York')
  setCampus()
})
