import { createFileRoute } from '@tanstack/react-router'
import { InvitationsPage } from '../../../features/admin/invitations'
import { useUniversity } from '../../../features/auth/use-me'

function Page() {
  const { slug, membership } = useUniversity()
  return <InvitationsPage slug={slug} userRole={membership.role} />
}

export const Route = createFileRoute('/$slug/admin/invitations')({ component: Page })
