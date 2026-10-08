import { expect, test, type Page } from '@playwright/test'
import {
  api, createUniversity, demoTerm, latestMailLink, logLines, mails, pageAs, personas, runId, seedDemo, signIn, signOut,
} from './helpers'

const slug = `e2e-auth-${runId}`
const P = personas(slug)
const expired = 'That sign-in link has expired or was already used. Request a new one below.'

test.beforeAll(() => {
  seedDemo(slug)
})

const field = (page: Page) => page.getByLabel('University email or university ID')
const requestLink = async (page: Page, email: string) => {
  await field(page).fill(email)
  await page.getByRole('button', { name: 'Email me a sign-in link' }).click()
}

test('E-AUTH-01 sign-in page validation', async ({ page }) => {
  await page.goto('/sign-in')
  await page.getByRole('button', { name: 'Continue with Google' }).click()
  await expect(page.getByText('Enter your university email or university ID.'), 'signed out, sign-in: empty field error').toBeVisible()
  await expect(field(page), 'signed out, sign-in: focus on the field').toBeFocused()
  await field(page).fill('northfield')
  await page.getByRole('button', { name: 'Email me a sign-in link' }).click()
  await expect(page.getByText('Enter your email to get a sign-in link'), 'signed out, sign-in: needs an email').toBeVisible()
  await field(page).fill('a@')
  await page.getByRole('button', { name: 'Email me a sign-in link' }).click()
  await expect(page.getByText('Enter a valid email address.'), 'signed out, sign-in: invalid email').toBeVisible()
  await field(page).fill(`nowhere-${runId}`)
  await page.getByRole('button', { name: 'Continue with Google' }).click()
  await expect(
    page.getByRole('alert').filter({ hasText: `We couldn't find a university for “nowhere-${runId}”, or this sign-in option isn't available for it.` }),
    'signed out, sign-in: unknown university',
  ).toBeVisible()
})

test('E-AUTH-02 unknown address gets the same answer and no mail', async ({ page }) => {
  const before = logLines()
  const nobody = `nobody@unknown-${runId}.test`
  const sentinel = `sentinel@${slug}.edu`
  await page.goto('/sign-in')
  await requestLink(page, nobody)
  await expect(page.getByRole('heading', { name: 'Check your email' }), 'signed out, sign-in: check your email').toBeVisible()
  await expect(
    page.getByText(`If ${nobody} can sign in to UniCourse, we've sent it a link. The link works once and expires in 15 minutes.`),
    'signed out, sign-in: same answer for an unknown address',
  ).toBeVisible()
  const res = await page.request.post('/api/v1/auth/magic-link', { data: { email: sentinel }, headers: { 'Sec-Fetch-Site': 'same-origin' } })
  expect(res.status(), 'signed out, API: sentinel request accepted').toBe(202)
  await latestMailLink(sentinel, before)
  expect(mails(before).some((m) => m.to === nobody), 'signed out, mail log: no mail to the unknown address').toBe(false)
})

test('E-AUTH-03 resend waits 30 s and says so', async ({ page }) => {
  await page.clock.install()
  await page.goto('/sign-in')
  await requestLink(page, P.tomasz.email)
  const resend = page.getByRole('button', { name: 'Send the link again' })
  await expect(resend, 'signed out, Check your email: resend disabled').toBeDisabled()
  await expect(page.getByText('You can send it again in 30 s.'), 'signed out, Check your email: countdown').toBeVisible()
  await page.clock.runFor(31_000)
  await expect(resend, 'signed out, Check your email: resend enabled after 30 s').toBeEnabled()
  await resend.click()
  await expect(page.getByRole('status').filter({ hasText: 'Sent again.' }), 'signed out, Check your email: sent again').toBeVisible()
  await expect(resend, 'signed out, Check your email: resend disabled again').toBeDisabled()
  await page.getByRole('button', { name: 'use a different email' }).click()
  await expect(field(page), 'signed out, sign-in: form back').toBeVisible()
})

test('E-AUTH-04 a used sign-in link is refused', async ({ browser }) => {
  const a = await (await browser.newContext()).newPage()
  const before = logLines()
  await a.goto('/sign-in')
  await requestLink(a, P.priya.email)
  const link = await latestMailLink(P.priya.email, before)
  await a.goto(link)
  await a.getByRole('button', { name: 'Continue signing in' }).click()
  await a.waitForURL(`**/${slug}/me`)
  await a.context().close()
  const b = await (await browser.newContext()).newPage()
  await b.goto(link)
  await b.getByRole('button', { name: 'Continue signing in' }).click()
  await b.waitForURL((u) => u.pathname === '/sign-in' && u.search === '?error=validation_failed')
  await expect(b.getByRole('alert').filter({ hasText: expired }), 'priya, used link: expired copy').toBeVisible()
  await b.context().close()
})

test('E-AUTH-05 a malformed token is refused', async ({ page }) => {
  await page.goto('/api/v1/auth/magic-link/not-a-token')
  await expect(page.getByRole('heading', { name: 'Sign in to UniCourse' }), 'signed out, confirm page: heading').toBeVisible()
  await page.getByRole('button', { name: 'Continue signing in' }).click()
  await page.waitForURL((u) => u.pathname === '/sign-in' && u.searchParams.has('error'))
  await expect(page.getByRole('alert').filter({ hasText: expired }), 'signed out, malformed token: expired copy').toBeVisible()
})

test('E-AUTH-06 sign-out ends the session', async ({ page }) => {
  await signIn(page, P.daniel.email, slug)
  await signOut(page)
  await page.goto(`/${slug}/me`)
  await page.waitForURL((u) => u.pathname === '/sign-in' && u.searchParams.get('next') === `/${slug}/me`)
  expect((await page.request.get('/api/v1/me')).status(), 'daniel, after sign-out: /me is 401').toBe(401)
})

test('E-AUTH-07 a deep link while signed out returns to it after sign-in', async ({ page }) => {
  const dest = `/${slug}/catalog?q=algebra`
  await page.goto(dest)
  await page.waitForURL((u) => u.pathname === '/sign-in' && u.searchParams.get('next') === dest)
  await signIn(page, P.meiLing.email, slug, dest)
  await expect(page.getByRole('heading', { name: 'Catalog', level: 1 }), 'student, Catalog: heading').toBeVisible()
  await expect(page.getByRole('searchbox', { name: 'Find a course' }), 'student, Catalog: query kept').toHaveValue('algebra')
})

test('E-AUTH-08 an allowed domain without a membership', async ({ page }) => {
  await signIn(page, `stranger@${slug}.edu`, slug, '/sign-in?error=no_membership')
  await expect(
    page.getByRole('alert').filter({
      hasText: "You're signed in, but your account isn't a member of any university yet. Ask your administrator for an invitation.",
    }),
    'stranger, sign-in: no membership',
  ).toBeVisible()
})

test('E-AUTH-09 a signed-in member on a wrong slug', async ({ browser }) => {
  const page = await pageAs(browser, P.jonas.email, slug)
  await page.goto(`/nope-${runId}/me`)
  await expect(page.getByRole('heading', { name: "You don't have access to this university" }), 'student, wrong slug: heading').toBeVisible()
  await expect(
    page.getByText(`Your account (${P.jonas.email}) isn't a member of the university at this address.`),
    'student, wrong slug: account line',
  ).toBeVisible()
  await page.getByRole('link', { name: 'Go to Northfield University' }).click()
  await expect(page, 'student, wrong slug: back to own university').toHaveURL(new RegExp(`/${slug}/me$`))
  await page.context().close()
})

// Admin invites `guest`, the invitee follows the mailed link and signs in on the invite page.
async function acceptInvite(browser: Parameters<typeof pageAs>[0], admin: Page, guest: string) {
  await api(admin, 'post', `/api/v1/universities/${slug}/invitations`, { email: guest, role: 'member' })
  const link = await latestMailLink(guest, 0, '/invite/')
  const page = await (await browser.newContext()).newPage()
  await page.goto(link)
  await expect(page, 'invitee, invite link: sign-in page').toHaveURL(/^[^?]*\/sign-in(\?|$)/)
  await expect(page.getByText(`You've been invited. Sign in as ${guest} to accept.`), 'invitee, invite page: banner').toBeVisible()
  await expect(field(page), 'invitee, invite page: prefilled email').toHaveValue(guest)
  await signIn(page, guest, slug)
  return page
}

test('E-AUTH-10 invitation accept from the mailed link, outside-domain invitee', async ({ browser }) => {
  const guest = `guest@guest-${runId}.test`
  const admin = await pageAs(browser, P.admin.email, slug)
  const page = await acceptInvite(browser, admin, guest)
  await expect(page.getByRole('heading', { name: 'My sections' }), 'invitee, My sections: heading').toBeVisible()
  await expect(page.getByText(`No sections in ${demoTerm}`), 'invitee, My sections: empty').toBeVisible()
  await admin.goto(`/${slug}/admin/invitations`)
  const row = admin.getByRole('row', { name: new RegExp(guest) })
  await expect(row, 'admin, Invitations: accepted').toContainText('Accepted')
  await expect(row.getByRole('button', { name: `Revoke invitation for ${guest}` }), 'admin, Invitations: no revoke').toHaveCount(0)
  await page.context().close()
  await admin.context().close()
})

test.fixme('E-AUTH-11: F1 accepted outside-domain invitee gets no sign-in link', async ({ browser }) => {
  const guest = `guest2@guest-${runId}.test`
  const admin = await pageAs(browser, P.admin.email, slug)
  const page = await acceptInvite(browser, admin, guest)
  await signOut(page)
  await signIn(page, guest, slug)
  await page.context().close()
  await admin.context().close()
})

test('E-AUTH-12 the 5-per-email limit', async ({ page }) => {
  const email = `limit@${slug}.edu`
  for (let i = 0; i < 5; i++) {
    const res = await page.request.post('/api/v1/auth/magic-link', { data: { email }, headers: { 'Sec-Fetch-Site': 'same-origin' } })
    expect(res.status(), `signed out, API: request ${i + 1} accepted`).toBe(202)
  }
  await page.goto('/sign-in')
  await requestLink(page, email)
  await expect(page.getByRole('alert').filter({ hasText: 'Too many sign-in attempts. Try again in an hour.' }), 'signed out, sign-in: rate limited').toBeVisible()
})

test('E-AUTH-13 a person in two universities switches between them', async ({ browser }) => {
  const hslug = `e2e-auth-h-${runId}`
  createUniversity({ name: `Harbor College ${runId}`, slug: hslug, domain: `harbor-${runId}.edu`, adminEmail: `dean@harbor-${runId}.edu` })
  const dean = await (await browser.newContext()).newPage()
  await signIn(dean, `dean@harbor-${runId}.edu`, hslug)
  await api(dean, 'post', `/api/v1/universities/${hslug}/invitations`, { email: P.fatima.email, role: 'member' })
  await dean.context().close()
  const page = await (await browser.newContext()).newPage()
  await signIn(page, P.fatima.email, slug, `/${hslug}/me`)
  const select = page.getByRole('combobox', { name: 'University' })
  await expect(select.getByRole('option', { name: `Harbor College ${runId}` }), 'student, header: Harbor option').toHaveCount(1)
  await expect(select.getByRole('option', { name: 'Northfield University' }), 'student, header: Northfield option').toHaveCount(1)
  await select.selectOption({ label: 'Northfield University' })
  await expect(page, 'student, header: switched to Northfield').toHaveURL(new RegExp(`/${slug}/me$`))
  await expect(page.getByRole('listitem').filter({ hasText: 'CS101-001' }), 'student, My sections: CS101-001').toBeVisible()
  await page.context().close()
})

test('E-AUTH-14 a member sets their display name', async ({ browser }) => {
  const page = await pageAs(browser, P.lucas.email, slug)
  await page.goto(`/${slug}/me`)
  await page.getByRole('button', { name: /^Account/ }).click()
  const dialog = page.getByRole('dialog', { name: 'Your account' })
  await expect(dialog.getByText(`Signed in as ${P.lucas.email}`), 'student, account dialog: email').toBeVisible()
  await dialog.getByLabel('Name').clear()
  await dialog.getByRole('button', { name: 'Save' }).click()
  await expect(dialog.getByText('Enter a name of up to 100 characters.'), 'student, account dialog: empty name error').toBeVisible()
  await dialog.getByLabel('Name').fill('Lucas M.')
  await dialog.getByRole('button', { name: 'Save' }).click()
  await expect(dialog, 'student, account dialog: closed after save').toBeHidden()
  await expect(page.getByRole('button', { name: /^Account Lucas M\./ }), 'student, header: new name').toBeVisible()
  await page.context().close()
})
