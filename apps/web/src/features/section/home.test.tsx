import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { SectionDescription } from './home'
import { mine, renderSection, sectionFixture } from './test-helpers'

afterEach(() => vi.unstubAllGlobals())

function setup(description: string, o: { userId?: string; role?: 'student'; put?: () => { status?: number; body?: unknown } } = {}) {
  const fn = mockApi({
    'GET /universities/a/me/sections': () => ({ body: mine(o.role) }),
    'PUT /universities/a/sections/s1/description': o.put ?? (() => ({ body: {} })),
    'GET /universities/a/sections/s1': () => ({ body: sectionFixture({ description }) }),
  })
  renderSection({
    path: '/a/sections/s1',
    ui: <SectionDescription slug="a" section={sectionFixture({ description })} />,
    userId: o.userId,
  })
  return fn
}
const heading = () => screen.queryByRole('heading', { name: 'About this section' })

it('invites an editor to add a description', async () => {
  setup('', { userId: 'u-ins' })
  expect(await screen.findByRole('heading', { name: 'About this section' })).toBeInTheDocument()
  expect(screen.getByText(/No description yet\./)).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Add description' })).toBeInTheDocument()
})

it('shows nothing to others when empty', async () => {
  setup('', { role: 'student' })
  await new Promise((r) => setTimeout(r, 50))
  expect(heading()).toBeNull()
})

it('shows the text to a reader without an edit button', async () => {
  setup('Line 1\nLine 2', { role: 'student' })
  expect(await screen.findByRole('heading', { name: 'About this section' })).toBeInTheDocument()
  expect(screen.getByText(/Line 1/)).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /description/ })).toBeNull()
})

it('edits and saves', async () => {
  const fn = setup('Old', { userId: 'u-ins' })
  fireEvent.click(await screen.findByRole('button', { name: 'Edit description' }))
  expect(screen.getByRole('dialog', { name: 'Edit description' })).toBeInTheDocument()
  const box = screen.getByLabelText('Description')
  expect(box).toHaveValue('Old')
  expect(screen.getByText(/3 \/ 10,000 characters/)).toBeInTheDocument()
  fireEvent.change(box, { target: { value: 'New text' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save description' }))
  await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Description saved.'))
  expect(screen.queryByRole('dialog')).toBeNull()
  const put = fn.mock.calls.find(([r]) => r.method === 'PUT')![0]
  expect(await put.json()).toEqual({ description: 'New text' })
})

it('shows the field error and the forbidden banner', async () => {
  setup('Old', {
    userId: 'u-ins',
    put: () => ({ status: 400, body: { error: { code: 'validation_failed', message: 'x', details: [{ field: 'description', message: 'must be at most 10000 characters' }] } } }),
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Edit description' }))
  fireEvent.change(screen.getByLabelText('Description'), { target: { value: 'x' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save description' }))
  expect(await screen.findByText('Use at most 10,000 characters of plain text.')).toBeInTheDocument()
  expect(screen.getByLabelText('Description')).toHaveAttribute('aria-invalid', 'true')
})

it('shows the forbidden banner', async () => {
  setup('Old', { userId: 'u-ins', put: () => ({ status: 403, body: { error: { code: 'forbidden', message: 'no', details: [] } } }) })
  fireEvent.click(await screen.findByRole('button', { name: 'Edit description' }))
  fireEvent.change(screen.getByLabelText('Description'), { target: { value: 'x' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save description' }))
  expect(await screen.findByText("Only this section's staff and administrators can edit its description.")).toBeInTheDocument()
})
