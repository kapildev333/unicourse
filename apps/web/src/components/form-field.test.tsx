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
  expect(input).toHaveAccessibleDescription(/required/)
})

it('merges an existing aria-describedby', () => {
  render(<FormField label="Name" name="name" error="required"><input aria-describedby="hint" /></FormField>)
  const input = screen.getByLabelText('Name')
  expect(input.getAttribute('aria-describedby')).toMatch(/^hint /)
  expect(input).toHaveAccessibleDescription(/required/)
})

it('keeps ids unique across fields sharing a name', () => {
  render(<><FormField label="A" name="x"><input /></FormField><FormField label="B" name="x"><input /></FormField></>)
  expect(screen.getByLabelText('A')).not.toBe(screen.getByLabelText('B'))
})

it('puts the hint id before the error id', () => {
  render(<FormField label="Code" name="code" hint="Two letters" error="bad"><input /></FormField>)
  const input = screen.getByLabelText('Code')
  const ids = input.getAttribute('aria-describedby')!.split(' ')
  expect(ids).toHaveLength(2)
  expect(document.getElementById(ids[0])).toHaveTextContent('Two letters')
  expect(document.getElementById(ids[1])).toHaveTextContent('bad')
})
