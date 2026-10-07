import { Link, Outlet } from '@tanstack/react-router'
import { ErrorBanner } from '../../components/error-banner'
import { PageHeader } from '../../components/page-header'
import { TabNav, tabLink } from '../../components/tab-nav'
import { useUniversity } from '../auth/use-me'

export function AdminLayout() {
  const { slug, membership } = useUniversity()
  if (membership.role === 'member') {
    return <ErrorBanner title="No access">The admin area is for university and department administrators.</ErrorBanner>
  }
  return (
    <>
      <PageHeader title="Admin" />
      <TabNav label="Admin sections">
        {membership.role === 'admin' && (
          <>
            <Link
              to="/$slug/admin/departments"
              params={{ slug }}
              {...tabLink}
            >
              Departments
            </Link>
            <Link
              to="/$slug/admin/terms"
              params={{ slug }}
              {...tabLink}
            >
              Terms
            </Link>
            <Link
              to="/$slug/admin/members"
              params={{ slug }}
              {...tabLink}
            >
              Members
            </Link>
          </>
        )}
        <Link
          to="/$slug/admin/invitations"
          params={{ slug }}
          {...tabLink}
        >
          Invitations
        </Link>
        {membership.role === 'admin' && (
          <Link
            to="/$slug/admin/audit"
            params={{ slug }}
            {...tabLink}
          >
            Audit log
          </Link>
        )}
      </TabNav>
      <Outlet />
    </>
  )
}
