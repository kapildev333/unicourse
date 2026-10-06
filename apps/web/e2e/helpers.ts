import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import { expect, type Page } from '@playwright/test'
import { adminBin, apiLog, e2eEnv } from './env'

export const runId = Date.now().toString(36)

export function createUniversity(o: { name: string; slug: string; domain: string; adminEmail: string }): string {
  const out = execFileSync(
    adminBin,
    ['create-university', '--name', o.name, '--slug', o.slug, '--domains', o.domain, '--timezone', 'UTC', '--admin-email', o.adminEmail],
    { env: e2eEnv, encoding: 'utf8' },
  )
  return out.trim().split('\n').pop()!
}

// Newest sign-in link mailed to `email` (dev mail is one JSON log line per message), polled until it appears.
export async function latestMailLink(email: string, notBefore = 0): Promise<string> {
  let found = ''
  await expect
    .poll(
      () => {
        const lines = fs.readFileSync(apiLog, 'utf8').split('\n')
        for (let i = lines.length - 1; i >= notBefore; i--) {
          if (!lines[i].includes('"msg":"mail"')) continue
          try {
            const m = JSON.parse(lines[i])
            const url = /https?:\/\/\S+\/api\/v1\/auth\/magic-link\/\S+/.exec(m.text)?.[0]
            if (m.to === email && url) return (found = url)
          } catch {
            // partial line
          }
        }
        return ''
      },
      { timeout: 15_000 },
    )
    .not.toBe('')
  return found
}

export const logLines = () => fs.readFileSync(apiLog, 'utf8').split('\n').length

export async function signIn(page: Page, email: string, slug: string) {
  const before = logLines()
  await page.goto('/sign-in')
  await page.getByLabel('Email', { exact: true }).fill(email)
  await page.getByRole('button', { name: 'Email me a sign-in link' }).click()
  await expect(page.getByRole('heading', { name: 'Check your email' })).toBeVisible()
  await page.goto(await latestMailLink(email, before - 1))
  await page.getByRole('button', { name: 'Continue signing in' }).click()
  await page.waitForURL(new RegExp(`/${slug}/me$`))
}

export async function signOut(page: Page) {
  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page).toHaveURL(/\/sign-in/)
}
