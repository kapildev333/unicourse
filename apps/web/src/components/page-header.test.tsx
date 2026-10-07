import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { PageHeader } from './page-header'

it('renders an h1 by default', () => {
  render(<PageHeader title="Admin" description="d" />)
  expect(screen.getByRole('heading', { level: 1, name: 'Admin' })).toBeInTheDocument()
})

it('as h2 renders the in-page tab heading (20 px bold, tighter rhythm)', () => {
  render(<PageHeader as="h2" title="Terms" description="d" action={<button>Go</button>} />)
  const h = screen.getByRole('heading', { level: 2, name: 'Terms' })
  expect(h).toHaveClass('text-xl', 'font-bold')
  expect(screen.queryByRole('heading', { level: 1 })).toBeNull()
  expect(h.parentElement?.parentElement).toHaveClass('mb-6')
  expect(screen.getByText('d')).toHaveClass('mt-1')
})
