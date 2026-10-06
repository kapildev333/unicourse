import { Navigate, createFileRoute } from '@tanstack/react-router'
import { useUniversity } from '../../../features/auth/use-me'

// `as` casts: see adminTo in admin-layout.tsx (routes arrive in P1-L4b/c).
// Members never get here: AdminLayout renders the "No access" panel instead of the Outlet.
function AdminIndex() {
  const { slug, membership } = useUniversity()
  return membership.role === 'admin' ? (
    <Navigate to={'/$slug/admin/departments' as '/$slug/admin'} params={{ slug }} replace />
  ) : (
    <Navigate to={'/$slug/admin/invitations' as '/$slug/admin'} params={{ slug }} replace />
  )
}

export const Route = createFileRoute('/$slug/admin/')({ component: AdminIndex })
