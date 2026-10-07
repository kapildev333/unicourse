// Screen capture for the UI/UX benchmark (not part of `npm run e2e`): `npm run screens`.
// Output: e2e/.tmp/screens/*.png and index.md. Dark/light via emulated prefers-color-scheme (theme stays "System").
import fs from 'node:fs'
import path from 'node:path'
import { expect, test, type Page } from '@playwright/test'
import { tmpDir } from './env'
import { createUniversity, runId, signIn } from './helpers'

const slug = `northfield-${runId}`
const dom = `northfield-${runId}.edu`
const A = `/api/v1/universities/${slug}`
const H = { 'Sec-Fetch-Site': 'same-origin' }
const year = new Date().getFullYear()
const out = path.join(tmpDir, 'screens')
const index: string[] = []

async function api(page: Page, method: 'get' | 'post' | 'put' | 'patch', url: string, data?: unknown) {
  const res = await page.request[method](url, { headers: H, data })
  expect(res.ok(), `${method} ${url}: ${res.status()} ${await res.text()}`).toBeTruthy()
  return res.status() === 204 ? null : res.json()
}

async function upload(page: Page, url: string, csv: string, commit = true) {
  const file = { name: 'upload.csv', mimeType: 'text/csv', buffer: Buffer.from(csv) }
  const pv = await page.request.post(`${url}/import/preview`, { headers: H, multipart: { file } })
  expect(pv.ok(), await pv.text()).toBeTruthy()
  if (!commit) return
  const { checksum } = await pv.json()
  const res = await page.request.post(`${url}/import`, { headers: H, multipart: { file, checksum } })
  expect(res.ok(), await res.text()).toBeTruthy()
}

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

const people = [
  ['Amara Okafor', 'amara.okafor'], ['Priya Raman', 'priya.raman'], ['Tomasz Zielinski', 'tomasz.zielinski'], ['Daniel Reyes', 'daniel.reyes'],
  ['Mei-Ling Chou', 'mei-ling.chou'], ['Jonas Berg', 'jonas.berg'], ['Fatima Al-Sayed', 'fatima.alsayed'], ['Lucas Moreau', 'lucas.moreau'],
].map(([name, local]) => ({ name, email: `${local}@${dom}` }))
const prof = `elena.marsh@${dom}`
const ta = `sofia.alvarez@${dom}`
const prof2 = `david.mensah@${dom}`
const newHire = `ingrid.larsen@${dom}`

// code, title, credits, department, meetings [weekday, start, end, room], student indexes, instructor
// (Amara, index 0, is in three of the five so the Catalog shows enrolled and open cards)
const courses = [
  ['CS101', 'Introduction to Computer Science', 4, 'CS', [[1, '10:00', '11:20', 'Hall B 204'], [3, '10:00', '11:20', 'Hall B 204']], [0, 1, 2, 3, 4, 5, 6, 7], prof],
  ['CS210', 'Data Structures and Algorithms', 4, 'CS', [[2, '10:00', '11:20', 'Engineering Building 310'], [4, '10:00', '11:20', 'Engineering Building 310']], [0, 1, 2, 3, 4], prof],
  ['MATH201', 'Linear Algebra', 4, 'MATH', [[1, '13:00', '14:20', 'Science Center 112'], [3, '13:00', '14:20', 'Science Center 112']], [0, 1, 2, 3, 5, 6], prof],
  ['ECON110', 'Principles of Microeconomics', 3, 'ECON', [[2, '13:30', '14:50', 'Mercer Hall 101'], [4, '13:30', '14:50', 'Mercer Hall 101']], [4, 5, 6, 7], prof2],
  ['BIO150', 'Cell Biology', 4, 'BIO', [[3, '15:00', '16:20', 'Life Sciences 120'], [5, '09:00', '10:20', 'Life Sciences 120']], [1, 3, 7], prof2],
] as const
const depts = [['Computer Science', 'CS'], ['Mathematics', 'MATH'], ['Economics', 'ECON'], ['Biology', 'BIO']]
const esc = (t: string) => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const csv = (rows: string[][]) => ['email,displayName', ...rows.map((r) => r.join(','))].join('\n') + '\n'
const usScale = [['A', 93, 4], ['A-', 90, 3.7], ['B+', 87, 3.3], ['B', 83, 3], ['B-', 80, 2.7], ['C+', 77, 2.3], ['C', 73, 2], ['C-', 70, 1.7], ['D+', 67, 1.3], ['D', 63, 1], ['D-', 60, 0.7], ['F', 0, 0]]
const csMid = [87.5, 62, 94, 71.25, 78, 55.5, 89, 81.75]

test('capture every screen', async ({ browser, page }) => {
  test.setTimeout(1_200_000)
  fs.mkdirSync(out, { recursive: true })
  const ctxs = async (email: string) => {
    const c = await browser.newContext()
    const p = await c.newPage()
    await signIn(p, email, slug)
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

  await test.step('university, admin, and empty states', async () => {
    createUniversity({ name: 'Northfield University', slug, domain: dom, adminEmail: `registrar@${dom}` })
    await signIn(page, `registrar@${dom}`, slug)
    await snap(page, '10-my-sections-empty-admin', 'My sections with nothing yet, admin signed in')
    await page.goto(`/${slug}/catalog`)
    await snap(page, '11-catalog-empty-admin', 'Catalog before any term exists, admin signed in')
    await page.goto(`/${slug}/courses`)
    await snap(page, '12-courses-empty-admin', 'Courses list with no courses, admin signed in')
    await page.goto(`/${slug}/courses/sections`)
    await snap(page, '13-sections-empty-admin', 'Sections list with no sections, admin signed in')
  })

  const dep: Record<string, string> = {}
  const sec: Record<string, string> = {}
  await test.step('seed catalog, staff, rosters', async () => {
    for (const [name, code] of depts) dep[code] = (await api(page, 'post', `${A}/departments`, { name, code })).id
    await api(page, 'post', `${A}/terms`, { name: `Academic Year ${year}`, startsOn: `${year}-01-01`, endsOn: `${year}-12-31` })
    const term = (await api(page, 'get', `${A}/terms`)).items.find((t: { name: string }) => t.name.includes('Academic'))
    await api(page, 'post', `${A}/terms`, { name: `Spring ${year + 1}`, startsOn: `${year + 1}-01-12`, endsOn: `${year + 1}-05-15` })
    for (const [code, title, credits, d, meets] of courses) {
      const c = await api(page, 'post', `${A}/courses`, { departmentId: dep[d], code, title, credits })
      const s = await api(page, 'post', `${A}/sections`, {
        courseId: c.id, termId: term.id, number: '001',
        meetings: meets.map(([weekday, startsAt, endsAt, location]) => ({ weekday, startsAt, endsAt, location })),
      })
      sec[code] = s.id
    }
    for (const email of [prof, ta, prof2, newHire]) await api(page, 'post', `${A}/invitations`, { email, role: 'member' })
    await api(page, 'post', `${A}/invitations`, { email: `ops@${dom}`, role: 'admin' })
    for (const [code, , , , , , who] of courses) await api(page, 'post', `${A}/sections/${sec[code]}/staff`, { email: who, role: 'instructor' })
    await api(page, 'post', `${A}/sections/${sec.CS101}/staff`, { email: ta, role: 'ta' })
    for (const [code, , , , , idx] of courses)
      await upload(page, `${A}/sections/${sec[code]}/enrollments`, csv(idx.map((i) => [people[i].email, people[i].name])))
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
  const prof2Page = await ctxs(prof2)
  const stu = await ctxs(people[0].email)
  // staff names, so student and roster screens show people rather than emails
  for (const [p, name] of [[profPage, 'Elena Marsh'], [taPage, 'Sofia Alvarez'], [prof2Page, 'David Mensah'], [page, 'Renata Quinn']] as const)
    await api(p, 'patch', '/api/v1/me', { displayName: name })
  for (const p of people.slice(1)) await (await ctxs(p.email)).context().close()
  // ingrid.larsen is never signed in, so Invitations keeps one pending row

  await test.step('assessments and scores', async () => {
    const meta: Record<string, [string, string]> = { CS101: ['Midterm Exam', 'Final Exam'], CS210: ['Problem Set Average', 'Final Project'], MATH201: ['Midterm Exam', 'Final Exam'], ECON110: ['Midterm Exam', 'Final Exam'], BIO150: ['Lab Practical', 'Final Exam'] }
    let s = 0
    for (const [code, , , , , , who] of courses) {
      s++
      const owner = who === prof ? profPage : prof2Page
      const enr = (await api(page, 'get', `${A}/sections/${sec[code]}/enrollments`)).items as { userId: string; email: string }[]
      const idx = (e: { email: string }) => people.findIndex((p) => p.email === e.email)
      for (const [n, name] of meta[code].entries()) {
        const a = await api(owner, 'post', `${A}/sections/${sec[code]}/assessments`, { name, maxPoints: 100, weight: n ? 60 : 40, published: true })
        const scores = enr
          .map((e) => {
            const i = idx(e)
            const pts = code === 'CS101' && !n ? csMid[i] : Math.round((48 + ((i * 37 + s * 19 + n * 13) % 51) + (i % 4) * 0.25) * 4) / 4
            return { userId: e.userId, points: pts }
          })
          // one missing score left in place per section: the last-enrolled student has no Final
          .filter((_x, k, all) => !(n === 1 && k === all.length - 1))
        await api(owner, 'put', `${A}/sections/${sec[code]}/assessments/${a.id}/scores`, { scores })
      }
    }
    const scale = await api(page, 'get', `${A}/grade-scale`)
    if (!scale.rows?.length) await api(page, 'put', `${A}/grade-scale`, { rows: usScale.map(([letter, minPercent, points]) => ({ letter, minPercent, points })) })
  })

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
    await snap(page, '45-section-detail-admin', 'Section detail CS101 (staff, meeting times, roster), admin')
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
    await expect(profPage.getByText('Must total 100% to finalize')).toBeVisible()
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
