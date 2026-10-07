import { render, screen } from '@testing-library/react'
import { StrictMode } from 'react'
import { expect, it } from 'vitest'
import { Dialog } from './dialog'

it('labels each dialog by its own title', () => {
  render(
    <>
      <Dialog open={false} onClose={() => {}} title="One">a</Dialog>
      <Dialog open={false} onClose={() => {}} title="Two">b</Dialog>
    </>,
  )
  const [d1, d2] = Array.from(document.querySelectorAll('dialog'))
  const id1 = d1.getAttribute('aria-labelledby')!
  const id2 = d2.getAttribute('aria-labelledby')!
  expect(id1).not.toBe(id2)
  expect(d1.querySelector(`[id="${id1}"]`)).toHaveTextContent('One')
  expect(d2.querySelector(`[id="${id2}"]`)).toHaveTextContent('Two')
  expect(screen.queryByRole('dialog')).toBeNull() // closed
})

it('centres itself: preflight zeroes the margin that centres a modal dialog', () => {
  render(<Dialog open={false} onClose={() => {}} title="T">a</Dialog>)
  expect(document.querySelector('dialog')).toHaveClass('m-auto')
})

it('size lg widens the dialog to max-w-2xl', () => {
  render(<Dialog open={false} onClose={() => {}} title="T" size="lg">a</Dialog>)
  expect(document.querySelector('dialog')).toHaveClass('max-w-2xl')
})

it('returns focus to the trigger when closed or unmounted', () => {
  HTMLDialogElement.prototype.showModal ||= function (this: HTMLDialogElement) {
    this.setAttribute('open', '')
  }
  HTMLDialogElement.prototype.close ||= function (this: HTMLDialogElement) {
    this.removeAttribute('open')
  }
  const { rerender, unmount } = render(
    <>
      <button>trigger</button>
      <Dialog open={false} onClose={() => {}} title="T">
        <input aria-label="x" />
      </Dialog>
    </>,
  )
  const trigger = screen.getByText('trigger')
  const dlg = (open: boolean) => (
    <>
      <button>trigger</button>
      <Dialog open={open} onClose={() => {}} title="T">
        <input aria-label="x" />
      </Dialog>
    </>
  )
  trigger.focus()
  rerender(dlg(true))
  ;(document.querySelector('input') as HTMLElement).focus()
  expect(document.activeElement).not.toBe(trigger)
  rerender(dlg(false))
  expect(document.activeElement).toBe(trigger)
  unmount()
})

it('returns focus to the trigger of a conditionally mounted dialog under StrictMode', () => {
  // like a browser, showModal moves focus into the dialog
  HTMLDialogElement.prototype.showModal = function (this: HTMLDialogElement) {
    this.setAttribute('open', '')
    this.querySelector('input')?.focus()
  }
  HTMLDialogElement.prototype.close = function (this: HTMLDialogElement) {
    this.removeAttribute('open')
  }
  const ui = (shown: boolean) => (
    <StrictMode>
      <button>trigger</button>
      {shown && (
        <Dialog open onClose={() => {}} title="T">
          <input aria-label="x" />
        </Dialog>
      )}
    </StrictMode>
  )
  const { rerender } = render(ui(false))
  const trigger = screen.getByText('trigger')
  trigger.focus()
  rerender(ui(true))
  rerender(ui(false))
  expect(document.activeElement).toBe(trigger)
})
