import { expect, test, type Page } from '@playwright/test'
import { api, expectNoSideScroll, logLines, mails, pageAs, personas, runId, sectionIds, seedDemo } from './helpers'

// Spec 21 §6.1: the bell, the Notifications page and the announcement-mail opt-out.
// The cases share one section (CS210-001: Elena teaches, Amara and Priya are enrolled) and run in order.
test.describe.configure({ mode: 'serial' })

const slug = `e2e-ntf-${runId}`
const P = personas(slug)
const A = `/api/v1/universities/${slug}`
let sid = ''

test.beforeAll(async ({ browser }) => {
  seedDemo(slug)
  const elena = await pageAs(browser, P.elena.email, slug)
  sid = (await sectionIds(elena, slug))['CS210-001']
  await elena.context().close()
})

const status = (page: Page, text: string) => page.getByRole('status').filter({ hasText: text })
const notifications = (page: Page) => page.goto(`/${slug}/notifications`)
const rows = (page: Page) => page.getByRole('list', { name: 'Notifications' })

test('E-NOTIF-01 an announcement notifies the student, not the author, and opening it marks it read', async ({ browser }) => {
  const elena = await pageAs(browser, P.elena.email, slug)
  await api(elena, 'post', `${A}/sections/${sid}/announcements`, { title: 'Lab moved to Friday', body: 'Room 4.', email: false })
  await elena.goto(`/${slug}/me`)
  await notifications(elena)
  await expect(elena.getByRole('heading', { name: 'Notifications', level: 1 }), 'instructor, Notifications: heading').toBeVisible()
  // the demo seed leaves older rows; only the author's own new announcement must be absent
  await expect(rows(elena).getByRole('listitem').first(), 'instructor, Notifications: list loaded').toBeVisible()
  await expect(elena.getByRole('link', { name: 'New announcement: Lab moved to Friday' }), 'instructor, Notifications: author gets none').toHaveCount(0)
  await elena.context().close()

  // Jonas is not enrolled in CS210-001: an unenrolled student gets nothing
  const jonas = await pageAs(browser, P.jonas.email, slug)
  await notifications(jonas)
  await expect(rows(jonas).getByRole('listitem').first(), 'unenrolled student, Notifications: list loaded').toBeVisible()
  await expect(jonas.getByRole('link', { name: 'New announcement: Lab moved to Friday' }), 'unenrolled student, Notifications: gets none').toHaveCount(0)
  await jonas.context().close()

  const amara = await pageAs(browser, P.amara.email, slug)
  await amara.goto(`/${slug}/me`)
  const bell = amara.getByRole('link', { name: /^Notifications, \d+ unread$/ })
  await expect(bell, 'student, header: bell shows unread').toBeVisible()
  await bell.click()
  await expect(amara.getByRole('heading', { name: 'Notifications', level: 1 }), 'student, Notifications: heading').toBeVisible()
  const row = rows(amara).getByRole('listitem').filter({ hasText: 'Lab moved to Friday' })
  await expect(row.getByRole('button', { name: 'Mark as read' }), 'student, Notifications: new announcement unread').toBeVisible()
  await amara.getByRole('link', { name: 'New announcement: Lab moved to Friday' }).click()
  await expect(amara, 'student: opens the Announcements tab').toHaveURL(new RegExp(`/${slug}/sections/${sid}/announcements$`))
  await notifications(amara)
  await expect(row, 'student, Notifications: row stays listed').toBeVisible()
  await expect(row.getByRole('button', { name: 'Mark as read' }), 'student, Notifications: row is read').toHaveCount(0)
  await amara.context().close()
})

test('E-NOTIF-02 released scores notify the student once', async ({ browser }) => {
  const elena = await pageAs(browser, P.elena.email, slug)
  const now = Date.now()
  const q = await api(elena, 'post', `${A}/sections/${sid}/assessments`, {
    kind: 'quiz',
    name: 'Notify quiz',
    opensAt: new Date(now - 60_000).toISOString(),
    closesAt: new Date(now + 3600_000).toISOString(),
    weight: 0,
  })
  const priya = await pageAs(browser, P.priya.email, slug)
  await notifications(priya)
  await expect(priya.getByRole('link', { name: 'New quiz: Notify quiz' }), 'student, Notifications: new quiz row').toBeVisible()
  await expect(priya.getByRole('link', { name: 'Grades released: Notify quiz' }), 'student, Notifications: nothing released before publishing').toHaveCount(0)

  await api(elena, 'patch', `${A}/sections/${sid}/assessments/${q.id}`, { published: true })
  await api(elena, 'patch', `${A}/sections/${sid}/assessments/${q.id}`, { published: true })
  await elena.context().close()
  await notifications(priya)
  await expect(priya.getByRole('link', { name: 'New quiz: Notify quiz' }), 'student, Notifications: new quiz row after publish').toBeVisible()
  await expect(priya.getByRole('link', { name: 'Grades released: Notify quiz' }), 'student, Notifications: released once').toHaveCount(1)
  await priya.context().close()
})

test('E-NOTIF-03 Mark all as read clears the count', async ({ browser }) => {
  const priya = await pageAs(browser, P.priya.email, slug)
  await notifications(priya)
  await expect(priya.getByRole('link', { name: /^Notifications, \d+ unread$/ }), 'student, header: unread count').toBeVisible()
  await priya.getByRole('button', { name: 'Mark all as read' }).click()
  await expect(status(priya, 'All notifications marked as read.'), 'student, Notifications: all read notice').toBeVisible()
  await expect(priya.getByRole('link', { name: 'Notifications', exact: true }), 'student, header: bell cleared').toBeVisible()
  await expect(priya.getByRole('button', { name: /^Mark as read/ }), 'student, Notifications: no row left unread').toHaveCount(0)
  await expect(priya.getByRole('button', { name: 'Mark all as read' }), 'student, Notifications: button disabled').toBeDisabled()
  await priya.context().close()
})

test('E-NOTIF-04 turning announcement mail off keeps the notification and stops the mail', async ({ browser }) => {
  const elena = await pageAs(browser, P.elena.email, slug)
  const before = logLines()
  await api(elena, 'post', `${A}/sections/${sid}/announcements`, { title: 'Mail on', body: 'x', email: true })
  await expect.poll(() => mails(before).filter((m) => m.to === P.amara.email).length, { message: 'mail on: amara is mailed' }).toBe(1)

  const amara = await pageAs(browser, P.amara.email, slug)
  await amara.goto(`/${slug}/me`)
  await amara.getByRole('button', { name: /^Account/ }).click()
  const dlg = amara.getByRole('dialog', { name: 'Your account' })
  await expect(dlg.getByLabel('Email me new announcements'), 'student, account dialog: on by default').toBeChecked()
  await dlg.getByLabel('Email me new announcements').uncheck()
  await dlg.getByRole('button', { name: 'Save' }).click()
  await expect(dlg, 'student, account dialog: closes after save').toBeHidden()

  const mark = logLines()
  await api(elena, 'post', `${A}/sections/${sid}/announcements`, { title: 'Mail off', body: 'x', email: true })
  // The outbox order is not per recipient, so fence with a third announcement: once Priya's fence mail is
  // logged, the outbox has drained past the "Mail off" rows (older next_attempt_at) and Amara's would be logged too.
  await api(elena, 'post', `${A}/sections/${sid}/announcements`, { title: 'Mail fence', body: 'x', email: true })
  await expect.poll(() => mails(mark).filter((m) => m.to === P.priya.email).length, { message: 'mail off and fence: priya is mailed twice' }).toBe(2)
  expect(mails(mark).filter((m) => m.to === P.amara.email), 'mail off: amara is not mailed').toHaveLength(0)
  await notifications(amara)
  await expect(amara.getByRole('link', { name: 'New announcement: Mail off' }), 'student, Notifications: row still appears').toBeVisible()
  await amara.context().close()
  await elena.context().close()
})

test('E-NOTIF-05 the Notifications page fits a 390 px phone', async ({ browser }) => {
  const page = await pageAs(browser, P.amara.email, slug)
  await page.setViewportSize({ width: 390, height: 844 })
  await notifications(page)
  await expect(rows(page).getByRole('listitem').first(), 'student at 390, Notifications: rows').toBeVisible()
  await expectNoSideScroll(page, 'student at 390, Notifications')
  await page.context().close()
})
