import { useMutation } from '@tanstack/react-query'
import { Megaphone, Plus } from 'lucide-react'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { LoadMore } from '../../components/load-more'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { formatDateTime } from '../../lib/format'
import { RemoveDialog } from '../courses/remove-dialog'
import { authorName } from './access'
import { AnnouncementDialog } from './announcement-dialog'
import { PlainText } from './plain-text'
import { announcementsKey, useAnnouncements, useContentRefresh, type Announcement, type Section } from './queries'
import { SectionFrame } from './section-frame'

export function AnnouncementsPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {(section, access) => <Announcements slug={slug} section={section} canEdit={access.canEdit} />}
    </SectionFrame>
  )
}

function Announcements({ slug, section, canEdit }: { slug: string; section: Section; canEdit: boolean }) {
  const list = useAnnouncements(slug, section.id)
  const refresh = useContentRefresh(slug)
  const [dialog, setDialog] = useState<'new' | Announcement | null>(null)
  const [deleting, setDeleting] = useState<Announcement | null>(null)
  const [notice, setNotice] = useState('')
  const headingRef = useRef<HTMLHeadingElement>(null)
  const { items } = list

  const del = useMutation({
    mutationFn: (a: Announcement) =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/announcements/{announcementId}', {
          params: { path: { slug, id: section.id, announcementId: a.id } },
        }),
      ),
    onSuccess: async () => {
      await refresh(announcementsKey(slug, section.id))
      setDeleting(null)
      setNotice('Announcement deleted.')
      headingRef.current?.focus() // the trigger no longer exists
    },
  })

  const newButton = (
    <Button onClick={() => setDialog('new')}>
      <Plus aria-hidden="true" />
      New announcement
    </Button>
  )

  let body: React.ReactNode
  if (list.isPending) body = <Loading label="Loading announcements…" />
  else if (list.error && items.length === 0) {
    const forbidden = (list.error as { status?: number }).status === 403
    body = forbidden ? (
      <ErrorBanner title="No access">Only this section's staff and enrolled students can read its announcements.</ErrorBanner>
    ) : (
      <ErrorBanner title="Couldn't load announcements" onRetry={list.refetch}>
        {errorMessage(list.error)}
      </ErrorBanner>
    )
  } else if (items.length === 0) {
    body = canEdit ? (
      <EmptyState
        icon={Megaphone}
        title="No announcements yet"
        body="Post news, changes and reminders for this section. Everyone enrolled can read them, and they get a copy by email unless you turn it off."
        action={newButton}
      />
    ) : (
      <EmptyState icon={Megaphone} title="No announcements yet" body="Staff haven't posted anything in this section yet." />
    )
  } else {
    body = (
      <>
        {list.error && (
          <div className="mb-4">
            <ErrorBanner title="Couldn't load announcements" onRetry={list.fetchNextPage}>
              {errorMessage(list.error)}
            </ErrorBanner>
          </div>
        )}
        <ul aria-labelledby="announcements-h" className="flex flex-col gap-4">
          {items.map((a) => (
            <li key={a.id}>
              <article aria-labelledby={`ann-${a.id}`} className="rounded border border-line bg-surface p-4 sm:p-6">
                <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2">
                  <h3 id={`ann-${a.id}`} className="min-w-0 text-base font-bold wrap-break-word">
                    {a.title}
                  </h3>
                  {canEdit && (
                    <div className="flex shrink-0 gap-1">
                      <Button variant="ghost" size="sm" aria-label={`Edit ${a.title}`} onClick={() => setDialog(a)}>
                        Edit
                      </Button>
                      <Button
                        variant="ghost-danger"
                        size="sm"
                        aria-label={`Delete ${a.title}`}
                        onClick={() => {
                          del.reset()
                          setDeleting(a)
                        }}
                      >
                        Delete
                      </Button>
                    </div>
                  )}
                </div>
                <p className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-ink-muted">
                  <span>
                    Posted by {authorName(a.author)} · <time dateTime={a.createdAt}>{formatDateTime(a.createdAt)}</time>
                    {a.updatedAt !== a.createdAt && ' · Edited'}
                  </span>
                  {a.emailed && <Badge tone="neutral">Emailed</Badge>}
                </p>
                <PlainText text={a.body} className="mt-3 max-w-prose text-base text-ink" />
              </article>
            </li>
          ))}
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
  }

  return (
    <>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-4">
        <h2 id="announcements-h" ref={headingRef} tabIndex={-1} className="text-xl font-bold focus:outline-none">
          Announcements
        </h2>
        {canEdit && items.length > 0 && newButton}
      </div>
      <StatusMessage message={notice} />
      {body}
      {dialog && (
        <AnnouncementDialog
          slug={slug}
          sectionId={section.id}
          announcement={dialog === 'new' ? undefined : dialog}
          onClose={() => setDialog(null)}
          onSaved={(m) => {
            setDialog(null)
            setNotice(m)
          }}
        />
      )}
      {deleting && (
        <RemoveDialog
          open
          title="Delete announcement?"
          description={`"${deleting.title}" will be removed for everyone in the section. Emails already sent can't be recalled.`}
          confirmLabel="Delete announcement"
          pendingLabel="Deleting…"
          pending={del.isPending}
          error={del.error}
          copy={{ forbidden: "Only this section's staff and administrators can delete announcements." }}
          onConfirm={() => del.mutate(deleting)}
          onClose={() => setDeleting(null)}
        />
      )}
    </>
  )
}
