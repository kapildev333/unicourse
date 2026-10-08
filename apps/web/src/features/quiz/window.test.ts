import { expect, it } from 'vitest'
import { fromLocalInput, toLocalInput, windowLine, windowState } from './window'

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
