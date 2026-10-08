import { expect, test, type Page } from '@playwright/test'
import { demoTerm, personas, runId, seedDemo, signIn, type PersonaKey } from './helpers'

const slug = `e2e-demo-${runId}`
const P = personas(slug)

test.beforeAll(() => {
  seedDemo(slug)
})

const nav = (page: Page) => page.getByRole('navigation', { name: 'Main' })
const list = (page: Page) => page.getByRole('list', { name: `My sections in ${demoTerm}` })
const card = (page: Page, code: string) => list(page).getByRole('listitem').filter({ hasText: code })

async function noStaffNav(page: Page, who: string) {
  await expect(nav(page).getByRole('link', { name: 'Courses' }), `${who}, nav: no Courses`).toHaveCount(0)
  await expect(nav(page).getByRole('link', { name: 'Admin' }), `${who}, nav: no Admin`).toHaveCount(0)
}

test('E-DEMO-01 admin sees the admin area', async ({ page }) => {
  await signIn(page, P.admin.email, slug)
  await expect(page.getByRole('banner').getByText('Northfield University'), 'admin, banner: university name').toBeVisible()
  await expect(nav(page).getByRole('link', { name: 'Courses' }), 'admin, nav: Courses').toBeVisible()
  await expect(nav(page).getByRole('link', { name: 'Admin' }), 'admin, nav: Admin').toBeVisible()
  await expect(page.getByText(`No sections in ${demoTerm}`), 'admin, My sections: empty').toBeVisible()
  await expect(page.getByRole('link', { name: 'Browse the catalog' }), 'admin, My sections: catalog link').toBeVisible()
  await expect(page.getByRole('link', { name: 'Go to Courses' }), 'admin, My sections: courses link').toBeVisible()
  await nav(page).getByRole('link', { name: 'Admin' }).click()
  await expect(page.getByRole('navigation', { name: 'Admin sections' }).getByRole('link'), 'admin, Admin: tabs').toHaveText([
    'Departments', 'Terms', 'Members', 'Invitations', 'Audit log',
  ])
})

test('E-DEMO-02 department admin sees only Invitations and own department courses', async ({ page }) => {
  await signIn(page, P.deptAdmin.email, slug)
  await expect(nav(page).getByRole('link', { name: 'Courses' }), 'department admin, nav: Courses').toBeVisible()
  await nav(page).getByRole('link', { name: 'Admin' }).click()
  await expect(page.getByRole('navigation', { name: 'Admin sections' }).getByRole('link'), 'department admin, Admin: tabs').toHaveText(['Invitations'])
  await nav(page).getByRole('link', { name: 'Courses' }).click()
  await expect(page.getByRole('cell', { name: 'CS101', exact: true }), 'department admin, Courses: CS101').toBeVisible()
  await expect(page.getByRole('cell', { name: 'CS210', exact: true }), 'department admin, Courses: CS210').toBeVisible()
  await expect(page.getByRole('cell', { name: 'MATH201', exact: true }), 'department admin, Courses: no MATH201').toHaveCount(0)
})

test('E-DEMO-03 Elena teaches three sections; CS101 finalized, CS210 open', async ({ page }) => {
  await signIn(page, P.elena.email, slug)
  await noStaffNav(page, 'instructor')
  await expect(list(page).getByRole('listitem'), 'instructor, My sections: count').toHaveCount(3)
  for (const code of ['CS101-001', 'CS210-001', 'MATH201-001']) {
    await expect(card(page, code), `instructor, My sections: ${code} role`).toContainText('Instructor')
    await expect(card(page, code).getByRole('link', { name: 'Gradebook' }), `instructor, My sections: ${code} gradebook`).toBeVisible()
  }
  await test.step('README: CS101 is finalised, CS210 is open', async () => {
    await card(page, 'CS101-001').getByRole('link', { name: 'Gradebook' }).click()
    await expect(page.getByRole('heading', { name: /Gradebook/ }), 'instructor, CS101 gradebook: finalized').toContainText('Finalized')
    await expect(page.getByRole('button', { name: 'Reopen grades' }), 'instructor, CS101 gradebook: reopen').toBeVisible()
    const row = page.getByRole('row', { name: /Lucas Moreau/ })
    await expect(row, 'instructor, CS101 gradebook: Lucas Incomplete').toContainText('Incomplete')
    await expect(row, 'instructor, CS101 gradebook: Lucas Changed').toContainText('Changed')
    await page.goBack()
    await card(page, 'CS210-001').getByRole('link', { name: 'Gradebook' }).click()
    await expect(page.getByRole('button', { name: 'Finalize grades' }), 'instructor, CS210 gradebook: finalize enabled').toBeEnabled()
  })
})

test('E-DEMO-04 David teaches two sections', async ({ page }) => {
  await signIn(page, P.david.email, slug)
  await noStaffNav(page, 'instructor')
  await expect(list(page).getByRole('listitem'), 'instructor, My sections: count').toHaveCount(2)
  for (const code of ['ECON110-001', 'BIO150-001']) await expect(card(page, code), `instructor, My sections: ${code}`).toContainText('Instructor')
})

test('E-DEMO-05 Sofia is TA on CS101', async ({ page }) => {
  await signIn(page, P.sofia.email, slug)
  await noStaffNav(page, 'TA')
  await expect(list(page).getByRole('listitem'), 'TA, My sections: count').toHaveCount(1)
  await expect(card(page, 'CS101-001'), 'TA, My sections: CS101 role').toContainText('TA')
  await expect(card(page, 'CS101-001').getByRole('link', { name: 'Gradebook' }), 'TA, My sections: gradebook').toBeVisible()
})

const students: [number, PersonaKey, string[]][] = [
  [6, 'amara', ['CS101', 'CS210', 'MATH201']],
  [7, 'priya', ['CS101', 'CS210', 'MATH201', 'BIO150']],
  [8, 'tomasz', ['CS101', 'CS210', 'MATH201']],
  [9, 'daniel', ['CS101', 'CS210', 'MATH201', 'BIO150']],
  [10, 'meiLing', ['CS101', 'CS210', 'ECON110']],
  [11, 'jonas', ['CS101', 'MATH201', 'ECON110']],
  [12, 'fatima', ['CS101', 'MATH201', 'ECON110']],
  [13, 'lucas', ['CS101', 'ECON110', 'BIO150']],
]
for (const [n, key, codes] of students) {
  test(`E-DEMO-${String(n).padStart(2, '0')} ${key} is a student in ${codes.join(', ')}`, async ({ page }) => {
    await signIn(page, P[key].email, slug)
    await noStaffNav(page, 'student')
    await expect(list(page).getByRole('listitem'), 'student, My sections: count').toHaveCount(codes.length)
    for (const code of codes) {
      await expect(card(page, code), `student, My sections: ${code} role`).toContainText('Student')
      await expect(card(page, code).getByRole('link', { name: 'Gradebook' }), `student, My sections: ${code} no gradebook`).toHaveCount(0)
    }
  })
}

test('E-DEMO-14 Ingrid is TA on CS210 at first sign-in', async ({ page }) => {
  await signIn(page, P.ingrid.email, slug)
  await noStaffNav(page, 'invited TA')
  await expect(list(page).getByRole('listitem'), 'invited TA, My sections: count').toHaveCount(1)
  await expect(card(page, 'CS210-001'), 'invited TA, My sections: CS210 role').toContainText('TA')
})
