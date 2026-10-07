import { fireEvent, render, screen } from '@testing-library/react'
import { useState } from 'react'
import { expect, it, vi } from 'vitest'
import { SegmentedControl } from './segmented-control'

const opts = [
  { value: 'list', label: 'List' },
  { value: 'schedule', label: 'Schedule' },
]

it('renders pressed buttons; click and arrow keys call onChange', () => {
  const onChange = vi.fn()
  const Host = () => {
    const [v, setV] = useState('list')
    return <SegmentedControl label="View" options={opts} value={v} onChange={(x) => (setV(x), onChange(x))} />
  }
  render(<Host />)
  const list = screen.getByRole('button', { name: 'List' })
  const sched = screen.getByRole('button', { name: 'Schedule' })
  expect(list).toHaveAttribute('aria-pressed', 'true')
  expect(list).toHaveClass('bg-accent-soft', 'text-ink')
  expect(sched).toHaveAttribute('aria-pressed', 'false')
  fireEvent.keyDown(list, { key: 'ArrowRight' })
  expect(onChange).toHaveBeenLastCalledWith('schedule')
  expect(sched).toHaveAttribute('aria-pressed', 'true')
  expect(document.activeElement).toBe(sched)
  fireEvent.keyDown(sched, { key: 'ArrowLeft' })
  expect(onChange).toHaveBeenLastCalledWith('list')
  fireEvent.click(sched)
  expect(onChange).toHaveBeenLastCalledWith('schedule')
})
