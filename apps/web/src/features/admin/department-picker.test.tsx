import { fireEvent, screen, waitFor } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { DepartmentPicker } from './department-picker'

afterEach(() => vi.unstubAllGlobals())

const cs = { id: '1', name: 'Computer Science', code: 'CS' }
const math = { id: '2', name: 'Mathematics', code: 'MATH' }
const L = 'GET /universities/x/departments'

function Host() {
  const [v, setV] = useState<string[]>(['1'])
  return <DepartmentPicker slug="x" value={v} onChange={setV} />
}

it('shows no search field when the list fits one page', async () => {
  mockApi({ [L]: () => ({ body: { items: [cs, math] } }) })
  renderWithQuery(<Host />)
  await screen.findByText('Mathematics')
  expect(screen.queryByLabelText('Search departments')).not.toBeInTheDocument()
})

it('searches with q past 200 rows and keeps a filtered-out tick', async () => {
  const fn = mockApi({
    [L]: (req) =>
      new URL(req.url).searchParams.get('q')
        ? { body: { items: [math] } }
        : { body: { items: [cs, math], nextCursor: 'n' } },
  })
  renderWithQuery(<Host />)
  await screen.findByText('Mathematics')
  fireEvent.change(screen.getByLabelText('Search departments'), { target: { value: 'ma' } })
  await waitFor(() =>
    expect(fn.mock.calls.some((c) => new URL((c[0] as Request).url).searchParams.get('q') === 'ma')).toBe(true),
  )
  await waitFor(() => expect(screen.queryByText('Computer Science')).not.toBeInTheDocument())
  fireEvent.change(screen.getByLabelText('Search departments'), { target: { value: '' } })
  expect(await screen.findByLabelText(/Computer Science/)).toBeChecked()
})

it('says nothing matches, not "No departments yet", when a search is empty', async () => {
  mockApi({
    [L]: (req) =>
      new URL(req.url).searchParams.get('q') ? { body: { items: [] } } : { body: { items: [cs, math], nextCursor: 'n' } },
  })
  renderWithQuery(<Host />)
  await screen.findByText('Mathematics')
  fireEvent.change(screen.getByLabelText('Search departments'), { target: { value: 'zzz' } })
  expect(await screen.findByText('No departments match.')).toBeInTheDocument()
  expect(screen.queryByText(/No departments yet/)).not.toBeInTheDocument()
})
