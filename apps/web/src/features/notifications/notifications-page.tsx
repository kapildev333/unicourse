import { Link } from '@tanstack/react-router'
import { Bell, CheckCheck, ClipboardList, FileText, GraduationCap, Megaphone, type LucideIcon } from 'lucide-react'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { LoadMore } from '../../components/load-more'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { StatusMessage } from '../../components/status-message'
import { formatDateTime } from '../../lib/format'
import { useMarkAllRead, useMarkRead, useNotifications, useUnreadCount, type Notification } from './queries'

const icons: Record<Notification['kind'], LucideIcon> = {
  announcement: Megaphone,
  assessment_created: ClipboardList,
  scores_published: FileText,
  grades_finalized: GraduationCap,
}

function copy(n: Notification): string {
  const what = n.assessment?.kind === 'assignment' ? 'assignment' : 'quiz'
  switch (n.kind) {
    case 'announcement':
      return `New announcement: ${n.subject}`
    case 'assessment_created':
      return `New ${what}: ${n.subject}`
    case 'scores_published':
      return `Grades released: ${n.subject}`
    case 'grades_finalized':
      return `Final grades posted for ${n.course.title}`
  }
}

/** The link to the notification's source; the page it opens is the one that explains the notification. */
function Target(props: { slug: string; n: Notification; onOpen: () => void }) {
  const { slug, n, onOpen } = props
  const className = `${n.readAt ? 'font-medium' : 'font-bold'} text-accent underline-offset-4 hover:text-accent-hover hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus`
  const text = copy(n)
  const common = { className, onClick: onOpen, children: text }
  const id = n.section.id
  if (n.kind === 'announcement') return <Link to="/$slug/sections/$id/announcements" params={{ slug, id }} {...common} />
  if (n.kind === 'grades_finalized' || n.assessment?.kind === 'manual' || !n.assessment)
    return <Link to="/$slug/grades" params={{ slug }} {...common} />
  const params = { slug, id, assessmentId: n.assessment.id }
  return n.assessment.kind === 'assignment' ? (
    <Link to="/$slug/sections/$id/assignments/$assessmentId" params={params} {...common} />
  ) : (
    <Link to="/$slug/sections/$id/quizzes/$assessmentId" params={params} {...common} />
  )
}

export function NotificationsPage({ slug }: { slug: string }) {
  const list = useNotifications(slug)
  const count = useUnreadCount(slug).data?.count ?? 0
  const markRead = useMarkRead(slug)
  const markAll = useMarkAllRead(slug)
  const { items } = list

  let body: React.ReactNode
  if (list.isPending) body = <Loading variant="table" columns={['Notification', 'Time']} label="Loading notifications…" />
  else if (list.error && items.length === 0)
    body = (
      <ErrorBanner title="Couldn't load your notifications" onRetry={list.refetch}>
        {errorMessage(list.error)}
      </ErrorBanner>
    )
  else if (items.length === 0)
    body = (
      <EmptyState
        icon={Bell}
        title="No notifications yet"
        body="New announcements, quizzes, assignments and released grades in your sections appear here."
      />
    )
  else
    body = (
      <>
        {list.error && (
          <div className="mb-4">
            <ErrorBanner title="Couldn't load your notifications" onRetry={list.hasNextPage ? list.fetchNextPage : list.refetch}>
              {errorMessage(list.error)}
            </ErrorBanner>
          </div>
        )}
        <ul aria-label="Notifications" className="border-t border-line">
          {items.map((n) => {
            const Icon = icons[n.kind]
            return (
              <li key={n.id} className="flex flex-wrap items-start gap-x-3 gap-y-2 border-b border-line py-4">
                <Icon aria-hidden="true" className="mt-1 shrink-0 text-ink-muted" />
                <div className="min-w-0 flex-1">
                  <Target slug={slug} n={n} onOpen={() => !n.readAt && markRead.mutate(n.id)} />
                  <p className="mt-1 text-sm text-ink-muted">
                    {n.course.code}-{n.section.number} · <time dateTime={n.createdAt}>{formatDateTime(n.createdAt)}</time>
                  </p>
                </div>
                {!n.readAt && (
                  <div className="flex shrink-0 items-center gap-2">
                    <Badge tone="neutral">New</Badge>
                    <Button
                      variant="ghost-neutral"
                      size="sm"
                      aria-label={`Mark as read: ${copy(n)}`}
                      onClick={() => markRead.mutate(n.id)}
                    >
                      Mark as read
                    </Button>
                  </div>
                )}
              </li>
            )
          })}
        </ul>
        <LoadMore
          hasNextPage={list.hasNextPage}
          isFetchingNextPage={list.isFetchingNextPage}
          onClick={list.fetchNextPage}
          shown={items.length}
          showCount={false}
        />
      </>
    )

  return (
    <>
      <PageHeader
        title="Notifications"
        description="Announcements, new work and grades from your sections."
        action={
          <Button variant="secondary" disabled={count === 0 || markAll.isPending} onClick={() => markAll.mutate()}>
            <CheckCheck aria-hidden="true" />
            Mark all as read
          </Button>
        }
      />
      <StatusMessage message={markAll.isSuccess ? 'All notifications marked as read.' : ''} />
      {(markAll.error || markRead.error) && (
        <div className="mb-4">
          <ErrorBanner title="Couldn't mark notifications as read">{errorMessage(markAll.error ?? markRead.error)}</ErrorBanner>
        </div>
      )}
      {body}
    </>
  )
}
