import { expect, test, type Page } from '@playwright/test'
import { demoTerm, expectNoSideScroll, pageAs, personas, runId, sectionIds, seedDemo } from './helpers'

const slug = `e2e-grd-${runId}`
const P = personas(slug)
const A = `/api/v1/universities/${slug}`
let ids: Record<string, string> = {}

test.beforeAll(() => {
  seedDemo(slug)
})

async function gradebook(page: Page, code: string) {
  if (!ids[code]) ids = await sectionIds(page, slug)
  await page.goto(`/${slug}/sections/${ids[code]}/grades`)
  await expect(page.getByRole('heading', { name: 'Gradebook' }), 'gradebook page: heading').toBeVisible()
}

const row = (page: Page, name: string) => page.getByRole('row', { name: new RegExp(name) })
const cell = (page: Page, name: string, assessment: string) => page.getByRole('textbox', { name: `Score for ${name}, ${assessment}` })
const status = (page: Page, text: string) => page.getByRole('status').filter({ hasText: text })

test('E-GRADES-01 assessment create, edit, delete; weights block Finalize', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await gradebook(page, 'CS210-001')
  const dlg = page.getByRole('dialog', { name: 'New assessment' })

  await page.getByRole('button', { name: 'New assessment' }).first().click()
  await dlg.getByRole('button', { name: 'Create assessment' }).click()
  await expect(dlg.getByText('Enter a name of at most 100 characters.'), 'elena, CS210 New assessment: name error').toBeVisible()
  await expect(dlg.getByText('Enter maximum points greater than 0.'), 'elena, CS210 New assessment: max points error').toBeVisible()
  await expect(dlg.getByText('Enter a weight from 0 to 100.'), 'elena, CS210 New assessment: weight error').toBeVisible()

  await dlg.getByLabel('Name').fill('Final Project')
  await dlg.getByLabel('Max points').fill('100')
  await dlg.getByLabel('Weight').fill('10')
  await dlg.getByRole('button', { name: 'Create assessment' }).click()
  await expect(dlg.getByText('This section already has an assessment with this name.'), 'elena, CS210 New assessment: duplicate name').toBeVisible()

  await dlg.getByLabel('Name').fill('Quiz 1')
  await dlg.getByLabel('Max points').fill('20')
  await dlg.getByRole('button', { name: 'Create assessment' }).click()
  await expect(status(page, 'Quiz 1 created.'), 'elena, CS210 gradebook: created notice').toBeVisible()
  await expect(page.getByText('Weights total 110%'), 'elena, CS210 gradebook: weights total').toBeVisible()
  await expect(page.getByText('Must total 100% to finalize', { exact: true }), 'elena, CS210 gradebook: hint').toBeVisible()
  await expect(page.getByRole('button', { name: 'Finalize grades' }), 'elena, CS210 gradebook: Finalize disabled').toBeDisabled()
  await expect(page.getByText('Weights must total 100% to finalize. They total 110%.'), 'elena, CS210 gradebook: weights banner').toBeVisible()

  const res = await page.request.post(`${A}/sections/${ids['CS210-001']}/final-grades`, { data: {} })
  expect(res.status(), 'elena, CS210 API: finalize with weights 110').toBe(400)
  const body = await res.json()
  expect(body.error.code, 'elena, CS210 API: error code').toBe('validation_failed')
  expect(body.error.details.some((d: { field: string }) => d.field === 'weights'), 'elena, CS210 API: weights detail').toBeTruthy()

  await page.getByRole('button', { name: 'Edit Quiz 1' }).click()
  const edit = page.getByRole('dialog', { name: 'Edit Quiz 1' })
  await edit.getByLabel('Weight').fill('0')
  await edit.getByRole('button', { name: 'Save changes' }).click()
  await expect(status(page, 'Quiz 1 saved.'), 'elena, CS210 gradebook: saved notice').toBeVisible()
  await expect(page.getByRole('button', { name: 'Finalize grades' }), 'elena, CS210 gradebook: Finalize enabled').toBeEnabled()

  await page.getByRole('button', { name: 'Edit Quiz 1' }).click()
  await page.getByRole('dialog', { name: 'Edit Quiz 1' }).getByRole('button', { name: 'Delete assessment' }).click()
  const del = page.getByRole('dialog', { name: 'Delete Quiz 1?' })
  await expect(del.getByText("Delete Quiz 1? Its 0 scores are deleted too. This can't be undone."), 'elena, CS210 Delete dialog: copy').toBeVisible()
  await del.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(status(page, 'Quiz 1 deleted.'), 'elena, CS210 gradebook: deleted notice').toBeVisible()
  await page.context().close()
})

test('E-GRADES-02 score cell validation, bonus, blank', async ({ browser }) => {
  const page = await pageAs(browser, P.david.email, slug)
  await gradebook(page, 'ECON110-001')
  const mei = row(page, 'Mei-Ling Chou')
  const mid = cell(page, 'Mei-Ling Chou', 'Midterm Exam')

  await mid.fill('abc')
  await mid.blur()
  await expect(mei.getByText('Enter a number like 42 or 42.5.'), 'david, ECON110 gradebook: non-number').toBeVisible()
  await expect(mid, 'david, ECON110 gradebook: cell invalid').toHaveAttribute('aria-invalid', 'true')

  await mid.fill('250')
  await mid.blur()
  await expect(mei.getByText('Enter at most 200 (twice the maximum).'), 'david, ECON110 gradebook: over twice max').toBeVisible()

  await mid.fill('150')
  await mid.blur()
  await expect(mei.getByText('Saved'), 'david, ECON110 gradebook: bonus saved').toBeVisible()

  const fin = cell(page, 'Mei-Ling Chou', 'Final Exam')
  await fin.fill('')
  await fin.blur()
  await expect(mei.getByText('Saved'), 'david, ECON110 gradebook: blank saved').toBeVisible()
  await page.reload()
  await expect(row(page, 'Mei-Ling Chou').getByText('Missing', { exact: true }), 'david, ECON110 gradebook: Missing').toBeVisible()
  await expect(row(page, 'Mei-Ling Chou').getByText('(1 missing)'), 'david, ECON110 gradebook: 1 missing').toBeVisible()
  await page.context().close()
})

test('E-GRADES-03 scores CSV preview errors and commit', async ({ browser }) => {
  const page = await pageAs(browser, P.david.email, slug)
  await gradebook(page, 'BIO150-001')
  await page.getByRole('button', { name: 'Import scores' }).click()
  const dlg = page.getByRole('dialog', { name: 'Import scores' })
  await dlg.getByLabel('Assessment').selectOption({ label: 'Final Exam' })
  const csv = (rows: string[]) => ({ name: 'scores.csv', mimeType: 'text/csv', buffer: Buffer.from(['email,points', ...rows].join('\n') + '\n') })
  await dlg.getByLabel('CSV file').setInputFiles(csv([`nobody@nowhere-${runId}.test,10`, `${P.jonas.email},10`, `${P.priya.email},abc`]))

  await expect(dlg.getByText('3 rows checked: 0 to add, 0 to update, 0 unchanged, 3 with errors.'), 'david, BIO150 Import scores: summary').toBeVisible()
  await expect(dlg.getByRole('cell', { name: 'unknown email' }), 'david, BIO150 Import scores: unknown email').toBeVisible()
  await expect(dlg.getByRole('cell', { name: 'not enrolled in this section' }), 'david, BIO150 Import scores: not enrolled').toBeVisible()
  await expect(dlg.getByRole('cell', { name: /^points / }), 'david, BIO150 Import scores: bad points').toBeVisible()
  await expect(dlg.getByRole('button', { name: 'Import 0 scores' }), 'david, BIO150 Import scores: disabled').toBeDisabled()
  await expect(dlg.getByText('Fix the 3 rows with errors and choose the file again.'), 'david, BIO150 Import scores: hint').toBeVisible()

  await dlg.getByLabel('CSV file').setInputFiles(csv([`${P.priya.email},99.5`, `${P.daniel.email},12.25`]))
  await dlg.getByRole('button', { name: 'Import 2 scores' }).click()
  await expect(status(page, 'Scores imported for Final Exam: 0 added, 2 updated.'), 'david, BIO150 gradebook: import notice').toBeVisible()
  await expect(cell(page, 'Priya Raman', 'Final Exam'), 'david, BIO150 gradebook: imported value').toHaveValue('99.5')
  await page.context().close()
})

test('E-GRADES-04 publish and unpublish, what the student sees', async ({ browser }) => {
  const david = await pageAs(browser, P.david.email, slug)
  const priya = await pageAs(browser, P.priya.email, slug)
  await gradebook(david, 'BIO150-001')
  const box = david.getByRole('checkbox', { name: 'Published Lab Practical' })
  await box.click()
  await expect(box, 'david, BIO150 gradebook: Lab Practical unchecked').not.toBeChecked()
  await expect(status(david, 'Lab Practical unpublished. Students no longer see its scores.'), 'david, BIO150 gradebook: unpublished').toBeVisible()

  await priya.goto(`/${slug}/grades`)
  const card = priya.getByRole('listitem').filter({ has: priya.getByRole('link', { name: /^BIO150-001/ }) })
  await expect(card.getByText('No grade yet'), 'priya, My grades: no grade').toBeVisible()
  await expect(card.getByText('Published scores (1)'), 'priya, My grades: 1 published').toBeVisible()

  await box.click()
  await expect(status(david, 'Lab Practical published. Students can now see its scores.'), 'david, BIO150 gradebook: published').toBeVisible()
  await priya.reload()
  await expect(card.getByText('Current'), 'priya, My grades: Current').toBeVisible()
  await expect(card.getByText('% so far'), 'priya, My grades: % so far').toBeVisible()
  await expect(card.getByText('Counts published scores only.'), 'priya, My grades: published only').toBeVisible()
  await expect(card.getByText('Published scores (2)'), 'priya, My grades: 2 published').toBeVisible()
  await david.context().close()
  await priya.context().close()
})

test('E-GRADES-05 finalize with an override, change a final grade, reopen', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await gradebook(page, 'MATH201-001')
  await page.getByRole('button', { name: 'Finalize grades' }).click()
  const dlg = page.getByRole('dialog', { name: 'Finalize grades for MATH201-001' })
  await expect(dlg.getByText(/^6 students · 1 with missing scores · Preview:/), 'elena, MATH201 Finalize dialog: summary').toBeVisible()
  await dlg.getByLabel('Grade for Fatima Al-Sayed').selectOption({ label: 'Withdrawn' })
  await dlg.getByRole('button', { name: 'Finalize 6 grades' }).click()
  await expect(dlg.getByText('Enter a reason for this change.'), 'elena, MATH201 Finalize dialog: reason required').toBeVisible()
  await dlg.getByLabel('Reason for Fatima Al-Sayed').fill('Withdrew in week 9')
  await dlg.getByRole('button', { name: 'Finalize 6 grades' }).click()
  await expect(status(page, 'Final grades recorded for 6 students.'), 'elena, MATH201 gradebook: finalized notice').toBeVisible()

  await expect(page.getByRole('heading', { name: 'Gradebook' }), 'elena, MATH201 gradebook: Finalized').toContainText('Finalized')
  for (const h of ['Running %', 'Running letter', 'Final']) {
    await expect(page.getByRole('columnheader', { name: h, exact: true }), `elena, MATH201 gradebook: column ${h}`).toBeVisible()
  }
  await expect(row(page, 'Fatima Al-Sayed'), 'elena, MATH201 gradebook: Fatima withdrawn').toContainText('Withdrawn')
  await expect(row(page, 'Fatima Al-Sayed'), 'elena, MATH201 gradebook: Fatima changed').toContainText('Changed')

  const res = await page.request.post(`${A}/sections/${ids['MATH201-001']}/final-grades`, { data: {} })
  expect(res.status(), 'elena, MATH201 API: finalize twice').toBe(409)

  await page.getByRole('button', { name: 'Change final grade for Amara Okafor' }).click()
  const ch = page.getByRole('dialog', { name: 'Change final grade for Amara Okafor' })
  await ch.getByLabel('Grade').selectOption({ label: 'B+' })
  await ch.getByRole('button', { name: 'Change grade' }).click()
  await expect(ch.getByText('Enter a reason for this change.'), 'elena, MATH201 Change dialog: reason required').toBeVisible()
  await ch.getByLabel('Reason').fill('Regraded final')
  await ch.getByRole('button', { name: 'Change grade' }).click()
  await expect(status(page, 'Final grade for Amara Okafor changed to B+.'), 'elena, MATH201 gradebook: changed notice').toBeVisible()
  await expect(row(page, 'Amara Okafor'), 'elena, MATH201 gradebook: Amara B+').toContainText('B+')

  await page.getByRole('button', { name: 'Reopen grades' }).click()
  const re = page.getByRole('dialog', { name: 'Reopen grades for MATH201-001?' })
  await expect(re.getByText("This deletes the 6 final grades. Students stop seeing their final letters until you finalize again. Scores and assessments are kept."), 'elena, MATH201 Reopen dialog: copy').toBeVisible()
  await re.getByRole('button', { name: 'Reopen grades' }).click()
  await expect(status(page, 'Grades reopened. Finalize again when scores are ready.'), 'elena, MATH201 gradebook: reopened notice').toBeVisible()
  await expect(page.getByRole('button', { name: 'Finalize grades' }), 'elena, MATH201 gradebook: Finalize back').toBeVisible()
  await expect(page.getByRole('columnheader', { name: 'Percent', exact: true }), 'elena, MATH201 gradebook: Percent column').toBeVisible()
  await page.context().close()
})

test('E-GRADES-06 My grades with an Incomplete and no GPA yet', async ({ browser }) => {
  const page = await pageAs(browser, P.lucas.email, slug)
  await page.goto(`/${slug}/grades`)
  await expect(page.getByRole('heading', { name: 'My grades' }), 'lucas, My grades: heading').toBeVisible()
  await expect(page.getByText('Your GPA appears once a section has a final grade.'), 'lucas, My grades: GPA hint').toBeVisible()
  await expect(page.getByText('Counts finalized sections only. 7.0 credits in progress.'), 'lucas, My grades: credits').toBeVisible()
  await expect(page.getByText('No term GPA yet'), 'lucas, My grades: term GPA').toBeVisible()
  const card = (code: string) => page.getByRole('listitem').filter({ has: page.getByRole('link', { name: new RegExp(`^${code}`) }) })
  await expect(card('CS101-001'), 'lucas, My grades: CS101 Incomplete').toContainText('Incomplete')
  await expect(card('CS101-001'), 'lucas, My grades: CS101 not counted').toContainText('Not counted in GPA')
  await expect(card('ECON110-001'), 'lucas, My grades: ECON110 Current').toContainText('Current')
  await expect(card('ECON110-001'), 'lucas, My grades: ECON110 so far').toContainText('% so far')
  await expect(page.getByRole('heading', { name: 'Gradebooks' }), 'lucas, My grades: no Gradebooks').toHaveCount(0)
  await page.getByText('How letters and grade points are assigned').click()
  await expect(page.getByRole('table', { name: 'Grade scale' }), 'lucas, My grades: scale table').toBeVisible()
  await page.context().close()
})

test('E-GRADES-07 what a TA may do', async ({ browser }) => {
  const page = await pageAs(browser, P.sofia.email, slug)
  await gradebook(page, 'CS101-001')
  for (const name of ['New assessment', 'Finalize grades', 'Reopen grades']) {
    await expect(page.getByRole('button', { name }), `sofia, CS101 gradebook: no ${name}`).toHaveCount(0)
  }
  await expect(page.getByRole('checkbox', { name: /^Published/ }), 'sofia, CS101 gradebook: no publish checkbox').toHaveCount(0)
  await expect(page.getByText('Published').first(), 'sofia, CS101 gradebook: Published badge').toBeVisible()
  await expect(page.getByRole('button', { name: 'Import scores' }), 'sofia, CS101 gradebook: Import scores').toBeVisible()
  await expect(page.getByRole('button', { name: 'Export CSV' }), 'sofia, CS101 gradebook: Export CSV').toBeVisible()
  await expect(page.getByText('Final grades are recorded. Score changes do not change them until an instructor reopens grades.'), 'sofia, CS101 gradebook: finalized note').toBeVisible()

  const c = cell(page, 'Amara Okafor', 'Final Exam')
  await c.fill('81')
  await c.blur()
  await expect(row(page, 'Amara Okafor').getByText('Saved'), 'sofia, CS101 gradebook: score saved').toBeVisible()

  const [d] = await Promise.all([page.waitForEvent('download'), page.getByRole('button', { name: 'Export CSV' }).click()])
  expect(d.suggestedFilename(), 'sofia, CS101 gradebook: export file name').toBe('CS101-001-gradebook.csv')

  const a = await page.request.post(`${A}/sections/${ids['CS101-001']}/assessments`, { data: { name: 'X', maxPoints: 10, weight: 0 } })
  expect(a.status(), 'sofia, CS101 API: create assessment').toBe(403)
  const f = await page.request.post(`${A}/sections/${ids['CS101-001']}/final-grades`, { data: {} })
  expect(f.status(), 'sofia, CS101 API: finalize').toBe(403)
  await page.context().close()
})

test('E-GRADES-08 a student never reaches the gradebook', async ({ browser }) => {
  const page = await pageAs(browser, P.amara.email, slug)
  ids = Object.keys(ids).length ? ids : await sectionIds(page, slug)
  await page.goto(`/${slug}/sections/${ids['CS101-001']}`)
  await expect(page.getByRole('navigation', { name: 'Section pages' }).getByRole('link'), 'amara, CS101 section: tabs').toHaveText(['Home', 'Announcements', 'Materials', 'Participants'])
  await page.goto(`/${slug}/me`)
  const card = page.getByRole('list', { name: `My sections in ${demoTerm}` }).getByRole('listitem').filter({ hasText: 'CS101-001' })
  await expect(card, 'amara, My sections: CS101-001 card').toBeVisible()
  await expect(card.getByRole('link', { name: 'Gradebook' }), 'amara, My sections: no Gradebook link').toHaveCount(0)
  await page.context().close()
})

test('E-GRADES-09 department admin powers on own and other departments', async ({ browser }) => {
  const page = await pageAs(browser, P.deptAdmin.email, slug)
  await gradebook(page, 'CS101-001')
  await expect(page.getByRole('button', { name: 'Reopen grades' }), 'owen, CS101 gradebook: Reopen').toBeVisible()
  await expect(page.getByRole('button', { name: 'New assessment' }).first(), 'owen, CS101 gradebook: New assessment').toBeVisible()
  await expect(page.getByRole('link', { name: 'Grade scale' }), 'owen, CS101 gradebook: Grade scale link').toBeVisible()
  await page.goto(`/${slug}/sections/${ids['ECON110-001']}/grades`)
  await expect(page.getByRole('alert'), 'owen, ECON110 gradebook: No access').toContainText('No access')
  await expect(page.getByRole('alert'), 'owen, ECON110 gradebook: copy').toContainText("Only this section's staff and administrators can open its gradebook.")
  await page.context().close()
})

test('E-GRADES-11 members read the scale only', async ({ browser }) => {
  const page = await pageAs(browser, P.amara.email, slug)
  await page.goto(`/${slug}/grade-scale`)
  await expect(page.getByRole('table', { name: 'Grade scale' }), 'amara, Grade scale: table').toBeVisible()
  await expect(page.getByRole('button', { name: 'Save scale' }), 'amara, Grade scale: no Save').toHaveCount(0)
  await expect(page.getByRole('main').getByRole('textbox'), 'amara, Grade scale: no inputs').toHaveCount(0)
  await page.context().close()
})

test('E-GRADES-12 gradebook at phone width', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await page.setViewportSize({ width: 390, height: 844 })
  await gradebook(page, 'CS101-001')
  await expectNoSideScroll(page, 'elena, CS101 gradebook')
  const b = await cell(page, 'Amara Okafor', 'Midterm Exam').boundingBox()
  expect(b!.x + b!.width, 'elena, CS101 gradebook: first score cell fits at 390').toBeLessThanOrEqual(390)
  await page.context().close()
})

// Runs last: it changes the university's scale.
test('E-GRADES-10 grade scale editor: validation, preset, save, snapshot', async ({ browser }) => {
  const page = await pageAs(browser, P.admin.email, slug)
  await page.goto(`/${slug}/grade-scale`)
  await expect(page.getByRole('heading', { name: 'Grade scale' }), 'admin, Grade scale: heading').toBeVisible()
  await page.getByRole('button', { name: 'Add row' }).click()
  await expect(page.getByText('Unsaved changes'), 'admin, Grade scale: unsaved').toBeVisible()
  await page.getByRole('button', { name: 'Save scale' }).click()
  await expect(page.getByText('Enter a letter.'), 'admin, Grade scale: letter required').toBeVisible()

  await page.getByRole('textbox', { name: 'Minimum percent, row 2' }).fill('93')
  await page.getByRole('button', { name: 'Save scale' }).click()
  await expect(page.getByText('Each minimum percent can appear only once.'), 'admin, Grade scale: duplicate minimum').toBeVisible()

  await page.getByRole('button', { name: 'Discard changes' }).click()
  await expect(page.getByText('Unsaved changes'), 'admin, Grade scale: discarded').toHaveCount(0)

  await page.getByRole('button', { name: 'Remove row 12' }).click()
  await page.getByRole('button', { name: 'Save scale' }).click()
  await expect(page.getByRole('alert'), 'admin, Grade scale: zero row').toContainText('One row must start at 0 so every percent gets a letter.')
  await page.getByRole('button', { name: 'Discard changes' }).click()

  await page.getByRole('button', { name: 'Load Canadian preset' }).click()
  await expect(page.getByRole('textbox', { name: 'Letter, row 1', exact: true }), 'admin, Grade scale: Canadian row 1').toHaveValue('A+')
  await page.getByRole('button', { name: 'Save scale' }).click()
  await expect(status(page, 'Grade scale saved. Final grades already recorded keep their letters.'), 'admin, Grade scale: saved').toBeVisible()

  await gradebook(page, 'CS101-001')
  await expect(row(page, 'Amara Okafor').getByText('B', { exact: true }), 'admin, CS101 gradebook: snapshot letter').toBeVisible()

  await page.goto(`/${slug}/grade-scale`)
  await page.getByRole('button', { name: 'Load US preset' }).click()
  await page.getByRole('button', { name: 'Save scale' }).click()
  await expect(status(page, 'Grade scale saved. Final grades already recorded keep their letters.'), 'admin, Grade scale: US restored').toBeVisible()
  await page.context().close()
})
