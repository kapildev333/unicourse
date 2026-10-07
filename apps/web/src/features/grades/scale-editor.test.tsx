import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { canadianPreset, usPreset } from './presets'
import { ScaleEditorPage } from './scale-editor'

afterEach(() => vi.unstubAllGlobals())

const G = 'GET /universities/a/grade-scale'
const P = 'PUT /universities/a/grade-scale'
// Server order is highest first; send shuffled to prove the editor sorts.
const scale = { rows: [...usPreset].reverse() }

function renderPage(role: string, routes: Parameters<typeof mockApi>[0]) {
  const fn = mockApi({ [G]: () => ({ body: scale }), ...routes })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  client.setQueryData(['me'], {
    user: { id: 'u0', email: 'x@a.edu', displayName: 'X' },
    memberships: [{ university: { id: 'a', slug: 'a', name: 'A U' }, role, departmentIds: [] }],
  })
  render(
    <QueryClientProvider client={client}>
      <ScaleEditorPage slug="a" />
    </QueryClientProvider>,
  )
  return fn
}
const puts = (fn: ReturnType<typeof mockApi>) => fn.mock.calls.filter(([r]) => r.method === 'PUT')
const body = async (fn: ReturnType<typeof mockApi>) => JSON.parse(await puts(fn)[0][0].text())
const input = (name: string) => screen.getByLabelText(name) as HTMLInputElement

it('admin edits a point value and saves the rows as numbers', async () => {
  const fn = renderPage('admin', { [P]: (r) => r.text().then((t) => ({ body: JSON.parse(t) })) })
  await screen.findByLabelText('Letter, row 1')
  expect(input('Letter, row 1').value).toBe('A')
  expect(input('Letter, row 12').value).toBe('F')
  fireEvent.change(input('Grade points, row 1'), { target: { value: '4.3' } })
  expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Save scale' }))
  expect(await screen.findByText(/Grade scale saved/)).toBeInTheDocument()
  expect(puts(fn)).toHaveLength(1)
  expect(await body(fn)).toEqual({ rows: usPreset.map((r, i) => (i === 0 ? { ...r, points: 4.3 } : r)) })
  expect(screen.queryByText('Unsaved changes')).not.toBeInTheDocument()
})

it('loads the Canadian preset and saves it', async () => {
  const fn = renderPage('admin', { [P]: () => ({ body: { rows: canadianPreset } }) })
  await screen.findByLabelText('Letter, row 1')
  fireEvent.click(screen.getByRole('button', { name: 'Load Canadian preset' }))
  expect(screen.getByLabelText('Letter, row 13')).toBeInTheDocument()
  expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Save scale' }))
  await screen.findByText(/Grade scale saved/)
  expect(await body(fn)).toEqual({ rows: canadianPreset })
})

it('flags a duplicate letter inline and sends nothing', async () => {
  const fn = renderPage('admin', {})
  await screen.findByLabelText('Letter, row 1')
  fireEvent.change(input('Letter, row 2'), { target: { value: 'A' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save scale' }))
  const el = input('Letter, row 2')
  await waitFor(() => expect(el).toHaveFocus())
  expect(el).toHaveAttribute('aria-invalid', 'true')
  expect(screen.getByText('Each letter can appear only once.')).toBeInTheDocument()
  expect(puts(fn)).toHaveLength(0)
})

it('asks for a row at 0 in the banner', async () => {
  const fn = renderPage('admin', {})
  await screen.findByLabelText('Letter, row 1')
  fireEvent.click(screen.getByRole('button', { name: 'Remove row 12' }))
  fireEvent.click(screen.getByRole('button', { name: 'Save scale' }))
  expect(await screen.findByRole('alert')).toHaveTextContent('One row must start at 0 so every percent gets a letter.')
  expect(puts(fn)).toHaveLength(0)
})

it('maps a server validation error onto the row', async () => {
  renderPage('admin', {
    [P]: () => ({
      status: 400,
      body: { error: { code: 'validation_failed', message: 'm', details: [{ field: 'rows[2].minPercent', message: 'duplicate minPercent' }] } },
    }),
  })
  await screen.findByLabelText('Letter, row 1')
  fireEvent.click(screen.getByRole('button', { name: 'Save scale' }))
  const el = input('Minimum percent, row 3')
  await waitFor(() => expect(el).toHaveAttribute('aria-invalid', 'true'))
  expect(screen.getByText('Enter a percent from 0 to 100. Each can appear only once.')).toBeInTheDocument()
})

it('dept_admin sees the table read-only with the administrators-only note', async () => {
  renderPage('dept_admin', {})
  expect(await screen.findByRole('table', { name: 'Grade scale' })).toBeInTheDocument()
  expect(screen.getByText('93%')).toBeInTheDocument()
  expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Save scale' })).not.toBeInTheDocument()
  expect(screen.getByText('Only university administrators can change the grade scale.')).toBeInTheDocument()
})

it('shows each row range from the next higher minimum', async () => {
  renderPage('admin', {})
  await screen.findByLabelText('Letter, row 1')
  expect(input('Letter, row 1').closest('tr')).toHaveTextContent('93 to 100')
  expect(input('Letter, row 2').closest('tr')).toHaveTextContent('90 to 92.99')
  fireEvent.change(input('Minimum percent, row 2'), { target: { value: '91' } })
  expect(input('Letter, row 2').closest('tr')).toHaveTextContent('91 to 92.99')
})

it('puts Save in the header, sizes the columns and makes Remove a neutral ghost', async () => {
  renderPage('admin', {})
  await screen.findByLabelText('Letter, row 1')
  const save = screen.getByRole('button', { name: 'Save scale' })
  expect(save.closest('form')).toBeNull()
  expect(save).toHaveAttribute('form')
  expect(input('Letter, row 1').className).toContain('w-20')
  expect(input('Grade points, row 1').className).toContain('w-28')
  expect(screen.getByRole('button', { name: 'Remove row 1' }).className).not.toContain('text-danger')
})

it('shows Unsaved changes as muted text and never together with the saved notice', async () => {
  renderPage('admin', { [P]: (r) => r.text().then((t) => ({ body: JSON.parse(t) })) })
  await screen.findByLabelText('Letter, row 1')
  fireEvent.change(input('Grade points, row 1'), { target: { value: '4.3' } })
  expect(screen.getByText('Unsaved changes')).toHaveClass('text-ink-muted')
  fireEvent.click(screen.getByRole('button', { name: 'Save scale' }))
  await screen.findByText(/Grade scale saved/)
  fireEvent.change(input('Grade points, row 1'), { target: { value: '4.2' } })
  expect(screen.queryByText(/Grade scale saved/)).not.toBeInTheDocument()
  expect(screen.getByText('Unsaved changes')).toBeInTheDocument()
})
