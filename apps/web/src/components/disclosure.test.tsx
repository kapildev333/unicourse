import { fireEvent, render } from '@testing-library/react'
import { expect, it } from 'vitest'
import { Disclosure } from './disclosure'

it('swaps the chevron with the open state; icon is aria-hidden; summary is list-none', () => {
  const { container } = render(<Disclosure summary="Details">body</Disclosure>)
  const d = container.querySelector('details')!
  const summary = container.querySelector('summary')!
  expect(summary).toHaveClass('list-none')
  expect(summary.querySelector('svg')).toHaveAttribute('aria-hidden', 'true')
  expect(summary.querySelector('svg')).toHaveClass('lucide-chevron-right')
  d.open = true
  fireEvent(d, new Event('toggle'))
  expect(summary.querySelector('svg')).toHaveClass('lucide-chevron-down')
  expect(summary.querySelector('svg')).not.toHaveClass('rotate-90')
  d.open = false
  fireEvent(d, new Event('toggle'))
  expect(summary.querySelector('svg')).toHaveClass('lucide-chevron-right')
})
