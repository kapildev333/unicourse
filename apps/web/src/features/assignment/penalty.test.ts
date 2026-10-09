import { expect, it } from 'vitest'
import { penalised } from './penalty'

it('penalised rounds half up to 2 decimals', () => {
  expect(penalised(15, 20)).toBe('12.00')
  expect(penalised(12.5, 11)).toBe('11.13')
  expect(penalised(0.01, 50)).toBe('0.01')
  expect(penalised(20, 0)).toBe('20.00')
  expect(penalised(20, 100)).toBe('0.00')
})
