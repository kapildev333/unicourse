import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { ListCount } from './list-count'

it('shows the bare count', () => {
  render(<ListCount shown={5} />)
  expect(screen.getByRole('status')).toHaveTextContent(/^Showing 5$/)
})

it('pluralises the noun', () => {
  const { rerender } = render(<ListCount shown={5} noun="section" />)
  expect(screen.getByRole('status')).toHaveTextContent(/^Showing 5 sections$/)
  rerender(<ListCount shown={1} noun="section" />)
  expect(screen.getByRole('status')).toHaveTextContent(/^Showing 1 section$/)
})
