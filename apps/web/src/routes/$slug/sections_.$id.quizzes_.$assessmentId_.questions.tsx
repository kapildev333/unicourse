import { createFileRoute } from '@tanstack/react-router'
import { QuestionsPage } from '../../features/quiz/question-editor'

export const Route = createFileRoute('/$slug/sections_/$id/quizzes_/$assessmentId_/questions')({
  component: function Page() {
    const { slug, id, assessmentId } = Route.useParams()
    return <QuestionsPage slug={slug} sectionId={id} assessmentId={assessmentId} />
  },
})
