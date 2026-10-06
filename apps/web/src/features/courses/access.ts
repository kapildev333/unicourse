import type { components } from '../../api/types'
import type { MyMembership } from '../auth/use-me'

/** Admin or dept_admin: sees the Courses area (display only; the server enforces). */
export function canManageCourses(m: MyMembership): boolean {
  return m.role === 'admin' || m.role === 'dept_admin'
}

export function canManageDepartment(m: MyMembership, departmentId: string): boolean {
  return m.role === 'admin' || (m.role === 'dept_admin' && m.departmentIds.includes(departmentId))
}

type Section = components['schemas']['Section']

export type SectionAccess = {
  manager: boolean
  staffRole?: 'instructor' | 'ta'
  canManageInstructors: boolean
  canManageTAs: boolean
  canViewRoster: boolean
  canEditRoster: boolean
}

/** Display only; the server authorises every request. Not staff and not a manager means student. */
export function sectionAccess(m: MyMembership, userId: string, section: Section): SectionAccess {
  const manager = canManageDepartment(m, section.course.departmentId)
  const staffRole = section.staff.find((s) => s.userId === userId)?.role
  return {
    manager,
    staffRole,
    canManageInstructors: manager,
    canManageTAs: manager || staffRole === 'instructor',
    canViewRoster: manager || !!staffRole,
    canEditRoster: manager,
  }
}

export function personName(p: { displayName?: string; email: string }): string {
  return p.displayName || p.email
}
