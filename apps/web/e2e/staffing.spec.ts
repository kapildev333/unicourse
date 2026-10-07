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
const year = String(new Date().getFullYear()) // the app picks the term by local date

const csv = (rows: string[]) => ({ name: 'roster.csv', mimeType: 'text/csv', buffer: Buffer.from(['email,displayName', ...rows].join('\n') + '\n') })

async function openSection(page: Page) {
  await page.goto(`/${slugA}/courses/sections`)
  await page.getByRole('link', { name: 'CS101' }).click()
}

// Fill the section dialog's single Monday meeting row (the dialog is already open).
async function mondayMeeting(page: Page, starts: string, ends: string, location: string) {
  const dlg = page.getByRole('dialog')
  await dlg.getByRole('button', { name: 'Add meeting time' }).click()
  const row = dlg.getByRole('group', { name: 'Meeting 1' })
  await row.getByLabel('Day').selectOption({ label: 'Monday' })
  await row.getByLabel('Starts').fill(starts)
  await row.getByLabel('Ends').fill(ends)
  await row.getByLabel('Location').fill(location)
}

test('staffing flow across two universities', async ({ browser, page }) => {
  const sectionUrl = { v: '' }

  await test.step('create university A and sign in as its admin', async () => {
    const url = createUniversity({ name: `Uni A ${runId}`, slug: slugA, domain: dA, adminEmail: adminA })
    await page.goto(url)
    await expect(page.getByLabel('University email or university ID')).toHaveValue(adminA)
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
    await page.getByRole('link', { name: 'Participants' }).click()
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
    await expect(p.getByRole('listitem').filter({ hasText: 'CS101' })).toContainText('Instructor')
    await ctx.close()
  })

  await test.step('roster import: duplicate line blocks, clean file imports', async () => {
    await page.goto(`${sectionUrl.v}/participants`)
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

  await test.step('every student signs in, sees the section, and the roster lists all three', async () => {
    for (const s of students) {
      const ctx = await browser.newContext()
      const p = await ctx.newPage()
      await signIn(p, s, slugA)
      await expect(p.getByRole('listitem').filter({ hasText: 'CS101' })).toContainText('Student')
      await ctx.close()
    }
    await page.goto(`${sectionUrl.v}/participants`)
    for (const s of students) await expect(page.getByRole('cell', { name: s, exact: true }).first()).toBeVisible()
  })

  await test.step('admin: overlapping Monday meetings on CS101 and CS102, a clashing MA201 taught by the professor', async () => {
    await page.goto(`/${slugA}/admin/departments`)
    await page.getByRole('button', { name: 'New department' }).first().click()
    await page.getByLabel('Name').fill('Mathematics')
    await page.getByLabel('Code').fill('MA')
    await page.getByRole('button', { name: 'Create department' }).click()
    await expect(page.getByRole('cell', { name: 'Mathematics', exact: true })).toBeVisible()

    await page.goto(`/${slugA}/courses`)
    for (const [dept, code, title] of [
      ['CS · Computer Science', 'CS102', 'Data Structures'],
      ['MA · Mathematics', 'MA201', 'Linear Algebra'],
    ]) {
      await page.getByRole('button', { name: 'New course' }).first().click()
      const dlg = page.getByRole('dialog')
      await dlg.getByLabel('Department').selectOption({ label: dept })
      await dlg.getByLabel('Code').fill(code)
      await dlg.getByLabel('Title').fill(title)
      await dlg.getByRole('button', { name: 'Create course' }).click()
      await expect(page.getByRole('cell', { name: code, exact: true })).toBeVisible()
    }

    await page.goto(`/${slugA}/courses/sections`)
    await page.getByRole('button', { name: 'Edit CS101 section 001' }).click()
    await mondayMeeting(page, '10:00', '11:15', 'Room 101')
    await page.getByRole('dialog').getByRole('button', { name: 'Save changes' }).click()
    await expect(page.getByRole('dialog')).toHaveCount(0)

    for (const [course, starts, ends, room] of [
      ['CS102 · Data Structures', '10:30', '12:00', 'Room 202'],
      ['MA201 · Linear Algebra', '11:00', '12:00', 'Room 303'],
    ]) {
      await page.getByRole('button', { name: 'New section' }).first().click()
      const dlg = page.getByRole('dialog')
      await dlg.getByLabel('Course').selectOption({ label: course })
      await dlg.getByLabel('Section number').fill('001')
      await mondayMeeting(page, starts, ends, room)
      await dlg.getByRole('button', { name: 'Create section' }).click()
      await expect(page.getByRole('dialog')).toHaveCount(0)
    }
    await expect(page.getByRole('link', { name: 'CS102' })).toBeVisible()
    await expect(page.getByRole('link', { name: 'MA201' })).toBeVisible()
  })

  await test.step('admin: roster CSV puts the student in CS102, the professor teaches MA201', async () => {
    await page.getByRole('link', { name: 'CS102' }).click()
    await page.getByRole('link', { name: 'Participants' }).click()
    await page.getByRole('button', { name: 'Import roster' }).first().click()
    const dlg = page.getByRole('dialog')
    await dlg.getByLabel('CSV file').setInputFiles(csv([`${students[0]},Student`]))
    await expect(dlg.getByRole('button', { name: /^Import 1 row/ })).toBeEnabled()
    await dlg.getByRole('button', { name: /^Import 1 row/ }).click()
    await expect(page.getByText(/Import finished/)).toBeVisible()
    await expect(page.getByRole('cell', { name: students[0], exact: true }).first()).toBeVisible()
    await page.getByLabel('Add by email').fill(instr)
    await page.getByRole('button', { name: 'Add instructor' }).click()
    await expect(page.getByRole('button', { name: `Remove ${instr} as Instructor` })).toBeVisible()

    await page.goto(`/${slugA}/courses/sections`)
    await page.getByRole('link', { name: 'MA201' }).click()
    await page.getByRole('link', { name: 'Participants' }).click()
    await page.getByLabel('Add by email').fill(instr)
    await page.getByRole('button', { name: 'Add instructor' }).click()
    await expect(page.getByRole('button', { name: `Remove ${instr} as Instructor` })).toBeVisible()
  })

  await test.step('student: schedule shows both sections, times and clash lines', async () => {
    const ctx = await browser.newContext()
    const p = await ctx.newPage()
    await signIn(p, students[0], slugA)
    await p.getByRole('button', { name: 'Schedule' }).click()
    const monday = p.getByRole('region', { name: 'Monday' })
    await expect(p.getByRole('region', { name: 'Sunday' })).toBeVisible()
    await expect(monday.getByRole('link', { name: /^CS101-001/ })).toContainText('10:00–11:15')
    await expect(monday.getByRole('link', { name: /^CS101-001/ })).toContainText('Clashes with CS102-001')
    await expect(monday.getByRole('link', { name: /^CS102-001/ })).toContainText('10:30–12:00')
    await expect(monday.getByRole('link', { name: /^CS102-001/ })).toContainText('Clashes with CS101-001')
    await expect(monday.getByRole('link', { name: /^CS101-001/ })).toContainText(instr)
    await expect(monday.getByRole('link', { name: /^CS102-001/ })).toContainText(instr)

    await expect(p.getByRole('link', { name: 'Courses' })).toHaveCount(0)
    await expect(p.getByRole('link', { name: 'Admin' })).toHaveCount(0)

    await p.getByRole('link', { name: 'Catalog' }).click()
    await p.getByRole('searchbox', { name: 'Find a course' }).fill('MA201')
    await p.keyboard.press('Enter')
    await expect(p.getByRole('listitem')).toHaveCount(1)
    const row = p.getByRole('listitem').filter({ hasText: 'MA201-001' })
    await expect(row).toContainText(instr)
    await expect(row).toContainText('Mon 11:00–12:00')
    await expect(row).toContainText('Clashes with CS101-001, CS102-001')

    await row.getByRole('link', { name: 'MA201-001' }).click()
    await expect(p.getByRole('heading', { name: 'Meeting times' })).toBeVisible()
    await expect(p.getByText('Monday 11:00–12:00 · Room 303')).toBeVisible()
    const strip = p.getByRole('navigation', { name: 'Section pages' })
    await expect(strip.getByRole('link')).toHaveText(['Home', 'Participants'])
    await strip.getByRole('link', { name: 'Participants' }).click()
    await expect(p.getByRole('heading', { name: /^Staff/ })).toBeVisible()
    await expect(p.getByRole('heading', { name: 'Roster' })).toHaveCount(0)
    await ctx.close()
  })

  await test.step('TA accepts the invitation and sees the roster without instructor controls', async () => {
    await signOut(page)
    await signIn(page, ta, slugA)
    await expect(page.getByRole('listitem').filter({ hasText: 'CS101' })).toContainText('TA')
    await page.getByRole('link', { name: 'CS101' }).click()
    await page.getByRole('link', { name: 'Participants' }).click()
    for (const s of students) await expect(page.getByRole('cell', { name: s, exact: true }).first()).toBeVisible()
    await expect(page.getByRole('button', { name: 'Add instructor' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: `Remove ${instr} as Instructor` })).toHaveCount(0)
  })

  await test.step('university B sees none of it', async () => {
    const adminB = `root@${dB}`
    createUniversity({ name: `Uni B ${runId}`, slug: slugB, domain: dB, adminEmail: adminB })
    const ctx = await browser.newContext()
    const p = await ctx.newPage()
    await signIn(p, adminB, slugB)
    await expect(p.getByText('No terms yet')).toBeVisible()
    await p.goto(`/${slugB}/courses`)
    await expect(p.getByRole('button', { name: 'New course' }).first()).toBeVisible()
    await expect(p.getByRole('cell', { name: 'CS101', exact: true })).toHaveCount(0)
    await p.goto(`/${slugB}/courses/sections`)
    await expect(p.getByRole('button', { name: 'New section' }).first()).toBeVisible()
    await expect(p.getByRole('link', { name: 'CS101' })).toHaveCount(0)
    await p.goto(`/${slugA}/me`)
    await expect(p.getByRole('heading', { name: "You don't have access to this university" })).toBeVisible()
    const res = await p.request.get(`/api/v1/universities/${slugA}/sections`)
    expect(res.status()).toBe(403)
    const sectionId = new URL(sectionUrl.v).pathname.split('/').filter(Boolean).pop()!
    const own = `/api/v1/universities/${slugB}/sections/${sectionId}`
    expect((await p.request.get(own)).status()).toBe(404)
    expect((await p.request.get(`${own}/enrollments`)).status()).toBe(404)
    await ctx.close()
  })
  // Spec 4 (grading). Credits: the course dialog has no credits field on this branch, so they are set
  // with PATCH /courses/{id} through the admin's session cookie (page.request).
  const api = `/api/v1/universities/${slugA}`
  const gradesUrl = (id: string) => `/${slugA}/sections/${id}/grades`
  const ids = { cs101: new URL(sectionUrl.v).pathname.split('/').filter(Boolean).pop()!, ma201: '' }
  const [s1, s2, s3] = students
  const row = (p: Page, email: string) => p.getByRole('row', { name: new RegExp(email) })
  const newAssessment = async (p: Page, name: string, max: string, weight: string) => {
    await p.getByRole('button', { name: 'New assessment' }).first().click()
    const dlg = p.getByRole('dialog')
    await dlg.getByLabel('Name').fill(name)
    await dlg.getByLabel('Max points').fill(max)
    await dlg.getByLabel('Weight (%)').fill(weight)
    await dlg.getByRole('button', { name: 'Create assessment' }).click()
    await expect(p.getByRole('button', { name: `Edit ${name}` })).toBeVisible()
  }

  await test.step('grading: credits 4 on CS101, 2 on MA201, student enrolled in MA201', async () => {
    const ctx = await browser.newContext()
    const p = await ctx.newPage()
    await signIn(p, adminA, slugA)
    const list = await (await p.request.get(`${api}/courses`)).json()
    for (const [code, credits] of [['CS101', 4], ['MA201', 2]] as const) {
      const c = list.items.find((x: { code: string }) => x.code === code)
      const res = await p.request.patch(`${api}/courses/${c.id}`, { data: { credits } })
      expect(res.status()).toBe(200)
    }
    await p.goto(`/${slugA}/courses/sections`)
    await p.getByRole('link', { name: 'MA201' }).click()
    ids.ma201 = new URL(p.url()).pathname.split('/').filter(Boolean).pop()!
    await p.getByRole('link', { name: 'Participants' }).click()
    await p.getByRole('button', { name: 'Import roster' }).first().click()
    const dlg = p.getByRole('dialog')
    await dlg.getByLabel('CSV file').setInputFiles(csv([`${s1},Student`]))
    await dlg.getByRole('button', { name: /^Import 1 row/ }).click()
    await expect(p.getByText(/Import finished/)).toBeVisible()
    await ctx.close()
  })

  const profCtx = await browser.newContext()
  const prof = await profCtx.newPage()
  await signIn(prof, instr, slugA)

  await test.step('instructor: CS101 weighted assessments, hand score, CSV scores, publish, finalize', async () => {
    await prof.goto(gradesUrl(ids.cs101))
    await newAssessment(prof, 'Midterm', '100', '40')
    await expect(prof.getByRole('button', { name: 'Finalize grades' })).toBeDisabled() // weights total 40%
    await newAssessment(prof, 'Final', '200', '60')
    await expect(prof.getByText('Weights total 100%')).toBeVisible()
    await expect(prof.getByRole('button', { name: 'Finalize grades' })).toBeEnabled()

    // by hand: s1 Midterm 95
    const cell = row(prof, s1).getByRole('textbox', { name: /Midterm$/ })
    await cell.fill('95')
    await cell.blur()
    await expect(row(prof, s1).getByText('Saved')).toBeVisible()

    // by CSV: Final scores for all three
    await prof.getByRole('button', { name: 'Import scores' }).click()
    const dlg = prof.getByRole('dialog')
    await dlg.getByLabel('Assessment').selectOption({ label: 'Final' })
    await dlg.getByLabel('CSV file').setInputFiles({
      name: 'final.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(`email,points\n${s1},180\n${s2},150\n${s3},120\n`),
    })
    await expect(dlg.getByRole('button', { name: 'Import 3 scores' })).toBeEnabled()
    await dlg.getByRole('button', { name: 'Import 3 scores' }).click()
    await expect(prof.getByRole('dialog')).toHaveCount(0)
    await expect(row(prof, s1).getByRole('textbox', { name: /Final$/ })).toHaveValue('180')
    await expect(row(prof, s3).getByRole('textbox', { name: /Final$/ })).toHaveValue('120')

    // publish both
    for (const n of ['Midterm', 'Final']) {
      const box = prof.getByRole('checkbox', { name: `Published ${n}` })
      await box.click()
      await expect(box).toBeChecked()
    }

    // s1: 0.4 * 95 + 0.6 * 90 = 92 -> A-; s2: 0.6 * 75 = 45 and s3: 0.6 * 60 = 36 (missing Midterm = 0) -> F
    await prof.getByRole('button', { name: 'Finalize grades' }).click()
    const fin = prof.getByRole('dialog')
    await expect(fin.getByText(/3 students · 2 with missing scores · Preview: A- 1 · F 2/)).toBeVisible()
    await fin.getByRole('button', { name: 'Finalize 3 grades' }).click()
    await expect(prof.getByRole('dialog')).toHaveCount(0)
    await expect(prof.getByRole('columnheader', { name: 'Final', exact: true })).toBeVisible()
    await expect(row(prof, s1)).toContainText('A- 3.70')
    await expect(row(prof, s2)).toContainText('F 0.00')
    await expect(prof.getByRole('button', { name: 'Reopen grades' })).toBeVisible()
  })

  await test.step('instructor: MA201 one assessment at 100%, score, finalize', async () => {
    await prof.goto(gradesUrl(ids.ma201))
    await newAssessment(prof, 'Exam', '100', '100')
    const cell = prof.getByRole('textbox', { name: /Exam$/ })
    await cell.fill('83')
    await cell.blur()
    await expect(prof.getByText('Saved', { exact: true })).toBeVisible()
    await prof.getByRole('button', { name: 'Finalize grades' }).click()
    await prof.getByRole('dialog').getByRole('button', { name: 'Finalize 1 grades' }).click()
    await expect(prof.getByRole('dialog')).toHaveCount(0)
    await expect(row(prof, s1)).toContainText('B 3.00') // 83% is exactly the B cut
  })

  await test.step('instructor: CSV export has the header and the student row', async () => {
    const res = await prof.request.get(`${api}/sections/${ids.cs101}/gradebook`, { headers: { Accept: 'text/csv' } })
    expect(res.status()).toBe(200)
    expect(res.headers()['content-type']).toContain('text/csv')
    const lines = (await res.text()).replace(/^﻿/, '').trim().split(/\r?\n/)
    expect(lines[0]).toBe('email,displayName,Midterm,Final,percent,letter,final')
    expect(lines.find((l) => l.startsWith(`${s1},`))).toBe(`${s1},Student,95.00,180.00,92.00,A-,A-`)
    await profCtx.close()
  })

  await test.step('student: final letters, term and cumulative GPA, published scores, no gradebook', async () => {
    const ctx = await browser.newContext()
    const p = await ctx.newPage()
    await signIn(p, s1, slugA)
    await p.goto(`/${slugA}/grades`)
    // CS101 A- 3.7 x 4 credits + MA201 B 3.0 x 2 credits: (14.8 + 6) / 6 = 3.4667 -> 3.47; CS102 has no final grade
    await expect(p.getByText('Term GPA 3.47 · 6.0 credits')).toBeVisible()
    await expect(p.getByText('Cumulative GPA').locator('..')).toContainText('3.47')
    const cs = p.getByRole('listitem').filter({ has: p.getByRole('link', { name: /^CS101-001/ }) })
    await expect(cs).toContainText('4.0 credits')
    await expect(cs).toContainText('A-3.70 grade points · 92.00%')
    await expect(cs.getByText('Published scores (2)')).toBeVisible()
    await cs.getByText('Published scores (2)').click()
    await expect(cs).toContainText('Midterm · 95 / 100 · 40%')
    await expect(cs).toContainText('Final · 180 / 200 · 60%')
    const ma = p.getByRole('listitem').filter({ has: p.getByRole('link', { name: /^MA201-001/ }) })
    await expect(ma).toContainText('2.0 credits')
    await expect(ma).toContainText('B3.00 grade points · 83.00%')
    await p.goto(gradesUrl(ids.cs101))
    await expect(p.getByText('No access').first()).toBeVisible()
    await expect(p.getByRole('button', { name: 'New assessment' })).toHaveCount(0)
    expect((await p.request.get(`${api}/sections/${ids.cs101}/gradebook`)).status()).toBe(403)
    await ctx.close()
  })
})
