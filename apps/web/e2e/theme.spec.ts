import { expect, test } from '@playwright/test'
import { createUniversity, runId, signIn, signOut } from './helpers'

const slug = `et-${runId}`
const domain = `t-${runId}.edu`
const admin = `root@${domain}`

test('the dark theme survives a reload and a sign-out', async ({ page }) => {
  createUniversity({ name: `Uni T ${runId}`, slug, domain, adminEmail: admin })
  await signIn(page, admin, slug)
  const html = page.locator('html')
  await expect(html).not.toHaveAttribute('data-theme') // System by default
  await page.getByRole('group', { name: 'Theme' }).getByRole('button', { name: 'Dark' }).click()
  await expect(html).toHaveAttribute('data-theme', 'dark')
  await page.reload()
  await expect(html).toHaveAttribute('data-theme', 'dark') // set by /theme-boot.js, not React
  await expect(page.getByRole('button', { name: 'Dark' })).toHaveAttribute('aria-pressed', 'true')
  await signOut(page)
  await expect(html).toHaveAttribute('data-theme', 'dark')
})
