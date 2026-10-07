import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { Table } from './table'

const body = (
  <tbody>
    <tr>
      <td data-label="Name">Ada</td>
    </tr>
  </tbody>
)

it('stack adds ui-table-stack and keeps data-label on cells', () => {
  render(<Table caption="People" stack>{body}</Table>)
  expect(screen.getByRole('table')).toHaveClass('ui-table', 'ui-table-stack')
  expect(screen.getByRole('cell', { name: 'Ada' })).toHaveAttribute('data-label', 'Name')
})

it('without stack there is no stack class; the scroll wrapper sits in a min-w-0 parent', () => {
  render(<Table caption="People">{body}</Table>)
  const table = screen.getByRole('table')
  expect(table).not.toHaveClass('ui-table-stack')
  expect(table.parentElement).toHaveClass('overflow-x-auto', 'relative')
  expect(table.parentElement!.parentElement).toHaveClass('min-w-0')
})
