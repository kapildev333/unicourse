import fs from 'node:fs'
import { expect, test, type Page } from '@playwright/test'
import { apiLog } from './env'
import { api, createUniversity, pageAs, personas, runId, sectionIds, seedDemo, signIn } from './helpers'

// Spec 21 §6.2-§6.3: the Upcoming panel, the calendar feed and dates in the university's zone.
test.describe.configure({ mode: 'serial' })

const slug = `e2e-upc-${runId}`
const P = personas(slug)
const A = `/api/v1/universities/${slug}`
const DAY = 24 * 3600_000
let sid = ''
let feedUrl = ''

test.beforeAll(async ({ browser }) => {
  seedDemo(slug)
  const elena = await pageAs(browser, P.elena.email, slug)
  sid = (await sectionIds(elena, slug))['CS210-001']
  const now = Date.now()
  const iso = (ms: number) => new Date(ms).toISOString()
  await api(elena, 'post', `${A}/sections/${sid}/assessments`, { kind: 'quiz', name: 'Open quiz', opensAt: iso(now - 3600_000), closesAt: iso(now + 3 * DAY), weight: 0 })
  await api(elena, 'post', `${A}/sections/${sid}/assessments`, { kind: 'assignment', name: 'Due essay', opensAt: iso(now - 3600_000), closesAt: iso(now + 2 * DAY), maxPoints: 10, weight: 0 })
  const done = await api(elena, 'post', `${A}/sections/${sid}/assessments`, { kind: 'assignment', name: 'Done essay', opensAt: iso(now - 3600_000), closesAt: iso(now + 2 * DAY), maxPoints: 10, weight: 0 })
  await elena.context().close()

  const amara = await pageAs(browser, P.amara.email, slug)
  await amara.goto(`/${slug}/sections/${sid}/assignments/${done.id}`)
  await amara.getByLabel('Your answer').fill('Finished.')
  await amara.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(amara.getByRole('status').filter({ hasText: 'Submission saved.' }), 'student, assignment: saved').toBeVisible()
  await amara.context().close()
})

const upcoming = (page: Page) => page.getByRole('list', { name: 'Upcoming work' })
const item = (page: Page, name: string) => upcoming(page).getByRole('listitem').filter({ hasText: name })

test('E-UPC-01 a student sees the open quiz and the due assignment, not the submitted one', async ({ browser }) => {
  const page = await pageAs(browser, P.amara.email, slug)
  await page.goto(`/${slug}/me`)
  await expect(page.getByRole('heading', { name: 'Upcoming', level: 2 }), 'student, My sections: Upcoming heading').toBeVisible()
  await expect(item(page, 'Open quiz'), 'student, Upcoming: open quiz').toContainText('CS210-001 · Quiz · Closes')
  await expect(item(page, 'Due essay'), 'student, Upcoming: due assignment').toContainText('CS210-001 · Assignment · Due')
  await expect(item(page, 'Done essay'), 'student, Upcoming: submitted assignment is gone').toHaveCount(0)
  await page.context().close()
})

test('E-UPC-02 staff see how many have submitted', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await page.goto(`/${slug}/me`)
  await expect(item(page, 'Done essay').getByText(/^1 of \d+ submitted$/), 'instructor, Upcoming: Done essay 1 submitted').toBeVisible()
  await expect(item(page, 'Due essay').getByText(/^0 of \d+ submitted$/), 'instructor, Upcoming: Due essay 0 submitted').toBeVisible()
  await expect(item(page, 'Open quiz').getByText(/^0 of \d+ submitted$/), 'instructor, Upcoming: quiz 0 submitted').toBeVisible()
  await page.context().close()
})

test('E-UPC-03 calendar feed: create, fetch, rotate, old link 404, turn off', async ({ browser }) => {
  const page = await pageAs(browser, P.amara.email, slug)
  await page.goto(`/${slug}/me`)
  const open = async () => {
    await page.getByRole('button', { name: 'Calendar feed' }).click()
    return page.getByRole('dialog', { name: 'Calendar feed' })
  }
  let dlg = await open()
  await dlg.getByRole('button', { name: 'Create link' }).click()
  const link = dlg.getByLabel('Calendar link')
  feedUrl = await link.inputValue()
  expect(feedUrl, 'student, Calendar feed: URL shown').toContain('/calendar-feeds/')
  const res = await page.request.get(feedUrl)
  expect(res.status(), 'feed fetch: status').toBe(200)
  expect(res.headers()['content-type'], 'feed fetch: content type').toContain('text/calendar')
  const body = await res.text()
  expect(body, 'feed fetch: VEVENT lines').toContain('BEGIN:VEVENT')
  expect(body, 'feed fetch: carries the open quiz').toContain('Open quiz')
  expect(body, 'feed fetch: leaves out submitted work').not.toContain('Done essay')
  await dlg.getByRole('button', { name: 'Done' }).click()

  dlg = await open()
  await expect(dlg.getByText(/Your private link was created on/), 'student, Calendar feed: link exists, URL not shown again').toBeVisible()
  await dlg.getByRole('button', { name: 'Create new link' }).click()
  await expect(dlg.getByText('The old link stops working in every calendar that uses it.'), 'student, Calendar feed: rotate confirm').toBeVisible()
  await dlg.getByRole('button', { name: 'Create new link' }).click()
  const fresh = await dlg.getByLabel('Calendar link').inputValue()
  expect(fresh, 'rotation: a different URL').not.toBe(feedUrl)
  expect((await page.request.get(feedUrl)).status(), 'rotation: old URL 404').toBe(404)
  expect((await page.request.get(fresh)).status(), 'rotation: new URL 200').toBe(200)
  await dlg.getByRole('button', { name: 'Done' }).click()

  dlg = await open()
  await dlg.getByRole('button', { name: 'Turn off' }).click()
  await expect(dlg.getByRole('button', { name: 'Create link' }), 'student, Calendar feed: turned off').toBeVisible()
  expect((await page.request.get(fresh)).status(), 'turn off: URL 404').toBe(404)
  const log = fs.readFileSync(apiLog, 'utf8')
  for (const url of [feedUrl, fresh]) {
    expect(log, 'neither feed token reaches the API log').not.toContain(url.split('/calendar-feeds/')[1].replace(/\.ics$/, ''))
  }
  await page.context().close()
})

// The runner's zone must not matter: the browser is pinned to New York.
test.describe('dates show in the university zone', () => {
  const zone = { timezoneId: 'America/New_York' }

  test('E-UPC-04 a UTC university names the zone, the New York one does not', async ({ browser }) => {
    const uslug = `e2e-upc-u-${runId}`
    const domain = `upc-${runId}.edu`
    const root = `root@${domain}`
    createUniversity({ name: `Zone ${runId}`, slug: uslug, domain, adminEmail: root })
    const ctx = await browser.newContext(zone)
    const page = await ctx.newPage()
    await signIn(page, root, uslug)
    const U = `/api/v1/universities/${uslug}`
    const y = new Date().getFullYear()
    const dept = await api(page, 'post', `${U}/departments`, { name: 'Computing', code: 'CS' })
    const term = await api(page, 'post', `${U}/terms`, { name: 'Zone term', startsOn: `${y - 1}-01-01`, endsOn: `${y + 1}-12-31` })
    const course = await api(page, 'post', `${U}/courses`, { departmentId: dept.id, code: 'CS900', title: 'Zones' })
    const sec = await api(page, 'post', `${U}/sections`, { courseId: course.id, termId: term.id, number: '001' })
    await api(page, 'post', `${U}/sections/${sec.id}/staff`, { email: root, role: 'instructor' })
    const closes = new Date()
    closes.setUTCDate(closes.getUTCDate() + 1)
    closes.setUTCHours(15, 0, 0, 0)
    await api(page, 'post', `${U}/sections/${sec.id}/assessments`, {
      kind: 'quiz', name: 'Zone quiz', opensAt: new Date(Date.now() - 3600_000).toISOString(), closesAt: closes.toISOString(), weight: 0,
    })
    await page.goto(`/${uslug}/me`)
    await expect(item(page, 'Zone quiz'), 'UTC university admin in a New York browser, Upcoming: UTC named').toContainText(/Closes .*3:00 PM UTC/)
    await page.goto(`/${uslug}/sections/${sec.id}/quizzes`)
    await page.getByRole('button', { name: 'Edit Zone quiz' }).click()
    const dlg = page.getByRole('dialog')
    await expect(dlg.getByLabel('Closes'), 'UTC university, quiz dialog: Closes is UTC wall time').toHaveValue(/T15:00$/)
    await expect(dlg.getByText('Times are in UTC.'), 'UTC university, quiz dialog: zone hint').toBeVisible()
    await ctx.close()

    // seeded Northfield is America/New_York: same browser zone, no suffix
    const ny = await browser.newContext(zone)
    const nyPage = await ny.newPage()
    await signIn(nyPage, P.amara.email, slug)
    await nyPage.goto(`/${slug}/me`)
    const text = await item(nyPage, 'Open quiz').innerText()
    expect(text, 'New York university in a New York browser, Upcoming: no zone suffix').not.toMatch(/\b(UTC|EDT|EST)\b/)
    await ny.close()
  })
})
