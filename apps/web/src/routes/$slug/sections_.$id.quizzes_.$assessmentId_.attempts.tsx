import { createFileRoute } from '@tanstack/react-router'
import { AttemptsPage } from '../../features/quiz/attempts-review'

export const Route = createFileRoute('/$slug/sections_/$id/quizzes_/$assessmentId_/attempts')({
  validateSearch: (s: Record<string, unknown>): { user?: string } => ({ user: typeof s.user === 'string' ? s.user : undefined }),
  component: function Page() {
    const { slug, id, assessmentId } = Route.useParams()
    const { user } = Route.useSearch()
    return <AttemptsPage slug={slug} sectionId={id} assessmentId={assessmentId} user={user} />
  },
})
