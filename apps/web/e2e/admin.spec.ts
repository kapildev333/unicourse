import { expect, test } from '@playwright/test'
import { H, api, demoTerm, expectNoSideScroll, logLines, mails, pageAs, personas, runId, seedDemo, springTerm, year } from './helpers'

const slug = `e2e-adm-${runId}`
const P = personas(slug)
const A = `/api/v1/universities/${slug}`

test.beforeAll(() => {
  seedDemo(slug)
})

const status = (page: import('@playwright/test').Page, text: string) => page.getByRole('status').filter({ hasText: text })
const row = (page: import('@playwright/test').Page, name: string | RegExp) => page.getByRole('row', { name })

test('E-ADMIN-01 department create, rename, delete', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  await admin.goto(`/${slug}/admin/departments`)
  await admin.getByRole('button', { name: 'New department' }).first().click()
  const dlg = admin.getByRole('dialog', { name: 'New department' })
  await dlg.getByRole('button', { name: 'Create department' }).click()
  await expect(dlg.getByText('Enter a name of up to 100 characters.'), 'admin, New department: name error').toBeVisible()
  await expect(dlg.getByText('Use 2–10 capital letters or digits.'), 'admin, New department: code error').toBeVisible()
  await expect(dlg.getByLabel('Name'), 'admin, New department: focus on first error').toBeFocused()
  await dlg.getByLabel('Name').fill('Chemistry')
  await dlg.getByLabel('Code').fill('CS')
  await dlg.getByRole('button', { name: 'Create department' }).click()
  await expect(dlg.getByText('A department with this code already exists.'), 'admin, New department: duplicate code').toBeVisible()
  await dlg.getByLabel('Code').fill('CHEM')
  await dlg.getByRole('button', { name: 'Create department' }).click()
  await expect(status(admin, 'Department “CHEM” created.'), 'admin, Departments: created').toBeVisible()
  await expect(admin.getByRole('cell', { name: 'Chemistry', exact: true }), 'admin, Departments: new cell').toBeVisible()

  await admin.getByRole('button', { name: 'Edit Chemistry' }).click()
  const edit = admin.getByRole('dialog', { name: 'Edit department' })
  await edit.getByLabel('Name').fill('Chemistry and Biochemistry')
  await edit.getByRole('button', { name: 'Save changes' }).click()
  await expect(status(admin, 'Department “CHEM” updated.'), 'admin, Departments: updated').toBeVisible()
  await expect(admin.getByRole('cell', { name: 'Chemistry and Biochemistry', exact: true }), 'admin, Departments: renamed cell').toBeVisible()

  await admin.getByRole('button', { name: 'Delete Chemistry and Biochemistry' }).click()
  const del = admin.getByRole('dialog', { name: 'Delete department?' })
  await expect(del.getByText("Chemistry and Biochemistry (CHEM) will be removed. This can't be undone."), 'admin, Delete department: confirmation').toBeVisible()
  await del.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(status(admin, 'Department “CHEM” deleted.'), 'admin, Departments: deleted').toBeVisible()
  await expect(admin.getByRole('cell', { name: 'Chemistry and Biochemistry', exact: true }), 'admin, Departments: gone').toHaveCount(0)
  await admin.context().close()
})

test('E-ADMIN-02 a department in use cannot be deleted', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  await admin.goto(`/${slug}/admin/departments`)
  await admin.getByRole('button', { name: 'Delete Computer Science' }).click()
  const del = admin.getByRole('dialog', { name: 'Delete department?' })
  await del.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(del, 'admin, Delete department: in-use refusal').toContainText('This department still has courses or department admins. Move or remove them first.')
  await del.getByRole('button', { name: 'Cancel' }).click()
  await expect(admin.getByRole('cell', { name: 'Computer Science', exact: true }), 'admin, Departments: still present').toBeVisible()
  await admin.context().close()
})

test('E-ADMIN-03 term create, edit, status, delete', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  const y = year - 1
  const summer = `Summer ${y}`
  await admin.goto(`/${slug}/admin/terms`)
  await expect(row(admin, new RegExp(demoTerm)), 'admin, Terms: current term').toContainText('Current')
  await expect(row(admin, new RegExp(springTerm)), 'admin, Terms: upcoming term').toContainText('Upcoming')

  await admin.getByRole('button', { name: 'New term' }).first().click()
  const dlg = admin.getByRole('dialog', { name: 'New term' })
  await dlg.getByLabel('Name').fill(summer)
  await dlg.getByLabel('Starts on').fill(`${y}-06-30`)
  await dlg.getByLabel('Ends on').fill(`${y}-06-01`)
  await dlg.getByRole('button', { name: 'Create term' }).click()
  await expect(dlg.getByText('End date must be after the start date.'), 'admin, New term: date order').toBeVisible()
  await dlg.getByLabel('Ends on').fill(`${y}-08-15`)
  await dlg.getByRole('button', { name: 'Create term' }).click()
  await expect(status(admin, `Term “${summer}” created.`), 'admin, Terms: created').toBeVisible()
  await expect(row(admin, new RegExp(summer)), 'admin, Terms: past term').toContainText('Past')

  await admin.getByRole('button', { name: 'New term' }).first().click()
  const dup = admin.getByRole('dialog', { name: 'New term' })
  await dup.getByLabel('Name').fill(springTerm)
  await dup.getByLabel('Starts on').fill(`${year + 1}-01-01`)
  await dup.getByLabel('Ends on').fill(`${year + 1}-06-01`)
  await dup.getByRole('button', { name: 'Create term' }).click()
  await expect(dup.getByText('A term with this name already exists.'), 'admin, New term: duplicate name').toBeVisible()
  await dup.getByRole('button', { name: 'Cancel' }).click()

  await admin.getByRole('button', { name: `Edit ${summer}` }).click()
  const edit = admin.getByRole('dialog', { name: 'Edit term' })
  await edit.getByLabel('Name').fill(`${summer} archive`)
  await edit.getByRole('button', { name: 'Save changes' }).click()
  await expect(status(admin, `Term “${summer} archive” updated.`), 'admin, Terms: updated').toBeVisible()

  await admin.getByRole('button', { name: `Delete ${summer} archive` }).click()
  const del = admin.getByRole('dialog', { name: 'Delete term?' })
  await expect(del.getByText(`${summer} archive will be removed. This can't be undone.`), 'admin, Delete term: confirmation').toBeVisible()
  await del.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(status(admin, `Term “${summer} archive” deleted.`), 'admin, Terms: deleted').toBeVisible()
  await admin.context().close()
})

test('E-ADMIN-04 a term with sections cannot be deleted', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  await admin.goto(`/${slug}/admin/terms`)
  await admin.getByRole('button', { name: `Delete ${demoTerm}` }).click()
  const del = admin.getByRole('dialog', { name: 'Delete term?' })
  await del.getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(del, 'admin, Delete term: has sections').toContainText('This term still has sections. Delete them first.')
  await admin.context().close()
})

test('E-ADMIN-05 members list, role change, remove', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  await admin.goto(`/${slug}/admin/members`)
  for (const n of ['Renata Quinn', 'Owen Park', 'Priya Raman']) await expect(row(admin, new RegExp(n)), `admin, Members: ${n}`).toBeVisible()
  await expect(row(admin, /Priya Raman/), 'admin, Members: Priya role and status').toContainText(/Member.*Active/)

  await admin.getByRole('button', { name: `Change role for ${P.priya.email}` }).click()
  const dlg = admin.getByRole('dialog', { name: 'Change role for Priya Raman' })
  await dlg.getByLabel('Role').selectOption({ label: 'Department admin' })
  await dlg.getByRole('button', { name: 'Save role' }).click()
  await expect(dlg.getByText('Choose at least one department for a department admin.'), 'admin, Change role: needs department').toBeVisible()
  await dlg.getByRole('group', { name: 'Departments' }).getByLabel('Mathematics').check()
  await dlg.getByRole('button', { name: 'Save role' }).click()
  await expect(status(admin, 'Priya Raman is now Department admin.'), 'admin, Members: role changed').toBeVisible()
  await expect(row(admin, /Priya Raman/), 'admin, Members: Priya new role').toContainText('Department admin')
  await expect(row(admin, /Priya Raman/), 'admin, Members: Priya department').toContainText('MATH')

  await admin.getByRole('button', { name: `Remove ${P.jonas.email}` }).click()
  const rm = admin.getByRole('dialog', { name: 'Remove member?' })
  await expect(
    rm.getByText(`${P.jonas.email} will lose access to Northfield University. Their course staff assignments are not changed.`),
    'admin, Remove member: confirmation',
  ).toBeVisible()
  await rm.getByRole('button', { name: 'Remove', exact: true }).click()
  await expect(status(admin, `${P.jonas.email} removed.`), 'admin, Members: removed').toBeVisible()
  await expect(row(admin, /Jonas Berg/), 'admin, Members: Jonas gone').toHaveCount(0)
  await admin.context().close()
})

test('E-ADMIN-06 the only admin stays an admin', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  await admin.goto(`/${slug}/admin/members`)
  await admin.getByRole('button', { name: `Change role for ${P.admin.email}` }).click()
  const dlg = admin.getByRole('dialog', { name: /Change role for/ })
  await dlg.getByLabel('Role').selectOption({ label: 'Member' })
  await dlg.getByRole('button', { name: 'Save role' }).click()
  await expect(dlg, 'admin, Change role: only admin').toContainText("This is the university's only admin. Make someone else an admin first.")
  await dlg.getByRole('button', { name: 'Cancel' }).click()
  await admin.getByRole('button', { name: `Remove ${P.admin.email}` }).click()
  const rm = admin.getByRole('dialog', { name: 'Remove member?' })
  await rm.getByRole('button', { name: 'Remove', exact: true }).click()
  await expect(rm, 'admin, Remove member: only admin').toContainText("This is the university's only admin and can't be removed.")
  await admin.context().close()
})

test('E-ADMIN-07 invitations list, validation, send, revoke', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  const hire = `new.hire@${slug}.edu`
  await admin.goto(`/${slug}/admin/invitations`)
  await expect(row(admin, new RegExp(P.ingrid.email)), 'admin, Invitations: Ingrid pending').toContainText('Pending')
  await expect(admin.getByRole('button', { name: `Revoke invitation for ${P.ingrid.email}` }), 'admin, Invitations: revoke button').toBeVisible()

  await admin.getByRole('button', { name: 'Invite someone' }).first().click()
  const dlg = admin.getByRole('dialog', { name: 'Invite someone' })
  await dlg.getByLabel('Email').fill('x')
  await dlg.getByRole('button', { name: 'Send invitation' }).click()
  await expect(dlg.getByText('Enter a valid email address.'), 'admin, Invitations dialog: email field error').toBeVisible()
  await dlg.getByLabel('Email').fill(P.amara.email)
  await dlg.getByRole('button', { name: 'Send invitation' }).click()
  await expect(dlg.getByText('This email already has access or a pending invitation.'), 'admin, Invitations dialog: duplicate').toBeVisible()
  await dlg.getByLabel('Email').fill(hire)
  await dlg.getByLabel('Role').selectOption({ label: 'Department admin' })
  await dlg.getByRole('button', { name: 'Send invitation' }).click()
  await expect(dlg.getByText('Choose at least one department for a department admin.'), 'admin, Invitations dialog: needs department').toBeVisible()
  await dlg.getByLabel('Role').selectOption({ label: 'Member' })
  const before = logLines()
  await dlg.getByRole('button', { name: 'Send invitation' }).click()
  await expect(status(admin, `Invitation sent to ${hire}.`), 'admin, Invitations: sent').toBeVisible()
  await expect(row(admin, new RegExp(hire)), 'admin, Invitations: new row pending').toContainText('Pending')
  await expect
    .poll(() => mails(before).some((m) => m.to === hire && m.subject === 'You are invited to Northfield University on UniCourse'), { message: 'admin, Invitations: invitation mail' })
    .toBe(true)

  await admin.getByRole('button', { name: `Revoke invitation for ${hire}` }).click()
  const rv = admin.getByRole('dialog', { name: 'Revoke invitation?' })
  await expect(rv.getByText(`${hire} won't be able to join with this invitation.`), 'admin, Revoke invitation: confirmation').toBeVisible()
  await rv.getByRole('button', { name: 'Revoke', exact: true }).click()
  await expect(status(admin, `Invitation for ${hire} revoked.`), 'admin, Invitations: revoked').toBeVisible()
  await expect(row(admin, new RegExp(hire)), 'admin, Invitations: row gone').toHaveCount(0)
  await admin.context().close()
})

test('E-ADMIN-08 audit log entries and filters', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  const d = await api(admin, 'post', `${A}/departments`, { name: 'Physics', code: 'PHYS' })
  await api(admin, 'patch', `${A}/departments/${d.id}`, { name: 'Applied Physics' })
  expect((await admin.request.delete(`${A}/departments/${d.id}`, { headers: H })).status()).toBe(204)

  await admin.goto(`/${slug}/admin/audit`)
  await admin.getByLabel('Entity').selectOption({ label: 'Department' })
  const rows = admin.getByRole('row')
  await expect(rows.nth(1), 'admin, Audit log: newest').toContainText('Department deleted')
  await expect(rows.nth(2), 'admin, Audit log: second').toContainText('Department updated')
  await expect(rows.nth(3), 'admin, Audit log: third').toContainText('Department created')
  await expect(rows.nth(1), 'admin, Audit log: actor').toContainText('Renata Quinn')
  await expect(rows.nth(1), 'admin, Audit log: target').toContainText('Department Applied Physics')

  await rows.nth(2).getByText('Show changes').click()
  await expect(rows.nth(2).getByText('Before'), 'admin, Audit log: Before').toBeVisible()
  await expect(rows.nth(2).getByText('After'), 'admin, Audit log: After').toBeVisible()
  await expect(rows.nth(2).getByText('Applied Physics (changed)'), 'admin, Audit log: changed value').toBeVisible()

  // Other entities: the filter must really apply, and the term, member and invitation cases above left entries.
  const byEntity: [string, string[]][] = [
    ['Term', ['Term deleted']],
    ['Membership', ['Member removed', 'Member updated']],
    ['Invitation', ['Invitation revoked', 'Invitation sent']],
  ]
  for (const [entity, actions] of byEntity) {
    await admin.getByLabel('Entity').selectOption({ label: entity })
    for (const [i, a] of actions.entries()) await expect(rows.nth(i + 1), `admin, Audit log ${entity}: row ${i + 1}`).toContainText(a)
  }
  await admin.getByLabel('Entity').selectOption({ label: 'All' })
  await admin.getByLabel('Changed by').selectOption({ label: 'Owen Park' })
  await expect(rows.nth(1), 'admin, Audit log: Owen filter').toContainText('Owen Park')

  const t = new Date(Date.now() + 86_400_000)
  const tomorrow = `${t.getFullYear()}-${String(t.getMonth() + 1).padStart(2, '0')}-${String(t.getDate()).padStart(2, '0')}`
  await admin.getByLabel('From').fill(tomorrow)
  await expect(admin.getByText('No changes match these filters'), 'admin, Audit log: empty').toBeVisible()
  await expect(admin.getByText('Try a wider date range or fewer filters.'), 'admin, Audit log: empty hint').toBeVisible()
  await admin.getByRole('button', { name: 'Clear filters' }).first().click()
  await expect(rows.nth(1), 'admin, Audit log: rows back').toBeVisible()
  await admin.context().close()
})

test('E-ADMIN-09 a department admin sees a narrower view', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  await api(admin, 'post', `${A}/invitations`, { email: `ops@${slug}.edu`, role: 'admin' })
  await admin.context().close()

  const owen = await pageAs(browser, P.deptAdmin.email, slug)
  await owen.goto(`/${slug}/admin`)
  await expect(owen, 'department admin, Admin: lands on Invitations').toHaveURL(/\/admin\/invitations$/)
  await expect(owen.getByRole('navigation', { name: 'Admin sections' }).getByRole('link'), 'department admin, Admin: tabs').toHaveText(['Invitations'])
  await expect(row(owen, new RegExp(P.ingrid.email)), 'department admin, Invitations: Ingrid').toBeVisible()
  await expect(owen.getByRole('cell', { name: `ops@${slug}.edu`, exact: true }), 'department admin, Invitations: no admin invitations').toHaveCount(0)

  await owen.getByRole('button', { name: 'Invite someone' }).first().click()
  const dlg = owen.getByRole('dialog', { name: 'Invite someone' })
  await expect(dlg.getByText('Role: Member'), 'department admin, Invitations dialog: fixed role').toBeVisible()
  await expect(dlg.getByRole('combobox', { name: 'Role' }), 'department admin, Invitations dialog: no role select').toHaveCount(0)
  await dlg.getByLabel('Email').fill(`ta.candidate@${slug}.edu`)
  await dlg.getByRole('button', { name: 'Send invitation' }).click()
  await expect(status(owen, `Invitation sent to ta.candidate@${slug}.edu.`), 'department admin, Invitations: sent').toBeVisible()

  for (const p of ['members', 'audit']) {
    await owen.goto(`/${slug}/admin/${p}`)
    await expect(owen.getByRole('alert'), `department admin, ${p}: no permission`).toContainText("You don't have permission to view this.")
  }
  const res = await owen.request.post(`${A}/invitations`, { headers: H, data: { email: `x@${slug}.edu`, role: 'admin' } })
  expect(res.status(), 'department admin, API: admin invitation').toBe(403)
  await owen.context().close()
})

test('E-ADMIN-10 a member has no admin area', async ({ browser }) => {
  const amara = await pageAs(browser, P.amara.email, slug)
  await amara.goto(`/${slug}/admin`)
  const alert = amara.getByRole('alert')
  await expect(alert, 'member, Admin: title').toContainText('No access')
  await expect(alert, 'member, Admin: body').toContainText('The admin area is for university and department administrators.')
  expect((await amara.request.get(`${A}/memberships`)).status(), 'member, API: memberships').toBe(403)
  await amara.context().close()
})

test('E-ADMIN-11 admin tables at phone width', async ({ browser }) => {
  const admin = await pageAs(browser, P.admin.email, slug)
  await admin.setViewportSize({ width: 390, height: 844 })
  await admin.goto(`/${slug}/admin/members`)
  await expect(admin.getByText('Priya Raman'), 'admin, Members: Priya').toBeVisible()
  await expectNoSideScroll(admin, 'admin, Members')
  await expect(admin.getByRole('button', { name: `Change role for ${P.priya.email}` }), 'admin, Members: change role').toBeVisible()
  await admin.getByRole('navigation', { name: 'Admin sections' }).getByRole('link', { name: 'Audit log' }).click()
  await expect(admin.getByRole('heading', { name: 'Audit log' }), 'admin, Audit log: heading').toBeVisible()
  await expectNoSideScroll(admin, 'admin, Audit log')
  await admin.context().close()
})
