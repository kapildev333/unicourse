import { expect, it } from 'vitest'
import { safeNext } from './safe-next'

it('keeps same-origin paths', () => {
  expect(safeNext('/a/b?x=1')).toBe('/a/b?x=1')
})

it('rejects everything else', () => {
  for (const s of ['//evil.com', '/\\evil.com', 'https://x', 'a/b', '/a\nb', '/' + 'a'.repeat(512), undefined, 5]) {
    expect(safeNext(s)).toBeUndefined()
  }
})
