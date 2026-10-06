import { createFileRoute } from '@tanstack/react-router'
import { AuditLogPage } from '../../../features/admin/audit-log'

function Page() {
  const { slug } = Route.useParams()
  return <AuditLogPage slug={slug} />
}

export const Route = createFileRoute('/$slug/admin/audit')({ component: Page })
