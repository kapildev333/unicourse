// Screen capture for the UI/UX benchmark (not part of `npm run e2e`): `npm run screens`.
// Output: e2e/.tmp/screens/*.png and index.md. Dark/light via emulated prefers-color-scheme (theme stays "System").
import fs from 'node:fs'
import path from 'node:path'
import { expect, test, type Page } from '@playwright/test'
import { tmpDir } from './env'
import { api, createUniversity, H, personas, runId, sectionIds, seedDemo, signIn } from './helpers'

const slug = `northfield-${runId}`
const dom = `northfield-${runId}.edu`
const A = `/api/v1/universities/${slug}`
const out = path.join(tmpDir, 'screens')
const index: string[] = []

async function settle(page: Page) {
  await expect(page.locator('h1').first()).toBeVisible()
  await page.waitForLoadState('networkidle')
  await expect(page.locator('.ui-skeleton')).toHaveCount(0)
}

// Four captures (1440/390 x light/dark); dialogs skip 390 dark. `base` is everything before the theme.
async function snap(page: Page, base: string, desc: string, o: { dialog?: boolean; noSettle?: boolean } = {}) {
  if (!o.noSettle) await settle(page)
  for (const w of [1440, 390])
    for (const scheme of ['light', 'dark'] as const) {
      if (o.dialog && w === 390 && scheme === 'dark') continue
      await page.setViewportSize({ width: w, height: w === 1440 ? 900 : 844 })
      await page.emulateMedia({ colorScheme: scheme })
      await page.waitForTimeout(200)
      const file = `${base}-${scheme}-${w}.png`
      await page.screenshot({ path: path.join(out, file), fullPage: true })
      index.push(`${file} - ${desc}`)
      if (w === 390) {
        const width = await page.evaluate(() => document.documentElement.scrollWidth)
        expect.soft(width, file).toBeLessThanOrEqual(390)
        if (width > 390) index.push(`WIDE ${file} ${width}`)
      }
    }
  await page.setViewportSize({ width: 1440, height: 900 }) // leave the page desktop-sized for the next action
  await page.emulateMedia({ colorScheme: 'light' })
}

const P = personas(slug)
const people = [P.amara, P.priya, P.tomasz, P.daniel, P.meiLing, P.jonas, P.fatima, P.lucas]
const prof = P.elena.email
const ta = P.sofia.email
const newHire = P.ingrid.email

const esc = (t: string) => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const csv = (rows: string[][]) => ['email,displayName', ...rows.map((r) => r.join(','))].join('\n') + '\n'

test('capture every screen', async ({ browser, page }) => {
  test.setTimeout(1_200_000)
  fs.mkdirSync(out, { recursive: true })
  const ctxs = async (email: string, uni = slug) => {
    const c = await browser.newContext()
    const p = await c.newPage()
    await signIn(p, email, uni)
    return p
  }

  await test.step('signed-out auth screens', async () => {
    await page.goto('/sign-in')
    await snap(page, '01-auth-sign-in-signedout', 'Sign-in page, nobody signed in')
    await page.getByLabel('University email or university ID').fill(`amara.okafor@${dom}`)
    await page.getByRole('button', { name: 'Email me a sign-in link' }).click()
    await expect(page.getByRole('heading', { name: 'Check your email' })).toBeVisible()
    await snap(page, '02-auth-check-email-signedout', 'Magic link sent, "Check your email", nobody signed in')
  })

  const dep: Record<string, string> = {}
  const sec: Record<string, string> = {}
  await test.step('empty states (a university with nothing in it), then the demo world', async () => {
    const blank = `blank-${runId}`
    createUniversity({ name: 'Northfield University', slug: blank, domain: `${blank}.edu`, adminEmail: `registrar@${blank}.edu` })
    const ep = await ctxs(`registrar@${blank}.edu`, blank)
    await snap(ep, '10-my-sections-empty-admin', 'My sections with nothing yet, admin signed in')
    await ep.goto(`/${blank}/catalog`)
    await snap(ep, '11-catalog-empty-admin', 'Catalog before any term exists, admin signed in')
    await ep.goto(`/${blank}/courses`)
    await snap(ep, '12-courses-empty-admin', 'Courses list with no courses, admin signed in')
    await ep.goto(`/${blank}/courses/sections`)
    await snap(ep, '13-sections-empty-admin', 'Sections list with no sections, admin signed in')
    await ep.context().close()

    // the demo world (Spec 7), then what the captures need beyond it: CS101 open, as the older seed left it
    seedDemo(slug)
    await signIn(page, P.admin.email, slug)
    const ids = await sectionIds(page, slug)
    for (const code of ['CS101', 'CS210', 'MATH201', 'ECON110', 'BIO150']) sec[code] = ids[`${code}-001`]
    const res = await page.request.delete(`${A}/sections/${sec.CS101}/final-grades`, { headers: H })
    expect(res.ok(), await res.text()).toBeTruthy()
    const depList = (await api(page, 'get', `${A}/departments`)).items as { id: string; code: string }[]
    for (const d of depList) dep[d.code] = d.id
  })

  await test.step('invite link while signed out', async () => {
    const c = await browser.newContext()
    const p = await c.newPage()
    // invitations are not mailed (P1-L1 §8) and the web ignores the token, so the link shape is built by hand
    await p.goto(`/invite/${runId}?email=${encodeURIComponent(newHire)}`)
    await expect(p.getByLabel('University email or university ID')).toHaveValue(newHire)
    await snap(p, '03-auth-invite-link-signedout', `Invitation link for ${newHire} opened signed out (lands on sign-in with the email filled)`)
    await c.close()
  })

  const profPage = await ctxs(prof)
  const taPage = await ctxs(ta)
  const stu = await ctxs(people[0].email)
  for (const p of people.slice(1)) await (await ctxs(p.email)).context().close()
  // ingrid.larsen is never signed in, so Invitations keeps one pending row

  await test.step('student and instructor home', async () => {
    await stu.goto(`/${slug}/me`)
    await snap(stu, '20-my-sections-list-student', 'My sections as student Amara Okafor, list view')
    await stu.getByRole('button', { name: 'Schedule' }).click()
    await expect(stu.getByRole('region', { name: 'Monday' })).toBeVisible()
    await snap(stu, '21-my-sections-schedule-student', 'My sections as student, weekly schedule view')
    await profPage.goto(`/${slug}/me`)
    await snap(profPage, '22-my-sections-list-instructor', 'My sections as instructor Elena Marsh')
    await profPage.getByRole('button', { name: 'Schedule' }).click()
    await expect(profPage.getByRole('region', { name: 'Monday' })).toBeVisible()
    await snap(profPage, '23-my-sections-schedule-instructor', 'My sections as instructor, weekly schedule view')
    await stu.goto(`/${slug}/catalog`)
    await snap(stu, '30-catalog-results-student', 'Catalog with section cards, student signed in')
    await stu.goto(`/${slug}/catalog?departmentId=${dep.CS}`)
    await snap(stu, '31-catalog-filter-student', 'Catalog filtered to Computer Science, student signed in')
    await stu.goto(`/${slug}/catalog?q=zzzz`)
    await snap(stu, '32-catalog-empty-result-student', 'Catalog search with no match, student signed in')
  })

  await test.step('courses, sections, dialogs (admin)', async () => {
    await page.goto(`/${slug}/courses`)
    await snap(page, '40-courses-list-admin', 'Courses list, admin signed in')
    await page.getByRole('button', { name: 'Edit CS101' }).click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await snap(page, '41-courses-course-dialog-admin', 'Edit course dialog open (CS101), admin', { dialog: true })
    await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
    await page.getByRole('button', { name: 'Delete BIO150' }).click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await snap(page, '42-courses-remove-dialog-admin', 'Remove course confirmation dialog open, admin', { dialog: true })
    await page.getByRole('button', { name: 'Cancel' }).click()

    await page.goto(`/${slug}/courses/sections`)
    await snap(page, '43-sections-list-admin', 'Sections list, admin signed in')
    await page.getByRole('button', { name: 'New section' }).first().click()
    const d = page.getByRole('dialog')
    await d.getByLabel('Course').selectOption({ label: 'CS210 · Data Structures and Algorithms' })
    await d.getByLabel('Section number').fill('002')
    await d.getByRole('button', { name: 'Add meeting time' }).click()
    const row = d.getByRole('group', { name: 'Meeting 1' })
    await row.getByLabel('Day').selectOption({ label: 'Friday' })
    await row.getByLabel('Starts').fill('14:00')
    await row.getByLabel('Ends').fill('15:20')
    await row.getByLabel('Location').fill('Engineering Building 215')
    await snap(page, '44-sections-section-dialog-admin', 'New section dialog with one meeting row added, admin', { dialog: true })
    await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()

    await page.goto(`/${slug}/sections/${sec.CS101}`)
    await snap(page, '45-section-detail-admin', 'Section Home CS101 (title band, description, meeting times), admin')
    await stu.goto(`/${slug}/sections/${sec.CS101}`)
    await snap(stu, '45a-section-home-student', 'Section Home CS101 as student Amara Okafor')
    await stu.goto(`/${slug}/sections/${sec.CS101}/materials`)
    await snap(stu, '45b-section-materials-student', 'Section Materials CS101 (weeks and files) as student Amara Okafor')
    await page.goto(`/${slug}/sections/${sec.CS101}/participants`)
    await snap(page, '45c-section-participants-admin', 'Section Participants CS101 (staff, roster), admin')
    await page.getByRole('button', { name: 'Import roster' }).first().click()
    await page.getByRole('dialog').getByLabel('CSV file').setInputFiles({
      name: 'roster.csv', mimeType: 'text/csv',
      buffer: Buffer.from(csv([[people[0].email, people[0].name], [people[1].email, people[1].name], [`hannah.lindqvist@${dom}`, 'Hannah Lindqvist'], [`omar.haddad@${dom}`, 'Omar Haddad']])),
    })
    await expect(page.getByRole('dialog').getByRole('button', { name: /^Import \d+ row/ })).toBeVisible()
    await snap(page, '46-section-csv-import-dialog-admin', 'Roster CSV import dialog with a preview, admin', { dialog: true })
    await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
    await page.getByRole('button', { name: new RegExp(`^Remove (${[people[7].name, people[7].email].map(esc).join('|')}) from the roster$`) }).click()
    await expect(page.getByRole('dialog')).toBeVisible()
    await snap(page, '47-section-remove-dialog-admin', 'Remove from roster confirmation dialog open, admin', { dialog: true })
    await page.getByRole('button', { name: 'Cancel' }).click()
  })

  await test.step('admin pages', async () => {
    for (const [n, p, desc] of [
      ['50', 'departments', 'Departments'], ['51', 'terms', 'Terms'], ['52', 'members', 'Members'],
      ['53', 'invitations', 'Invitations (some pending)'], ['54', 'audit', 'Audit log'],
    ])
      {
        await page.goto(`/${slug}/admin/${p}`)
        await snap(page, `${n}-admin-${p}-admin`, `${desc}, admin signed in`)
      }
  })

  await test.step('grading', async () => {
    const g = (id: string) => `/${slug}/sections/${id}/grades`
    await profPage.goto(g(sec.CS101))
    await snap(profPage, '60-grades-gradebook-instructor', 'Gradebook CS101 before finalizing, instructor')
    await profPage.getByRole('button', { name: 'Edit Midterm Exam' }).click()
    await expect(profPage.getByRole('dialog')).toBeVisible()
    await snap(profPage, '61-grades-assessment-dialog-instructor', 'Assessment dialog open (Midterm Exam), instructor', { dialog: true })
    await profPage.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
    await profPage.getByRole('button', { name: 'Finalize grades' }).click()
    await expect(profPage.getByRole('dialog')).toBeVisible()
    await snap(profPage, '62-grades-finalize-preview-instructor', 'Finalize grades preview dialog open, instructor', { dialog: true })
    await profPage.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()
    for (const code of ['CS101', 'MATH201']) await api(profPage, 'post', `${A}/sections/${sec[code]}/final-grades`, { overrides: [] })
    await profPage.goto(g(sec.CS101))
    await expect(profPage.getByRole('button', { name: 'Reopen grades' })).toBeVisible()
    await snap(profPage, '63-grades-gradebook-finalized-instructor', 'Gradebook CS101 after finalizing, instructor')
    await taPage.goto(g(sec.CS101))
    await snap(taPage, '64-grades-gradebook-ta', 'Gradebook CS101 as teaching assistant Sofia Alvarez')
    await stu.goto(`/${slug}/grades`)
    await snap(stu, '65-grades-my-grades-student', 'My grades with letters and GPA, student Amara Okafor')
    await page.goto(`/${slug}/grade-scale`)
    await snap(page, '66-grades-grade-scale-admin', 'Grade scale editor, admin signed in')
  })

  await test.step('loading skeleton and error banner', async () => {
    const delayed = '**/api/v1/universities/*/courses?*'
    await page.route(delayed, () => {}) // never answers: the skeleton stays up until the next goto drops the request
    void page.goto(`/${slug}/courses`).catch(() => {})
    await page.waitForTimeout(500)
    await expect(page.locator('.ui-skeleton').first()).toBeVisible()
    await snap(page, '70-states-loading-skeleton-admin', 'Courses list while its data loads (skeleton), admin', { noSettle: true })
    await page.unroute(delayed)
    await page.goto(`/${slug}/me`)
    const failed = '**/api/v1/universities/*/sections?*'
    await page.route(failed, (r) => r.fulfill({ status: 500, contentType: 'application/json', body: '{"error":"internal"}' }))
    await page.goto(`/${slug}/courses/sections`)
    await expect(page.getByRole('alert')).toBeVisible({ timeout: 30_000 })
    await snap(page, '71-states-error-banner-admin', 'Sections list when the API answers 500 (error banner), admin')
    await page.unroute(failed)
  })

  await test.step('success and warning states', async () => {
    await page.goto(`/${slug}/grade-scale`)
    await page.getByRole('button', { name: 'Save scale' }).click()
    await expect(page.getByRole('status').filter({ hasText: 'Grade scale saved' })).toBeVisible()
    await snap(page, '72-states-success-admin', 'Grade scale saved (success StatusMessage), admin')
    // CS210 has weights 40 + 60; a third assessment makes 110, which shows the "Must total 100%" warning badge
    await api(profPage, 'post', `${A}/sections/${sec.CS210}/assessments`, { name: 'Extra Credit', maxPoints: 10, weight: 10, published: true })
    await profPage.goto(`/${slug}/sections/${sec.CS210}/grades`)
    await expect(profPage.getByText('Must total 100% to finalize', { exact: true })).toBeVisible()
    await snap(profPage, '73-states-weight-warning-instructor', 'Gradebook CS210 with weights totalling 110 (warning badge), instructor')
  })

  await test.step('no access page', async () => {
    const slugB = `harbor-${runId}`
    createUniversity({ name: 'Harbor College', slug: slugB, domain: `harbor-${runId}.edu`, adminEmail: `dean@harbor-${runId}.edu` })
    const c = await browser.newContext()
    const p = await c.newPage()
    await signIn(p, `dean@harbor-${runId}.edu`, slugB)
    await p.goto(`/${slug}/me`)
    await expect(p.getByRole('heading', { name: "You don't have access to this university" })).toBeVisible()
    await snap(p, '04-auth-no-access-admin', "Harbor admin (another university) opening Northfield's address")
    await c.close()
  })

  index.sort()
  fs.writeFileSync(path.join(out, 'index.md'), `# Screens\n\n${index.map((l) => `- ${l}`).join('\n')}\n`)
})
