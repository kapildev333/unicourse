import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import { act } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { SignInPage } from './sign-in-page'

beforeEach(() => localStorage.clear())
afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

const idField = () => screen.getByLabelText('University email or university ID')

const submitMagic = (value: string) => {
  fireEvent.change(idField(), { target: { value } })
  fireEvent.click(screen.getByRole('button', { name: /Email me a sign-in link/ }))
}

it('renders one field, the magic link button and both providers', () => {
  renderWithQuery(<SignInPage />)
  expect(screen.getAllByRole('textbox')).toHaveLength(1)
  expect(screen.getByText(/or your university's short name such as northfield/)).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Continue with Google' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Continue with Microsoft' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Email me a sign-in link' })).toBeInTheDocument()
})

it('shows the inline error when the magic link is asked for a non-email value', () => {
  const fetchMock = mockApi({})
  renderWithQuery(<SignInPage />)
  submitMagic('northfield')
  expect(screen.getByText('Enter your email to get a sign-in link')).toBeInTheDocument()
  expect(fetchMock).not.toHaveBeenCalled()
})

it('shows the invitation info row with the email and prefills the field', () => {
  renderWithQuery(<SignInPage email="new-hire@a.edu" />)
  expect(screen.getByText(/You've been invited\. Sign in as/).textContent).toBe(
    "You've been invited. Sign in as new-hire@a.edu to accept.",
  )
  expect(screen.getByText('new-hire@a.edu')).toHaveClass('whitespace-nowrap')
  expect(idField()).toHaveValue('new-hire@a.edu')
})

it('marks the last used method from localStorage and records the one used', async () => {
  localStorage.setItem('unicourse:last-sign-in', 'google')
  mockApi({ 'POST /auth/magic-link': () => ({ status: 202 }) })
  renderWithQuery(<SignInPage />)
  expect(within(screen.getByRole('button', { name: /Continue with Google/ })).getByText('Last used')).toBeInTheDocument()
  expect(screen.getAllByText('Last used')).toHaveLength(1)
  submitMagic('alice@a.edu')
  await screen.findByText('Check your email')
  expect(localStorage.getItem('unicourse:last-sign-in')).toBe('magic')
})

it('resends the same email, says so, and waits 30 s before the next resend', async () => {
  vi.useFakeTimers({ shouldAdvanceTime: true })
  const fetchMock = mockApi({ 'POST /auth/magic-link': () => ({ status: 202 }) })
  renderWithQuery(<SignInPage />)
  submitMagic('alice@a.edu')
  await screen.findByText('Check your email')
  const again = screen.getByRole('button', { name: 'Send the link again' })
  expect(again).toBeDisabled()
  expect(screen.getByText(/You can send it again in 30 s/)).toBeInTheDocument()
  await act(() => vi.advanceTimersByTimeAsync(30_000))
  expect(again).toBeEnabled()
  fireEvent.click(again)
  expect(await screen.findByText('Sent again.')).toBeInTheDocument()
  expect(fetchMock).toHaveBeenCalledTimes(2)
  expect(await fetchMock.mock.calls[1][0].json()).toEqual({ email: 'alice@a.edu' })
  expect(again).toBeDisabled()
})

it('"use a different email" is an inline button that returns to the form', async () => {
  mockApi({ 'POST /auth/magic-link': () => ({ status: 202 }) })
  renderWithQuery(<SignInPage />)
  submitMagic('alice@a.edu')
  await screen.findByText('Check your email')
  fireEvent.click(screen.getByRole('button', { name: 'use a different email' }))
  expect(idField()).toBeInTheDocument()
})

it('requests a magic link and shows the check-your-email screen', async () => {
  const fetchMock = mockApi({ 'POST /auth/magic-link': () => ({ status: 202 }) })
  renderWithQuery(<SignInPage />)
  submitMagic('alice@a.edu')
  expect(await screen.findByRole('heading', { name: 'Check your email' })).toBeInTheDocument()
  expect(screen.getByText(/alice@a\.edu/)).toBeInTheDocument()
  expect(await fetchMock.mock.calls[0][0].json()).toEqual({ email: 'alice@a.edu' })
})

it('shows the envelope message on 429', async () => {
  mockApi({
    'POST /auth/magic-link': () => ({
      status: 429,
      body: { error: { code: 'rate_limited', message: 'Too many requests, try again later', details: [] } },
    }),
  })
  renderWithQuery(<SignInPage />)
  submitMagic('alice@a.edu')
  expect(await screen.findByRole('alert')).toHaveTextContent('Too many requests, try again later')
})

it('shows callback error copy', () => {
  renderWithQuery(<SignInPage error="forbidden" />)
  expect(screen.getByRole('alert')).toHaveTextContent(
    "This email can't sign in to UniCourse. Ask your university administrator to invite you.",
  )
})

it('forwards next only when it is a same-origin path', async () => {
  const fetchMock = mockApi({ 'POST /auth/magic-link': () => ({ status: 202 }) })
  const bad = renderWithQuery(<SignInPage next="//evil.com" />)
  submitMagic('a@a.edu')
  await screen.findByText('Check your email')
  expect(await fetchMock.mock.calls[0][0].json()).toEqual({ email: 'a@a.edu' })
  bad.unmount()

  renderWithQuery(<SignInPage next="/x/admin" />)
  submitMagic('a@a.edu')
  await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2))
  expect(await fetchMock.mock.calls[1][0].json()).toEqual({ email: 'a@a.edu', next: '/x/admin' })
})

it('starts provider sign-in with a university id and redirects', async () => {
  const assign = vi.fn()
  vi.stubGlobal('location', { ...window.location, assign })
  const fetchMock = mockApi({ 'POST /auth/oidc/google/start': () => ({ body: { url: 'https://idp.example/auth' } }) })
  renderWithQuery(<SignInPage />)
  fireEvent.change(screen.getByLabelText('University email or university ID'), { target: { value: 'example' } })
  fireEvent.click(screen.getByRole('button', { name: 'Continue with Google' }))
  await waitFor(() => expect(assign).toHaveBeenCalledWith('https://idp.example/auth'))
  expect(await fetchMock.mock.calls[0][0].json()).toEqual({ slug: 'example' })
})

it('rejects an empty identifier locally and focuses the field', async () => {
  const fetchMock = mockApi({})
  renderWithQuery(<SignInPage />)
  fireEvent.click(screen.getByRole('button', { name: 'Continue with Google' }))
  expect(await screen.findByText('Enter your university email or university ID.')).toBeInTheDocument()
  expect(fetchMock).not.toHaveBeenCalled()
  await waitFor(() => expect(screen.getByLabelText('University email or university ID')).toHaveFocus())
})

const notFound = { error: { code: 'not_found', message: 'nope', details: [] } }

it('focuses the banner when the provider start returns 404 and quotes the submitted value', async () => {
  mockApi({ 'POST /auth/oidc/google/start': () => ({ status: 404, body: notFound }) })
  renderWithQuery(<SignInPage />)
  const input = screen.getByLabelText('University email or university ID')
  fireEvent.change(input, { target: { value: 'nowhere' } })
  fireEvent.click(screen.getByRole('button', { name: 'Continue with Google' }))
  const banner = await screen.findByRole('alert')
  expect(banner).toHaveTextContent('“nowhere”')
  await waitFor(() => expect(banner).toHaveFocus())
  fireEvent.change(input, { target: { value: 'edited' } })
  expect(banner).toHaveTextContent('“nowhere”')
})

it('focuses the banner on magic-link 429', async () => {
  mockApi({
    'POST /auth/magic-link': () => ({
      status: 429,
      body: { error: { code: 'rate_limited', message: 'Slow down', details: [] } },
    }),
  })
  renderWithQuery(<SignInPage />)
  submitMagic('alice@a.edu')
  const banner = await screen.findByRole('alert')
  await waitFor(() => expect(banner).toHaveFocus())
})

it('does not record Last used when the provider start fails', async () => {
  mockApi({ 'POST /auth/oidc/google/start': () => ({ status: 404, body: notFound }) })
  renderWithQuery(<SignInPage />)
  fireEvent.change(idField(), { target: { value: 'nowhere' } })
  fireEvent.click(screen.getByRole('button', { name: 'Continue with Google' }))
  await screen.findByRole('alert')
  expect(localStorage.getItem('unicourse:last-sign-in')).toBeNull()
})

it('rate_limited callback copy names the real window, an hour', () => {
  renderWithQuery(<SignInPage error="rate_limited" />)
  expect(screen.getByRole('alert')).toHaveTextContent('Too many sign-in attempts. Try again in an hour.')
})
