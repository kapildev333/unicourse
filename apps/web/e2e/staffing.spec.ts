import { expect, test, type Page } from '@playwright/test'
import { createUniversity, runId, signIn, signOut } from './helpers'

const slugA = `ea-${runId}`
const slugB = `eb-${runId}`
const dA = `a-${runId}.edu`
const dB = `b-${runId}.edu`
const adminA = `root@${dA}`
const instr = `prof@${dA}`
const ta = `ta@${dA}`
const students = [1, 2, 3].map((i) => `s${i}@${dA}`)
const year = String(new Date().getUTCFullYear())

const csv = (rows: string[]) => ({ name: 'roster.csv', mimeType: 'text/csv', buffer: Buffer.from(['email,displayName', ...rows].join('\n') + '\n') })

async function openSection(page: Page) {
  await page.goto(`/${slugA}/courses/sections`)
  await page.getByRole('link', { name: 'CS101' }).click()
}

test('staffing flow across two universities', async ({ browser, page }) => {
  const sectionUrl = { v: '' }

  await test.step('create university A and sign in as its admin', async () => {
    const url = createUniversity({ name: `Uni A ${runId}`, slug: slugA, domain: dA, adminEmail: adminA })
    await page.goto(url)
    await expect(page.getByLabel('Email', { exact: true })).toHaveValue(adminA)
    await signIn(page, adminA, slugA)
  })

  await test.step('department and term', async () => {
    await page.goto(`/${slugA}/admin/departments`)
    await page.getByRole('button', { name: 'New department' }).first().click()
    await page.getByLabel('Name').fill('Computer Science')
    await page.getByLabel('Code').fill('CS')
    await page.getByRole('button', { name: 'Create department' }).click()
    await expect(page.getByRole('cell', { name: 'Computer Science', exact: true })).toBeVisible()

    await page.goto(`/${slugA}/admin/terms`)
    await page.getByRole('button', { name: 'New term' }).first().click()
    await page.getByLabel('Name').fill(`Fall ${year}`)
    await page.getByLabel('Starts on').fill(`${year}-01-01`)
    await page.getByLabel('Ends on').fill(`${year}-12-31`)
    await page.getByRole('button', { name: 'Create term' }).click()
    await expect(page.getByRole('cell', { name: `Fall ${year}`, exact: true })).toBeVisible()
  })

  await test.step('course and section', async () => {
    await page.goto(`/${slugA}/courses`)
    await page.getByRole('button', { name: 'New course' }).first().click()
    const dlg = page.getByRole('dialog')
    await dlg.getByLabel('Department').selectOption({ index: 1 })
    await dlg.getByLabel('Code').fill('CS101')
    await dlg.getByLabel('Title').fill('Intro to CS')
    await dlg.getByRole('button', { name: 'Create course' }).click()
    await expect(page.getByRole('cell', { name: 'CS101', exact: true })).toBeVisible()

    await page.goto(`/${slugA}/courses/sections`)
    await page.getByRole('button', { name: 'New section' }).first().click()
    const sdlg = page.getByRole('dialog')
    await sdlg.getByLabel('Course').selectOption({ index: 1 })
    await sdlg.getByLabel('Section number').fill('001')
    await sdlg.getByRole('button', { name: 'Create section' }).click()
    await expect(page.getByRole('link', { name: 'CS101' })).toBeVisible()
  })

  await test.step('invite instructor as member, add instructor and TA by email', async () => {
    await page.goto(`/${slugA}/admin/invitations`)
    await page.getByRole('button', { name: /New invitation|Invite/ }).first().click()
    await page.getByLabel('Email').fill(instr)
    await page.getByRole('button', { name: 'Send invitation' }).click()
    await expect(page.getByRole('cell', { name: instr, exact: true })).toBeVisible()

    await openSection(page)
    sectionUrl.v = page.url()
    await page.getByLabel('Add by email').fill(instr)
    await page.getByRole('button', { name: 'Add instructor' }).click()
    await expect(page.getByText(`${instr} Invited`)).toBeVisible()

    await page.getByLabel('Add by email').fill(ta)
    await page.getByRole('button', { name: 'Add TA' }).click()
    await expect(page.getByText(`${ta} Invited`)).toBeVisible()
  })

  await test.step('instructor sees the section', async () => {
    const ctx = await browser.newContext()
    const p = await ctx.newPage()
    await signIn(p, instr, slugA)
    await expect(p.getByRole('cell', { name: 'Instructor', exact: true })).toBeVisible()
    await expect(p.getByRole('link', { name: 'CS101' })).toBeVisible()
    await ctx.close()
  })

  await test.step('roster import: duplicate line blocks, clean file imports', async () => {
    await page.goto(sectionUrl.v)
    await page.getByRole('button', { name: 'Import roster' }).first().click()
    const dlg = page.getByRole('dialog')
    await dlg.getByLabel('CSV file').setInputFiles(csv([...students.map((s) => `${s},Student`), `${students[0]},Student`]))
    await expect(dlg.getByText(/duplicate of line/)).toBeVisible()
    await expect(dlg.getByRole('button', { name: /^Import/ })).toBeDisabled()

    await dlg.getByLabel('CSV file').setInputFiles(csv(students.map((s) => `${s},Student`)))
    await expect(dlg.getByText(/3 to invite/)).toBeVisible()
    await dlg.getByRole('button', { name: 'Import 3 rows' }).click()
    await expect(page.getByText(/Import finished: 3 invited/)).toBeVisible()
  })

  await test.step('student signs in and appears on the roster', async () => {
    const ctx = await browser.newContext()
    const p = await ctx.newPage()
    await signIn(p, students[0], slugA)
    await expect(p.getByRole('link', { name: 'CS101' })).toBeVisible()
    await ctx.close()
    await page.goto(sectionUrl.v)
    await expect(page.getByRole('cell', { name: students[0], exact: true }).first()).toBeVisible()
  })

  await test.step('TA accepts the invitation and sees the roster without instructor controls', async () => {
    await signOut(page)
    await signIn(page, ta, slugA)
    await expect(page.getByRole('cell', { name: 'TA', exact: true })).toBeVisible()
    await page.getByRole('link', { name: 'CS101' }).click()
    await expect(page.getByRole('cell', { name: students[0], exact: true }).first()).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add instructor' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: `Remove ${instr} as Instructor` })).toHaveCount(0)
  })

  await test.step('university B sees none of it', async () => {
    const adminB = `root@${dB}`
    createUniversity({ name: `Uni B ${runId}`, slug: slugB, domain: dB, adminEmail: adminB })
    const ctx = await browser.newContext()
    const p = await ctx.newPage()
    await signIn(p, adminB, slugB)
    await expect(p.getByText('No sections yet')).toBeVisible()
    await p.goto(`/${slugB}/courses`)
    await expect(p.getByRole('cell', { name: 'CS101', exact: true })).toHaveCount(0)
    await p.goto(`/${slugB}/courses/sections`)
    await expect(p.getByRole('link', { name: 'CS101' })).toHaveCount(0)
    await p.goto(`/${slugA}/me`)
    await expect(p.getByRole('heading', { name: "You don't have access to this university" })).toBeVisible()
    const res = await p.request.get(`/api/v1/universities/${slugA}/sections`)
    expect(res.status()).toBe(403)
    await ctx.close()
  })
})
