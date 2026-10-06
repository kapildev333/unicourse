import { createFileRoute } from '@tanstack/react-router'
import { MembersPage } from '../../../features/admin/members'

function Page() {
  const { slug } = Route.useParams()
  return <MembersPage slug={slug} />
}

export const Route = createFileRoute('/$slug/admin/members')({ component: Page })
