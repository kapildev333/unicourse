import { Link, Outlet } from '@tanstack/react-router'
import { ErrorBanner } from '../../components/error-banner'
import { PageHeader } from '../../components/page-header'
import { TabNav, tabLink } from '../../components/tab-nav'
import { useUniversity } from '../auth/use-me'
import { canManageCourses } from './access'

export function CoursesLayout() {
  const { slug, membership } = useUniversity()
  if (!canManageCourses(membership)) {
    return (
      <>
        <PageHeader title="Courses" />
        <ErrorBanner title="No access">
          Courses are managed by university and department administrators. Your sections are under My sections.
        </ErrorBanner>
      </>
    )
  }
  return (
    <>
      <TabNav label="Courses sections">
        <Link to="/$slug/courses" params={{ slug }} activeOptions={{ exact: true }} {...tabLink}>
          Courses
        </Link>
        <Link to="/$slug/courses/sections" params={{ slug }} {...tabLink}>
          Sections
        </Link>
      </TabNav>
      <Outlet />
    </>
  )
}
