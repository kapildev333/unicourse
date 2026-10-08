import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import { expect, test, type Page } from '@playwright/test'
import { api, pageAs, personas, runId, sectionIds, seedDemo } from './helpers'

const slug = `e2e-cnt-${runId}`
const P = personas(slug)
const A = `/api/v1/universities/${slug}`
const pdf = fileURLToPath(new URL('./fixtures/syllabus.pdf', import.meta.url))

let ids: Record<string, string> = {}

test.beforeAll(async ({ browser }) => {
  seedDemo(slug)
  const admin = await pageAs(browser, P.admin.email, slug)
  ids = await sectionIds(admin, slug)
  await admin.context().close()
})

const at = (page: Page, code: string, tab = '') => page.goto(`/${slug}/sections/${ids[code]}${tab}`)
const article = (page: Page, title: string) => page.getByRole('article').filter({ hasText: title })
const status = (page: Page, text: string) => page.getByRole('status').filter({ hasText: text })
const weekCard = (page: Page, title: string) =>
  page.getByRole('listitem').filter({ has: page.getByRole('heading', { name: title, level: 3 }) })

test('E-CONTENT-01 a TA edits the instructor\'s announcement', async ({ browser }) => {
  const page = await pageAs(browser, P.sofia.email, slug)
  await at(page, 'CS101-001', '/announcements')
  await page.getByRole('button', { name: 'Edit Welcome to CS101' }).click()
  const dlg = page.getByRole('dialog', { name: 'Edit announcement' })
  await expect(dlg.getByText('Edits are not emailed again.'), 'TA, edit dialog: no re-send note').toBeVisible()
  await expect(dlg.getByLabel('Also email everyone in this section'), 'TA, edit dialog: no email checkbox').toHaveCount(0)
  await dlg.getByLabel('Title').fill('Welcome to CS101 (updated)')
  await dlg.getByRole('button', { name: 'Save changes' }).click()
  await expect(status(page, 'Announcement saved. Edits are not emailed.'), 'TA, Announcements: saved notice').toBeVisible()
  const art = article(page, 'Welcome to CS101 (updated)')
  await expect(art, 'TA, Announcements: author kept').toContainText('Posted by Elena Marsh')
  await expect(art, 'TA, Announcements: edited marker').toContainText('· Edited')
  await page.context().close()
})

test('E-CONTENT-02 an instructor deletes an announcement', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await at(page, 'CS101-001', '/announcements')
  await page.getByRole('button', { name: 'Delete Midterm grades are posted' }).click()
  const dlg = page.getByRole('dialog', { name: 'Delete announcement?' })
  await expect(
    dlg.getByText('"Midterm grades are posted" will be removed for everyone in the section. Emails already sent can\'t be recalled.'),
    'instructor, delete dialog: copy',
  ).toBeVisible()
  await dlg.getByRole('button', { name: 'Delete announcement' }).click()
  await expect(status(page, 'Announcement deleted.'), 'instructor, Announcements: deleted notice').toBeVisible()
  await expect(page.getByRole('heading', { name: 'Midterm grades are posted' }), 'instructor, Announcements: gone').toHaveCount(0)
  await page.context().close()
})

test('E-CONTENT-03 announcement validation, posting without email', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await at(page, 'MATH201-001', '/announcements')
  await page.getByRole('button', { name: 'New announcement' }).first().click()
  const dlg = page.getByRole('dialog', { name: 'New announcement' })
  await dlg.getByRole('button', { name: 'Post announcement' }).click()
  await expect(dlg.getByText('Enter a title of at most 200 characters, on one line.'), 'instructor, new announcement: title error').toBeVisible()
  await expect(dlg.getByText('Write a message of at most 20,000 characters.'), 'instructor, new announcement: message error').toBeVisible()
  await dlg.getByLabel('Title').fill('Quiz moved')
  await dlg.getByLabel('Message').fill('The quiz moves to Friday.')
  await dlg.getByLabel('Also email everyone in this section').uncheck()
  await dlg.getByRole('button', { name: 'Post announcement' }).click()
  await expect(status(page, 'Announcement posted. No email was sent.'), 'instructor, Announcements: no-email notice').toBeVisible()
  await expect(article(page, 'Quiz moved').getByText('Emailed'), 'instructor, Announcements: no Emailed marker').toHaveCount(0)
  await page.context().close()
})

test('E-CONTENT-04 announcements page through Load more', async ({ browser }) => {
  const page = await pageAs(browser, P.admin.email, slug)
  for (let i = 1; i <= 20; i++) {
    await api(page, 'post', `${A}/sections/${ids['BIO150-001']}/announcements`, { title: `Note ${i}`, body: 'x', email: false })
  }
  await at(page, 'BIO150-001', '/announcements')
  await expect(page.getByRole('article'), 'admin, Announcements: first page').toHaveCount(20)
  await expect(page.getByRole('heading', { name: 'Lab safety briefing' }), 'admin, Announcements: oldest not yet shown').toHaveCount(0)
  await page.getByRole('button', { name: 'Load more' }).click()
  await expect(page.getByRole('article'), 'admin, Announcements: after Load more').toHaveCount(21)
  await expect(page.getByRole('article').last(), 'admin, Announcements: oldest last').toContainText('Lab safety briefing')
  await page.context().close()
})

test('E-CONTENT-05 week rename, reorder, delete', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  const W = `${A}/sections/${ids['CS210-001']}/weeks`
  const w2 = await api(page, 'post', W, { title: 'Week 2: Trees' })
  await api(page, 'post', W, { title: 'Week 3: Graphs' })
  const up = await page.request.post(`${W}/${w2.id}/materials`, {
    headers: { 'Sec-Fetch-Site': 'same-origin' },
    multipart: { file: { name: 'syllabus.pdf', mimeType: 'application/pdf', buffer: fs.readFileSync(pdf) } },
  })
  expect(up.status(), 'instructor, upload to Week 2').toBe(201)

  await at(page, 'CS210-001', '/materials')
  await page.getByRole('button', { name: 'Edit Week 3: Graphs' }).click()
  const dlg = page.getByRole('dialog', { name: 'Edit week' })
  await dlg.getByLabel('Title').fill('Week 3: Graphs and paths')
  await dlg.getByLabel('Notes').fill('Read chapter 9.')
  await dlg.getByRole('button', { name: 'Save changes' }).click()
  await expect(status(page, 'Week 3: Graphs and paths saved.'), 'instructor, Materials: week saved').toBeVisible()
  await expect(page.getByText('Read chapter 9.'), 'instructor, Materials: notes').toBeVisible()

  await page.getByRole('button', { name: 'Move Week 3: Graphs and paths up' }).click()
  await expect(status(page, 'Week 3: Graphs and paths moved up.'), 'instructor, Materials: moved').toBeVisible()
  await expect(page.getByRole('heading', { level: 3 }), 'instructor, Materials: week order').toHaveText([
    'Week 1: Arrays and linked lists', 'Week 3: Graphs and paths', 'Week 2: Trees',
  ])
  await expect(page.getByRole('button', { name: 'Move Week 1: Arrays and linked lists up' }), 'instructor, Materials: first week cannot move up').toBeDisabled()

  await page.getByRole('button', { name: 'Delete Week 2: Trees' }).click()
  const d1 = page.getByRole('dialog', { name: 'Delete Week 2: Trees?' })
  await expect(d1.getByText('Its 1 file will be deleted too. This can\'t be undone.'), 'instructor, delete week: file count').toBeVisible()
  await d1.getByRole('button', { name: 'Delete week' }).click()
  await expect(status(page, 'Week 2: Trees deleted.'), 'instructor, Materials: Week 2 deleted').toBeVisible()

  await page.getByRole('button', { name: 'Delete Week 3: Graphs and paths' }).click()
  const d2 = page.getByRole('dialog', { name: 'Delete Week 3: Graphs and paths?' })
  await expect(d2.getByText('The week is empty. It will be removed for everyone in the section.'), 'instructor, delete week: empty').toBeVisible()
  await d2.getByRole('button', { name: 'Delete week' }).click()
  await expect(status(page, 'Week 3: Graphs and paths deleted.'), 'instructor, Materials: Week 3 deleted').toBeVisible()
  await page.context().close()
})

test('E-CONTENT-06 material rename, move, delete', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await at(page, 'CS101-001', '/materials')

  await page.getByRole('button', { name: 'Rename Course syllabus' }).click()
  const r = page.getByRole('dialog', { name: 'Rename file' })
  await expect(r.getByText('File name: syllabus.pdf. Downloads keep the file name.'), 'instructor, rename dialog: copy').toBeVisible()
  await r.getByLabel('Title').fill('Syllabus (v2)')
  await r.getByRole('button', { name: 'Save title' }).click()
  await expect(status(page, 'Renamed to Syllabus (v2).'), 'instructor, Materials: renamed').toBeVisible()
  await expect(page.getByRole('link', { name: 'Download Syllabus (v2)' }), 'instructor, Materials: new link').toBeVisible()

  await page.getByRole('button', { name: 'Move Lecture 1 slides to another week' }).click()
  const m = page.getByRole('dialog', { name: 'Move file' })
  await m.getByLabel('Week').selectOption({ label: 'Week 3: Functions' })
  await m.getByRole('button', { name: 'Move file' }).click()
  await expect(status(page, 'Lecture 1 slides moved to Week 3: Functions.'), 'instructor, Materials: moved').toBeVisible()
  await expect(weekCard(page, 'Week 3: Functions').getByRole('link', { name: 'Download Lecture 1 slides' }), 'instructor, Materials: file in Week 3').toBeVisible()

  await page.getByRole('button', { name: 'Delete Code samples' }).click()
  const d = page.getByRole('dialog', { name: 'Delete Code samples?' })
  await expect(d.getByText('Students will no longer see or download it. This can\'t be undone.'), 'instructor, delete file dialog: copy').toBeVisible()
  await d.getByRole('button', { name: 'Delete file' }).click()
  await expect(status(page, 'Code samples deleted.'), 'instructor, Materials: file deleted').toBeVisible()
  await expect(page.getByRole('link', { name: 'Download Code samples' }), 'instructor, Materials: link gone').toHaveCount(0)
  await page.context().close()
})

test('E-CONTENT-07 a disallowed file type is refused, the rest uploads', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await at(page, 'CS210-001', '/materials')
  await page.getByLabel('Files for Week 1: Arrays and linked lists').setInputFiles([
    { name: 'notes.html', mimeType: 'text/html', buffer: Buffer.from('<p>x</p>') },
    { name: 'syllabus.pdf', mimeType: 'application/pdf', buffer: fs.readFileSync(pdf) },
  ])
  const alert = page.getByRole('alert')
  await expect(alert, 'instructor, Materials: failure count').toContainText('1 file not uploaded')
  await expect(alert, 'instructor, Materials: failure reason').toContainText("notes.html: This file type can't be uploaded.")
  await expect(status(page, 'Uploaded 1 of 2 files to Week 1: Arrays and linked lists.'), 'instructor, Materials: partial notice').toBeVisible()
  await expect(page.getByRole('link', { name: 'Download syllabus.pdf' }), 'instructor, Materials: uploaded file').toBeVisible()

  const weeks = await api(page, 'get', `${A}/sections/${ids['CS210-001']}/weeks`)
  const w1 = (weeks.items ?? weeks).find((w: { title: string }) => w.title === 'Week 1: Arrays and linked lists')
  const res = await page.request.post(`${A}/sections/${ids['CS210-001']}/weeks/${w1.id}/materials`, {
    headers: { 'Sec-Fetch-Site': 'same-origin' },
    multipart: { file: { name: 'diagram.svg', mimeType: 'image/svg+xml', buffer: Buffer.from('<svg xmlns="http://www.w3.org/2000/svg"/>') } },
  })
  expect(res.status(), 'instructor, API: svg refused').toBe(400)
  expect((await res.json()).error.details[0].field, 'instructor, API: error names the file field').toBe('file')
  await page.context().close()
})

test('E-CONTENT-08 description limit; department admin edits own department only', async ({ browser }) => {
  const page = await pageAs(browser, P.deptAdmin.email, slug)
  await at(page, 'CS210-001')
  await page.getByRole('button', { name: 'Edit description' }).click()
  const dlg = page.getByRole('dialog', { name: 'Edit description' })
  await dlg.getByLabel('Description').fill('a'.repeat(10_001))
  await dlg.getByRole('button', { name: 'Save description' }).click()
  await expect(dlg.getByText('Use at most 10,000 characters of plain text.'), 'department admin, description dialog: limit error').toBeVisible()
  await dlg.getByLabel('Description').fill('Updated by the department.')
  await dlg.getByRole('button', { name: 'Save description' }).click()
  await expect(status(page, 'Description saved.'), 'department admin, Home: saved notice').toBeVisible()
  await expect(page.getByText('Updated by the department.'), 'department admin, Home: new description').toBeVisible()

  await at(page, 'ECON110-001')
  await expect(page.getByRole('heading', { level: 1 }), 'department admin, ECON110 Home: loaded').toBeVisible()
  await expect(page.getByRole('button', { name: 'Edit description' }), 'department admin, ECON110 Home: no edit').toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Add description' }), 'department admin, ECON110 Home: no add').toHaveCount(0)
  await page.context().close()
})

test('E-CONTENT-09 a student cannot write content', async ({ browser }) => {
  const page = await pageAs(browser, P.amara.email, slug)
  const S = `${A}/sections/${ids['CS101-001']}`
  const list = await api(page, 'get', `${S}/announcements`)
  const office = (list.items ?? list).find((a: { title: string }) => a.title === 'TA office hours')
  const H = { 'Sec-Fetch-Site': 'same-origin' }
  const patch = await page.request.patch(`${S}/announcements/${office.id}`, { headers: H, data: { title: 'x' } })
  expect(patch.status(), 'student, API: edit announcement').toBe(403)
  const post = await page.request.post(`${S}/weeks`, { headers: H, data: { title: 'x' } })
  expect(post.status(), 'student, API: add week').toBe(403)
  await page.context().close()
})
