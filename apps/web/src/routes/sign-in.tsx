import { createFileRoute } from '@tanstack/react-router'
import { SignInPage } from '../features/auth/sign-in-page'

const str = (v: unknown) => (typeof v === 'string' ? v : undefined)

export const Route = createFileRoute('/sign-in')({
  validateSearch: (s: Record<string, unknown>): { next?: string; error?: string; email?: string } => ({
    next: str(s.next),
    error: str(s.error),
    email: str(s.email),
  }),
  component: SignInRoute,
})

function SignInRoute() {
  return <SignInPage {...Route.useSearch()} />
}
