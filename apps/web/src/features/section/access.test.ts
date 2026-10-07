import { expect, it } from 'vitest'
import { authorName, contentAccess } from './access'
import { sectionFixture } from './test-helpers'

const m = (role: string, departmentIds: string[] = []) =>
  ({ university: { id: 'a', slug: 'a', name: 'A' }, role, departmentIds }) as never
const s = sectionFixture({
  staff: [
    { userId: 'i', role: 'instructor', email: 'i@a.edu', displayName: '' },
    { userId: 't', role: 'ta', email: 't@a.edu', displayName: '' },
  ],
})
const row = (a: ReturnType<typeof contentAccess>) => [a.canEdit, a.canRead, a.canGrades, a.ownRole]

it('contentAccess truth table', () => {
  expect(row(contentAccess(m('admin'), 'x', s))).toEqual([true, true, true, undefined])
  expect(row(contentAccess(m('dept_admin', ['d1']), 'x', s))).toEqual([true, true, true, undefined])
  expect(row(contentAccess(m('dept_admin', ['d2']), 'x', s))).toEqual([false, false, false, undefined])
  expect(row(contentAccess(m('member'), 'i', s))).toEqual([true, true, true, 'instructor'])
  expect(row(contentAccess(m('member'), 't', s))).toEqual([true, true, true, 'ta'])
  expect(row(contentAccess(m('member'), 'x', s, 'student'))).toEqual([false, true, false, 'student'])
  expect(row(contentAccess(m('member'), 'x', s))).toEqual([false, false, false, undefined])
})

it('authorName falls back to email, then a placeholder', () => {
  expect(authorName({ displayName: 'Dana' })).toBe('Dana')
  expect(authorName({ displayName: '' })).toBe('Unnamed member')
  expect(authorName({ displayName: '', email: 'x@a.edu' })).toBe('x@a.edu')
})
