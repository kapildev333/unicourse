import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import type { Assessment, Submission } from '../quiz/queries'
import { mine, renderSection, sectionFixture } from '../section/test-helpers'
import { AssignmentPage } from './submission-form'

afterEach(() => vi.unstubAllGlobals())

const S = '/universities/a/sections/s1'
const asg = (over: Partial<Assessment> = {}): Assessment => ({
  id: 'q1',
  sectionId: 's1',
  kind: 'assignment',
  name: 'Essay 1',
  description: 'Write about <b>trees</b>.',
  maxPoints: 20,
  weight: 20,
  opensAt: '2020-01-01T09:00:00Z',
  closesAt: '2099-01-01T09:00:00Z',
  published: false,
  position: 1,
  ...over,
})
const file = (id: string, name: string) => ({ id, fileName: name, contentType: 'application/pdf', sizeBytes: 1234, position: 1, uploadedAt: '2026-10-09T15:12:00Z' })
const sub = (over: Partial<Submission> = {}): Submission => ({ userId: 'u0', body: 'My essay', submittedAt: '2026-10-09T15:12:00Z', files: [], late: false, penaltyPercent: 0, ...over })
const err = (status: number, code: string, details: object[] = []) => ({ status, body: { error: { code, message: 'm', details } } })
const M = `${S}/assessments/q1/submissions/me`
const sent = (fn: ReturnType<typeof mockApi>, method: string) => fn.mock.calls.filter(([r]) => r.method === method)

function setup(a: Assessment, s: Submission | null, routes: Routes = {}) {
  const fn = mockApi({
    [`GET ${S}`]: () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: mine('student') }),
    'GET /upload-policy': () => ({ body: { maxBytes: 104857600, allowedExtensions: ['pdf', 'png'] } }),
    [`GET ${S}/assessments`]: () => ({ body: { items: [a] } }),
    [`GET ${M}`]: () => (s ? { body: s } : err(404, 'not_found')),
    ...routes,
  })
  renderSection({ path: '/a/sections/s1/assignments/q1', ui: <AssignmentPage slug="a" sectionId="s1" assessmentId="q1" />, userId: 'u0' })
  return fn
}

function stubXhr(reply: (n: number) => { status: number; body: unknown }) {
  const posts: FormData[] = []
  class Xhr {
    upload = {}
    status = 0
    statusText = ''
    responseText = ''
    onload?: () => void
    open() {}
    setRequestHeader() {}
    getResponseHeader() {
      return null
    }
    send(form: FormData) {
      posts.push(form)
      void Promise.resolve().then(() => {
        const r = reply(posts.length)
        this.status = r.status
        this.responseText = JSON.stringify(r.body)
        this.onload?.()
      })
    }
  }
  vi.stubGlobal('XMLHttpRequest', Xhr)
  return posts
}

it('upcoming: description, no form', async () => {
  setup(asg({ opensAt: '2098-01-01T00:00:00Z' }), null)
  expect(await screen.findByText(/^This assignment opens /)).toBeInTheDocument()
  expect(screen.getByText('Write about <b>trees</b>.')).toBeInTheDocument() // plain text, never markup
  expect(screen.queryByRole('textbox')).toBeNull()
  expect(screen.queryByRole('button', { name: 'Save' })).toBeNull()
})

it('open: shows the answer, window and Submitted time; Save PUTs the body', async () => {
  const fn = setup(asg(), sub(), { [`PUT ${M}`]: (r) => r.json().then((b) => ({ body: sub(b) })) })
  const box = await screen.findByRole('textbox', { name: 'Your answer' })
  expect(box).toHaveValue('My essay')
  expect(screen.getByText(/^Opens .* · Due /)).toBeInTheDocument()
  expect(screen.getByText(/^Submitted /)).toBeInTheDocument()
  fireEvent.change(box, { target: { value: ' New text ' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(await screen.findByText('Submission saved.')).toBeInTheDocument()
  expect(await sent(fn, 'PUT')[0][0].json()).toEqual({ body: 'New text' })
})

it('empty answer and no files: local error, no request', async () => {
  const fn = setup(asg(), null)
  await screen.findByText('Not submitted yet.')
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(await screen.findByText('Write an answer or upload a file.')).toBeInTheDocument()
  expect(sent(fn, 'PUT')).toHaveLength(0)
})

it('400 on body shows the field copy', async () => {
  setup(asg(), null, { [`PUT ${M}`]: () => err(400, 'validation_failed', [{ field: 'body', message: 'must be at most 20000 characters' }]) })
  fireEvent.change(await screen.findByRole('textbox', { name: 'Your answer' }), { target: { value: 'x' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect(await screen.findByText('Use at most 20,000 characters of plain text.')).toBeInTheDocument()
})

it('lists files with cookie-bound download links and Remove deletes', async () => {
  const fn = setup(asg(), sub({ files: [file('f1', 'essay.pdf')] }), { [`DELETE ${M}/files/f1`]: () => ({ status: 204 }) })
  const dl = await screen.findByRole('link', { name: 'Download essay.pdf' })
  expect(dl).toHaveAttribute('href', `/api/v1${M}/files/f1/download`)
  expect(screen.getByText('1 of 10')).toBeInTheDocument()
  fireEvent.click(screen.getByRole('button', { name: 'Remove essay.pdf' }))
  expect(await screen.findByText('essay.pdf removed.')).toBeInTheDocument()
  expect(sent(fn, 'DELETE')).toHaveLength(1)
})

it('upload: native input with accept; two files make two POSTs with status', async () => {
  setup(asg(), sub())
  const posts = stubXhr(() => ({ status: 201, body: sub() }))
  const input = await screen.findByLabelText('Files for your submission')
  await waitFor(() => expect(input).toHaveAttribute('accept', '.pdf,.png'))
  expect(screen.getByText(/^Up to 10 files, each up to 100 MB/)).toBeInTheDocument()
  fireEvent.change(input, { target: { files: [new File(['x'], 'a.pdf'), new File(['x'], 'b.png')] } })
  await waitFor(() => expect(screen.getAllByRole('status').map((s) => s.textContent)).toContain('Uploaded 2 files to your submission.'))
  expect(posts).toHaveLength(2)
  expect(posts[0].get('file')).toBeInstanceOf(File)
})

it('refuses files beyond 10 before sending and lists per-file errors', async () => {
  const many = Array.from({ length: 9 }, (_, i) => file(`f${i}`, `f${i}.pdf`))
  setup(asg(), sub({ files: many }))
  const posts = stubXhr(() => ({ status: 201, body: sub() }))
  fireEvent.change(await screen.findByLabelText('Files for your submission'), {
    target: { files: [new File(['x'], 'a.pdf'), new File(['x'], 'b.pdf'), new File(['x'], 'c.exe')] },
  })
  const alert = await screen.findByRole('alert')
  await waitFor(() => expect(alert).toHaveTextContent('b.pdf: A submission can have at most 10 files.'))
  expect(alert).toHaveTextContent('c.exe')
  expect(posts).toHaveLength(1)
})

it('409 on save shows the due-date banner and the read-only view', async () => {
  setup(asg(), sub(), { [`PUT ${M}`]: () => err(409, 'conflict') })
  fireEvent.click(await screen.findByRole('button', { name: 'Save' }))
  expect(await screen.findByText('The due date has passed')).toBeInTheDocument()
  expect(screen.getByText('Your submission can no longer be changed.', { selector: 'div' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Save' })).toBeNull()
  expect(screen.getByText('My essay')).toBeInTheDocument()
})

it('closed: read-only with files, or "You didn\'t submit"', async () => {
  setup(asg({ closesAt: '2021-01-01T00:00:00Z' }), sub({ files: [file('f1', 'essay.pdf')] }))
  expect(await screen.findByText('The due date has passed. Your submission can no longer be changed.')).toBeInTheDocument()
  expect(screen.getByRole('link', { name: 'Download essay.pdf' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: /Save|Remove|Upload/ })).toBeNull()
})

it('closed without a submission', async () => {
  setup(asg({ closesAt: '2021-01-01T00:00:00Z' }), null)
  expect(await screen.findByText("You didn't submit this assignment.")).toBeInTheDocument()
})

it('grade block only once published', async () => {
  setup(asg({ closesAt: '2021-01-01T00:00:00Z', published: true }), sub({ score: 17, feedback: 'Good use of sources.' }))
  expect(await screen.findByText('Score 17 / 20')).toBeInTheDocument()
  expect(screen.getByText('Good use of sources.')).toBeInTheDocument()
})

it('no grade block while unpublished', async () => {
  setup(asg({ closesAt: '2021-01-01T00:00:00Z' }), sub({ score: 17 }))
  await screen.findByText('Your submission')
  expect(screen.queryByRole('heading', { name: 'Grade' })).toBeNull()
})

it('400 on body with another message shows the server sentence', async () => {
  setup(asg(), null, { [`PUT ${M}`]: () => err(400, 'validation_failed', [{ field: 'body', message: 'a submission needs text or a file' }]) })
  fireEvent.change(await screen.findByRole('textbox', { name: 'Your answer' }), { target: { value: 'x' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save' }))
  expect((await screen.findAllByText('A submission needs text or a file.')).length).toBeGreaterThan(0)
  expect(screen.queryByText('Use at most 20,000 characters of plain text.')).toBeNull()
})
