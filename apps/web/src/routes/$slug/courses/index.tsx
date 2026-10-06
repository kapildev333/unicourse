import { createFileRoute } from '@tanstack/react-router'
import { CourseListPage } from '../../../features/courses/course-list'

function Page() {
  const { slug } = Route.useParams()
  const { departmentId } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <CourseListPage
      slug={slug}
      departmentId={departmentId}
      onDepartmentChange={(id) => void navigate({ search: { departmentId: id || undefined }, replace: true })}
    />
  )
}

export const Route = createFileRoute('/$slug/courses/')({
  validateSearch: (s: Record<string, unknown>): { departmentId?: string } => ({
    departmentId: typeof s.departmentId === 'string' ? s.departmentId : undefined,
  }),
  component: Page,
})
