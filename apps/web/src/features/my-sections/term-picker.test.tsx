import { fireEvent, screen, waitFor } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { TermPicker } from './term-picker'

afterEach(() => vi.unstubAllGlobals())

const t = (id: string, name: string) => ({ id, name, startsOn: '2026-01-01', endsOn: '2026-05-01' })
const all = [t('1', 'Fall 2026'), t('2', 'Spring 2027')]
const L = 'GET /universities/x/terms'

const far = t('9', 'Winter 2040') // beyond the first 200: only the search returns it

function Host({ more }: { more: boolean }) {
  const [v, setV] = useState('1')
  const [picked, setPicked] = useState<ReturnType<typeof t>>()
  return (
    <TermPicker
      slug="x"
      terms={all}
      more={more}
      picked={picked}
      value={v}
      onChange={(id, term) => {
        setV(id)
        setPicked(term)
      }}
    />
  )
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

it('lets a term beyond the loaded list be picked and stay selected after the search is cleared', async () => {
  mockApi({ [L]: () => ({ body: { items: [far] } }) })
  renderWithQuery(<Host more />)
  fireEvent.change(screen.getByLabelText('Search terms'), { target: { value: 'win' } })
  await screen.findByRole('option', { name: 'Winter 2040' })
  fireEvent.change(screen.getByLabelText('Term'), { target: { value: '9' } })
  fireEvent.change(screen.getByLabelText('Search terms'), { target: { value: '' } })
  await waitFor(() => expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual(['Winter 2040', 'Fall 2026', 'Spring 2027']))
  expect(screen.getByLabelText('Term')).toHaveValue('9')
})
