import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import { expect, type Browser, type BrowserContext, type Page } from '@playwright/test'
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
export async function latestMailLink(email: string, notBefore = 0, path = '/api/v1/auth/magic-link/'): Promise<string> {
  let found = ''
  const re = new RegExp(`https?://\\S+${path.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\S+`)
  await expect
    .poll(
      () => {
        const lines = fs.readFileSync(apiLog, 'utf8').split('\n')
        for (let i = lines.length - 1; i >= notBefore; i--) {
          if (!lines[i].includes('"msg":"mail"')) continue
          try {
            const m = JSON.parse(lines[i])
            const url = re.exec(m.text)?.[0]
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

export const logLines = () => fs.readFileSync(apiLog, 'utf8').split('\n').length - 1

// Signs in by magic link and waits until `${url.pathname}${url.search}` === landing.
// Navigates to /sign-in only when the page is not already showing its form, so `?next=` and `?email=` survive.
export async function signIn(page: Page, email: string, slug: string, landing = `/${slug}/me`) {
  const before = logLines()
  const field = page.getByLabel('University email or university ID')
  const sent = page.getByRole('heading', { name: 'Check your email' })
  if (new URL(page.url(), 'http://x').pathname === '/sign-in') await expect(field.or(sent)).toBeVisible()
  if (new URL(page.url(), 'http://x').pathname !== '/sign-in' || (await sent.isVisible())) await page.goto('/sign-in')
  await field.fill(email)
  await page.getByRole('button', { name: 'Email me a sign-in link' }).click()
  await expect(sent).toBeVisible()
  await page.goto(await latestMailLink(email, before))
  await page.getByRole('button', { name: 'Continue signing in' }).click()
  await page.waitForURL((u) => `${u.pathname}${u.search}` === landing)
}

export async function signOut(page: Page) {
  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page).toHaveURL(/\/sign-in/)
}

export const H = { 'Sec-Fetch-Site': 'same-origin' }

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- response shapes are asserted by the callers
export async function api(page: Page, method: 'get' | 'post' | 'put' | 'patch', url: string, data?: unknown): Promise<any> {
  const res = await page.request[method](url, { headers: H, data })
  expect(res.ok(), `${method} ${url}: ${res.status()} ${await res.text()}`).toBeTruthy()
  return res.status() === 204 ? null : res.json()
}

export async function upload(page: Page, url: string, csv: string, commit = true): Promise<void> {
  const file = { name: 'upload.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) }
  const pv = await page.request.post(`${url}/import/preview`, { headers: H, multipart: { file } })
  expect(pv.ok(), await pv.text()).toBeTruthy()
  if (!commit) return
  const { checksum } = await pv.json()
  const res = await page.request.post(`${url}/import`, { headers: H, multipart: { file, checksum } })
  expect(res.ok(), await res.text()).toBeTruthy()
}

export const year = new Date().getFullYear()
export const demoTerm = `Academic Year ${year}`
export const springTerm = `Spring ${year + 1}`

export type PersonaKey =
  | 'admin' | 'deptAdmin' | 'elena' | 'david' | 'sofia'
  | 'amara' | 'priya' | 'tomasz' | 'daniel' | 'meiLing' | 'jonas' | 'fatima' | 'lucas' | 'ingrid'
export type Persona = { email: string; name: string }

// Mirrors apps/api/internal/seed/demo.json; demo.spec.ts is the guard that it still matches.
const table: Record<PersonaKey, [string, string]> = {
  admin: ['registrar', 'Renata Quinn'],
  deptAdmin: ['owen.park', 'Owen Park'],
  elena: ['elena.marsh', 'Elena Marsh'],
  david: ['david.mensah', 'David Mensah'],
  sofia: ['sofia.alvarez', 'Sofia Alvarez'],
  amara: ['amara.okafor', 'Amara Okafor'],
  priya: ['priya.raman', 'Priya Raman'],
  tomasz: ['tomasz.zielinski', 'Tomasz Zielinski'],
  daniel: ['daniel.reyes', 'Daniel Reyes'],
  meiLing: ['mei-ling.chou', 'Mei-Ling Chou'],
  jonas: ['jonas.berg', 'Jonas Berg'],
  fatima: ['fatima.alsayed', 'Fatima Al-Sayed'],
  lucas: ['lucas.moreau', 'Lucas Moreau'],
  ingrid: ['ingrid.larsen', 'Ingrid Larsen'],
}

// Pure: the demo personas of `slug`. No I/O.
export function personas(slug: string): Record<PersonaKey, Persona> {
  return Object.fromEntries(
    Object.entries(table).map(([k, [local, name]]) => [k, { email: `${local}@${slug}.edu`, name }]),
  ) as Record<PersonaKey, Persona>
}

// Runs `unicourse-admin seed-demo --slug <slug>`; call from test.beforeAll only.
export function seedDemo(slug: string): Record<PersonaKey, Persona> {
  execFileSync(adminBin, ['seed-demo', '--slug', slug], { env: e2eEnv, stdio: ['ignore', 'ignore', 'inherit'] })
  return personas(slug)
}

const sessions = new Map<string, Awaited<ReturnType<BrowserContext['storageState']>>>()

// A page in a new context signed in as `email`; the session is cached per email in this worker.
// The caller closes page.context(). Never sign out on such a page.
export async function pageAs(browser: Browser, email: string, slug: string): Promise<Page> {
  let state = sessions.get(email)
  if (!state) {
    const ctx = await browser.newContext()
    await signIn(await ctx.newPage(), email, slug)
    state = await ctx.storageState()
    sessions.set(email, state)
    await ctx.close()
  }
  return (await browser.newContext({ storageState: state })).newPage()
}

export type Mail = { to: string; subject: string; text: string }

// Parsed `"msg":"mail"` lines of the API log from line index `notBefore` on; partial lines skipped.
// `mails(logLines())` taken before an action returns exactly the mails written after it.
export function mails(notBefore = 0): Mail[] {
  const out: Mail[] = []
  for (const line of fs.readFileSync(apiLog, 'utf8').split('\n').slice(notBefore)) {
    if (!line.includes('"msg":"mail"')) continue
    try {
      const m = JSON.parse(line)
      out.push({ to: m.to, subject: m.subject, text: m.text })
    } catch {
      // partial line
    }
  }
  return out
}

// `<course code>-<number>` to section id, every term.
export async function sectionIds(page: Page, slug: string): Promise<Record<string, string>> {
  const res = await api(page, 'get', `/api/v1/universities/${slug}/sections?limit=200`)
  return Object.fromEntries(res.items.map((s: { course: { code: string }; number: string; id: string }) => [`${s.course.code}-${s.number}`, s.id]))
}

export async function expectNoSideScroll(page: Page, what: string): Promise<void> {
  const w = await page.evaluate(() => ({ doc: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }))
  expect(w.doc, `${what}: page is ${w.doc} px wide at ${w.view} px`).toBeLessThanOrEqual(w.view)
}
