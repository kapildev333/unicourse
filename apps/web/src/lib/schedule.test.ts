import { describe, expect, it } from 'vitest'
import { assignLanes, classCode, clashesFor, currentTerm, formatMeeting, overlaps, toMinutes } from './schedule'

const s = (weekday: number, startsAt: string, endsAt: string) => ({ weekday, startsAt, endsAt })

describe('overlaps', () => {
  it.each([
    [s(1, '10:00', '11:00'), s(1, '11:00', '12:00'), false],
    [s(1, '10:00', '11:01'), s(1, '11:00', '12:00'), true],
    [s(1, '10:00', '11:00'), s(1, '10:00', '11:00'), true],
    [s(1, '09:00', '12:00'), s(1, '10:00', '11:00'), true],
    [s(1, '10:00', '11:00'), s(2, '10:00', '11:00'), false],
  ])('%j vs %j -> %s (both orders)', (a, b, want) => {
    expect(overlaps(a, b)).toBe(want)
    expect(overlaps(b, a)).toBe(want)
  })
})

describe('clashesFor', () => {
  const a = { id: 'a', meetings: [s(1, '10:00', '11:00'), s(3, '10:00', '11:00')] }
  const b = { id: 'b', meetings: [s(1, '10:30', '11:30'), s(3, '10:30', '11:30')] }
  const c = { id: 'c', meetings: [s(1, '10:00', '10:30')] }
  const none = { id: 'n', meetings: [] }
  it('excludes itself, lists each once, keeps order', () => {
    expect(clashesFor(a, [a, c, b]).map((x) => x.id)).toEqual(['c', 'b'])
  })
  it('no meetings clashes with nothing', () => {
    expect(clashesFor(none, [a, b])).toEqual([])
    expect(clashesFor(a, [none])).toEqual([])
  })
  it('per-meeting use', () => {
    expect(clashesFor({ id: 'a', meetings: [a.meetings[1]] }, [b, c]).map((x) => x.id)).toEqual(['b'])
  })
})

describe('currentTerm', () => {
  const past = { startsOn: '2026-01-10', endsOn: '2026-05-20' }
  const fall = { startsOn: '2026-08-25', endsOn: '2026-12-15' }
  const winter = { startsOn: '2027-01-05', endsOn: '2027-02-01' }
  const later = { startsOn: '2027-03-01', endsOn: '2027-06-01' }
  it('containing today, inclusive bounds', () => {
    expect(currentTerm([past, fall], '2026-10-06')).toBe(fall)
    expect(currentTerm([fall], '2026-08-25')).toBe(fall)
    expect(currentTerm([fall], '2026-12-15')).toBe(fall)
  })
  it('several containing -> earliest start', () => {
    const o = { startsOn: '2026-09-01', endsOn: '2026-12-01' }
    expect(currentTerm([o, fall], '2026-10-06')).toBe(fall)
  })
  it('next upcoming, not the later one', () => {
    expect(currentTerm([later, winter, past], '2026-12-20')).toBe(winter)
  })
  it('none', () => {
    expect(currentTerm([past], '2026-10-06')).toBeUndefined()
    expect(currentTerm([], '2026-10-06')).toBeUndefined()
  })
})

describe('formatting', () => {
  it('classCode', () => expect(classCode({ course: { code: 'CS101' }, number: '001' })).toBe('CS101-001'))
  it('formatMeeting', () => {
    const m = { ...s(1, '10:00', '11:15'), location: 'Room 101' }
    expect(formatMeeting(m)).toBe('Mon 10:00–11:15 · Room 101')
    expect(formatMeeting(m, true)).toBe('Monday 10:00–11:15 · Room 101')
    expect(formatMeeting({ ...m, location: '' })).toBe('Mon 10:00–11:15')
  })
  it('toMinutes', () => expect(toMinutes('09:05')).toBe(545))
})

describe('assignLanes', () => {
  it('no overlap', () => expect(assignLanes([s(1, '09:00', '10:00'), s(1, '11:00', '12:00')])).toEqual({ lane: [0, 0], lanes: 1 }))
  it('touching shares a lane', () => expect(assignLanes([s(1, '09:00', '10:00'), s(1, '10:00', '11:00')]).lanes).toBe(1))
  it('overlapping', () => expect(assignLanes([s(1, '09:00', '10:30'), s(1, '10:00', '11:00')])).toEqual({ lane: [0, 1], lanes: 2 }))
  it('empty', () => expect(assignLanes([])).toEqual({ lane: [], lanes: 1 }))
})
