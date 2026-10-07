import { fireEvent, render, screen } from '@testing-library/react'
import { expect, it, vi } from 'vitest'
import { ScoreCell } from './score-cell'

function setup(over: Partial<Parameters<typeof ScoreCell>[0]> = {}) {
  const p = { onChange: vi.fn(), onCommit: vi.fn(), onRevert: vi.fn(), onPasteLines: vi.fn() }
  render(
    <table>
      <tbody>
        <tr>
          <ScoreCell label="Score for A, T" row={0} col={0} value="12" status="idle" {...p} {...over} />
          <ScoreCell label="Other" row={0} col={1} value="" status="idle" {...p} />
        </tr>
      </tbody>
    </table>,
  )
  return { ...p, input: screen.getByLabelText(over.label ?? 'Score for A, T') as HTMLInputElement }
}

it('ArrowRight moves only with the caret at the end', () => {
  const { input } = setup()
  input.focus()
  input.setSelectionRange(0, 0)
  fireEvent.keyDown(input, { key: 'ArrowRight' })
  expect(input).toHaveFocus()
  input.setSelectionRange(2, 2)
  fireEvent.keyDown(input, { key: 'ArrowRight' })
  expect(screen.getByLabelText('Other')).toHaveFocus()
})

it('Escape reverts and blur commits', () => {
  const { input, onRevert, onCommit } = setup()
  fireEvent.keyDown(input, { key: 'Escape' })
  fireEvent.blur(input)
  expect(onRevert).toHaveBeenCalled()
  expect(onCommit).toHaveBeenCalled()
})

it('paste: multi-line takes the first column; a single value is left to the browser', () => {
  const { input, onPasteLines } = setup()
  fireEvent.paste(input, { clipboardData: { getData: () => '1\tx\n2\ty\n' } })
  expect(onPasteLines).toHaveBeenCalledWith(['1', '2'])
  onPasteLines.mockClear()
  fireEvent.paste(input, { clipboardData: { getData: () => '7\n' } })
  expect(onPasteLines).not.toHaveBeenCalled()
})

it('error state links the message and marks the input invalid', () => {
  const { input } = setup({ status: 'error', message: 'Nope' })
  expect(input).toHaveAttribute('aria-invalid', 'true')
  expect(input).toHaveAccessibleDescription('Nope')
})
