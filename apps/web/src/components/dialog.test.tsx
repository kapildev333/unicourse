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
