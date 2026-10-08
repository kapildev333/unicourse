import { fireEvent, screen, waitFor } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { TermPicker } from './term-picker'

afterEach(() => vi.unstubAllGlobals())

const t = (id: string, name: string) => ({ id, name, startsOn: '2026-01-01', endsOn: '2026-05-01' })
const all = [t('1', 'Fall 2026'), t('2', 'Spring 2027')]
const L = 'GET /universities/x/terms'

function Host({ more }: { more: boolean }) {
  const [v, setV] = useState('1')
  return <TermPicker slug="x" terms={all} more={more} value={v} onChange={setV} />
}

it('has no search field without a next page', () => {
  mockApi({})
  renderWithQuery(<Host more={false} />)
  expect(screen.queryByLabelText('Search terms')).not.toBeInTheDocument()
})

it('searches with q and keeps the chosen term in the select', async () => {
  const fn = mockApi({ [L]: () => ({ body: { items: [t('2', 'Spring 2027')] } }) })
  renderWithQuery(<Host more />)
  fireEvent.change(screen.getByLabelText('Search terms'), { target: { value: 'spr' } })
  await waitFor(() =>
    expect(fn.mock.calls.some((c) => new URL((c[0] as Request).url).searchParams.get('q') === 'spr')).toBe(true),
  )
  const opts = () => screen.getAllByRole('option').map((o) => o.textContent)
  await waitFor(() => expect(opts()).toEqual(['Fall 2026', 'Spring 2027']))
  expect(screen.getByLabelText('Term')).toHaveValue('1')
})
