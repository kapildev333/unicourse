import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { Button } from './button'

it('ghost-danger is muted at rest and red on hover and focus', () => {
  render(<Button variant="ghost-danger">Delete</Button>)
  expect(screen.getByRole('button')).toHaveClass('text-ink-muted', 'hover:text-danger', 'focus-visible:text-danger')
})

it('ghost-neutral is muted and darkens on hover', () => {
  render(<Button variant="ghost-neutral">Remove</Button>)
  expect(screen.getByRole('button')).toHaveClass('text-ink-muted', 'hover:text-ink', 'hover:bg-canvas')
})
