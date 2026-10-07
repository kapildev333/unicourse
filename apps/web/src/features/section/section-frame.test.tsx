import { screen } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { SectionFrame } from './section-frame'
import { renderSection } from './test-helpers'

afterEach(() => vi.unstubAllGlobals())

it('titles a failed section load "Couldn\'t load section" and offers a retry', async () => {
  mockApi({
    'GET /universities/a/sections/s1': () => ({ status: 500, body: { error: { code: 'internal', message: 'x', details: [] } } }),
    'GET /universities/a/me/sections': () => ({ body: { terms: [] } }),
  })
  renderSection({ path: '/a/sections/s1', ui: <SectionFrame slug="a" sectionId="s1">{() => null}</SectionFrame> })
  expect(await screen.findByText("Couldn't load section")).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Try again' })).toBeInTheDocument()
})
