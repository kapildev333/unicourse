import { Navigate, createFileRoute } from '@tanstack/react-router'
import { useUniversity } from '../../../features/auth/use-me'

// Members never get here: AdminLayout renders the "No access" panel instead of the Outlet.
function AdminIndex() {
  const { slug, membership } = useUniversity()
  return membership.role === 'admin' ? (
    <Navigate to="/$slug/admin/departments" params={{ slug }} replace />
  ) : (
    <Navigate to="/$slug/admin/invitations" params={{ slug }} replace />
  )
}

export const Route = createFileRoute('/$slug/admin/')({ component: AdminIndex })
