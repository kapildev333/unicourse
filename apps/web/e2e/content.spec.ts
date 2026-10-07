import fs from 'node:fs'
import { fileURLToPath } from 'node:url'
import { expect, test, type Page } from '@playwright/test'
import { createUniversity, runId, signIn } from './helpers'
import { apiLog } from './env'

const slug = `ec-${runId}`
const domain = `c-${runId}.edu`
const admin = `root@${domain}`
const instr = `prof@${domain}`
const ta = `ta@${domain}`
const member = `member@${domain}`
const students = [1, 2].map((i) => `s${i}@${domain}`)
const year = String(new Date().getFullYear()) // the app picks the term by local date

const fixture = (name: string) => fileURLToPath(new URL(`./fixtures/${name}`, import.meta.url))
const pdf = fixture('syllabus.pdf')
const zip = fixture('code-samples.zip')
const pptx = fixture('slides.pptx')

const description = 'Welcome to CS101.\nWe meet on Mondays.'
const annA = { title: 'Welcome from the instructor', body: 'Read the syllabus before week 1.' }
const annB = { title: 'TA office hours', body: 'Fridays at noon.' }

// Wait until the dev mail log holds a message to `to` whose subject contains `text`.
async function expectMail(to: string, text: string) {
  await expect
    .poll(
      () =>
        fs
          .readFileSync(apiLog, 'utf8')
          .split('\n')
          .some((l) => {
            if (!l.includes('"msg":"mail"')) return false
            try {
              const m = JSON.parse(l)
              return m.to === to && String(m.subject).includes(text)
            } catch {
              return false // partial line
            }
          }),
      { timeout: 15_000 },
    )
    .toBe(true)
}

async function postAnnouncement(page: Page, a: { title: string; body: string }) {
  await page.getByRole('button', { name: 'New announcement' }).first().click()
  const dlg = page.getByRole('dialog')
  await dlg.getByLabel('Title').fill(a.title)
  await dlg.getByLabel('Message').fill(a.body)
  await expect(dlg.getByLabel('Also email everyone in this section')).toBeChecked()
  await dlg.getByRole('button', { name: 'Post announcement' }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.getByRole('article').filter({ hasText: a.title }).getByText('Emailed')).toBeVisible()
}

test('section content: description, announcements, materials, access', async ({ browser, page }) => {
  let sectionUrl = ''
  const as = async (email: string) => {
    const ctx = await browser.newContext()
    const p = await ctx.newPage()
    await signIn(p, email, slug)
    return p
  }

  await test.step('admin seeds department, term, course, section, members, staff and roster', async () => {
    await page.goto(createUniversity({ name: `Uni C ${runId}`, slug, domain, adminEmail: admin }))
    await signIn(page, admin, slug)

    await page.goto(`/${slug}/admin/departments`)
    await page.getByRole('button', { name: 'New department' }).first().click()
    await page.getByLabel('Name').fill('Computer Science')
    await page.getByLabel('Code').fill('CS')
    await page.getByRole('button', { name: 'Create department' }).click()
    await expect(page.getByRole('cell', { name: 'Computer Science', exact: true })).toBeVisible()

    await page.goto(`/${slug}/admin/terms`)
    await page.getByRole('button', { name: 'New term' }).first().click()
    await page.getByLabel('Name').fill(`Fall ${year}`)
    await page.getByLabel('Starts on').fill(`${year}-01-01`)
    await page.getByLabel('Ends on').fill(`${year}-12-31`)
    await page.getByRole('button', { name: 'Create term' }).click()
    await expect(page.getByRole('cell', { name: `Fall ${year}`, exact: true })).toBeVisible()

    await page.goto(`/${slug}/courses`)
    await page.getByRole('button', { name: 'New course' }).first().click()
    const dlg = page.getByRole('dialog')
    await dlg.getByLabel('Department').selectOption({ index: 1 })
    await dlg.getByLabel('Code').fill('CS101')
    await dlg.getByLabel('Title').fill('Intro to CS')
    await dlg.getByRole('button', { name: 'Create course' }).click()
    await expect(page.getByRole('cell', { name: 'CS101', exact: true })).toBeVisible()

    await page.goto(`/${slug}/courses/sections`)
    await page.getByRole('button', { name: 'New section' }).first().click()
    const sdlg = page.getByRole('dialog')
    await sdlg.getByLabel('Course').selectOption({ index: 1 })
    await sdlg.getByLabel('Section number').fill('001')
    await sdlg.getByRole('button', { name: 'Create section' }).click()
    await expect(page.getByRole('link', { name: 'CS101' })).toBeVisible()

    for (const email of [instr, ta, member]) {
      await page.goto(`/${slug}/admin/invitations`)
      await page
        .getByRole('button', { name: /New invitation|Invite/ })
        .first()
        .click()
      await page.getByLabel('Email').fill(email)
      await page.getByRole('button', { name: 'Send invitation' }).click()
      await expect(page.getByRole('cell', { name: email, exact: true })).toBeVisible()
    }

    await page.goto(`/${slug}/courses/sections`)
    await page.getByRole('link', { name: 'CS101' }).click()
    sectionUrl = page.url()
    await page.getByRole('link', { name: 'Participants' }).click()
    await page.getByLabel('Add by email').fill(instr)
    await page.getByRole('button', { name: 'Add instructor' }).click()
    await expect(page.getByText(`${instr} Invited`)).toBeVisible()
    await page.getByLabel('Add by email').fill(ta)
    await page.getByRole('button', { name: 'Add TA' }).click()
    await expect(page.getByText(`${ta} Invited`)).toBeVisible()

    await page.getByRole('button', { name: 'Import roster' }).first().click()
    const rdlg = page.getByRole('dialog')
    await rdlg.getByLabel('CSV file').setInputFiles({
      name: 'roster.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(['email,displayName', ...students.map((s) => `${s},Student`)].join('\n') + '\n'),
    })
    await rdlg.getByRole('button', { name: 'Import 2 rows' }).click()
    await expect(page.getByText(/Import finished: 2 invited/)).toBeVisible()
  })

  // Announcement mail goes only to active members, so everyone accepts their invitation first.
  await test.step('everyone invited signs in once', async () => {
    for (const email of [ta, ...students, member]) await (await as(email)).context().close()
  })

  const prof = await as(instr)

  await test.step('instructor writes the description', async () => {
    await prof.goto(sectionUrl)
    await prof.getByRole('button', { name: 'Add description' }).click()
    const dlg = prof.getByRole('dialog')
    await dlg.getByLabel('Description').fill(description)
    await dlg.getByRole('button', { name: 'Save description' }).click()
    await expect(prof.getByRole('dialog')).toHaveCount(0)
    await expect(prof.getByText('Description saved.')).toBeVisible()
    await expect(prof.getByText('Welcome to CS101.')).toBeVisible()
    await expect(prof.getByText('We meet on Mondays.')).toBeVisible()
  })

  await test.step('instructor posts an announcement, emailed', async () => {
    await prof.goto(`${sectionUrl}/announcements`)
    await postAnnouncement(prof, annA)
    await expectMail(students[0], annA.title)
  })

  await test.step('instructor adds Week 1 and uploads the PDF and the ZIP', async () => {
    await prof.goto(`${sectionUrl}/materials`)
    await prof.getByRole('button', { name: 'Add week' }).first().click()
    const dlg = prof.getByRole('dialog')
    await expect(dlg.getByLabel('Title')).toHaveValue('Week 1')
    await dlg.getByRole('button', { name: 'Add week' }).click()
    await expect(prof.getByRole('heading', { name: 'Week 1', level: 3 })).toBeVisible()

    await prof.getByLabel('Files for Week 1').setInputFiles([pdf, zip])
    // finalLine: "Uploaded N files to W." when every file made it ("N of M" only on partial failure)
    await expect(prof.getByRole('status').filter({ hasText: 'Uploaded 2 files to Week 1.' })).toBeVisible()
    await expect(prof.getByRole('link', { name: 'Download syllabus.pdf' })).toBeVisible()
    await expect(prof.getByRole('link', { name: 'Download code-samples.zip' })).toBeVisible()
    await expect(prof.getByRole('link', { name: /^Download / })).toHaveCount(2)
  })

  const taPage = await as(ta)

  await test.step('TA posts a second announcement and uploads the PPTX', async () => {
    await taPage.goto(`${sectionUrl}/announcements`)
    await postAnnouncement(taPage, annB)
    await taPage.goto(`${sectionUrl}/materials`)
    await taPage.getByLabel('Files for Week 1').setInputFiles(pptx)
    await expect(taPage.getByRole('status').filter({ hasText: 'Uploaded 1 file to Week 1.' })).toBeVisible()
    await expect(taPage.getByRole('link', { name: 'Download slides.pptx' })).toBeVisible()
  })

  await test.step('student reads everything and downloads the PDF intact', async () => {
    const s = await as(students[0])
    await s.goto(sectionUrl)
    await expect(s.getByText('We meet on Mondays.')).toBeVisible()
    await expect(s.getByRole('button', { name: /description/ })).toHaveCount(0)

    await s.getByRole('link', { name: 'Announcements' }).click()
    const titles = s.getByRole('article').getByRole('heading')
    await expect(titles).toHaveText([annB.title, annA.title])

    await s.getByRole('link', { name: 'Materials' }).click()
    await expect(s.getByRole('heading', { name: 'Week 1', level: 3 })).toBeVisible()
    await expect(s.getByRole('link', { name: /^Download / })).toHaveCount(3)

    const [download] = await Promise.all([
      s.waitForEvent('download'),
      s.getByRole('link', { name: 'Download syllabus.pdf' }).click(),
    ])
    const saved = await download.path()
    expect(fs.readFileSync(saved).equals(fs.readFileSync(pdf))).toBe(true)

    // no staff controls on either page
    await expect(
      s.getByRole('button', { name: /^(Add week|Edit |Rename |Move |Delete |Upload files|New announcement)/ }),
    ).toHaveCount(0)
    await s.goto(`${sectionUrl}/announcements`)
    await expect(s.getByRole('heading', { name: annA.title })).toBeVisible()
    await expect(
      s.getByRole('button', { name: /^(Add week|Edit |Rename |Move |Delete |Upload files|New announcement)/ }),
    ).toHaveCount(0)
    await s.context().close()
  })

  await test.step('a member who is neither staff nor enrolled sees only the description', async () => {
    const m = await as(member)
    await m.goto(sectionUrl)
    await expect(m.getByText('We meet on Mondays.')).toBeVisible()
    await expect(m.getByRole('navigation', { name: 'Section pages' })).toHaveCount(0)
    for (const tab of ['announcements', 'materials']) {
      await m.goto(`${sectionUrl}/${tab}`)
      await expect(m.getByText('No access').first()).toBeVisible()
    }
    await m.context().close()
  })

  await prof.context().close()
  await taPage.context().close()
})
