import { Link, Outlet } from '@tanstack/react-router'
import { ErrorBanner } from '../../components/error-banner'
import { useUniversity } from '../auth/use-me'

const tabBase =
  '-mb-px border-b-2 px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-focus'
const tabActive = { className: 'border-accent text-accent' }
const tabInactive = { className: 'border-transparent text-ink-muted hover:text-ink' }

export function AdminLayout() {
  const { slug, membership } = useUniversity()
  if (membership.role === 'member') {
    return <ErrorBanner title="No access">The admin area is for university and department administrators.</ErrorBanner>
  }
  return (
    <>
      <nav aria-label="Admin sections" className="mb-6 flex gap-1 border-b border-line">
        {membership.role === 'admin' && (
          <>
            <Link
              to="/$slug/admin/departments"
              params={{ slug }}
              className={tabBase}
              activeProps={tabActive}
              inactiveProps={tabInactive}
            >
              Departments
            </Link>
            <Link
              to="/$slug/admin/terms"
              params={{ slug }}
              className={tabBase}
              activeProps={tabActive}
              inactiveProps={tabInactive}
            >
              Terms
            </Link>
            <Link
              to="/$slug/admin/members"
              params={{ slug }}
              className={tabBase}
              activeProps={tabActive}
              inactiveProps={tabInactive}
            >
              Members
            </Link>
          </>
        )}
        <Link
          to="/$slug/admin/invitations"
          params={{ slug }}
          className={tabBase}
          activeProps={tabActive}
          inactiveProps={tabInactive}
        >
          Invitations
        </Link>
        {membership.role === 'admin' && (
          <Link
            to="/$slug/admin/audit"
            params={{ slug }}
            className={tabBase}
            activeProps={tabActive}
            inactiveProps={tabInactive}
          >
            Audit log
          </Link>
        )}
      </nav>
      <Outlet />
    </>
  )
}
