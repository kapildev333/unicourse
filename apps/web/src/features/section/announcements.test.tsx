import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, type Routes } from '../../test-utils'
import { AnnouncementsPage } from './announcements'
import { mine, renderSection, sectionFixture } from './test-helpers'

afterEach(() => vi.unstubAllGlobals())

const ann = (n: number, over: object = {}) => ({
  id: `a${n}`,
  sectionId: 's1',
  title: `Post ${n}`,
  body: `Body ${n}`,
  author: { userId: 'u-ins', displayName: 'Dana Reyes', email: 'd@a.edu' },
  emailed: false,
  createdAt: '2026-10-05T14:03:00Z',
  updatedAt: '2026-10-05T14:03:00Z',
  ...over,
})
const err = (status: number, code: string, details: unknown[] = []) => ({ status, body: { error: { code, message: 'm', details } } })

function setup(routes: Routes, o: { editor?: boolean } = {}) {
  const editor = o.editor ?? true
  const fn = mockApi({
    'GET /universities/a/sections/s1': () => ({ body: sectionFixture() }),
    'GET /universities/a/me/sections': () => ({ body: mine(editor ? undefined : 'student') }),
    ...routes,
  })
  renderSection({ path: '/a/sections/s1/announcements', ui: <AnnouncementsPage slug="a" sectionId="s1" />, userId: editor ? 'u-ins' : 'u0' })
  return fn
}
const list = (items: unknown[], nextCursor?: string) => () => ({ body: { items, nextCursor } })
const A = 'GET /universities/a/sections/s1/announcements'
const sent = async (fn: ReturnType<typeof mockApi>, method: string) =>
  (await fn.mock.calls.find(([r]) => r.method === method)![0].json()) as unknown

it('lists newest first with Emailed and Edited markers', async () => {
  setup({ [A]: list([ann(2, { emailed: true, updatedAt: '2026-10-06T10:00:00Z' }), ann(1)]) })
  await screen.findByText('Post 2')
  expect(screen.getAllByRole('heading', { level: 3 }).map((h) => h.textContent)).toEqual(['Post 2', 'Post 1'])
  const [first, second] = screen.getAllByRole('article')
  expect(within(first).getByText('Emailed')).toBeInTheDocument()
  expect(within(first).getByText(/Posted by Dana Reyes/)).toHaveTextContent('· Edited')
  expect(within(second).queryByText('Emailed')).toBeNull()
  expect(within(second).getByText(/Posted by Dana Reyes/)).not.toHaveTextContent('Edited')
})

it('loads more with the cursor', async () => {
  const seen: (string | null)[] = []
  setup({
    [A]: (req) => {
      const c = new URL(req.url).searchParams.get('cursor')
      seen.push(c)
      return { body: c ? { items: [ann(3)] } : { items: [ann(2), ann(1)], nextCursor: 'c2' } }
    },
  })
  fireEvent.click(await screen.findByRole('button', { name: 'Load more' }))
  await screen.findByText('Post 3')
  expect(seen).toEqual([null, 'c2'])
  expect(screen.queryByRole('button', { name: 'Load more' })).toBeNull()
})

it('shows edit controls to editors only', async () => {
  setup({ [A]: list([ann(1)]) })
  expect(await screen.findByRole('button', { name: 'New announcement' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Edit Post 1' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Delete Post 1' })).toBeInTheDocument()
})

it('hides edit controls from students', async () => {
  setup({ [A]: list([ann(1)]) }, { editor: false })
  await screen.findByText('Post 1')
  expect(screen.queryByRole('button', { name: 'New announcement' })).toBeNull()
  expect(screen.queryByRole('button', { name: /^Edit / })).toBeNull()
  expect(screen.queryByRole('button', { name: /^Delete / })).toBeNull()
})

it('has empty states for editors and readers', async () => {
  setup({ [A]: list([]) })
  await screen.findByText('No announcements yet')
  expect(screen.getAllByRole('button', { name: 'New announcement' })).toHaveLength(1)
})

it('has an empty state for students', async () => {
  setup({ [A]: list([]) }, { editor: false })
  expect(await screen.findByText("Staff haven't posted anything in this section yet.")).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'New announcement' })).toBeNull()
})

async function fillAndPost(title = 'Hello', body = 'World') {
  fireEvent.click(await screen.findByRole('button', { name: 'New announcement' }))
  fireEvent.change(screen.getByLabelText('Title'), { target: { value: title } })
  fireEvent.change(screen.getByLabelText('Message'), { target: { value: body } })
}

it('posts with the email box checked by default', async () => {
  const fn = setup({ [A]: list([ann(1)]), 'POST /universities/a/sections/s1/announcements': () => ({ status: 201, body: ann(9, { emailed: true }) }) })
  await fillAndPost()
  expect(screen.getByRole('checkbox', { name: 'Also email everyone in this section' })).toBeChecked()
  fireEvent.click(screen.getByRole('button', { name: 'Post announcement' }))
  await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Announcement posted and emailed to the section.'))
  expect(await sent(fn, 'POST')).toEqual({ title: 'Hello', body: 'World', email: true })
})

it('posts without email when unchecked', async () => {
  const fn = setup({ [A]: list([ann(1)]), 'POST /universities/a/sections/s1/announcements': () => ({ status: 201, body: ann(9) }) })
  await fillAndPost()
  fireEvent.click(screen.getByRole('checkbox', { name: 'Also email everyone in this section' }))
  fireEvent.click(screen.getByRole('button', { name: 'Post announcement' }))
  await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Announcement posted. No email was sent.'))
  expect(await sent(fn, 'POST')).toEqual({ title: 'Hello', body: 'World', email: false })
})

it('says when nobody was emailed', async () => {
  setup({ [A]: list([ann(1)]), 'POST /universities/a/sections/s1/announcements': () => ({ status: 201, body: ann(9) }) })
  await fillAndPost()
  fireEvent.click(screen.getByRole('button', { name: 'Post announcement' }))
  await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('nobody else is in this section yet'))
})

it('shows a field error and focuses it', async () => {
  setup({
    [A]: list([ann(1)]),
    'POST /universities/a/sections/s1/announcements': () => err(400, 'validation_failed', [{ field: 'title', message: 'must be at most 200 characters' }]),
  })
  await fillAndPost()
  fireEvent.click(screen.getByRole('button', { name: 'Post announcement' }))
  expect(await screen.findByText('Enter a title of at most 200 characters, on one line.')).toBeInTheDocument()
  const t = screen.getByLabelText('Title')
  expect(t).toHaveAttribute('aria-invalid', 'true')
  await waitFor(() => expect(t).toHaveFocus())
})

it('edits changed fields only, without the email box', async () => {
  const fn = setup({ [A]: list([ann(1)]), 'PATCH /universities/a/sections/s1/announcements/a1': () => ({ body: ann(1) }) })
  fireEvent.click(await screen.findByRole('button', { name: 'Edit Post 1' }))
  expect(screen.queryByRole('checkbox')).toBeNull()
  expect(screen.getByText('Edits are not emailed again.')).toBeInTheDocument()
  fireEvent.change(screen.getByLabelText('Message'), { target: { value: 'Changed' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save changes' }))
  await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Announcement saved. Edits are not emailed.'))
  expect(await sent(fn, 'PATCH')).toEqual({ body: 'Changed' })
})

it('deletes after confirming and focuses the heading', async () => {
  const fn = setup({ [A]: list([ann(1)]), 'DELETE /universities/a/sections/s1/announcements/a1': () => ({ status: 204 }) })
  fireEvent.click(await screen.findByRole('button', { name: 'Delete Post 1' }))
  const d = screen.getByRole('dialog', { name: 'Delete announcement?' })
  expect(d).toHaveTextContent('"Post 1" will be removed')
  fireEvent.click(within(d).getByRole('button', { name: 'Delete announcement' }))
  await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Announcement deleted.'))
  expect(fn.mock.calls.some(([r]) => r.method === 'DELETE')).toBe(true)
  expect(screen.getByRole('heading', { name: 'Announcements', level: 2 })).toHaveFocus()
})

it('shows no access on 403', async () => {
  setup({ [A]: () => err(403, 'forbidden') }, { editor: false })
  expect(await screen.findByText('No access')).toBeInTheDocument()
  expect(screen.getByText("Only this section's staff and enrolled students can read its announcements.")).toBeInTheDocument()
})
