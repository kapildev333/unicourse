import { gradebookAccess } from '../grades/grade'
import { sectionAccess } from '../courses/access'
import { useMe, useUniversity, type MyMembership } from '../auth/use-me'
import { useMySections } from '../my-sections/my-sections'
import type { Section } from './queries'

type Role = 'instructor' | 'ta' | 'student'

export type ContentAccess = {
  canEdit: boolean // manager or instructor/TA of the section
  canRead: boolean // canEdit or enrolled
  canGrades: boolean
  ownRole?: Role
}

/** Display only; the server authorises every request (spec 6 §4). */
export function contentAccess(m: MyMembership, userId: string, section: Section, myRole?: Role): ContentAccess {
  const { manager, staffRole } = sectionAccess(m, userId, section)
  const canEdit = manager || !!staffRole
  return {
    canEdit,
    canRead: canEdit || myRole === 'student',
    canGrades: gradebookAccess(m, userId, section).view,
    ownRole: staffRole ?? myRole,
  }
}

export function useContentAccess(slug: string, section: Section): ContentAccess {
  const { membership } = useUniversity()
  const me = useMe().data
  const myRole = useMySections(slug).data?.terms.flatMap((g) => g.sections).find((s) => s.id === section.id)?.myRole
  return contentAccess(membership, me?.user.id ?? '', section, myRole)
}

export const authorName = (a: { displayName: string; email?: string }) => a.displayName || a.email || 'Unnamed member'
