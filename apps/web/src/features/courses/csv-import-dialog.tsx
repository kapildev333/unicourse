import { useMutation } from '@tanstack/react-query'
import { Download } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { ApiError } from '../../api/errors'
import { commitImport, previewImport, type ImportCounts, type ImportPreview, type ImportTarget } from '../../api/upload'
import { ActionBadge } from '../../components/badge'
import { Button } from '../../components/button'
import { Dialog, DialogFooter } from '../../components/dialog'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { FormField, fileInputClass } from '../../components/form-field'
import { Table } from '../../components/table'

const CEILING = 200

const sentence = (s: string) => {
  const t = s.trim()
  if (!t) return t
  const c = t[0].toUpperCase() + t.slice(1)
  return /[.!?]$/.test(c) ? c : c + '.'
}

export function importSummary(_kind: ImportTarget['kind'], c: ImportCounts): string {
  const parts = [
    c.add > 0 && `${c.add} added`,
    c.invite > 0 && `${c.invite} invited`,
    c.exists > 0 && `${c.exists} already existed`,
  ].filter(Boolean)
  return `Import finished: ${parts.length ? parts.join(', ') : 'nothing changed'}.`
}

type Props = {
  open: boolean
  onClose(): void
  title: string
  target: ImportTarget
  columns: string[]
  hint?: string
  onDone(result: ImportPreview): void
}

export function CsvImportDialog(props: Props) {
  const { open, onClose, title } = props
  // the body mounts only while open, so every open starts from a clean state
  return (
    <Dialog open={open} onClose={onClose} title={title} wide>
      {open && <Body {...props} />}
    </Dialog>
  )
}

function Body({ onClose, target, columns, hint, onDone }: Props) {
  const [file, setFile] = useState<File>()
  const bannerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const [chooseAgain, setChooseAgain] = useState(false)
  const previewMut = useMutation({ mutationFn: (f: File) => previewImport(target, f) })
  const commitMut = useMutation({
    mutationFn: (v: { f: File; checksum: string }) => commitImport(target, v.f, v.checksum),
    onSuccess: (r) => onDone(r),
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

  function previewAgain() {
    // re-read the input: the file on disk may have changed, and a cleared input has no file to reuse
    const f = inputRef.current?.files?.[0]
    setChooseAgain(!f)
    if (!f) return
    setFile(f)
    setChooseAgain(false)
    commitMut.reset()
    previewMut.mutate(f)
  }
  function onFile(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0]
    setFile(f)
    commitMut.reset()
    previewMut.reset()
    if (f) previewMut.mutate(f)
  }

  const counts = preview?.counts
  const errorCount = (counts?.error ?? 0) + failed.size
  const todo = counts ? counts.add + counts.invite : 0
  const canImport = !!preview && !!file && !stale && failed.size === 0 && errorCount === 0 && todo > 0 && !commitMut.isPending && !commitMut.isSuccess
  let reason = ''
  if (!file) reason = 'Choose a file to preview.'
  else if (previewMut.isPending) reason = 'Checking the file…'
  else if (stale) reason = 'Preview the file again.'
  else if (errorCount > 0) reason = `Fix the ${errorCount} rows with errors and choose the file again.`
  else if (preview && todo === 0) reason = 'Nothing to import: every row already exists.'
  else if (!preview) reason = 'Choose a file to preview.'

  let status = ''
  if (previewMut.isPending && file) status = `Checking ${file.name}…`
  else if (failed.size > 0) status = `Import failed: ${failed.size} rows have errors. Fix the file and choose it again.`
  else if (preview && counts)
    status = `${preview.rows.length} rows checked: ${counts.add} to add, ${counts.invite} to invite, ${counts.exists} already exist, ${counts.error} with errors.`

  const rows = useMemo(() => {
    if (!preview) return []
    const bad = (r: ImportPreview['rows'][number]) => r.action === 'error' || failed.has(r.line)
    const byLine = [...preview.rows].sort((a, b) => a.line - b.line)
    return [...byLine.filter(bad), ...byLine.filter((r) => !bad(r))]
  }, [preview, failed])
  const shown = rows.slice(0, CEILING)
  const showMessage = rows.some((r) => failed.get(r.line) ?? r.message)

  let banner: React.ReactNode = null
  if (error && failed.size > 0) {
    banner = (
      <ErrorBanner ref={bannerRef} title="Import failed">
        {failed.size} rows have errors; they are marked below and listed first.
      </ErrorBanner>
    )
  } else if (error instanceof ApiError && error.details.some((d) => d.field === 'checksum') && file) {
    banner = (
      <ErrorBanner title="The file changed since the preview">
        Preview it again before importing.
        <div>
          <Button variant="secondary" size="sm" className="mt-2" onClick={previewAgain}>
            Preview again
          </Button>
        </div>
        {chooseAgain && <p className="mt-2">Choose the file again.</p>}
      </ErrorBanner>
    )
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

  const template = columns.join(',')
  return (
    <div className="flex flex-col gap-4">
      <div className="text-sm text-ink-muted">
        <p>
          The first line must be a header with these columns:{' '}
          <code className="rounded bg-canvas px-1.5 py-0.5 font-sans text-ink">{template}</code>
        </p>
        {hint && <p>{hint}</p>}
        <p>
          Columns are case-insensitive. Maximum 5 MB.{' '}
          <a
            download={`${target.kind}-template.csv`}
            href={'data:text/csv;charset=utf-8,' + encodeURIComponent(template + '\n')}
            className="inline-flex items-center gap-1 text-accent underline hover:text-accent-hover"
          >
            <Download />
            Download template
          </a>
        </p>
      </div>
      <FormField label="CSV file" name="file">
        <input
          ref={inputRef}
          type="file"
          accept=".csv,text/csv"
          className={fileInputClass}
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
      {preview && file && (
        <div>
          <div className="max-h-[55vh] overflow-y-auto">
            <Table caption={`Preview of ${file.name}`} stack>
              <thead>
                <tr>
                  <th scope="col" className="text-right">
                    Line
                  </th>
                  <th scope="col">Result</th>
                  {showMessage && <th scope="col">Message</th>}
                  {columns.map((c) => (
                    <th key={c} scope="col">
                      {c}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {shown.map((r) => {
                  const bad = r.action === 'error' || failed.has(r.line)
                  return (
                    <tr key={r.line} className={bad ? 'bg-danger-soft' : ''}>
                      <td data-label="Line" className="text-right">
                        {r.line}
                      </td>
                      <td data-label="Result">
                        <ActionBadge action={bad ? 'error' : r.action} />
                      </td>
                      {showMessage && (
                        <td data-label="Message" className={bad ? 'text-danger' : ''}>
                          {failed.get(r.line) ?? r.message}
                        </td>
                      )}
                      {columns.map((c) => (
                        <td key={c} data-label={c}>
                          {r.data[c.toLowerCase()] ?? ''}
                        </td>
                      ))}
                    </tr>
                  )
                })}
              </tbody>
            </Table>
          </div>
          {rows.length > CEILING && (
            <p className="mt-2 text-sm text-ink-muted">
              {errorCount > CEILING
                ? `Showing the first ${CEILING} of ${errorCount} rows with errors.`
                : `Showing ${CEILING} of ${rows.length} rows. Rows with errors are listed first.`}
            </p>
          )}
        </div>
      )}
      <DialogFooter>
        <p id="csv-import-reason" className="mr-auto text-sm text-ink-muted">
          {canImport ? '' : reason}
        </p>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button
          disabled={!canImport}
          aria-describedby={canImport ? undefined : 'csv-import-reason'}
          onClick={() => file && preview && commitMut.mutate({ f: file, checksum: preview.checksum })}
        >
          {commitMut.isPending ? 'Importing…' : preview ? `Import ${todo} rows` : 'Import'}
        </Button>
      </DialogFooter>
    </div>
  )
}
