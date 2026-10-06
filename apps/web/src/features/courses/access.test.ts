import { expect, it } from 'vitest'
import type { components } from '../../api/types'
import type { MyMembership } from '../auth/use-me'
import { personName, sectionAccess } from './access'

type Section = components['schemas']['Section']
const section = {
  course: { departmentId: 'd1' },
  staff: [
    { userId: 'ins', role: 'instructor' },
    { userId: 'ta', role: 'ta' },
  ],
} as Section
const m = (role: string, departmentIds: string[] = []) => ({ role, departmentIds }) as MyMembership
const none = { canManageInstructors: false, canManageTAs: false, canViewRoster: false, canEditRoster: false }

const cases: [string, MyMembership, string, Partial<ReturnType<typeof sectionAccess>>][] = [
  ['admin', m('admin'), 'x', { manager: true, canManageInstructors: true, canManageTAs: true, canViewRoster: true, canEditRoster: true }],
  ['dept_admin of the department', m('dept_admin', ['d1']), 'x', { manager: true, canManageInstructors: true, canManageTAs: true, canViewRoster: true, canEditRoster: true }],
  ['dept_admin of another department', m('dept_admin', ['d2']), 'x', { manager: false, ...none }],
  ['instructor', m('member'), 'ins', { staffRole: 'instructor', canManageTAs: true, canViewRoster: true }],
  ['TA', m('member'), 'ta', { staffRole: 'ta', canViewRoster: true }],
  ['student', m('member'), 'stu', {}],
]

it.each(cases)('%s', (_, membership, userId, want) => {
  expect(sectionAccess(membership, userId, section)).toMatchObject({ manager: false, ...none, ...want })
})

it('personName falls back to the email', () => {
  expect(personName({ displayName: 'Dana', email: 'd@a.edu' })).toBe('Dana')
  expect(personName({ displayName: '', email: 'd@a.edu' })).toBe('d@a.edu')
})
