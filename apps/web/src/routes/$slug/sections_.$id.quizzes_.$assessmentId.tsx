import { createFileRoute } from '@tanstack/react-router'
import { TakeQuizPage } from '../../features/quiz/take-quiz'

export const Route = createFileRoute('/$slug/sections_/$id/quizzes_/$assessmentId')({
  component: function Page() {
    const { slug, id, assessmentId } = Route.useParams()
    return <TakeQuizPage slug={slug} sectionId={id} assessmentId={assessmentId} />
  },
})
