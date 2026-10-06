import { useQuery, type UseQueryResult } from '@tanstack/react-query'
import { useParams } from '@tanstack/react-router'
import { api, unwrap } from '../../api/client'
import type { ApiError } from '../../api/errors'
import type { components } from '../../api/types'

export type Me = components['schemas']['Me']
export type MyMembership = components['schemas']['MyMembership']

export function useMe(): UseQueryResult<Me, ApiError> {
  return useQuery<Me, ApiError>({
    queryKey: ['me'],
    queryFn: () => unwrap(api.GET('/me')),
    staleTime: 300_000,
  })
}

/** Only valid under UniversityLayout, which guarantees the membership exists. */
export function useUniversity(): { slug: string; membership: MyMembership } {
  const { slug } = useParams({ strict: false }) as { slug?: string }
  const me = useMe().data
  const membership = me?.memberships.find((m) => m.university.slug === slug)
  if (!slug || !membership) throw new Error('useUniversity outside UniversityLayout')
  return { slug, membership }
}
