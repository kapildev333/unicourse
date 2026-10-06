import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { FormField } from './form-field'

it('associates the label with the input', () => {
  render(<FormField label="Name" name="name"><input /></FormField>)
  expect(screen.getByLabelText('Name')).toBeInTheDocument()
})

it('marks the input invalid and describes it with the error', () => {
  render(<FormField label="Name" name="name" error="required"><input /></FormField>)
  const input = screen.getByLabelText('Name')
  expect(input).toHaveAttribute('aria-invalid', 'true')
  expect(input).toHaveAccessibleDescription('required')
})
