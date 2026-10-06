import { Link, Outlet } from '@tanstack/react-router'
import { ErrorBanner } from '../../components/error-banner'
import { useUniversity } from '../auth/use-me'

// ponytail: admin sub-routes land in P1-L4b/c; until their files exist the typed router
// does not know these paths, so `to` is cast. Drop `adminTo` once the routes are generated.
const adminTo = (page: string) => `/$slug/admin/${page}` as '/$slug/admin'
const tabBase =
  '-mb-px border-b-2 border-transparent px-3 py-2 text-sm font-medium text-ink-muted hover:text-ink focus-visible:outline-2 focus-visible:outline-focus'
const tabActive = { className: 'border-accent text-accent', 'aria-current': 'page' as const }

export function AdminLayout() {
  const { slug, membership } = useUniversity()
  if (membership.role === 'member') {
    return (
      <ErrorBanner title="No access">The admin area is for university and department administrators.</ErrorBanner>
    )
  }
  return (
    <>
      <nav aria-label="Admin sections" className="mb-6 flex gap-1 border-b border-line">
        {membership.role === 'admin' && (
          <>
            <Link to={adminTo('departments')} params={{ slug }} className={tabBase} activeProps={tabActive}>
              Departments
            </Link>
            <Link to={adminTo('terms')} params={{ slug }} className={tabBase} activeProps={tabActive}>
              Terms
            </Link>
            <Link to={adminTo('members')} params={{ slug }} className={tabBase} activeProps={tabActive}>
              Members
            </Link>
          </>
        )}
        <Link to={adminTo('invitations')} params={{ slug }} className={tabBase} activeProps={tabActive}>
          Invitations
        </Link>
        {membership.role === 'admin' && (
          <Link to={adminTo('audit')} params={{ slug }} className={tabBase} activeProps={tabActive}>
            Audit log
          </Link>
        )}
      </nav>
      <Outlet />
    </>
  )
}
