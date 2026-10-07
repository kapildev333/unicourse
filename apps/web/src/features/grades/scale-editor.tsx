import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useRef, useState } from 'react'
import { api, unwrap } from '../../api/client'
import { ApiError } from '../../api/errors'
import { Button } from '../../components/button'
import { ErrorBanner } from '../../components/error-banner'
import { FormErrors, errorMessage, useFocusFirstError } from '../../components/form-errors'
import { inputClass } from '../../components/form-field'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { StatusMessage } from '../../components/status-message'
import { Table } from '../../components/table'
import { useMe } from '../auth/use-me'
import { fmt2, fmtNum, hasScaleErrors, sentence, validateScale, type ScaleErrors } from './grade'
import { canadianPreset, usPreset } from './presets'
import { useGradeScale, useScaleRefresh, type GradeScale, type GradeScaleRow } from './queries'

type Field = 'letter' | 'minPercent' | 'points'
type EditRow = { key: string; letter: string; minPercent: string; points: string }

const scaleCopy = {
  'validation_failed:letter': 'Enter a letter. Each letter can appear only once.',
  'validation_failed:minPercent': 'Enter a percent from 0 to 100. Each can appear only once.',
  'validation_failed:points': 'Enter grade points from 0 to 9.99.',
  'validation_failed:rows': 'One row must start at 0 so every percent gets a letter.',
  conflict: 'Someone else changed the scale at the same time. Reload the page and try again.',
  forbidden: 'Only university administrators can change the grade scale.',
}
const adminsOnly = 'Only university administrators can change the grade scale.'

const sorted = (rows: GradeScaleRow[]) => [...rows].sort((a, b) => b.minPercent - a.minPercent)

export function ScaleTable({ rows, caption }: { rows: GradeScaleRow[]; caption: string }) {
  return (
    <Table caption={caption}>
      <thead>
        <tr>
          <th scope="col">Letter</th>
          <th scope="col" className="text-right">Minimum percent</th>
          <th scope="col" className="text-right">Grade points</th>
        </tr>
      </thead>
      <tbody>
        {sorted(rows).map((r) => (
          <tr key={r.letter}>
            <th scope="row">{r.letter}</th>
            <td className="text-right">{fmtNum(r.minPercent)}%</td>
            <td className="text-right">{fmt2(r.points)}</td>
          </tr>
        ))}
      </tbody>
    </Table>
  )
}

let nextKey = 0
const toEdit = (rows: GradeScaleRow[]): EditRow[] =>
  sorted(rows).map((r) => ({
    key: `r${nextKey++}`,
    letter: r.letter,
    minPercent: fmtNum(r.minPercent),
    points: fmtNum(r.points),
  }))
const snapshot = (rows: EditRow[]) => JSON.stringify(rows.map(({ letter, minPercent, points }) => [letter, minPercent, points]))

/** Server detail fields (`rows[2].minPercent`, `rows`) mapped onto the editor's rows. */
function fromServer(error: unknown): ScaleErrors {
  const out: ScaleErrors = { rows: {} }
  if (!(error instanceof ApiError) || error.code !== 'validation_failed') return out
  for (const d of error.details) {
    const m = /^rows\[(\d+)\]\.(\w+)$/.exec(d.field)
    if (m && ['letter', 'minPercent', 'points'].includes(m[2])) {
      const i = Number(m[1])
      const f = m[2] as Field
      out.rows[i] = { ...out.rows[i], [f]: scaleCopy[`validation_failed:${f}`] ?? sentence(d.message) }
    } else if (d.field === 'rows') out.table = scaleCopy['validation_failed:rows']
  }
  return out
}

function ScaleForm({ slug, data }: { slug: string; data: GradeScale }) {
  const qc = useQueryClient()
  const refresh = useScaleRefresh(slug)
  const [seen, setSeen] = useState(data)
  const [rows, setRows] = useState(() => toEdit(data.rows))
  const [loaded, setLoaded] = useState(() => snapshot(toEdit(data.rows)))
  const [errors, setErrors] = useState<ScaleErrors | null>(null)
  const [serverError, setServerError] = useState<unknown>(null)
  const [notice, setNotice] = useState('')
  const formRef = useRef<HTMLFormElement>(null)
  const bannerRef = useRef<HTMLDivElement>(null)
  const focusKey = useRef<string | null>(null)
  const dirty = snapshot(rows) !== loaded

  const reset = (next: GradeScale) => {
    const r = toEdit(next.rows)
    setRows(r)
    setLoaded(snapshot(r))
  }
  if (data !== seen) {
    setSeen(data)
    if (!dirty) reset(data)
  }

  const m = useMutation({
    mutationFn: (body: { rows: GradeScaleRow[] }) =>
      unwrap(api.PUT('/universities/{slug}/grade-scale', { params: { path: { slug } }, body })),
    onSuccess: async (res) => {
      qc.setQueryData([slug, 'grade-scale'], res)
      setSeen(res)
      reset(res)
      setNotice('Grade scale saved. Final grades already recorded keep their letters.')
      await refresh()
    },
    onError: (e) => {
      const parsed = fromServer(e)
      if (hasScaleErrors(parsed)) setErrors(parsed)
      else setServerError(e)
    },
  })
  useFocusFirstError(formRef, errors ?? serverError, bannerRef)

  const edit = (key: string, f: Field, v: string) => setRows((rs) => rs.map((r) => (r.key === key ? { ...r, [f]: v } : r)))
  const load = (p: GradeScaleRow[]) => {
    setRows(toEdit(p))
    setErrors(null)
    setServerError(null)
  }

  function save(e: React.FormEvent) {
    e.preventDefault()
    setNotice('')
    setServerError(null)
    const v = validateScale(rows)
    if (hasScaleErrors(v)) return setErrors(v)
    setErrors(null)
    m.mutate({
      rows: rows.map((r) => ({ letter: r.letter.trim(), minPercent: Number(r.minPercent), points: Number(r.points) })),
    })
  }

  const cell = (r: EditRow, i: number, f: Field, label: string, decimals: boolean) => {
    const err = errors?.rows[i]?.[f]
    const id = `scale-${r.key}-${f}-error`
    return (
      <td>
        <input
          type="text"
          inputMode={decimals ? 'decimal' : undefined}
          className={`${inputClass} w-24`}
          aria-label={`${label}, row ${i + 1}`}
          aria-invalid={err ? true : undefined}
          aria-describedby={err ? id : undefined}
          value={r[f]}
          ref={
            f === 'letter'
              ? (el) => {
                  if (el && focusKey.current === r.key) {
                    focusKey.current = null
                    el.focus()
                  }
                }
              : undefined
          }
          onChange={(e) => edit(r.key, f, e.target.value)}
        />
        {err && (
          <p id={id} className="mt-1 text-xs text-danger">
            {err}
          </p>
        )}
      </td>
    )
  }

  return (
    <form ref={formRef} noValidate onSubmit={save} className="flex flex-col gap-4">
      <StatusMessage message={notice} />
      {errors?.table ? (
        <ErrorBanner ref={bannerRef} title="">
          {errors.table}
        </ErrorBanner>
      ) : (
        serverError != null && <FormErrors ref={bannerRef} error={serverError} copy={scaleCopy} />
      )}
      <div className="flex flex-wrap items-center gap-2">
        <Button variant="secondary" onClick={() => load(usPreset)}>
          Load US preset
        </Button>
        <Button variant="secondary" onClick={() => load(canadianPreset)}>
          Load Canadian preset
        </Button>
        {dirty && <span className="ml-auto text-sm text-ink-muted">Unsaved changes</span>}
      </div>
      <Table caption="Grade scale">
        <thead>
          <tr>
            <th scope="col">Letter</th>
            <th scope="col">Minimum percent</th>
            <th scope="col">Grade points</th>
            <th scope="col">
              <span className="sr-only">Actions</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={r.key}>
              {cell(r, i, 'letter', 'Letter', false)}
              {cell(r, i, 'minPercent', 'Minimum percent', true)}
              {cell(r, i, 'points', 'Grade points', true)}
              <td className="text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-danger hover:bg-danger-soft"
                  aria-label={`Remove row ${i + 1}`}
                  onClick={() => setRows((rs) => rs.filter((x) => x.key !== r.key))}
                >
                  Remove
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </Table>
      <div>
        <Button
          variant="secondary"
          onClick={() => {
            const key = `r${nextKey++}`
            focusKey.current = key
            setRows((rs) => [...rs, { key, letter: '', minPercent: '', points: '' }])
          }}
        >
          Add row
        </Button>
      </div>
      <div className="flex justify-end gap-2">
        {dirty && (
          <Button
            variant="secondary"
            onClick={() => {
              reset(data)
              setErrors(null)
              setServerError(null)
            }}
          >
            Discard changes
          </Button>
        )}
        <Button type="submit" disabled={m.isPending}>
          {m.isPending ? 'Saving…' : 'Save scale'}
        </Button>
      </div>
    </form>
  )
}

export function ScaleEditorPage({ slug }: { slug: string }) {
  const me = useMe()
  const scale = useGradeScale(slug)
  const isAdmin = me.data?.memberships.find((m) => m.university.slug === slug)?.role === 'admin'
  let body: React.ReactNode
  if (scale.isPending || me.isPending) body = <Loading />
  else if (scale.error) body = <ErrorBanner onRetry={() => void scale.refetch()}>{errorMessage(scale.error)}</ErrorBanner>
  else if (isAdmin) body = <ScaleForm slug={slug} data={scale.data} />
  else
    body = (
      <>
        <ScaleTable rows={scale.data.rows} caption="Grade scale" />
        <p className="mt-4 text-sm text-ink-muted">{adminsOnly}</p>
      </>
    )
  return (
    <>
      <PageHeader
        title="Grade scale"
        description="Letters, the minimum percent for each, and the grade points used for GPA. Final grades already recorded keep the letter they were given."
      />
      {body}
    </>
  )
}
