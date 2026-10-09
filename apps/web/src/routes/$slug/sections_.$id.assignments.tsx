import { createFileRoute } from '@tanstack/react-router'
import { AssignmentsPage } from '../../features/assignment/assignment-list'

export const Route = createFileRoute('/$slug/sections_/$id/assignments')({
  component: function Page() {
    const { slug, id } = Route.useParams()
    return <AssignmentsPage slug={slug} sectionId={id} />
  },
})
