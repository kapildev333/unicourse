import { expect, test, type Page } from '@playwright/test'
import { api, expectNoSideScroll, pageAs, personas, runId, sectionIds, seedDemo } from './helpers'

// Spec 11 §1, quiz half: create, take, automatic submission at the close, review, release, gradebook.
// The cases share one quiz and run in order.
test.describe.configure({ mode: 'serial' })

const slug = `e2e-qz-${runId}`
const P = personas(slug)
const A = `/api/v1/universities/${slug}`
const NAME = 'Week 3 check'
let sid = ''
let quizId = ''
let closeAt = 0 // epoch ms the quiz closes, set by E-QUIZ-02
let amara: Page // the student who never submits; her page stays open until the sweeper has run

test.beforeAll(async ({ browser }) => {
  seedDemo(slug)
  const elena = await pageAs(browser, P.elena.email, slug)
  sid = (await sectionIds(elena, slug))['CS210-001']
  await elena.context().close()
})

test.afterAll(async () => {
  await amara?.context().close()
})

const status = (page: Page, text: string) => page.getByRole('status').filter({ hasText: text })
const quizzes = (page: Page) => page.goto(`/${slug}/sections/${sid}/quizzes`)
const question = (page: Page, n: number) => page.getByRole('article', { name: new RegExp(`^Question ${n}\\b`) })
const group = (page: Page, prompt: string) => page.getByRole('group', { name: new RegExp(prompt) })
// value of an <input type="datetime-local"> for `ms`, in the machine's local time like the browser
const local = (ms: number) => {
  const d = new Date(ms)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}

test('E-QUIZ-01 the instructor creates the quiz and its three questions', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await quizzes(page)
  await expect(page.getByText('No quizzes yet'), 'instructor, Quizzes: empty state').toBeVisible()
  await page.getByRole('button', { name: 'New quiz' }).click()
  const dlg = page.getByRole('dialog', { name: 'New quiz' })
  await dlg.getByLabel('Title').fill(NAME)
  await dlg.getByLabel('Description').fill('Lists, stacks and searching.')
  await dlg.getByLabel('Opens').fill(local(Date.now() - 60_000))
  await dlg.getByLabel('Closes').fill(local(Date.now() + 30 * 60_000))
  await dlg.getByLabel('Weight (%)').fill('40')
  await dlg.getByRole('button', { name: 'Create quiz' }).click()
  await expect(status(page, `${NAME} created.`), 'instructor, Quizzes: created notice').toBeVisible()
  await expect(page.getByText('No questions yet. Weight 40%.'), 'instructor, Quizzes: row meta').toBeVisible()

  await page.getByRole('link', { name: `Questions for ${NAME}` }).click()
  await expect(page.getByRole('heading', { name: `Questions: ${NAME}` }), 'instructor, editor: heading').toBeVisible()

  await page.getByRole('button', { name: 'Add multiple choice' }).first().click()
  let q = question(page, 1)
  await q.getByLabel('Prompt', { exact: true }).fill('Which keyword defines a function in Python?')
  await q.getByLabel('Points', { exact: true }).fill('2')
  await q.getByLabel('Choice 1', { exact: true }).fill('def')
  await q.getByLabel('Choice 2', { exact: true }).fill('func')
  await q.getByLabel('Correct (choice 1)').check()

  await page.getByRole('button', { name: 'Add multiple choice' }).click()
  q = question(page, 2)
  await q.getByLabel('Prompt', { exact: true }).fill('Which structure is last in, first out?')
  await q.getByLabel('Points', { exact: true }).fill('2')
  await q.getByLabel('Choice 1', { exact: true }).fill('Queue')
  await q.getByLabel('Choice 2', { exact: true }).fill('Stack')
  await q.getByLabel('Correct (choice 2)').check()

  await page.getByRole('button', { name: 'Add fill in the blank' }).click()
  q = question(page, 3)
  await q.getByLabel('Prompt', { exact: true }).fill('Name the search that halves a sorted list each step.')
  await q.getByLabel('Points', { exact: true }).fill('3')
  await q.getByLabel('Accepted answers').fill('binary search')

  await page.getByRole('button', { name: 'Save questions' }).click()
  await expect(status(page, 'Questions saved: 3 questions, 7 points.'), 'instructor, editor: saved notice').toBeVisible()

  quizId = (await api(page, 'get', `${A}/sections/${sid}/assessments`)).items.find((a: { name: string }) => a.name === NAME).id
  await page.context().close()
})

test('E-QUIZ-02 a student starts, answers two questions, sees Saved, does not submit', async ({ browser }) => {
  // The dialog's time input has minute granularity; pin the close to 75 s from now so the sweeper's work is observed soon.
  const elena = await pageAs(browser, P.elena.email, slug)
  closeAt = Date.now() + 75_000
  await api(elena, 'patch', `${A}/sections/${sid}/assessments/${quizId}`, { closesAt: new Date(closeAt).toISOString() })
  await elena.context().close()

  amara = await pageAs(browser, P.amara.email, slug)
  await quizzes(amara)
  await expect(amara.getByText('Open', { exact: true }), 'student, Quizzes: Open badge').toBeVisible()
  await amara.getByRole('link', { name: `Start ${NAME}` }).click()
  await expect(amara.getByText('Lists, stacks and searching.'), 'student, quiz page: description').toBeVisible()
  await amara.getByRole('button', { name: 'Start quiz' }).click()

  await group(amara, 'Which keyword').getByLabel('def', { exact: true }).check()
  await expect(status(amara, 'Saved'), 'student, take page: Saved after a choice').toBeVisible()
  // spelled with a hyphen: the automatic grading does not match it, the TA will accept it
  await Promise.all([
    amara.waitForResponse((r) => r.url().includes('/attempts/me') && r.request().method() === 'PUT' && r.ok()),
    group(amara, 'Name the search').getByLabel('Your answer').fill('binary-search'),
  ])
  await expect(status(amara, 'Saved'), 'student, take page: Saved after the blank').toBeVisible()
  expect(Date.now(), 'the answers were saved before the close').toBeLessThan(closeAt)
})

test('E-QUIZ-03 a second student submits explicitly', async ({ browser }) => {
  const page = await pageAs(browser, P.priya.email, slug)
  await quizzes(page)
  await page.getByRole('link', { name: `Start ${NAME}` }).click()
  await page.getByRole('button', { name: 'Start quiz' }).click()
  await group(page, 'Which keyword').getByLabel('def', { exact: true }).check()
  await group(page, 'Which structure').getByLabel('Queue', { exact: true }).check()
  await group(page, 'Name the search').getByLabel('Your answer').fill('Binary  Search')
  await expect(status(page, 'Saved'), 'priya, take page: Saved').toBeVisible()
  await page.getByRole('button', { name: 'Submit quiz' }).click()
  const dlg = page.getByRole('dialog', { name: 'Submit quiz?' })
  await expect(dlg.getByText('You answered all 3 questions.'), 'priya, Submit dialog: count').toBeVisible()
  await dlg.getByRole('button', { name: 'Submit quiz' }).click()
  await expect(page.getByRole('heading', { name: 'Your results' }), 'priya, results view').toBeVisible()
  await expect(page.getByText(/^Submitted (?!automatically)/), 'priya, results: explicit submit').toBeVisible()
  await expect(page.getByText('appear here once staff release the scores'), 'priya, results: score hidden until released').toBeVisible()
  await page.context().close()
})

test('E-QUIZ-04 the quiz closes and the server submits the untouched attempt', async () => {
  test.setTimeout(150_000)
  const left = Math.max(0, closeAt - Date.now())
  await expect(amara.getByText(/^Submitted automatically at /), 'student, results: submitted automatically').toBeVisible({ timeout: left + 40_000 })
  await expect(amara.getByRole('heading', { name: 'Your results' }), 'student, results heading').toBeVisible()
  await expect(amara.getByText('appear here once staff release the scores'), 'student, results: still hidden').toBeVisible()
})

test('E-QUIZ-05 the TA accepts the fill-in, gives feedback, saves and releases scores', async ({ browser }) => {
  const page = await pageAs(browser, P.ingrid.email, slug)
  await quizzes(page)
  await page.getByRole('link', { name: `Attempts for ${NAME}` }).click()
  const row = page.getByRole('row', { name: /Amara Okafor/ })
  await expect(row.getByText('Auto', { exact: true }), 'TA, Attempts: Auto badge').toBeVisible()
  await expect(row.getByText('2 / 7'), 'TA, Attempts: automatic score').toBeVisible()
  await expect(page.getByRole('row', { name: /Priya Raman/ }).getByText('5 / 7'), 'TA, Attempts: priya automatic score').toBeVisible()
  await row.getByRole('link', { name: 'Amara Okafor' }).click()

  await expect(page.getByText(/Submitted automatically at /), 'TA, attempt: submitted automatically').toBeVisible()
  await expect(question(page, 3), 'TA, attempt: the unmatched answer').toContainText('Answer: binary-search')
  await page.getByLabel('Points for question 3').fill('3')
  await page.getByLabel('Feedback on question 3').fill('Spelling accepted.')
  await page.getByLabel('Feedback for Amara Okafor').fill('Good work, review stacks.')
  await page.getByRole('button', { name: 'Save review' }).click()
  await expect(status(page, 'Review saved. Amara Okafor: 5 / 7.'), 'TA, attempt: saved notice').toBeVisible()

  await page.getByRole('link', { name: 'All attempts' }).click()
  await page.getByRole('button', { name: `Release scores for ${NAME}` }).click()
  const dlg = page.getByRole('dialog', { name: `Release scores for ${NAME}?` })
  await dlg.getByRole('button', { name: 'Release scores' }).click()
  await expect(status(page, `${NAME}: scores released to students.`), 'TA, Attempts: released notice').toBeVisible()
  await page.context().close()
})

test('E-QUIZ-06 the student sees the score, the feedback and the correct answers', async () => {
  await quizzes(amara)
  await expect(amara.getByText('Score 5 / 7.'), 'student, Quizzes: score on the row').toBeVisible()
  await amara.getByRole('link', { name: `View ${NAME}` }).click()
  await expect(amara.getByText('Score 5 / 7', { exact: true }), 'student, results: total').toBeVisible()
  await expect(amara.getByText('Good work, review stacks.'), 'student, results: staff feedback').toBeVisible()
  await expect(amara.getByText('Correct answer: def'), 'student, results: key of question 1').toBeVisible()
  await expect(amara.getByText('Correct answer: Stack'), 'student, results: key of question 2').toBeVisible()
  await expect(question(amara, 3).getByText('Accepted answers: binary search'), 'student, results: key of question 3').toBeVisible()
  await expect(question(amara, 3).getByText('Feedback: Spelling accepted.'), 'student, results: question feedback').toBeVisible()
  await expect(question(amara, 3).getByText('3 / 3 points'), 'student, results: question 3 points').toBeVisible()
})

test('E-QUIZ-07 the gradebook column shows the same scores', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await page.goto(`/${slug}/sections/${sid}/grades`)
  await expect(page.getByRole('textbox', { name: `Score for Amara Okafor, ${NAME}` }), 'instructor, gradebook: amara').toHaveValue('5')
  await expect(page.getByRole('textbox', { name: `Score for Priya Raman, ${NAME}` }), 'instructor, gradebook: priya').toHaveValue('5')
  await page.context().close()
})

test('E-QUIZ-08 questions are locked once an attempt exists', async ({ browser }) => {
  const page = await pageAs(browser, P.elena.email, slug)
  await page.goto(`/${slug}/sections/${sid}/quizzes/${quizId}/questions`)
  await expect(page.getByText(/Students have started: you can fix prompts, points and answer keys/), 'instructor, editor: regrade note').toBeVisible()
  await expect(page.getByRole('button', { name: 'Save and regrade' }), 'instructor, editor: Save and regrade').toBeVisible()
  await expect(page.getByRole('button', { name: 'Add multiple choice' }), 'instructor, editor: structure locked').toBeDisabled()
  const res = await page.request.put(`${A}/sections/${sid}/assessments/${quizId}/questions`, {
    headers: { 'Sec-Fetch-Site': 'same-origin' },
    data: { items: [{ kind: 'blank', prompt: 'x', points: 1, answers: ['x'] }] },
  })
  expect(res.status(), 'instructor, API: id-less questions PUT after an attempt').toBe(400)
  expect((await res.json()).error.details[0].field, 'instructor, API: 400 on the missing id').toBe('items[0].id')
  await page.context().close()
})

test('E-QUIZ-09 a student cannot open the question editor', async ({ browser }) => {
  const page = await pageAs(browser, P.priya.email, slug)
  await page.goto(`/${slug}/sections/${sid}/quizzes/${quizId}/questions`)
  await expect(page.getByText("Only this section's staff can change its quizzes."), 'student, editor URL: no access').toBeVisible()
  await expect(page.getByLabel('Prompt'), 'student, editor URL: no form').toHaveCount(0)
  const res = await page.request.put(`${A}/sections/${sid}/assessments/${quizId}/questions`, {
    headers: { 'Sec-Fetch-Site': 'same-origin' },
    data: { items: [{ kind: 'blank', prompt: 'x', points: 1, answers: ['x'] }] },
  })
  expect(res.status(), 'student, API: questions PUT').toBe(403)
  await page.context().close()
})

test('E-QUIZ-10 the take page fits a 390 px phone', async ({ browser }) => {
  const elena = await pageAs(browser, P.elena.email, slug)
  const now = Date.now()
  const q = await api(elena, 'post', `${A}/sections/${sid}/assessments`, {
    kind: 'quiz',
    name: 'Phone check',
    description: 'A quiz with a long description so that wrapping is exercised on a narrow screen.',
    opensAt: new Date(now - 60_000).toISOString(),
    closesAt: new Date(now + 60 * 60_000).toISOString(),
    weight: 0,
  })
  await api(elena, 'put', `${A}/sections/${sid}/assessments/${q.id}/questions`, {
    items: [
      { kind: 'choice', prompt: 'Pick the second choice, which has a rather long text that must wrap inside the card.', points: 1, choices: [{ text: 'First choice with some words', correct: false }, { text: 'Second choice with some more words than the first', correct: true }] },
      { kind: 'blank', prompt: 'Type anything.', points: 1, answers: ['anything'] },
    ],
  })
  await elena.context().close()

  const page = await pageAs(browser, P.daniel.email, slug)
  await page.setViewportSize({ width: 390, height: 844 })
  await quizzes(page)
  await expectNoSideScroll(page, 'student at 390, Quizzes')
  await page.getByRole('link', { name: 'Start Phone check' }).click()
  await page.getByRole('button', { name: 'Start quiz' }).click()
  await expect(page.getByRole('button', { name: 'Submit quiz' }), 'student at 390, take page: Submit').toBeVisible()
  await expectNoSideScroll(page, 'student at 390, take page')
  await page.context().close()
})
