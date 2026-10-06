import { render, screen } from '@testing-library/react'
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
