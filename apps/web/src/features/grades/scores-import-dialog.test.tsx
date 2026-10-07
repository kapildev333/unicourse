import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { ScoresImportDialog } from './scores-import-dialog'

afterEach(() => vi.unstubAllGlobals())

const base = '/universities/a/sections/s1/assessments'
const asm = (id: string, name: string) => ({ id, sectionId: 's1', name, maxPoints: 40, weight: 40, published: false, position: 1 })
const counts = (add: number, update: number, error: number) => ({ add, update, exists: 0, error })
const row = (line: number, action: string, extra: object = {}) => ({
  line,
  action,
  data: { email: `s${line}@a.edu`, points: '30' },
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
    <ScoresImportDialog
      slug="a"
      sectionId="s1"
      assessments={[asm('a1', 'Midterm'), asm('a2', 'Final exam')]}
      onClose={() => {}}
      onDone={onDone}
    />,
  )
  return { fn, onDone }
}
const pick = () =>
  fireEvent.change(screen.getByLabelText('CSV file'), { target: { files: [new File(['x'], 's.csv', { type: 'text/csv' })] } })
const importBtn = () => screen.getByRole('button', { name: /^Import/ })
const calls = (fn: ReturnType<typeof mockApi>, key: string) =>
  fn.mock.calls
    .map((c) => c[0] as Request)
    .filter((r) => `${r.method} ${new URL(r.url).pathname.replace('/api/v1', '')}` === key)

it('previews with counts and lists error rows first', async () => {
  const P = `POST ${base}/a1/scores/import/preview`
  const { fn } = setup({
    [P]: () => ({
      body: { checksum: 'k', rows: [row(2, 'add'), row(3, 'error', { message: 'unknown email' })], counts: counts(1, 0, 1) },
    }),
  })
  pick()
  const msg = await screen.findByText('unknown email')
  expect(calls(fn, P)).toHaveLength(1)
  expect(msg.closest('tr')).toHaveTextContent('Error')
  const bodyRows = screen.getAllByRole('row').slice(1)
  expect(bodyRows[0]).toHaveTextContent('unknown email')
  expect(screen.getByText('Update').nextSibling).toHaveTextContent('0')
  expect(importBtn()).toBeDisabled()
})

it('imports with the checksum and reports the result', async () => {
  const result = { checksum: 'k', rows: [row(2, 'add'), row(3, 'add'), row(4, 'update')], counts: counts(2, 1, 0) }
  const C = `POST ${base}/a1/scores/import`
  const { fn, onDone } = setup({
    [`POST ${base}/a1/scores/import/preview`]: () => ({ body: result }),
    [C]: () => ({ body: result }),
    'GET /universities/a/sections/s1/gradebook': () => ({ body: {} }),
  })
  pick()
  await waitFor(() => expect(importBtn()).toBeEnabled())
  expect(importBtn()).toHaveTextContent('Import 3 scores')
  fireEvent.click(importBtn())
  await waitFor(() => expect(onDone).toHaveBeenCalledWith('Scores imported for Midterm: 2 added, 1 updated.'))
  expect(calls(fn, C)).toHaveLength(1)
})

it('switching the assessment clears the preview and targets the new one', async () => {
  const result = { checksum: 'k', rows: [row(2, 'add')], counts: counts(1, 0, 0) }
  const P2 = `POST ${base}/a2/scores/import/preview`
  const { fn } = setup({
    [`POST ${base}/a1/scores/import/preview`]: () => ({ body: result }),
    [P2]: () => ({ body: result }),
  })
  pick()
  await screen.findByText('s2@a.edu')
  fireEvent.change(screen.getByLabelText('Assessment'), { target: { value: 'a2' } })
  expect(screen.queryByText('s2@a.edu')).not.toBeInTheDocument()
  pick()
  await screen.findByText('s2@a.edu')
  expect(calls(fn, P2)).toHaveLength(1)
})

it('marks a failed line and asks to choose the file again on a checksum mismatch', async () => {
  const result = { checksum: 'k', rows: [row(2, 'add'), row(3, 'add')], counts: counts(2, 0, 0) }
  let fail = [{ field: 'line:3', message: 'not enrolled in this section' }]
  setup({
    [`POST ${base}/a1/scores/import/preview`]: () => ({ body: result }),
    [`POST ${base}/a1/scores/import`]: () => err(fail),
  })
  pick()
  await waitFor(() => expect(importBtn()).toBeEnabled())
  fireEvent.click(importBtn())
  expect((await screen.findByText('Not enrolled in this section.')).closest('tr')).toHaveTextContent('Error')
  fail = [{ field: 'checksum', message: 'changed' }]
  pick()
  await waitFor(() => expect(importBtn()).toBeEnabled())
  fireEvent.click(importBtn())
  expect(await screen.findByText('The file changed since the preview')).toBeInTheDocument()
  expect(screen.getAllByText(/Choose the file again/).length).toBeGreaterThan(0)
})
