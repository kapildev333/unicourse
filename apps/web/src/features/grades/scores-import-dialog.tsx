import { useMutation } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { ApiError } from '../../api/errors'
import type { components } from '../../api/types'
import { Badge } from '../../components/badge'
import { Button } from '../../components/button'
import { Dialog } from '../../components/dialog'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { FormField, inputClass } from '../../components/form-field'
import { Table } from '../../components/table'
import { sentence } from './grade'
import { useGradesRefresh, type Assessment } from './queries'

type Preview = components['schemas']['ScoreImportPreview']
type Row = Preview['rows'][number]

const CEILING = 200
const COLUMNS = ['email', 'points']
const badges = {
  add: { tone: 'accent', label: 'Add' },
  update: { tone: 'warning', label: 'Update' },
  exists: { tone: 'neutral', label: 'Exists' },
  error: { tone: 'danger', label: 'Error' },
} as const

// fetch sets the multipart boundary itself, so no Content-Type here.
function form(file: File, checksum?: string) {
  const f = new FormData()
  f.append('file', file)
  if (checksum !== undefined) f.append('checksum', checksum)
  return { body: f as never, bodySerializer: (b: unknown) => b as FormData }
}

/** Mount only while open (parent renders it conditionally) so every open starts clean. */
export function ScoresImportDialog(props: {
  slug: string
  sectionId: string
  assessments: Assessment[]
  onClose: () => void
  onDone: (message: string) => void
}) {
  const { slug, sectionId, assessments, onClose, onDone } = props
  const refresh = useGradesRefresh(slug)
  const [assessmentId, setAssessmentId] = useState(assessments[0]?.id ?? '')
  const [file, setFile] = useState<File>()
  const bannerRef = useRef<HTMLDivElement>(null)
  const params = { params: { path: { slug, id: sectionId, assessmentId } } }
  const previewMut = useMutation({
    mutationFn: (f: File) =>
      unwrap(
        api.POST('/universities/{slug}/sections/{id}/assessments/{assessmentId}/scores/import/preview', {
          ...params,
          ...form(f),
        }),
      ),
  })
  const commitMut = useMutation({
    mutationFn: (v: { f: File; checksum: string }) =>
      unwrap(
        api.POST('/universities/{slug}/sections/{id}/assessments/{assessmentId}/scores/import', {
          ...params,
          ...form(v.f, v.checksum),
        }),
      ),
    onSuccess: async (r) => {
      await refresh()
      const name = assessments.find((a) => a.id === assessmentId)?.name ?? ''
      onDone(`Scores imported for ${name}: ${r.counts.add} added, ${r.counts.update} updated.`)
    },
  })

  const preview = previewMut.data
  const commitError = commitMut.error instanceof ApiError ? commitMut.error : undefined
  const failed = useMemo(
    () =>
      new Map(
        (commitError?.code === 'validation_failed' ? commitError.details : [])
          .filter((d) => d.field.startsWith('line:'))
          .map((d) => [Number(d.field.slice(5)), sentence(d.message)] as const),
      ),
    [commitError],
  )
  const stale = commitError?.details.some((d) => d.field === 'checksum') ?? false
  const error = commitMut.error ?? previewMut.error

  useEffect(() => {
    if (failed.size > 0) bannerRef.current?.focus()
  }, [failed])

  function onFile(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    setFile(f)
    commitMut.reset()
    previewMut.reset()
    if (f) previewMut.mutate(f)
  }
  function onAssessment(e: React.ChangeEvent<HTMLSelectElement>) {
    setAssessmentId(e.target.value)
    setFile(undefined)
    commitMut.reset()
    previewMut.reset()
  }

  const counts = preview?.counts
  const errorCount = (counts?.error ?? 0) + failed.size
  const todo = counts ? counts.add + counts.update : 0
  const canImport =
    !!preview && !!file && !stale && errorCount === 0 && todo > 0 && !commitMut.isPending && !commitMut.isSuccess
  let reason = ''
  if (!file || !preview) reason = previewMut.isPending ? 'Checking the file…' : 'Choose a file to preview.'
  else if (stale) reason = 'Choose the file again.'
  else if (errorCount > 0) reason = `Fix the ${errorCount} rows with errors and choose the file again.`
  else if (todo === 0) reason = 'Nothing to import: every score is unchanged.'

  let status = ''
  if (previewMut.isPending && file) status = `Checking ${file.name}…`
  else if (failed.size > 0) status = `Import failed: ${failed.size} rows have errors. Fix the file and choose it again.`
  else if (preview && counts)
    status = `${preview.rows.length} rows checked: ${counts.add} to add, ${counts.update} to update, ${counts.exists} unchanged, ${counts.error} with errors.`

  const rows = useMemo(() => {
    if (!preview) return []
    const bad = (r: Row) => r.action === 'error' || failed.has(r.line)
    const byLine = [...preview.rows].sort((a, b) => a.line - b.line)
    return [...byLine.filter(bad), ...byLine.filter((r) => !bad(r))]
  }, [preview, failed])

  let banner: React.ReactNode = null
  if (failed.size > 0) {
    banner = (
      <ErrorBanner ref={bannerRef} title="Import failed">
        {failed.size} rows have errors; they are marked below and listed first.
      </ErrorBanner>
    )
  } else if (stale) {
    banner = <ErrorBanner title="The file changed since the preview">Choose the file again to preview it.</ErrorBanner>
  } else if (error instanceof ApiError && error.details.some((d) => d.field === 'file')) {
    banner = (
      <ErrorBanner title="This file can't be imported">
        {error.details
          .filter((d) => d.field === 'file')
          .map((d) => (
            <p key={d.message}>{sentence(d.message)}</p>
          ))}
      </ErrorBanner>
    )
  } else if (error) {
    banner = <ErrorBanner>{errorMessage(error)}</ErrorBanner>
  }

  const template = COLUMNS.join(',')
  return (
    <Dialog open onClose={onClose} title="Import scores" wide>
      <div className="flex flex-col gap-4">
        <FormField label="Assessment" name="assessment">
          <select className={inputClass} value={assessmentId} disabled={commitMut.isPending} onChange={onAssessment}>
            {assessments.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
        </FormField>
        <div className="text-sm text-ink-muted">
          <p>
            The first line must be a header with these columns:{' '}
            <code className="rounded bg-canvas px-1.5 py-0.5 text-ink">{template}</code>
          </p>
          <p>Points for students already scored are replaced.</p>
          <p>
            Columns are case-insensitive. Maximum 5 MB.{' '}
            <a
              download="scores-template.csv"
              href={'data:text/csv;charset=utf-8,' + encodeURIComponent(template + '\n')}
              className="text-accent underline"
            >
              Download template
            </a>
          </p>
        </div>
        <FormField label="CSV file" name="file">
          <input
            key={assessmentId}
            type="file"
            accept=".csv,text/csv"
            className="text-sm"
            disabled={commitMut.isPending}
            onClick={(e) => {
              // choosing the same (fixed) file again fires no change event unless the input is cleared
              e.currentTarget.value = ''
            }}
            onChange={onFile}
          />
        </FormField>
        <p role="status" aria-live="polite" className="text-sm">
          {status}
        </p>
        {banner}
        {counts && (
          <dl className="flex gap-4 text-sm">
            {(
              [
                ['Add', counts.add],
                ['Update', counts.update],
                ['Errors', errorCount],
              ] as const
            ).map(([k, n]) => (
              <div key={k} className="flex gap-1">
                <dt className="text-ink-muted">{k}</dt>
                <dd className={k === 'Errors' && n > 0 ? 'font-bold text-danger' : ''}>{n}</dd>
              </div>
            ))}
          </dl>
        )}
        {preview && file && (
          <div>
            <div className="max-h-[55vh] overflow-y-auto">
              <Table caption={`Preview of ${file.name}`}>
                <thead>
                  <tr>
                    <th scope="col">Line</th>
                    <th scope="col">Result</th>
                    <th scope="col">Message</th>
                    {COLUMNS.map((c) => (
                      <th key={c} scope="col">
                        {c}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {rows.slice(0, CEILING).map((r) => {
                    const bad = r.action === 'error' || failed.has(r.line)
                    const b = badges[bad ? 'error' : r.action]
                    return (
                      <tr key={r.line} className={bad ? 'bg-danger-soft' : ''}>
                        <td>{r.line}</td>
                        <td>
                          <Badge tone={b.tone}>{b.label}</Badge>
                        </td>
                        <td className={bad ? 'text-danger' : ''}>{failed.get(r.line) ?? r.message}</td>
                        {COLUMNS.map((c) => (
                          <td key={c}>{r.data[c] ?? ''}</td>
                        ))}
                      </tr>
                    )
                  })}
                </tbody>
              </Table>
            </div>
            {rows.length > CEILING && (
              <p className="mt-2 text-sm text-ink-muted">
                Showing {CEILING} of {rows.length} rows. Rows with errors are listed first.
              </p>
            )}
          </div>
        )}
        <div className="mt-2 flex items-center justify-between gap-4">
          <p id="scores-import-reason" className="text-sm text-ink-muted">
            {canImport ? '' : reason}
          </p>
          <div className="flex gap-2">
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button
              disabled={!canImport}
              aria-describedby={canImport ? undefined : 'scores-import-reason'}
              onClick={() => file && preview && commitMut.mutate({ f: file, checksum: preview.checksum })}
            >
              {commitMut.isPending ? 'Importing…' : preview ? `Import ${todo} scores` : 'Import scores'}
            </Button>
          </div>
        </div>
      </div>
    </Dialog>
  )
}
