import { useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { useCursorList } from '../../lib/use-cursor-list'

type S = components['schemas']
export type Section = S['Section']
export type Announcement = S['Announcement']

export const sectionKey = (slug: string, id: string) => [slug, 'sections', 'detail', id] as const
export const announcementsKey = (slug: string, id: string) => [slug, 'sections', id, 'announcements'] as const

export function useAnnouncements(slug: string, sectionId: string) {
  return useCursorList<Announcement>(announcementsKey(slug, sectionId), (cursor) =>
    unwrap(
      api.GET('/universities/{slug}/sections/{id}/announcements', {
        params: { path: { slug, id: sectionId }, query: { limit: 20, cursor } },
      }),
    ),
  )
}

/** Invalidates one content key and the audit log; await it before showing a status line. */
export function useContentRefresh(slug: string) {
  const qc = useQueryClient()
  return (key: readonly unknown[]) =>
    Promise.all([key, [slug, 'audit-log']].map((queryKey) => qc.invalidateQueries({ queryKey })))
}
