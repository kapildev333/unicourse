import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { AssessmentDialog } from './assessment-dialog'
import type { Assessment } from './queries'

afterEach(() => vi.unstubAllGlobals())

const A = '/universities/a/sections/s1/assessments'
const mid: Assessment = {
  id: 'a1',
  sectionId: 's1',
  name: 'Midterm',
  maxPoints: 40,
  weight: 40,
  published: false,
  position: 1,
}

function setup(routes: Parameters<typeof mockApi>[0], assessment?: Assessment) {
  const fn = mockApi({
    [`GET /universities/a/sections/s1/gradebook`]: () => ({ body: {} }),
    ...routes,
  })
  const onClose = vi.fn()
  const onSaved = vi.fn()
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <AssessmentDialog
        slug="a"
        sectionId="s1"
        assessment={assessment}
        scoredCount={3}
        otherWeight={60}
        onClose={onClose}
        onSaved={onSaved}
      />
    </QueryClientProvider>,
  )
  return { fn, onClose, onSaved }
}
const sent = (fn: ReturnType<typeof mockApi>, method: string) => fn.mock.calls.filter(([r]) => r.method === method)
const type = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } })

it('creates with numeric body', async () => {
  const { fn, onSaved } = setup({
    [`POST ${A}`]: (r) => r.json().then((b) => ({ status: 201, body: { ...mid, ...b } })),
  })
  type('Name', 'Midterm')
  type('Max points', '40')
  type('Weight (%)', '40')
  type('Due date', '2026-10-12')
  fireEvent.click(screen.getByRole('button', { name: 'Create assessment' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalledWith('Midterm created.'))
  expect(JSON.parse(await sent(fn, 'POST')[0][0].text())).toEqual({
    name: 'Midterm',
    maxPoints: 40,
    weight: 40,
    dueOn: '2026-10-12',
  })
})

it('rejects bad max points locally, focused, without a request', async () => {
  const { fn } = setup({})
  type('Name', 'Midterm')
  type('Weight (%)', '40')
  for (const v of ['0', 'abc']) {
    type('Max points', v)
    fireEvent.click(screen.getByRole('button', { name: 'Create assessment' }))
    expect(await screen.findByText('Enter maximum points greater than 0.')).toBeInTheDocument()
    await waitFor(() => expect(screen.getByLabelText('Max points')).toHaveFocus())
  }
  expect(sent(fn, 'POST')).toHaveLength(0)
})

it('shows the duplicate-name conflict under Name', async () => {
  setup({
    [`POST ${A}`]: () => ({
      status: 409,
      body: {
        error: {
          code: 'conflict',
          message: 'dup',
          details: [{ field: 'name', message: 'dup' }],
        },
      },
    }),
  })
  type('Name', 'Midterm')
  type('Max points', '40')
  type('Weight (%)', '40')
  fireEvent.click(screen.getByRole('button', { name: 'Create assessment' }))
  expect(await screen.findByText('This section already has an assessment with this name.')).toBeInTheDocument()
})

it('edit sends only the changed field; unchanged closes without a request', async () => {
  const { fn, onClose, onSaved } = setup(
    {
      [`PATCH ${A}/a1`]: (r) => r.json().then((b) => ({ body: { ...mid, ...b } })),
    },
    mid,
  )
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  expect(onClose).toHaveBeenCalled()
  expect(sent(fn, 'PATCH')).toHaveLength(0)
  type('Weight (%)', '35')
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalledWith('Midterm saved.'))
  expect(JSON.parse(await sent(fn, 'PATCH')[0][0].text())).toEqual({
    weight: 35,
  })
})

it('delete names the score count and deletes', async () => {
  const { fn, onSaved } = setup({ [`DELETE ${A}/a1`]: () => ({ status: 204 }) }, mid)
  fireEvent.click(screen.getByRole('button', { name: 'Delete assessment' }))
  expect(screen.getByText(/Its 3 scores are deleted too/)).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Delete' }))
  await waitFor(() => expect(onSaved).toHaveBeenCalledWith('Midterm deleted.'))
  expect(sent(fn, 'DELETE')).toHaveLength(1)
})

it('shows the other assessments total in the weight hint and sizes the inputs after inputClass', () => {
  setup({})
  expect(screen.getByText(/Other assessments total 60%\./)).toBeInTheDocument()
  expect(screen.getByLabelText('Max points').className).toMatch(/w-full.* w-32!/)
  expect(screen.getByLabelText('Weight (%)').className).toMatch(/w-full.* w-32!/)
  expect(screen.getByLabelText('Due date').className).toMatch(/w-full.* w-48!/)
})
