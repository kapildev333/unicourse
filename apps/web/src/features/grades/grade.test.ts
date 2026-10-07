import { expect, it } from 'vitest'
import {
  finalPercent, fmt2, gradebookAccess, letterFor, parsePoints, round2, totalsHundred, validateScale, weightTotal,
} from './grade'
import { canadianPreset, usPreset } from './presets'

const section = {
  id: 's1', courseId: 'c1', termId: 't1', number: '001',
  course: { id: 'c1', departmentId: 'd1', code: 'CS101', title: 'Intro' },
  term: { id: 't1', name: 'Fall', startsOn: '2026-08-25', endsOn: '2026-12-15' },
  staff: [
    { userId: 'i', role: 'instructor', email: 'i@a.edu' },
    { userId: 't', role: 'ta', email: 't@a.edu' },
  ],
  pendingStaff: [], meetings: [],
} as never
const mem = (role: string, departmentIds: string[] = []) =>
  ({ university: { id: 'a', slug: 'a', name: 'A' }, role, departmentIds }) as never

it('gradebookAccess by role', () => {
  const all = { view: true, scores: true, assessments: true, finalize: true }
  const none = { view: false, scores: false, assessments: false, finalize: false }
  expect(gradebookAccess(mem('member'), 'i', section)).toEqual(all)
  expect(gradebookAccess(mem('member'), 't', section)).toEqual({ ...none, view: true, scores: true })
  expect(gradebookAccess(mem('admin'), 'x', section)).toEqual({ ...none, view: true, finalize: true })
  expect(gradebookAccess(mem('dept_admin', ['d1']), 'x', section)).toEqual({ ...none, view: true, finalize: true })
  expect(gradebookAccess(mem('dept_admin', ['d2']), 'x', section)).toEqual(none)
  expect(gradebookAccess(mem('member'), 'x', section)).toEqual(none)
})

it('rounds half up', () => {
  expect(fmt2(92.995)).toBe('93.00')
  expect(fmt2(3.335)).toBe('3.34')
  expect(fmt2(2.5)).toBe('2.50')
  expect(round2(1.005)).toBe(1.01)
})

it('letterFor uses the unrounded percent', () => {
  const l = (p: number, rows = usPreset) => letterFor(rows, p)?.letter
  expect(l(92.99)).toBe('A-')
  expect(l(93)).toBe('A')
  expect(l(92.995)).toBe('A-')
  expect(l(0)).toBe('F')
  expect(l(100)).toBe('A')
  expect(letterFor([{ letter: 'A', minPercent: 90, points: 4 }], 10)).toBeUndefined()
})

it('parsePoints', () => {
  expect(parsePoints('', 40)).toEqual({ points: null })
  expect(parsePoints('34', 40)).toEqual({ points: 34 })
  expect(parsePoints('34.5', 40)).toEqual({ points: 34.5 })
  expect(parsePoints('80', 40)).toEqual({ points: 80 })
  for (const t of ['34.555', '-1', 'abc', '3,5'])
    expect(parsePoints(t, 40)).toEqual({ error: 'Enter a number like 42 or 42.5.' })
  expect(parsePoints('80.01', 40)).toEqual({ error: 'Enter at most 80 (twice the maximum).' })
})

const a = (id: string, weight: number, maxPoints = 40) => ({ id, weight, maxPoints }) as never

it('finalPercent', () => {
  expect(finalPercent([a('m', 40), a('f', 60)], { m: 30 })).toEqual({ percent: 30, missing: 1 })
  expect(finalPercent([a('m', 40), a('f', 60)], { m: 30, f: 20 })).toEqual({ percent: 60, missing: 0 })
  expect(finalPercent([a('m', 100), a('z', 0)], { m: 20 }).percent).toBe(50)
  expect(finalPercent([], {}).percent).toBe(0)
})

it('weights', () => {
  expect(totalsHundred([{ weight: 40 }, { weight: 60 }])).toBe(true)
  expect(totalsHundred([{ weight: 33.33 }, { weight: 33.33 }, { weight: 33.33 }])).toBe(false)
  expect(weightTotal([{ weight: 33.33 }, { weight: 33.33 }, { weight: 33.33 }])).toBe(99.99)
  expect(totalsHundred([{ weight: 40 }, { weight: 60.004 }])).toBe(true)
})

const row = (letter: string, minPercent: string, points = '1') => ({ letter, minPercent, points })

it('validateScale', () => {
  let e = validateScale([row('A', '90'), row('A', '0')])
  expect(e.rows[1]?.letter).toBe('Each letter can appear only once.')
  expect(e.rows[0]).toBeUndefined()
  e = validateScale([row('A', '90'), row('B', '90')])
  expect(e.rows[1]?.minPercent).toBe('Each minimum percent can appear only once.')
  expect(e.table).toMatch(/One row must start at 0/)
  expect(validateScale([]).table).toBe('Add at least one row.')
  expect(validateScale([row('A', '0', '10')]).rows[0]?.points).toBe('Enter grade points from 0 to 9.99.')
  expect(validateScale([row(' ', '101')]).rows[0]).toEqual({
    letter: 'Enter a letter.', minPercent: 'Enter a percent from 0 to 100.',
  })
  for (const [p, n] of [[usPreset, 12], [canadianPreset, 13]] as const) {
    expect(p).toHaveLength(n)
    expect(p.filter((r) => r.minPercent === 0)).toHaveLength(1)
    const ok = validateScale(p.map((r) => ({ letter: r.letter, minPercent: String(r.minPercent), points: String(r.points) })))
    expect(ok).toEqual({ rows: {} })
  }
})
