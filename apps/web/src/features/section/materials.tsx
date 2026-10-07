import { useMutation } from '@tanstack/react-query'
import { ArrowDown, ArrowUp, Download, FolderOpen, Plus, Upload } from 'lucide-react'
import { createElement, useEffect, useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { Button } from '../../components/button'
import { EmptyState } from '../../components/empty-state'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { StatusMessage } from '../../components/status-message'
import { formatDate } from '../../lib/format'
import { RemoveDialog } from '../courses/remove-dialog'
import { authorName } from './access'
import { ACCEPT, fileIcon, formatSize } from './files'
import { MoveDialog, RenameDialog } from './material-dialogs'
import { PlainText } from './plain-text'
import { uploadMaterial, useContentRefresh, useWeeks, weeksKey, type Material, type Section, type Week } from './queries'
import { SectionFrame } from './section-frame'
import { finalLine, runUploads, type Failed } from './upload'
import { WeekDialog, materialsForbidden } from './week-dialog'

export const downloadHref = (slug: string, sectionId: string, id: string) =>
  `/api/v1/universities/${encodeURIComponent(slug)}/sections/${sectionId}/materials/${id}/download`

type UploadState = {
  weekId: string
  line: string
  failed: Failed[]
  busy: boolean
}

export function MaterialsPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {(section, access) => <Materials slug={slug} section={section} canEdit={access.canEdit} />}
    </SectionFrame>
  )
}

function Materials({ slug, section, canEdit }: { slug: string; section: Section; canEdit: boolean }) {
  const q = useWeeks(slug, section.id)
  const refresh = useContentRefresh(slug)
  const weeks = q.data?.items ?? []
  const [weekDialog, setWeekDialog] = useState<'new' | Week | null>(null)
  const [renaming, setRenaming] = useState<Material | null>(null)
  const [moving, setMoving] = useState<Material | null>(null)
  const [delFile, setDelFile] = useState<Material | null>(null)
  const [delWeek, setDelWeek] = useState<Week | null>(null)
  const [notice, setNotice] = useState('')
  const [pageError, setPageError] = useState('')
  const [upload, setUpload] = useState<UploadState | null>(null)
  const [movingWeek, setMovingWeek] = useState('')
  const headingRef = useRef<HTMLHeadingElement>(null)
  const focusAfter = useRef('')
  const busy = !!upload?.busy

  // a keyed reorder can drop focus: put it back on the move button once the new order renders
  useEffect(() => {
    if (!focusAfter.current) return
    const [dir, id] = focusAfter.current.split(' ')
    const el = [`move-${dir}-${id}`, `move-${dir === 'up' ? 'down' : 'up'}-${id}`]
      .map((i) => document.getElementById(i) as HTMLButtonElement | null)
      .find((b) => b && !b.disabled)
    focusAfter.current = ''
    el?.focus()
  })

  useEffect(() => {
    if (!busy) return
    const stop = (e: BeforeUnloadEvent) => e.preventDefault()
    window.addEventListener('beforeunload', stop)
    return () => window.removeEventListener('beforeunload', stop)
  }, [busy])

  const path = { slug, id: section.id }
  const moveWeek = useMutation({
    mutationFn: ({ week, dir }: { week: Week; dir: 'up' | 'down' }) =>
      unwrap(
        api.PATCH('/universities/{slug}/sections/{id}/weeks/{weekId}', {
          params: { path: { ...path, weekId: week.id } },
          body: { position: week.position + (dir === 'up' ? -1 : 1) },
        }),
      ),
    onMutate: ({ week }) => {
      setMovingWeek(week.id)
      setPageError('')
    },
    onSuccess: async (_r, { week, dir }) => {
      await refresh(weeksKey(slug, section.id))
      focusAfter.current = `${dir} ${week.id}`
      setMovingWeek('') // the buttons must be enabled again before the effect looks for one
      setNotice(`${week.title} moved ${dir}.`)
    },
    onError: (e) => {
      setMovingWeek('')
      setPageError(errorMessage(e, { forbidden: materialsForbidden, not_found: 'This week was deleted. Reload the page.' }))
    },
  })

  const removeWeek = useMutation({
    mutationFn: (w: Week) =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/weeks/{weekId}', {
          params: { path: { ...path, weekId: w.id } },
        }),
      ),
    onSuccess: async (_r, w) => {
      await refresh(weeksKey(slug, section.id))
      setDelWeek(null)
      setNotice(`${w.title} deleted.`)
      headingRef.current?.focus() // the trigger no longer exists
    },
  })
  const removeFile = useMutation({
    mutationFn: (m: Material) =>
      unwrap(
        api.DELETE('/universities/{slug}/sections/{id}/materials/{materialId}', {
          params: { path: { ...path, materialId: m.id } },
        }),
      ),
    onSuccess: async (_r, m) => {
      await refresh(weeksKey(slug, section.id))
      setDelFile(null)
      setNotice(`${m.title} deleted.`)
      headingRef.current?.focus()
    },
  })

  async function onUpload(week: Week, files: File[]) {
    setNotice('')
    setUpload({ weekId: week.id, line: '', failed: [], busy: true })
    const r = await runUploads(
      files,
      (f) => uploadMaterial(slug, section.id, week.id, f),
      (line) => setUpload((u) => u && { ...u, line }),
    )
    await refresh(weeksKey(slug, section.id))
    setUpload({
      weekId: week.id,
      line: finalLine(r, week.title),
      failed: r.failed,
      busy: false,
    })
  }

  const addButton = (
    <Button onClick={() => setWeekDialog('new')}>
      <Plus aria-hidden="true" />
      Add week
    </Button>
  )

  let body: React.ReactNode
  if (q.isPending) body = <Loading label="Loading materials…" />
  else if (q.error) {
    body =
      q.error.status === 403 ? (
        <ErrorBanner title="No access">Only this section's staff and enrolled students can open its materials.</ErrorBanner>
      ) : (
        <ErrorBanner title="Couldn't load materials" onRetry={() => void q.refetch()}>
          {errorMessage(q.error)}
        </ErrorBanner>
      )
  } else if (weeks.length === 0) {
    body = canEdit ? (
      <EmptyState
        icon={FolderOpen}
        title="No weeks yet"
        body="Add a week, then upload slides, readings and other files to it. Everyone enrolled can download them."
        action={addButton}
      />
    ) : (
      <EmptyState icon={FolderOpen} title="No materials yet" body="Staff haven't added any weeks or files to this section yet." />
    )
  } else {
    body = (
      <ol aria-labelledby="materials-h" className="flex flex-col gap-6">
        {weeks.map((w, i) => (
          <WeekCard
            key={w.id}
            slug={slug}
            sectionId={section.id}
            week={w}
            weeks={weeks}
            first={i === 0}
            last={i === weeks.length - 1}
            canEdit={canEdit}
            upload={upload}
            moveBusy={movingWeek === w.id}
            onMove={(dir) => moveWeek.mutate({ week: w, dir })}
            onEdit={() => setWeekDialog(w)}
            onDelete={() => {
              removeWeek.reset()
              setDelWeek(w)
            }}
            onUpload={(files) => void onUpload(w, files)}
            onRename={setRenaming}
            onMoveFile={setMoving}
            onDeleteFile={(m) => {
              removeFile.reset()
              setDelFile(m)
            }}
          />
        ))}
      </ol>
    )
  }

  return (
    <>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-4">
        <h2 id="materials-h" ref={headingRef} tabIndex={-1} className="text-xl font-bold focus:outline-none">
          Materials
        </h2>
        {canEdit && weeks.length > 0 && addButton}
      </div>
      {canEdit && (
        <p className="mb-4 text-sm text-ink-muted">
          Files up to 100 MB: PDF, Word, PowerPoint, Excel, images (PNG, JPG, GIF, WebP), ZIP, TXT, Markdown and CSV.
        </p>
      )}
      <StatusMessage message={notice} />
      {pageError && (
        <div className="mb-4">
          <ErrorBanner>{pageError}</ErrorBanner>
        </div>
      )}
      {body}
      {weekDialog && (
        <WeekDialog
          slug={slug}
          sectionId={section.id}
          week={weekDialog === 'new' ? undefined : weekDialog}
          defaultTitle={`Week ${weeks.length + 1}`}
          onClose={() => setWeekDialog(null)}
          onSaved={(m) => {
            setWeekDialog(null)
            setPageError('')
            setNotice(m)
          }}
        />
      )}
      {renaming && (
        <RenameDialog
          slug={slug}
          sectionId={section.id}
          material={renaming}
          onClose={() => setRenaming(null)}
          onSaved={(m) => {
            setRenaming(null)
            setNotice(m)
          }}
        />
      )}
      {moving && (
        <MoveDialog
          slug={slug}
          sectionId={section.id}
          material={moving}
          weeks={weeks}
          onClose={() => setMoving(null)}
          onSaved={(m) => {
            setMoving(null)
            setNotice(m)
          }}
        />
      )}
      {delFile && (
        <RemoveDialog
          open
          title={`Delete ${delFile.title}?`}
          description="Students will no longer see or download it. This can't be undone."
          confirmLabel="Delete file"
          pendingLabel="Deleting…"
          pending={removeFile.isPending}
          error={removeFile.error}
          copy={{
            forbidden: materialsForbidden,
            not_found: 'This file was deleted. Reload the page.',
          }}
          onConfirm={() => removeFile.mutate(delFile)}
          onClose={() => setDelFile(null)}
        />
      )}
      {delWeek && (
        <RemoveDialog
          open
          title={`Delete ${delWeek.title}?`}
          description={
            delWeek.materials.length === 0
              ? 'The week is empty. It will be removed for everyone in the section.'
              : `Its ${delWeek.materials.length} ${delWeek.materials.length === 1 ? 'file' : 'files'} will be deleted too. This can't be undone.`
          }
          confirmLabel="Delete week"
          pendingLabel="Deleting…"
          pending={removeWeek.isPending}
          error={removeWeek.error}
          copy={{
            forbidden: materialsForbidden,
            not_found: 'This week was deleted. Reload the page.',
          }}
          onConfirm={() => removeWeek.mutate(delWeek)}
          onClose={() => setDelWeek(null)}
        />
      )}
    </>
  )
}

function WeekCard(props: {
  slug: string
  sectionId: string
  week: Week
  weeks: Week[]
  first: boolean
  last: boolean
  canEdit: boolean
  upload: UploadState | null
  moveBusy: boolean
  onMove: (dir: 'up' | 'down') => void
  onEdit: () => void
  onDelete: () => void
  onUpload: (files: File[]) => void
  onRename: (m: Material) => void
  onMoveFile: (m: Material) => void
  onDeleteFile: (m: Material) => void
}) {
  const { slug, sectionId, week: w, weeks, first, last, canEdit, upload, moveBusy } = props
  const inputRef = useRef<HTMLInputElement>(null)
  const mine = upload?.weekId === w.id ? upload : null
  const busy = !!upload?.busy
  const line = mine?.line ?? ''
  const failed = mine?.failed ?? []
  return (
    <li className="rounded border border-line bg-surface">
      <div className="flex flex-wrap items-start justify-between gap-x-4 gap-y-2 px-4 py-3 sm:px-6">
        <h3 id={`week-${w.id}`} className="min-w-0 pt-1 text-base font-bold wrap-break-word">
          {w.title}
        </h3>
        {canEdit && (
          <div className="flex flex-wrap items-center gap-1">
            <Button
              id={`move-up-${w.id}`}
              variant="ghost"
              size="sm"
              aria-label={`Move ${w.title} up`}
              title="Move up"
              disabled={first || moveBusy}
              onClick={() => props.onMove('up')}
            >
              <ArrowUp aria-hidden="true" />
            </Button>
            <Button
              id={`move-down-${w.id}`}
              variant="ghost"
              size="sm"
              aria-label={`Move ${w.title} down`}
              title="Move down"
              disabled={last || moveBusy}
              onClick={() => props.onMove('down')}
            >
              <ArrowDown aria-hidden="true" />
            </Button>
            <Button variant="ghost" size="sm" aria-label={`Edit ${w.title}`} onClick={props.onEdit}>
              Edit
            </Button>
            <Button variant="ghost-danger" size="sm" aria-label={`Delete ${w.title}`} onClick={props.onDelete}>
              Delete
            </Button>
            <Button
              variant="secondary"
              size="sm"
              aria-label={`Upload files to ${w.title}`}
              disabled={busy}
              title={busy ? 'Wait for the current upload to finish' : undefined}
              onClick={() => inputRef.current?.click()}
            >
              <Upload aria-hidden="true" />
              {mine?.busy ? 'Uploading…' : 'Upload files'}
            </Button>
            <input
              ref={inputRef}
              type="file"
              multiple
              accept={ACCEPT}
              hidden
              aria-label={`Files for ${w.title}`}
              onChange={(e) => {
                const files = Array.from(e.target.files ?? [])
                e.target.value = ''
                if (files.length) props.onUpload(files)
              }}
            />
          </div>
        )}
      </div>
      {w.notes && <PlainText text={w.notes} className="px-4 pb-3 text-sm text-ink sm:px-6" />}
      {canEdit && (
        <p role="status" className={`px-4 text-sm text-ink sm:px-6 ${line ? 'pb-3' : ''}`}>
          {line}
        </p>
      )}
      {canEdit && failed.length > 0 && (
        <div className="px-4 pb-3 sm:px-6">
          <ErrorBanner title={`${failed.length} ${failed.length === 1 ? 'file' : 'files'} not uploaded`}>
            <ul className="list-disc pl-5">
              {failed.map((f, i) => (
                <li key={i}>
                  <span className="font-bold">{f.name}</span>: {f.reason}
                </li>
              ))}
            </ul>
          </ErrorBanner>
        </div>
      )}
      {w.materials.length > 0 ? (
        <ul aria-labelledby={`week-${w.id}`} className="divide-y divide-line border-t border-line">
          {w.materials.map((m) => (
            <MaterialRow
              key={m.id}
              slug={slug}
              sectionId={sectionId}
              m={m}
              canEdit={canEdit}
              canMove={weeks.length > 1}
              onRename={props.onRename}
              onMoveFile={props.onMoveFile}
              onDeleteFile={props.onDeleteFile}
            />
          ))}
        </ul>
      ) : (
        <p className="border-t border-line px-4 py-3 text-sm text-ink-muted sm:px-6">No files in this week yet.</p>
      )}
    </li>
  )
}

function MaterialRow(props: {
  slug: string
  sectionId: string
  m: Material
  canEdit: boolean
  canMove: boolean
  onRename: (m: Material) => void
  onMoveFile: (m: Material) => void
  onDeleteFile: (m: Material) => void
}) {
  const { slug, sectionId, m, canEdit, canMove } = props
  return (
    <li className="flex flex-wrap items-start gap-x-3 gap-y-2 px-4 py-3 sm:flex-nowrap sm:px-6">
      {createElement(fileIcon(m.contentType), { 'aria-hidden': true, className: 'mt-0.5 shrink-0 text-ink-muted' })}
      <div className="min-w-0 flex-1">
        <p className="text-sm font-bold wrap-anywhere">{m.title}</p>
        <p className="text-sm text-ink-muted wrap-anywhere">
          {[m.title !== m.fileName && m.fileName, formatSize(m.sizeBytes), authorName(m.uploadedBy), formatDate(m.createdAt)]
            .filter(Boolean)
            .join(' · ')}
        </p>
      </div>
      <div className="flex w-full flex-wrap items-center gap-1 sm:w-auto sm:shrink-0">
        <a
          href={downloadHref(slug, sectionId, m.id)}
          download
          aria-label={`Download ${m.title}`}
          className="mr-2 inline-flex items-center gap-1 text-sm text-accent underline hover:text-accent-hover focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
        >
          <Download aria-hidden="true" />
          Download
        </a>
        {canEdit && (
          <>
            <Button variant="ghost" size="sm" aria-label={`Rename ${m.title}`} onClick={() => props.onRename(m)}>
              Rename
            </Button>
            {canMove && (
              <Button
                variant="ghost"
                size="sm"
                aria-label={`Move ${m.title} to another week`}
                onClick={() => props.onMoveFile(m)}
              >
                Move
              </Button>
            )}
            <Button variant="ghost-danger" size="sm" aria-label={`Delete ${m.title}`} onClick={() => props.onDeleteFile(m)}>
              Delete
            </Button>
          </>
        )}
      </div>
    </li>
  )
}
