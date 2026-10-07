import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { LoadMore } from './load-more'

const p = { hasNextPage: false, isFetchingNextPage: false, onClick: () => {}, shown: 3 }

it('shows the count by default and omits it with showCount={false}', () => {
  const { rerender } = render(<LoadMore {...p} />)
  expect(screen.getByRole('status')).toHaveTextContent('Showing 3')
  rerender(<LoadMore {...p} showCount={false} />)
  expect(screen.queryByRole('status')).toBeNull()
})
