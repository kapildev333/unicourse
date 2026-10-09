import { createFileRoute } from '@tanstack/react-router'
import { SubmissionsPage } from '../../features/assignment/submissions-grading'

export const Route = createFileRoute('/$slug/sections_/$id/assignments_/$assessmentId_/submissions')({
  validateSearch: (s: Record<string, unknown>): { user?: string } => (typeof s.user === 'string' && s.user ? { user: s.user } : {}),
  component: function Page() {
    const { slug, id, assessmentId } = Route.useParams()
    const { user } = Route.useSearch()
    return <SubmissionsPage slug={slug} sectionId={id} assessmentId={assessmentId} userId={user} />
  },
})
