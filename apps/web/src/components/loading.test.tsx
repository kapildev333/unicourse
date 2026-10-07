import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { Loading } from './loading'

it('default is three bars with the label', () => {
  const { container } = render(<Loading />)
  expect(screen.getByRole('status')).toHaveTextContent('Loading…')
  expect(container.querySelectorAll('.ui-skeleton')).toHaveLength(3)
})

it('table variant shows the column names and 5 rows of bars', () => {
  const { container } = render(<Loading variant="table" columns={['Name', 'Role']} />)
  expect(screen.getAllByRole('columnheader', { hidden: true }).map((h) => h.textContent)).toEqual(['Name', 'Role'])
  expect(container.querySelectorAll('tbody tr')).toHaveLength(5)
  expect(container.querySelectorAll('tbody .ui-skeleton')).toHaveLength(10)
  expect(screen.getByRole('status')).toHaveTextContent('Loading…')
})

it('cards variant shows 3 card blocks', () => {
  const { container } = render(<Loading variant="cards" />)
  expect(container.querySelectorAll('[data-skeleton-card]')).toHaveLength(3)
  expect(screen.getByRole('status')).toHaveTextContent('Loading…')
})
