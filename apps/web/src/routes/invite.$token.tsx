import { createFileRoute } from '@tanstack/react-router'
import { AcceptInvite } from '../features/auth/accept-invite'

export const Route = createFileRoute('/invite/$token')({
  validateSearch: (s: Record<string, unknown>): { email?: string } => ({
    email: typeof s.email === 'string' ? s.email : undefined,
  }),
  component: InvitePage,
})

function InvitePage() {
  return <AcceptInvite email={Route.useSearch().email} />
}
