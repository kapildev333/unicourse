import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { ThemeSwitch } from './theme-switch'

afterEach(() => {
  localStorage.clear()
  document.documentElement.removeAttribute('data-theme')
})

it('switches to dark, applies and stores it', () => {
  render(<ThemeSwitch />)
  fireEvent.click(screen.getByRole('button', { name: 'Dark' }))
  expect(screen.getByRole('button', { name: 'Dark' })).toHaveAttribute('aria-pressed', 'true')
  expect(screen.getByRole('button', { name: 'System' })).toHaveAttribute('aria-pressed', 'false')
  expect(document.documentElement).toHaveAttribute('data-theme', 'dark')
  expect(localStorage.getItem('unicourse:theme')).toBe('dark')
})
