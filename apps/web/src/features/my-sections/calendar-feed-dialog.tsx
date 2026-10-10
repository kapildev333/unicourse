import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import type { ApiError } from '../../api/errors'
import type { components } from '../../api/types'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { inputClass } from '../../components/form-field'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { formatDate } from '../../lib/format'

type Feed = components['schemas']['CalendarFeed']

/** Mount only while open. The URL exists in memory only after create or rotate; the server keeps a hash. */
export function CalendarFeedDialog({ slug, onClose }: { slug: string; onClose: () => void }) {
  const path = { params: { path: { slug } } }
  const key = [slug, 'calendar-feed']
  const client = useQueryClient()
  const [url, setUrl] = useState<string>()
  const [confirming, setConfirming] = useState(false)
  const [copied, setCopied] = useState('')
  const input = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (url) input.current?.focus() // the URL appears once; move focus to it
  }, [url])

  const feed = useQuery<Feed | null, ApiError>({
    queryKey: key,
    queryFn: () =>
      unwrap(api.GET('/universities/{slug}/me/calendar-feed', path)).catch((e: ApiError) => {
        if (e.status === 404) return null
        throw e
      }),
    retry: false,
  })
  const create = useMutation({
    mutationFn: () => unwrap(api.POST('/universities/{slug}/me/calendar-feed', path)),
    onSuccess: (d) => {
      setUrl(d.url)
      setConfirming(false)
      setCopied('')
      revoke.reset()
      client.setQueryData(key, { createdAt: d.createdAt })
    },
  })
  const revoke = useMutation({
    mutationFn: () => unwrap(api.DELETE('/universities/{slug}/me/calendar-feed', path)),
    onSuccess: () => {
      create.reset()
      client.setQueryData(key, null)
    },
  })
  const error = create.error ?? revoke.error ?? feed.error

  async function copy() {
    try {
      await navigator.clipboard.writeText(url ?? '')
      setCopied('Link copied.')
    } catch {
      input.current?.select()
      setCopied('Select the link and copy it.')
    }
  }

  let body: React.ReactNode
  let footer: React.ReactNode
  if (url) {
    body = (
      <div className="flex flex-col gap-3">
        <input
          ref={input}
          readOnly
          aria-label="Calendar link"
          value={url}
          className={inputClass}
          onFocus={(e) => e.currentTarget.select()}
        />
        <div>
          <Button variant="secondary" onClick={() => void copy()}>
            Copy link
          </Button>
        </div>
        {copied === 'Link copied.' ? (
          <StatusMessage message={copied} />
        ) : (
          <p role="status" className="text-sm text-ink-muted">
            {copied}
          </p>
        )}
        <p className="text-sm text-ink-muted">
          Copy it now: it is shown only once. Anyone with this link can see your schedule, so keep it private.
        </p>
      </div>
    )
    footer = <Button onClick={onClose}>Done</Button>
  } else if (feed.isPending) {
    body = <Loading />
  } else if (feed.isError) {
    body = (
      <ErrorBanner title="Couldn't load your calendar feed" onRetry={() => void feed.refetch()}>
        {errorMessage(feed.error)}
      </ErrorBanner>
    )
  } else if (!feed.data) {
    body = (
      <p className="text-sm text-ink-muted">
        Add your section meetings and quiz and assignment deadlines to Google, Apple or Outlook Calendar with a private link.
      </p>
    )
    footer = (
      <>
        <Button variant="ghost-neutral" onClick={onClose}>
          Cancel
        </Button>
        <Button disabled={create.isPending} onClick={() => create.mutate()}>
          Create link
        </Button>
      </>
    )
  } else if (confirming) {
    body = <p className="text-sm text-ink">The old link stops working in every calendar that uses it.</p>
    footer = (
      <>
        <Button variant="ghost-neutral" onClick={() => setConfirming(false)}>
          Cancel
        </Button>
        <Button disabled={create.isPending} onClick={() => create.mutate()}>
          Create new link
        </Button>
      </>
    )
  } else {
    body = (
      <p className="text-sm text-ink-muted">
        Your private link was created on {formatDate(feed.data.createdAt)}. It is shown only when it is created. If you lost
        it or shared it by mistake, create a new one.
      </p>
    )
    footer = (
      <>
        <Button variant="ghost-danger" className="mr-auto" disabled={revoke.isPending} onClick={() => revoke.mutate()}>
          Turn off
        </Button>
        <Button variant="ghost-neutral" onClick={onClose}>
          Cancel
        </Button>
        <Button variant="secondary" onClick={() => setConfirming(true)}>
          Create new link
        </Button>
      </>
    )
  }
  return (
    <Dialog open onClose={onClose} title="Calendar feed">
      {error && !feed.isError && (
        <div className="mb-4">
          <ErrorBanner>{errorMessage(error)}</ErrorBanner>
        </div>
      )}
      {body}
      {footer && <DialogFooter>{footer}</DialogFooter>}
    </Dialog>
  )
}
