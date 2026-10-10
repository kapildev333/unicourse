import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import { expect, test, type Page } from '@playwright/test'
import { api, expectNoSideScroll, pageAs, personas, runId, sectionIds, seedDemo } from './helpers'

// Spec 11 §1, assignment half: create, submit text and a PDF, replace the text, download, grade, release, My grades.
// The cases share one assignment and run in order.
test.describe.configure({ mode: 'serial' })

const slug = `e2e-asg-${runId}`
const other = `e2e-asg2-${runId}`
const P = personas(slug)
const P2 = personas(other)
const NAME = 'Essay 1'
const pdfPath = fileURLToPath(new URL('./fixtures/syllabus.pdf', import.meta.url))
let sid = ''

test.beforeAll(async ({ browser }) => {
  seedDemo(slug)
  seedDemo(other)
  const elena = await pageAs(browser, P.elena.email, slug)
  sid = (await sectionIds(elena, slug))['CS210-001']
  await elena.context().close()
})

const status = (page: Page, text: string) => page.getByRole('status').filter({ hasText: text })
const list = (page: Page) => page.goto(`/${slug}/sections/${sid}/assignments`)
const local = (ms: number) => new Date(ms).toISOString().slice(0, 16)
async function open(page: Page) {
  await list(page)
  await page.getByRole('link', { name: `View ${NAME}` }).click()
  await expect(page.getByRole('heading', { name: NAME }), 'student, assignment page: heading').toBeVisible()
}

test('E-ASG-01 the instructor creates Essay 1', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await list(page)
  await expect(page.getByText('No assignments yet'), 'instructor, Assignments: empty state').toBeVisible()
  await page.getByRole('button', { name: 'New assignment' }).click()
  const dlg = page.getByRole('dialog', { name: 'New assignment' })
  await dlg.getByLabel('Title').fill(NAME)
  await dlg.getByLabel('Description').fill('Write about one data structure.')
  await dlg.getByLabel('Opens').fill(local(Date.now() - 60_000))
  await dlg.getByLabel('Due').fill(local(Date.now() + 7 * 24 * 3600_000))
  await dlg.getByLabel('Max points').fill('20')
  await dlg.getByLabel('Weight (%)').fill('20')
  await dlg.getByRole('button', { name: 'Create assignment' }).click()
  await expect(status(page, `${NAME} created.`), 'instructor, Assignments: created notice').toBeVisible()
  const card = page.getByRole('article').filter({ hasText: NAME })
  await expect(card.getByText('Open', { exact: true }), 'instructor, Assignments: Open badge').toBeVisible()
  await expect(card, 'instructor, Assignments: max points and weight').toContainText('20 points')
  await page.context().close()
})

test('E-ASG-02 a student saves text, uploads a PDF and replaces the text once', async ({ browser }) => {
  const page = await pageAs(browser, P.amara.email, slug)
  await open(page)
  await expect(page.getByText('Write about one data structure.'), 'student, assignment: description').toBeVisible()
  await expect(page.getByText('Not submitted yet.'), 'student, assignment: not submitted').toBeVisible()
  await page.getByLabel('Your answer').fill('A stack is last in, first out.')
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(status(page, 'Submission saved.'), 'student, assignment: saved').toBeVisible()
  await expect(page.getByText(/^Submitted /), 'student, assignment: submitted time').toBeVisible()

  await page.getByLabel('Files for your submission').setInputFiles(pdfPath)
  await expect(page.getByLabel('Download syllabus.pdf'), 'student, assignment: PDF listed').toBeVisible()
  await expect(page.getByText('1 of 10'), 'student, assignment: file count').toBeVisible()

  await page.getByLabel('Your answer').fill('A stack is last in, first out; a queue is the opposite.')
  await page.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(status(page, 'Submission saved.'), 'student, assignment: saved again').toBeVisible()
  await page.reload()
  await expect(page.getByLabel('Your answer'), 'student, assignment: replaced text persists').toHaveValue('A stack is last in, first out; a queue is the opposite.')
  await expect(page.getByLabel('Download syllabus.pdf'), 'student, assignment: PDF persists').toBeVisible()
  await page.context().close()
})

test('E-ASG-03 the TA downloads the same PDF, grades 17 and releases', async ({ browser }) => {
  const page = await pageAs(browser, P.ingrid.email, slug)
  await list(page)
  await page.getByRole('link', { name: `Submissions for ${NAME}` }).click()
  const rows = page.getByRole('row')
  await expect(rows.filter({ hasText: 'Amara Okafor' }), 'TA, Submissions: amara row').toContainText('Not graded')
  await page.getByRole('link', { name: 'Amara Okafor' }).click()
  await expect(page.getByText('A stack is last in, first out; a queue is the opposite.'), 'TA, submission: replaced text').toBeVisible()

  const href = await page.getByLabel('Download syllabus.pdf').getAttribute('href')
  const res = await page.request.get(href!)
  expect(res.status(), 'TA, download: status').toBe(200)
  expect(Buffer.compare(await res.body(), fs.readFileSync(pdfPath)), 'TA, download: bytes equal the uploaded fixture').toBe(0)

  await page.getByLabel('Score for Amara Okafor').fill('17')
  await page.getByLabel('Feedback for Amara Okafor').fill('Clear and correct. Add an example next time.')
  await page.getByRole('button', { name: 'Save grade' }).click()
  await expect(status(page, 'Grade saved. Amara Okafor: 17 / 20.'), 'TA, submission: saved notice').toBeVisible()

  await page.getByRole('link', { name: 'All submissions' }).click()
  await expect(page.getByRole('row').filter({ hasText: 'Amara Okafor' }), 'TA, Submissions: graded row').toContainText('17 / 20')
  const student = await pageAs(browser, P.amara.email, slug)
  await open(student)
  await expect(student.getByText('Score 17 / 20'), 'student, assignment: score hidden before release').toHaveCount(0)
  await page.getByRole('button', { name: `Release scores for ${NAME}` }).click()
  await page.getByRole('dialog', { name: `Release scores for ${NAME}?` }).getByRole('button', { name: 'Release scores' }).click()
  await expect(status(page, `${NAME}: scores released to students.`), 'TA, Submissions: released notice').toBeVisible()
  await student.reload()
  await expect(student.getByText('Score 17 / 20'), 'student, assignment: score after release').toBeVisible()
  await student.context().close()
  await page.context().close()
})

test('E-ASG-04 the student sees 17 / 20 and the feedback; My grades lists it', async ({ browser }) => {
  const page = await pageAs(browser, P.amara.email, slug)
  await open(page)
  await expect(page.getByText('Score 17 / 20'), 'student, assignment: score').toBeVisible()
  await expect(page.getByText('Clear and correct. Add an example next time.'), 'student, assignment: feedback').toBeVisible()

  await page.goto(`/${slug}/grades`)
  const card = page.getByRole('listitem').filter({ has: page.getByRole('link', { name: /^CS210-001/ }) })
  await card.getByText(/^Published scores/).click()
  await expect(card.getByRole('listitem').filter({ hasText: NAME }), 'student, My grades: Essay 1 row').toContainText('17 / 20')
  await card.getByRole('link', { name: NAME }).click()
  await expect(page.getByRole('heading', { name: NAME }), 'student, My grades: link opens the assignment').toBeVisible()
  await page.context().close()
})

test('E-ASG-05 an .html upload is refused and leaves no submission', async ({ browser }) => {
  const page = await pageAs(browser, P.priya.email, slug)
  await open(page)
  await page.getByLabel('Files for your submission').setInputFiles({ name: 'page.html', mimeType: 'text/html', buffer: Buffer.from('<h1>hi</h1>') })
  const banner = page.getByRole('alert').filter({ hasText: '1 file not uploaded' })
  await expect(banner, 'priya, assignment: refused banner').toBeVisible()
  await expect(banner, 'priya, assignment: names the file').toContainText('page.html')
  await expect(page.getByText('No files attached.'), 'priya, assignment: nothing attached').toBeVisible()
  await expect(page.getByText('Not submitted yet.'), 'priya, assignment: still not submitted').toBeVisible()
  await page.context().close()
})

test('E-ASG-06 a student who submitted nothing has no row', async ({ browser }) => {
  const page = await pageAs(browser, P.ingrid.email, slug)
  await list(page)
  await page.getByRole('link', { name: `Submissions for ${NAME}` }).click()
  await expect(page.getByRole('row').filter({ hasText: 'Amara Okafor' }), 'TA, Submissions: amara row').toHaveCount(1)
  await expect(page.getByRole('row').filter({ hasText: 'Priya Raman' }), 'TA, Submissions: no priya row').toHaveCount(0)
  await page.context().close()
})

test('E-ASG-07 the assignment page fits a 390 px phone', async ({ browser }) => {
  const page = await pageAs(browser, P.priya.email, slug)
  await page.setViewportSize({ width: 390, height: 844 })
  await list(page)
  await expectNoSideScroll(page, 'student at 390, Assignments')
  await page.getByRole('link', { name: `View ${NAME}` }).click()
  await expect(page.getByRole('button', { name: 'Upload files' }), 'student at 390, assignment: Upload').toBeVisible()
  await expectNoSideScroll(page, 'student at 390, assignment page')
  await page.context().close()
})

test('E-ASG-08 a second university sees none of it', async ({ browser }) => {
  const page = await pageAs(browser, P2.elena.email, other)
  await page.goto(`/${slug}/sections/${sid}/assignments`)
  await expect(page.getByText(NAME), 'other university instructor, first university URL: no Essay 1').toHaveCount(0)
  await expect(page.getByRole('heading', { name: "You don't have access to this university" }), 'other university instructor, first university URL: refused').toBeVisible()
  const res = await page.request.get(`/api/v1/universities/${slug}/sections/${sid}/assessments`, { headers: { 'Sec-Fetch-Site': 'same-origin' } })
  expect([403, 404], 'other university instructor, API: first university assessments').toContain(res.status())
  const own = (await sectionIds(page, other))['CS210-001']
  await page.goto(`/${other}/sections/${own}/assignments`)
  await expect(page.getByText('No assignments yet'), 'other university instructor, own CS210-001: empty').toBeVisible()
  await page.context().close()
})

test('E-ASG-09 a late submission is penalised: 15 graded, 12.00 recorded and shown', async ({ browser }) => {
  const elena = await pageAs(browser, P.elena.email, slug)
  const now = Date.now()
  await api(elena, 'post', `/api/v1/universities/${slug}/sections/${sid}/assessments`, {
    kind: 'assignment',
    name: 'Late essay',
    opensAt: new Date(now - 48 * 3600_000).toISOString(),
    closesAt: new Date(now - 3600_000).toISOString(),
    lateUntil: new Date(now + 7 * 24 * 3600_000).toISOString(),
    latePenaltyPercent: 20,
    maxPoints: 20,
    weight: 0,
  })
  await elena.context().close()

  const student = await pageAs(browser, P.priya.email, slug)
  await list(student)
  await student.getByRole('link', { name: 'View Late essay' }).click()
  await expect(student.getByText('Past due', { exact: true }), 'student, late assignment: Past due badge').toBeVisible()
  await expect(student.getByText(/Late submissions are accepted until .* with a 20% penalty\./), 'student, late assignment: late sentence').toBeVisible()
  await student.getByLabel('Your answer').fill('Better late than never.')
  await student.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(status(student, 'Submission saved.'), 'student, late assignment: saved').toBeVisible()
  await expect(student.getByText(/^Submitted late on /), 'student, late assignment: late submitted line').toBeVisible()

  const ta = await pageAs(browser, P.ingrid.email, slug)
  await list(ta)
  await ta.getByRole('link', { name: 'Submissions for Late essay' }).click()
  await ta.getByRole('link', { name: 'Priya Raman' }).click()
  await ta.getByLabel('Score for Priya Raman').fill('15')
  await ta.getByRole('button', { name: 'Save grade' }).click()
  await expect(ta.getByText('Recorded as 12.00 after the 20% late penalty'), 'TA, submission: penalised value').toBeVisible()
  await ta.getByRole('link', { name: 'All submissions' }).click()
  await ta.getByRole('button', { name: 'Release scores for Late essay' }).click()
  await ta.getByRole('dialog', { name: 'Release scores for Late essay?' }).getByRole('button', { name: 'Release scores' }).click()
  await expect(status(ta, 'Late essay: scores released to students.'), 'TA, Submissions: released notice').toBeVisible()
  await ta.context().close()

  await student.reload()
  await expect(student.getByText('Score 15 / 20, 12.00 after the 20% late penalty'), 'student, late assignment: penalised score').toBeVisible()
  await student.context().close()

  const inst = await pageAs(browser, P.elena.email, slug)
  await inst.goto(`/${slug}/sections/${sid}/grades`)
  await expect(inst.getByRole('textbox', { name: 'Score for Priya Raman, Late essay' }), 'instructor, gradebook: 12').toHaveValue('12')
  await inst.context().close()
})
