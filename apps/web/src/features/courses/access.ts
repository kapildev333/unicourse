import type { MyMembership } from '../auth/use-me'

/** Admin or dept_admin: sees the Courses area (display only; the server enforces). */
export function canManageCourses(m: MyMembership): boolean {
  return m.role === 'admin' || m.role === 'dept_admin'
}

export function canManageDepartment(m: MyMembership, departmentId: string): boolean {
  return m.role === 'admin' || (m.role === 'dept_admin' && m.departmentIds.includes(departmentId))
}
