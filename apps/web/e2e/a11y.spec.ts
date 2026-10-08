import AxeBuilder from '@axe-core/playwright'
import { expect, test, type Locator, type Page } from '@playwright/test'
import { pageAs, personas, runId, sectionIds, seedDemo } from './helpers'

const slug = `e2e-a11y-${runId}`
const csv = { name: 'roster.csv', mimeType: 'text/csv', buffer: Buffer.from('email,displayName\nnew.student@' + slug + '.edu,New Student\n') }
const P = personas(slug)

test.beforeAll(() => {
  seedDemo(slug)
})

const h1 = (page: Page) => page.getByRole('heading', { level: 1 }).first()
const dialog = (page: Page, name: string) => page.getByRole('dialog', { name })

// Scans the current state of `page` in both colour schemes; `ready` is the level-1 heading or the named dialog.
async function scan(page: Page, role: string, pageName: string, ready: Locator = h1(page)) {
  for (const colorScheme of ['light', 'dark'] as const) {
    await page.emulateMedia({ colorScheme })
    await expect(ready).toBeVisible()
    await page.evaluate(() => Promise.all(document.getAnimations().map((a) => a.finished.catch(() => {})))) // colour transitions after the scheme switch
    const r = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze()
    const bad = r.violations.filter((v) => v.impact === 'serious' || v.impact === 'critical')
    expect(bad.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`), `${role}, ${pageName}, ${colorScheme}`).toEqual([])
  }
}

// One step per page: `go` brings the page to the state to scan.
async function visit(page: Page, role: string, pageName: string, go: () => Promise<unknown>, ready?: Locator) {
  await test.step(pageName, async () => {
    await go()
    await scan(page, role, pageName, ready)
  })
}

async function asPersona(browser: Parameters<typeof pageAs>[0], email: string, body: (page: Page) => Promise<void>) {
  const page = await pageAs(browser, email, slug)
  try {
    await body(page)
  } finally {
    await page.context().close()
  }
}

test('E-A11Y-01 signed out', async ({ page }) => {
  const role = 'signed out'
  await visit(page, role, '/sign-in', () => page.goto('/sign-in'))
  await visit(page, role, 'Check your email', async () => {
    await page.goto('/sign-in')
    await page.getByLabel('University email or university ID').fill(`nobody@unknown-${runId}.test`)
    await page.getByRole('button', { name: 'Email me a sign-in link' }).click()
  }, page.getByRole('heading', { name: 'Check your email' }))
  await visit(page, role, '/sign-in?error=forbidden', () => page.goto('/sign-in?error=forbidden'))
  await visit(page, role, 'API confirm page', () => page.goto('/api/v1/auth/magic-link/not-a-token'))
})

test('E-A11Y-02 student', async ({ browser }) => {
  const role = 'student'
  await asPersona(browser, P.amara.email, async (page) => {
    const ids = await sectionIds(page, slug)
    const s = `/${slug}/sections/${ids['CS101-001']}`
    await visit(page, role, '/me list', () => page.goto(`/${slug}/me`))
    await visit(page, role, '/me schedule', () => page.goto(`/${slug}/me?view=schedule`))
    await visit(page, role, '/catalog', () => page.goto(`/${slug}/catalog`))
    await visit(page, role, '/catalog no results', () => page.goto(`/${slug}/catalog?q=zzzz`))
    await visit(page, role, 'CS101 Home', () => page.goto(s))
    await visit(page, role, 'CS101 Announcements', () => page.goto(`${s}/announcements`))
    await visit(page, role, 'CS101 Materials', () => page.goto(`${s}/materials`))
    await visit(page, role, 'CS101 Participants', () => page.goto(`${s}/participants`))
    await visit(page, role, '/grades', () => page.goto(`/${slug}/grades`))
    await visit(page, role, '/grade-scale', () => page.goto(`/${slug}/grade-scale`))
    await visit(page, role, 'no access', () => page.goto(`/nope-${runId}/me`))
    await visit(page, role, 'dialog Your account', async () => {
      await page.goto(`/${slug}/me`)
      await page.getByRole('button', { name: /Account/ }).click()
    }, dialog(page, 'Your account'))
  })
})

test('E-A11Y-03 instructor', async ({ browser }) => {
  const role = 'instructor'
  await asPersona(browser, P.elena.email, async (page) => {
    const ids = await sectionIds(page, slug)
    const grades = (code: string) => `/${slug}/sections/${ids[code]}/grades`
    await visit(page, role, '/me', () => page.goto(`/${slug}/me`))
    await visit(page, role, 'CS210 gradebook', () => page.goto(grades('CS210-001')))
    await visit(page, role, 'dialog Edit Problem Set Average', async () => {
      await page.goto(grades('CS210-001'))
      await page.getByRole('button', { name: 'Edit Problem Set Average' }).click()
    }, dialog(page, 'Edit Problem Set Average'))
    await visit(page, role, 'dialog Finalize grades for CS210-001', async () => {
      await page.goto(grades('CS210-001'))
      await page.getByRole('button', { name: 'Finalize grades' }).click()
    }, dialog(page, 'Finalize grades for CS210-001'))
    await visit(page, role, 'CS101 gradebook (finalized)', () => page.goto(grades('CS101-001')))
    await visit(page, role, 'CS101 Materials, dialog Add week', async () => {
      await page.goto(`/${slug}/sections/${ids['CS101-001']}/materials`)
      await page.getByRole('button', { name: 'Add week' }).first().click()
    }, dialog(page, 'Add week'))
  })
})

test('E-A11Y-04 TA', async ({ browser }) => {
  const role = 'TA'
  await asPersona(browser, P.sofia.email, async (page) => {
    const ids = await sectionIds(page, slug)
    const s = `/${slug}/sections/${ids['CS101-001']}`
    await visit(page, role, 'CS101 gradebook', () => page.goto(`${s}/grades`))
    await visit(page, role, 'CS101 Announcements, dialog New announcement', async () => {
      await page.goto(`${s}/announcements`)
      await page.getByRole('button', { name: 'New announcement' }).first().click()
    }, dialog(page, 'New announcement'))
  })
})

test('E-A11Y-05 admin', async ({ browser }) => {
  const role = 'admin'
  await asPersona(browser, P.admin.email, async (page) => {
    const ids = await sectionIds(page, slug)
    for (const t of ['departments', 'terms', 'members', 'invitations']) {
      await visit(page, role, `/admin/${t}`, () => page.goto(`/${slug}/admin/${t}`))
    }
    await visit(page, role, '/admin/audit, Show changes open', async () => {
      await page.goto(`/${slug}/admin/audit`)
      await page.getByText('Show changes').first().click()
    })
    await visit(page, role, '/courses', () => page.goto(`/${slug}/courses`))
    await visit(page, role, 'dialog New course', async () => {
      await page.goto(`/${slug}/courses`)
      await page.getByRole('button', { name: 'New course' }).first().click()
    }, dialog(page, 'New course'))
    await visit(page, role, '/courses/sections', () => page.goto(`/${slug}/courses/sections`))
    await visit(page, role, 'dialog New section, one meeting row', async () => {
      await page.goto(`/${slug}/courses/sections`)
      await page.getByRole('button', { name: 'New section' }).first().click()
      await dialog(page, 'New section').getByRole('button', { name: 'Add meeting time' }).click()
    }, dialog(page, 'New section'))
    await visit(page, role, 'CS101 Participants, dialog Import roster with a CSV', async () => {
      await page.goto(`/${slug}/sections/${ids['CS101-001']}/participants`)
      await page.getByRole('button', { name: 'Import roster' }).first().click()
      await dialog(page, 'Import roster').getByLabel('CSV file').setInputFiles(csv)
      await expect(dialog(page, 'Import roster').getByText(/to invite/)).toBeVisible()
    }, dialog(page, 'Import roster'))
    await visit(page, role, '/grade-scale', () => page.goto(`/${slug}/grade-scale`))
  })
})

test('E-A11Y-06 department admin', async ({ browser }) => {
  const role = 'department admin'
  await asPersona(browser, P.deptAdmin.email, async (page) => {
    await visit(page, role, 'dialog Invite someone', async () => {
      await page.goto(`/${slug}/admin/invitations`)
      await page.getByRole('button', { name: 'Invite someone' }).first().click()
    }, dialog(page, 'Invite someone'))
    await visit(page, role, '/courses', () => page.goto(`/${slug}/courses`))
  })
})
