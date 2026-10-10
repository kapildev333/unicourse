import { afterEach, expect, it, vi } from 'vitest'
import { campusClock, formatDate, formatDateTime, fromZonedInput, setCampus, toZonedInput } from './format'

afterEach(() => {
  setCampus()
})
const ny = () => vi.spyOn(Intl.DateTimeFormat.prototype, 'resolvedOptions')

it('names the zone when the campus differs from the browser', () => {
  const spy = ny()
  spy.mockReturnValue({ ...Intl.DateTimeFormat().resolvedOptions(), timeZone: 'America/New_York' })
  setCampus({ timezone: 'UTC', locale: 'en-US' })
  expect(formatDateTime('2026-10-26T15:00:00Z')).toBe('Oct 26, 2026, 3:00 PM UTC')
  spy.mockRestore()
})

it('adds no suffix when the campus zone is the browser zone', () => {
  const spy = ny()
  spy.mockReturnValue({ ...Intl.DateTimeFormat().resolvedOptions(), timeZone: 'UTC' })
  setCampus({ timezone: 'UTC', locale: 'en-US' })
  expect(formatDateTime('2026-10-26T15:00:00Z')).toBe('Oct 26, 2026, 3:00 PM')
  spy.mockRestore()
})

it('falls back to the browser for an invalid zone or locale', () => {
  setCampus({ timezone: 'Not/AZone', locale: 'xx-!!' })
  expect(formatDateTime('2026-10-26T15:00:00Z')).not.toMatch(/Not/)
})

it('formats a calendar date without shifting it, and in the campus locale', () => {
  setCampus({ timezone: 'Pacific/Auckland', locale: 'en-US' })
  expect(formatDate('2026-10-26')).toBe('Oct 26, 2026')
})

it('datetime-local values are campus wall time', () => {
  setCampus({ timezone: 'UTC', locale: 'en-US' })
  expect(toZonedInput('2026-10-26T15:00:00Z')).toBe('2026-10-26T15:00')
  expect(fromZonedInput('2026-10-26T15:00')).toBe('2026-10-26T15:00:00.000Z')
  setCampus({ timezone: 'America/New_York', locale: 'en-US' })
  expect(fromZonedInput('2026-10-26T15:00')).toBe('2026-10-26T19:00:00.000Z')
  expect(fromZonedInput('')).toBeUndefined()
  expect(fromZonedInput('nonsense')).toBeUndefined()
})

it('handles the DST gap and fold in Europe/Berlin', () => {
  setCampus({ timezone: 'Europe/Berlin', locale: 'en-US' })
  expect(fromZonedInput('2026-03-29T02:30')).toBe('2026-03-29T01:30:00.000Z') // gap: moves forward to 03:30 CEST
  expect(toZonedInput('2026-03-29T01:30:00Z')).toBe('2026-03-29T03:30')
  expect(fromZonedInput('2026-10-25T02:30')).toBe('2026-10-25T01:30:00.000Z') // fold: the later (CET) instant
  expect(fromZonedInput('2026-07-01T12:00')).toBe('2026-07-01T10:00:00.000Z')
})

it('campusClock reads date, ISO weekday and minutes in the campus zone', () => {
  setCampus({ timezone: 'Pacific/Auckland', locale: 'en-US' })
  // Sun 2026-10-25 23:30Z is Monday 12:30 (NZDT +13) the next day
  expect(campusClock(new Date('2026-10-25T23:30:00Z'))).toEqual({ date: '2026-10-26', weekday: 1, minutes: 12 * 60 + 30 })
})
