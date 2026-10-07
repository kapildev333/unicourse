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

it('never uses a middle dot: counts read as plain words, the dot stays for two-part labels', () => {
  render(<ListCount shown={3} noun="change" total={9} />)
  expect(screen.getByRole('status').textContent).not.toContain('·')
})

it('shows the total when the API gives one', () => {
  render(<ListCount shown={50} noun="change" total={120} />)
  expect(screen.getByRole('status')).toHaveTextContent(/^Showing 50 of 120 changes$/)
})
