import { useQuery, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'
import type { ApiError } from '../../api/errors'
import type { components } from '../../api/types'
import { useCursorList } from '../../lib/use-cursor-list'

type S = components['schemas']
export type Section = S['Section']
export type Announcement = S['Announcement']
export type Week = S['Week']
export type Material = S['Material']
export type UploadPolicy = S['UploadPolicy']

/** The server's upload limit and allowlist; read once per session. */
export function useUploadPolicy() {
  return useQuery({ queryKey: ['upload-policy'], queryFn: () => unwrap(api.GET('/upload-policy')), staleTime: Infinity })
}

export const sectionKey = (slug: string, id: string) => [slug, 'sections', 'detail', id] as const
export const announcementsKey = (slug: string, id: string) => [slug, 'sections', id, 'announcements'] as const
export const weeksKey = (slug: string, id: string) => [slug, 'sections', id, 'weeks'] as const

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

export function useWeeks(slug: string, sectionId: string) {
  return useQuery<S['WeekList'], ApiError>({
    queryKey: weeksKey(slug, sectionId),
    queryFn: () => unwrap(api.GET('/universities/{slug}/sections/{id}/weeks', { params: { path: { slug, id: sectionId } } })),
  })
}
