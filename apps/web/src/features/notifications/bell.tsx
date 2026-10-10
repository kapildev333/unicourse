import { Link, useRouterState } from '@tanstack/react-router'
import { Bell as BellIcon } from 'lucide-react'
import { base, sizes, variants } from '../../components/button'
import { useUnreadCount } from './queries'

/** Header link to the notifications page with the unread count; icon-only below md so the phone header stays one row. */
export function Bell({ slug }: { slug: string }) {
  const count = useUnreadCount(slug).data?.count ?? 0
  const current = useRouterState({ select: (s) => /^\/[^/]+\/notifications\/?$/.test(s.location.pathname) })
  return (
    <Link
      to="/$slug/notifications"
      params={{ slug }}
      // ButtonLink has no title prop; same recipe as Button
      className={`${base} ${sizes.md} ${variants['ghost-neutral']} relative max-md:w-10 max-md:px-0`}
      aria-label={count > 0 ? `Notifications, ${count} unread` : 'Notifications'}
      aria-current={current ? 'page' : undefined}
      title="Notifications"
    >
      <BellIcon aria-hidden="true" />
      {count > 0 && (
        <span
          aria-hidden="true"
          className="absolute -top-1 -right-1 min-w-5 rounded-full bg-accent px-1 text-center text-xs font-bold text-on-accent"
        >
          {count > 99 ? '99+' : count}
        </span>
      )}
    </Link>
  )
}
