import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { FinalizeDialog, OverrideDialog } from './finalize-dialog'
import { usPreset } from './presets'
import type { Gradebook } from './queries'

afterEach(() => vi.unstubAllGlobals())

const POST = 'POST /universities/a/sections/s1/final-grades'
const gb = 'GET /universities/a/sections/s1/gradebook'
const asm = (id: string, name: string, maxPoints: number, weight: number) => ({
  id,
  sectionId: 's1',
  name,
  maxPoints,
  weight,
  published: false,
  position: 1,
})
const stu = (id: string, name: string, scores: Record<string, number>) => ({
  userId: `u-${id}`,
  email: `${id}@a.edu`,
  displayName: name,
  scores,
})
const gradebook = {
  assessments: [asm('a1', 'Midterm', 40, 40), asm('a2', 'Final exam', 100, 60)],
  students: [stu('erin', 'Erin Moe', { a1: 40, a2: 85 }), stu('sam', 'Sam Lee', { a1: 30 })],
  stats: { letters: {} },
} as unknown as Gradebook
const scale = { rows: usPreset }

function setup(routes: Parameters<typeof mockApi>[0] = {}) {
  const fn = mockApi({ [gb]: () => ({ body: gradebook }), ...routes })
  const onDone = vi.fn()
  const onClose = vi.fn()
  renderWithQuery(
    <FinalizeDialog
      slug="a"
      sectionId="s1"
      code="CS101-001"
      gradebook={gradebook}
      scale={scale}
      onClose={onClose}
      onDone={onDone}
    />,
  )
  return { fn, onDone, onClose }
}
const calls = (fn: ReturnType<typeof mockApi>, key: string) =>
  fn.mock.calls
    .map((c) => c[0] as Request)
    .filter((r) => `${r.method} ${new URL(r.url).pathname.replace('/api/v1', '')}` === key)
const err = (status: number, code: string, details: { field: string; message: string }[] = []) => ({
  status,
  body: { error: { code, message: 'x', details } },
})
const choose = (label: string, value: string) => fireEvent.change(screen.getByLabelText(label), { target: { value } })
const confirm = () => fireEvent.click(screen.getByRole('button', { name: /^Finalize \d+ grades/ }))

it('previews percent, missing count, letter and summary', () => {
  setup()
  expect(screen.getByText(/2 students · 1 with missing scores · Preview: /)).toHaveTextContent('A- 1 · F 1')
  expect(screen.getByRole('cell', { name: '30.00% 1 missing' })).toBeInTheDocument()
  expect(screen.getByText('1 missing')).toBeInTheDocument()
  expect(screen.getByRole('option', { name: 'Keep F' })).toBeInTheDocument()
  expect(screen.queryByRole('columnheader', { name: 'Letter' })).not.toBeInTheDocument()
  expect(screen.getByText(/Showing 2 of 2/)).toBeInTheDocument()
  // the student with a missing score sorts first, and the name cells are capped
  const rows = screen.getAllByRole('rowheader')
  expect(rows[0]).toHaveTextContent('Sam Lee')
  expect(rows[0]).toHaveAttribute('title', 'Sam Lee')
  expect(rows[0]).toHaveClass('max-w-48', 'truncate')
  expect(screen.getByRole('columnheader', { name: 'Student' })).toHaveClass('max-w-48', 'truncate')
})

it('blocks an override without a reason and focuses the input', () => {
  const { fn } = setup()
  choose('Grade for Sam Lee', 'status:incomplete')
  const reason = screen.getByLabelText('Reason for Sam Lee')
  confirm()
  expect(screen.getByText('Enter a reason for this change.')).toBeInTheDocument()
  expect(reason).toHaveFocus()
  expect(calls(fn, POST)).toHaveLength(0)
})

it('posts an empty body when nothing is overridden and reports success', async () => {
  const { fn, onDone } = setup({ [POST]: () => ({ body: { items: [] } }) })
  confirm()
  await waitFor(() => expect(onDone).toHaveBeenCalledWith('Final grades recorded for 2 students.'))
  expect(JSON.parse(await calls(fn, POST)[0].text())).toEqual({})
})

it('posts the override with its reason', async () => {
  const { fn, onDone } = setup({ [POST]: () => ({ body: { items: [] } }) })
  choose('Grade for Sam Lee', 'status:incomplete')
  fireEvent.change(screen.getByLabelText('Reason for Sam Lee'), { target: { value: 'Medical leave' } })
  confirm()
  await waitFor(() => expect(onDone).toHaveBeenCalled())
  expect(JSON.parse(await calls(fn, POST)[0].text())).toEqual({
    overrides: [{ userId: 'u-sam', status: 'incomplete', reason: 'Medical leave' }],
  })
})

it('shows the weights copy on 400 and disables confirm on 409', async () => {
  let reply = err(400, 'validation_failed', [{ field: 'weights', message: 'must total 100' }])
  const { onClose, fn } = setup({ [POST]: () => reply })
  confirm()
  expect(await screen.findByText('Assessment weights must total 100% before you finalize.')).toBeInTheDocument()
  reply = err(409, 'conflict')
  confirm()
  expect(await screen.findByText(/already finalized/)).toBeInTheDocument()
  expect(screen.getByRole('button', { name: /^Finalize \d+ grades/ })).toBeDisabled()
  fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
  expect(onClose).toHaveBeenCalled()
  expect(calls(fn, POST)).toHaveLength(2)
})

it('maps a server override error onto its row', async () => {
  setup({ [POST]: () => err(400, 'validation_failed', [{ field: 'overrides[0].reason', message: 'required' }]) })
  choose('Grade for Sam Lee', 'letter:B')
  fireEvent.change(screen.getByLabelText('Reason for Sam Lee'), { target: { value: 'x' } })
  confirm()
  expect(await screen.findByText('Enter a reason for this change.')).toBeInTheDocument()
  expect(screen.getByLabelText('Reason for Sam Lee')).toHaveAttribute('aria-invalid', 'true')
})

it('OverrideDialog patches the letter with a reason and blocks a missing reason', async () => {
  const PATCH = 'PATCH /universities/a/sections/s1/final-grades/u-erin'
  const fn = mockApi({ [PATCH]: () => ({ body: {} }), [gb]: () => ({ body: gradebook }) })
  const onDone = vi.fn()
  const student = { ...gradebook.students[0], final: { status: 'graded', letter: 'B+' } } as Gradebook['students'][number]
  renderWithQuery(
    <OverrideDialog slug="a" sectionId="s1" student={student} scale={scale} onClose={() => {}} onDone={onDone} />,
  )
  fireEvent.change(screen.getByLabelText('Grade'), { target: { value: 'letter:A-' } })
  fireEvent.click(screen.getByRole('button', { name: 'Change grade' }))
  expect(screen.getByText('Enter a reason for this change.')).toBeInTheDocument()
  expect(calls(fn, PATCH)).toHaveLength(0)
  fireEvent.change(screen.getByLabelText('Reason'), { target: { value: 'Regrade' } })
  fireEvent.click(screen.getByRole('button', { name: 'Change grade' }))
  await waitFor(() => expect(onDone).toHaveBeenCalledWith('Final grade for Erin Moe changed to A-.'))
  expect(JSON.parse(await calls(fn, PATCH)[0].text())).toEqual({ letter: 'A-', reason: 'Regrade' })
})
