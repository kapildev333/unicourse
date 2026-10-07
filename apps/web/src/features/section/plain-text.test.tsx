import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { PlainText, countHint } from './plain-text'

it('keeps line breaks', () => {
  const { container } = render(<PlainText text={'a\nb'} />)
  expect(container.textContent).toBe('a\nb')
  expect(container.firstElementChild).toHaveClass('whitespace-pre-wrap')
})

it('links http(s) URLs only and leaves trailing punctuation out', () => {
  render(<PlainText text="see https://a.edu/x." />)
  const a = screen.getByRole('link', { name: 'https://a.edu/x' })
  expect(a).toHaveAttribute('href', 'https://a.edu/x')
  expect(a).toHaveAttribute('target', '_blank')
  expect(a).toHaveAttribute('rel', 'noopener noreferrer')
})

it('renders other markup as text', () => {
  const { container } = render(<PlainText text="javascript:alert(1) <b>x</b>" />)
  expect(screen.queryByRole('link')).toBeNull()
  expect(container.querySelector('b')).toBeNull()
})

it('countHint counts code points', () => {
  expect(countHint('abc', 10000)).toBe('3 / 10,000 characters')
  expect(countHint('x'.repeat(10012), 10000)).toBe('10,012 / 10,000 characters, 12 too many')
  expect(countHint('😀', 10)).toBe('1 / 10 characters')
})
