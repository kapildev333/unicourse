import { createFileRoute } from '@tanstack/react-router'
import { DepartmentsPage } from '../../../features/admin/departments'

function Page() {
  const { slug } = Route.useParams()
  return <DepartmentsPage slug={slug} />
}

export const Route = createFileRoute('/$slug/admin/departments')({
  component: Page,
})
