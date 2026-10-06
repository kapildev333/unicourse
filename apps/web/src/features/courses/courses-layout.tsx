import { Link, Outlet } from '@tanstack/react-router'
import { ErrorBanner } from '../../components/error-banner'
import { PageHeader } from '../../components/page-header'
import { useUniversity } from '../auth/use-me'
import { canManageCourses } from './access'

const tabBase =
  '-mb-px border-b-2 px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-focus'
const tabActive = { className: 'border-accent text-accent' }
const tabInactive = { className: 'border-transparent text-ink-muted hover:text-ink' }

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
      <nav aria-label="Courses sections" className="mb-6 flex gap-1 border-b border-line">
        <Link
          to="/$slug/courses"
          params={{ slug }}
          activeOptions={{ exact: true }}
          className={tabBase}
          activeProps={tabActive}
          inactiveProps={tabInactive}
        >
          Courses
        </Link>
        <Link
          to="/$slug/courses/sections"
          params={{ slug }}
          className={tabBase}
          activeProps={tabActive}
          inactiveProps={tabInactive}
        >
          Sections
        </Link>
      </nav>
      <Outlet />
    </>
  )
}
