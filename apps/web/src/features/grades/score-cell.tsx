import { useId } from 'react'

export type CellStatus = 'idle' | 'pending' | 'saved' | 'error'
export type Cell = {
  draft: string
  sent?: string
  status: CellStatus
  message?: string
}

const cellInput =
  'h-8 w-20 rounded border border-line-strong bg-surface px-2 text-right text-sm text-ink focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-focus aria-[invalid=true]:border-danger'

/** Focus and select the input at (row, col) inside `table`, if any. */
export function focusCell(table: Element | null, row: number, col: number) {
  const el = table?.querySelector<HTMLInputElement>(`input[data-row="${row}"][data-col="${col}"]`)
  el?.focus()
  el?.select()
}

export function ScoreCell(props: {
  label: string
  row: number
  col: number
  value: string
  status: CellStatus
  message?: string
  missing?: boolean
  onChange: (text: string) => void
  onCommit: () => void
  onRevert: () => void
  onPasteLines: (lines: string[]) => void
}) {
  const { label, row, col, value, status, message, missing, onChange, onCommit, onRevert, onPasteLines } = props
  const msgId = useId()

  function onKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    const el = e.currentTarget
    const table = el.closest('table')
    const go = (r: number, c: number) => {
      e.preventDefault()
      focusCell(table, r, c)
    }
    const collapsed = el.selectionStart === el.selectionEnd
    if (e.key === 'Enter') go(row + (e.shiftKey ? -1 : 1), col)
    else if (e.key === 'ArrowDown') go(row + 1, col)
    else if (e.key === 'ArrowUp') go(row - 1, col)
    else if (e.key === 'ArrowLeft' && collapsed && el.selectionStart === 0) go(row, col - 1)
    else if (e.key === 'ArrowRight' && collapsed && el.selectionStart === el.value.length) go(row, col + 1)
    else if (e.key === 'Escape') onRevert()
  }

  function onPaste(e: React.ClipboardEvent<HTMLInputElement>) {
    const lines = e.clipboardData.getData('text').split(/\r?\n/)
    if (lines[lines.length - 1] === '') lines.pop()
    if (lines.length < 2) return
    e.preventDefault()
    onPasteLines(lines.map((l) => l.split('\t')[0].trim()))
  }

  return (
    <td className="text-right">
      <input
        type="text"
        inputMode="decimal"
        autoComplete="off"
        data-row={row}
        data-col={col}
        aria-label={label}
        aria-invalid={status === 'error' || undefined}
        aria-describedby={status === 'error' ? msgId : undefined}
        className={cellInput}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onFocus={(e) => e.currentTarget.select()}
        onBlur={onCommit}
        onKeyDown={onKeyDown}
        onPaste={onPaste}
      />
      {missing && status === 'idle' && !value && <p className="mt-1 text-xs text-ink-muted">Missing</p>}
      {status === 'pending' && <p className="mt-1 text-xs text-ink-muted">Saving…</p>}
      {status === 'saved' && <p className="mt-1 text-xs text-success">Saved</p>}
      {status === 'error' && (
        <p id={msgId} className="mt-1 ml-auto max-w-40 text-xs text-danger">
          {message}
        </p>
      )}
    </td>
  )
}
