import { screen, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi } from '../../test-utils'
import { SectionTabs } from './section-tabs'
import { mine, renderSection, sectionFixture } from './test-helpers'

afterEach(() => vi.unstubAllGlobals())

async function tabs(o: { userId?: string; role?: 'admin' | 'dept_admin' | 'member'; departmentIds?: string[]; mine?: ReturnType<typeof mine>; path?: string }) {
  mockApi({ 'GET /universities/a/me/sections': () => ({ body: o.mine ?? mine() }) })
  const path = o.path ?? '/a/sections/s1'
  renderSection({ path, ui: <SectionTabs slug="a" section={sectionFixture()} />, ...o })
  return path
}
const names = () =>
  within(screen.getByRole('navigation', { name: 'Section pages' })).getAllByRole('link').map((l) => l.textContent)

it('shows tabs per role', async () => {
  await tabs({ userId: 'u-ins' })
  await screen.findByRole('navigation', { name: 'Section pages' })
  expect(names()).toEqual(['Home', 'Announcements', 'Grades'])
})

it('shows a student Home and Announcements', async () => {
  await tabs({ mine: mine('student') })
  await screen.findByRole('navigation', { name: 'Section pages' })
  expect(names()).toEqual(['Home', 'Announcements'])
})

it('shows no strip to a plain member or another department admin', async () => {
  await tabs({})
  await new Promise((r) => setTimeout(r, 50))
  expect(screen.queryByRole('navigation', { name: 'Section pages' })).toBeNull()
})

it('hides the strip from a dept_admin of another department', async () => {
  await tabs({ role: 'dept_admin', departmentIds: ['d2'] })
  await new Promise((r) => setTimeout(r, 50))
  expect(screen.queryByRole('navigation', { name: 'Section pages' })).toBeNull()
})

it('marks the current tab', async () => {
  await tabs({ userId: 'u-ins', path: '/a/sections/s1/announcements' })
  const nav = await screen.findByRole('navigation', { name: 'Section pages' })
  expect(within(nav).getByRole('link', { name: 'Announcements' })).toHaveAttribute('aria-current', 'page')
  expect(within(nav).getByRole('link', { name: 'Home' })).not.toHaveAttribute('aria-current')
})

it('marks Home current on the section page', async () => {
  await tabs({ userId: 'u-ins' })
  expect(await screen.findByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page')
})
