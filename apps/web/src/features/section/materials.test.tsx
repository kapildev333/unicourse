import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { formatDate } from '../../lib/format'
import { mockApi, type Routes } from '../../test-utils'
import { MaterialsPage } from './materials'
import { mine, renderSection, sectionFixture } from './test-helpers'

afterEach(() => vi.unstubAllGlobals())

const mat = (id: string, weekId: string, over: object = {}) => ({
  id,
  weekId,
  title: 'Lecture 1 slides',
  fileName: 'lecture-1.pdf',
  contentType: 'application/pdf',
  sizeBytes: 1258291,
  position: 1,
  uploadedBy: { userId: 'u-ins', displayName: 'Dana Reyes', email: 'd@a.edu' },
  createdAt: '2026-10-05T14:03:00Z',
  ...over,
})
const week = (n: number, materials: unknown[] = [], over: object = {}) => ({
  id: `w${n}`,
  sectionId: 's1',
  title: `Week ${n}`,
  notes: '',
  position: n,
  materials,
  ...over,
})
const two = () => ({
  items: [
    week(
      1,
      [
        mat('m1', 'w1'),
        mat('m2', 'w1', {
          title: 'notes.zip',
          fileName: 'notes.zip',
          contentType: 'application/zip',
          sizeBytes: 482113,
        }),
      ],
      { notes: 'Read chapter 1.' },
    ),
    week(2),
  ],
})
const err = (status: number, code: string, details: unknown[] = []) => ({
  status,
  body: { error: { code, message: 'm', details } },
})
const W = 'GET /universities/a/sections/s1/weeks'

function setup(routes: Routes, o: { editor?: boolean } = {}) {
  const editor = o.editor ?? true
  const fn = mockApi({
    'GET /universities/a/sections/s1': () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({
      body: mine(editor ? undefined : 'student'),
    }),
    [W]: () => ({ body: two() }),
    ...routes,
  })
  renderSection({
    path: '/a/sections/s1/materials',
    ui: <MaterialsPage slug="a" sectionId="s1" />,
    userId: editor ? 'u-ins' : 'u0',
  })
  return fn
}
const statusText = () =>
  screen
    .getAllByRole('status')
    .map((s) => s.textContent)
    .join('|')
const sent = async (fn: ReturnType<typeof mockApi>, method: string) =>
  (await fn.mock.calls.find(([r]) => r.method === method)![0].json()) as unknown

it('lists weeks in order with rows', async () => {
  setup({})
  await screen.findByText('Lecture 1 slides')
  expect(screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent)).toEqual(['Week 1', 'Week 2'])
  expect(screen.getByText('Read chapter 1.')).toBeInTheDocument()
  expect(screen.getByText(`lecture-1.pdf · 1.2 MB · Dana Reyes · ${formatDate('2026-10-05T14:03:00Z')}`)).toBeInTheDocument()
})

it('links downloads', async () => {
  setup({})
  const a = await screen.findByRole('link', {
    name: 'Download Lecture 1 slides',
  })
  expect(a).toHaveAttribute('href', '/api/v1/universities/a/sections/s1/materials/m1/download')
  expect(a).toHaveAttribute('download')
})

it('shows readers no edit controls', async () => {
  setup({}, { editor: false })
  await screen.findByText('Lecture 1 slides')
  expect(screen.queryByRole('button', { name: 'Add week' })).toBeNull()
  expect(screen.queryByRole('button', { name: 'Upload files to Week 1' })).toBeNull()
  expect(screen.queryByRole('button', { name: /^Rename/ })).toBeNull()
  expect(screen.getAllByRole('link', { name: /^Download/ })).toHaveLength(2)
})

it('adds a week with a prefilled title', async () => {
  const fn = setup({
    'POST /universities/a/sections/s1/weeks': () => ({
      status: 201,
      body: week(3),
    }),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Add week' }))
  expect(screen.getByLabelText('Title')).toHaveValue('Week 3')
  fireEvent.click(
    within(screen.getByRole('dialog')).getByRole('button', {
      name: 'Add week',
    }),
  )
  await waitFor(() => expect(statusText()).toContain('Week 3 added.'))
  expect(await sent(fn, 'POST')).toEqual({ title: 'Week 3', notes: '' })
})

it('moves a week and keeps focus on a move button', async () => {
  const fn = setup({
    'PATCH /universities/a/sections/s1/weeks/w1': () => ({
      body: week(1, [], { position: 2 }),
    }),
  })
  await screen.findByText('Week 1')
  expect(screen.getByRole('button', { name: 'Move Week 1 up' })).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: 'Move Week 1 down' }))
  await waitFor(() => expect(statusText()).toContain('Week 1 moved down.'))
  expect(await sent(fn, 'PATCH')).toEqual({ position: 2 })
  await waitFor(() => expect(document.activeElement?.id).toMatch(/^move-(up|down)-w1$/))
})

it('names the file count when deleting a week', async () => {
  const fn = setup({
    'DELETE /universities/a/sections/s1/weeks/w1': () => ({ status: 204 }),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Week 1' }))
  expect(screen.getByText("Its 2 files will be deleted too. This can't be undone.")).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Delete week' }))
  await waitFor(() => expect(fn.mock.calls.some(([r]) => r.method === 'DELETE')).toBe(true))
  await waitFor(() => expect(statusText()).toContain('Week 1 deleted.'))
})

it('uploads valid files and lists refused ones', async () => {
  let release!: () => void
  const gate = new Promise<void>((r) => (release = r))
  const fn = setup({
    'POST /universities/a/sections/s1/weeks/w1/materials': async () => {
      await gate
      return { status: 201, body: mat('m9', 'w1') }
    },
  })
  await screen.findByText('Week 1')
  const pdf = new File(['x'], 'a.pdf', { type: 'application/pdf' })
  const exe = new File(['x'], 'b.exe')
  fireEvent.change(screen.getByLabelText('Files for Week 1'), {
    target: { files: [pdf, exe] },
  })
  await waitFor(() => expect(screen.getAllByRole('status').map((s) => s.textContent)).toContain('Uploading 1 of 1: a.pdf'))
  expect(screen.getByRole('button', { name: 'Upload files to Week 2' })).toBeDisabled()
  release()
  await waitFor(() =>
    expect(screen.getAllByRole('status').map((s) => s.textContent)).toContain('Uploaded 1 of 2 files to Week 1.'),
  )
  const alert = screen.getByRole('alert')
  expect(alert).toHaveTextContent('1 file not uploaded')
  expect(alert).toHaveTextContent("b.exe: This file type can't be uploaded.")
  const posts = fn.mock.calls.filter(([r]) => r.method === 'POST')
  expect(posts).toHaveLength(1)
  expect(await posts[0][0].text()).toContain('name="file"') // the stub's FormData reports filename="blob"; the browser sends the real name
})

it('continues after a per-file server error', async () => {
  let n = 0
  const fn = setup({
    'POST /universities/a/sections/s1/weeks/w1/materials': () =>
      ++n === 1
        ? err(400, 'validation_failed', [{ field: 'file', message: 'is larger than 100 MB' }])
        : { status: 201, body: mat('m9', 'w1') },
  })
  await screen.findByText('Week 1')
  const files = [new File(['x'], 'a.pdf'), new File(['x'], 'b.pdf')]
  fireEvent.change(screen.getByLabelText('Files for Week 1'), {
    target: { files },
  })
  const alert = await screen.findByRole('alert')
  expect(alert).toHaveTextContent('a.pdf: It is larger than 100 MB.')
  expect(fn.mock.calls.filter(([r]) => r.method === 'POST')).toHaveLength(2)
})

it('renames and moves a file', async () => {
  const fn = setup({
    'PATCH /universities/a/sections/s1/materials/m1': () => ({
      body: mat('m1', 'w1'),
    }),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Rename Lecture 1 slides' }))
  fireEvent.change(screen.getByLabelText('Title'), {
    target: { value: 'Slides' },
  })
  fireEvent.click(screen.getByRole('button', { name: 'Save title' }))
  await waitFor(() => expect(statusText()).toContain('Renamed to Slides.'))
  expect(await sent(fn, 'PATCH')).toEqual({ title: 'Slides' })

  fireEvent.click(
    screen.getByRole('button', {
      name: 'Move Lecture 1 slides to another week',
    }),
  )
  const sel = screen.getByLabelText('Week')
  expect(
    within(sel)
      .getAllByRole('option')
      .map((o) => o.textContent),
  ).toEqual(['Week 2'])
  fireEvent.click(screen.getByRole('button', { name: 'Move file' }))
  await waitFor(() => expect(statusText()).toContain('Lecture 1 slides moved to Week 2.'))
  expect(await fn.mock.calls.filter(([r]) => r.method === 'PATCH')[1][0].json()).toEqual({ weekId: 'w2' })
})

it('has empty states', async () => {
  setup({ [W]: () => ({ body: { items: [] } }) })
  await screen.findByText('No weeks yet')
  expect(screen.getAllByRole('button', { name: 'Add week' })).toHaveLength(1)
})

it('has a reader empty state and an empty week', async () => {
  setup({ [W]: () => ({ body: { items: [] } }) }, { editor: false })
  expect(await screen.findByText('No materials yet')).toBeInTheDocument()
})

it('says a week has no files', async () => {
  setup({ [W]: () => ({ body: { items: [week(1)] } }) })
  expect(await screen.findByText('No files in this week yet.')).toBeInTheDocument()
})

it('shows no access on 403', async () => {
  setup({ [W]: () => err(403, 'forbidden') })
  expect(await screen.findByText('No access')).toBeInTheDocument()
})
