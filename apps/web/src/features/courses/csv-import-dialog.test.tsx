import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { CsvImportDialog, importSummary } from './csv-import-dialog'

afterEach(() => vi.unstubAllGlobals())

const PREVIEW = 'POST /universities/a/courses/import/preview'
const COMMIT = 'POST /universities/a/courses/import'
const counts = (add: number, error: number) => ({ add, exists: 0, invite: 0, error })
const row = (line: number, action: string, extra: object = {}) => ({
  line,
  action,
  data: { departmentcode: 'CS', code: `C${line}`, title: 'T' },
  ...extra,
})
const err = (details: { field: string; message: string }[]) => ({
  status: 400,
  body: { error: { code: 'validation_failed', message: 'bad', details } },
})

function setup(routes: Parameters<typeof mockApi>[0]) {
  const fn = mockApi(routes)
  const onDone = vi.fn()
  renderWithQuery(
    <CsvImportDialog
      open
      onClose={() => {}}
      title="Import courses"
      target={{ kind: 'courses', slug: 'a' }}
      columns={['departmentCode', 'code', 'title']}
      onDone={onDone}
    />,
  )
  return { fn, onDone }
}
const pick = (n = 'c.csv') =>
  fireEvent.change(screen.getByLabelText('CSV file'), { target: { files: [new File(['x'], n, { type: 'text/csv' })] } })
const importBtn = () => screen.getByRole('button', { name: /^Import/ })
const calls = (fn: ReturnType<typeof mockApi>, key: string) =>
  fn.mock.calls.map((c) => c[0] as Request).filter((r) => `${r.method} ${new URL(r.url).pathname.replace('/api/v1', '')}` === key)

it('disables Import and shows the message when a row has an error', async () => {
  setup({
    [PREVIEW]: () => ({
      body: {
        checksum: 'k',
        rows: [
          row(2, 'add'),
          row(3, 'error', { message: 'unknown department code MATH', data: { departmentcode: 'MATH', code: 'M1', title: 'X' } }),
        ],
        counts: counts(1, 1),
      },
    }),
  })
  pick()
  const msg = await screen.findByText('unknown department code MATH')
  const tr = msg.closest('tr')!
  expect(tr).toHaveTextContent('Error')
  expect(tr).toHaveTextContent('3')
  expect(importBtn()).toBeDisabled()
  expect(screen.getAllByText(/Fix the 1 rows with errors/).length).toBeGreaterThan(0)
})

it('commits a clean preview with the checksum and reports the result', async () => {
  const result = { checksum: 'k', rows: [row(2, 'add'), row(3, 'add')], counts: counts(2, 0) }
  const { fn, onDone } = setup({ [PREVIEW]: () => ({ body: result }), [COMMIT]: () => ({ body: result }) })
  pick()
  await waitFor(() => expect(importBtn()).toBeEnabled())
  expect(importBtn()).toHaveTextContent('Import 2 rows')
  fireEvent.click(importBtn())
  await waitFor(() => expect(onDone).toHaveBeenCalledWith(result))
  const [req] = calls(fn, COMMIT)
  expect(req.headers.get('content-type')).toMatch(/^multipart\/form-data/)
  expect((await req.formData()).get('checksum')).toBe('k')
  expect(calls(fn, COMMIT)).toHaveLength(1)
})

it('marks the rows named by a failed commit and lists them first', async () => {
  setup({
    [PREVIEW]: () => ({ body: { checksum: 'k', rows: [row(2, 'add'), row(3, 'add')], counts: counts(2, 0) } }),
    [COMMIT]: () => err([{ field: 'line:3', message: 'duplicate row' }]),
  })
  pick()
  await waitFor(() => expect(importBtn()).toBeEnabled())
  fireEvent.click(importBtn())
  const msg = await screen.findByText('Duplicate row.')
  const tr = msg.closest('tr')!
  expect(tr).toHaveTextContent('Error')
  expect(tr).toHaveTextContent('3')
  expect(document.querySelector('tbody tr')).toBe(tr)
  expect(importBtn()).toBeDisabled()
  expect(screen.getByText('Import failed')).toBeInTheDocument()
})

it('offers Preview again after a checksum mismatch', async () => {
  const { fn } = setup({
    [PREVIEW]: () => ({ body: { checksum: 'k', rows: [row(2, 'add')], counts: counts(1, 0) } }),
    [COMMIT]: () => err([{ field: 'checksum', message: 'mismatch' }]),
  })
  pick()
  await waitFor(() => expect(importBtn()).toBeEnabled())
  fireEvent.click(importBtn())
  expect(await screen.findByText('The file changed since the preview')).toBeInTheDocument()
  expect(importBtn()).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: 'Preview again' }))
  await waitFor(() => expect(calls(fn, PREVIEW)).toHaveLength(2))
})

it('shows a file error under its own title', async () => {
  setup({ [PREVIEW]: () => err([{ field: 'file', message: 'missing column code' }]) })
  pick()
  expect(await screen.findByText("This file can't be imported")).toBeInTheDocument()
  expect(screen.getByText('Missing column code.')).toBeInTheDocument()
})

it('caps the DOM at 200 rows with errors first', async () => {
  const rows = Array.from({ length: 250 }, (_, i) =>
    i + 1 === 240 ? row(240, 'error', { message: 'bad' }) : row(i + 1, 'add'),
  )
  setup({ [PREVIEW]: () => ({ body: { checksum: 'k', rows, counts: counts(249, 1) } }) })
  pick()
  expect(await screen.findByText('Showing 200 of 250 rows. Rows with errors are listed first.')).toBeInTheDocument()
  const trs = document.querySelectorAll('tbody tr')
  expect(trs).toHaveLength(200)
  expect(trs[0]).toHaveTextContent('240')
})

it('offers a template download for the columns', () => {
  setup({})
  const href = screen.getByRole('link', { name: 'Download template' }).getAttribute('href')!
  expect(decodeURIComponent(href.split(',').slice(1).join(','))).toBe('departmentCode,code,title\n')
})

it('summarises counts, omitting zero parts', () => {
  expect(importSummary('courses', { add: 12, invite: 3, exists: 5, error: 0 })).toBe(
    'Import finished: 12 added, 3 invited, 5 already existed.',
  )
  expect(importSummary('courses', { add: 0, invite: 0, exists: 0, error: 0 })).toBe('Import finished: nothing changed.')
})
