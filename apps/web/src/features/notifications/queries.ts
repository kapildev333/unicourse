import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from '../../api/client'
import type { components } from '../../api/types'
import { useCursorList } from '../../lib/use-cursor-list'

export type Notification = components['schemas']['Notification']

export const notificationsKey = (slug: string) => [slug, 'notifications'] as const
export const unreadCountKey = (slug: string) => [slug, 'notifications', 'count'] as const
export const settingsKey = (slug: string) => [slug, 'notification-settings'] as const

/** The bell's count: polled once a minute while the tab is visible, refetched on focus (default). */
export function useUnreadCount(slug: string) {
  return useQuery({
    queryKey: unreadCountKey(slug),
    queryFn: () => unwrap(api.GET('/universities/{slug}/me/notifications/unread-count', { params: { path: { slug } } })),
    refetchInterval: 60_000,
    refetchIntervalInBackground: false,
    staleTime: 30_000,
  })
}

// The list key is [slug, 'notifications', 'list'] so invalidating notificationsKey refreshes list and count together.
export function useNotifications(slug: string) {
  return useCursorList<Notification>([...notificationsKey(slug), 'list'], (cursor) =>
    unwrap(api.GET('/universities/{slug}/me/notifications', { params: { path: { slug }, query: { limit: 20, cursor } } })),
  )
}

export function useMarkRead(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) =>
      unwrap(
        api.PATCH('/universities/{slug}/me/notifications/{id}', {
          params: { path: { slug, id } },
          body: { read: true },
        }),
      ),
    onSettled: () => qc.invalidateQueries({ queryKey: notificationsKey(slug) }),
  })
}

export function useMarkAllRead(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () =>
      unwrap(api.PATCH('/universities/{slug}/me/notifications', { params: { path: { slug } }, body: { read: true } })),
    onSuccess: () => qc.invalidateQueries({ queryKey: notificationsKey(slug) }),
  })
}

export function useNotificationSettings(slug: string, enabled: boolean) {
  return useQuery({
    queryKey: settingsKey(slug),
    enabled,
    queryFn: () => unwrap(api.GET('/universities/{slug}/me/notification-settings', { params: { path: { slug } } })),
  })
}

export function useSaveNotificationSettings(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (announcementMail: boolean) =>
      unwrap(
        api.PATCH('/universities/{slug}/me/notification-settings', {
          params: { path: { slug } },
          body: { announcementMail },
        }),
      ),
    onSuccess: (s) => qc.setQueryData(settingsKey(slug), s),
  })
}
