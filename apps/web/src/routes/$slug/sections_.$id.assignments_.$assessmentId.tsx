import { createFileRoute } from '@tanstack/react-router'
import { AssignmentPage } from '../../features/assignment/submission-form'

export const Route = createFileRoute('/$slug/sections_/$id/assignments_/$assessmentId')({
  component: function Page() {
    const { slug, id, assessmentId } = Route.useParams()
    return <AssignmentPage slug={slug} sectionId={id} assessmentId={assessmentId} />
  },
})
