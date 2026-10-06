import { fireEvent, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, it, vi } from 'vitest'
import { mockApi, renderWithQuery } from '../../test-utils'
import { SignInPage } from './sign-in-page'

afterEach(() => vi.unstubAllGlobals())

const submitMagic = (value: string) => {
  fireEvent.change(screen.getByLabelText('Email'), { target: { value } })
  fireEvent.submit(screen.getByRole('form', { name: 'Email me a sign-in link' }))
}

it('renders both providers and the email field', () => {
  renderWithQuery(<SignInPage />)
  expect(screen.getByRole('button', { name: 'Continue with Google' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Continue with Microsoft' })).toBeInTheDocument()
  expect(screen.getByLabelText('Email')).toBeInTheDocument()
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
