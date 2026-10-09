import { createFileRoute } from '@tanstack/react-router'
import { QuizzesPage } from '../../features/quiz/quiz-list'

export const Route = createFileRoute('/$slug/sections_/$id/quizzes')({
  component: function Page() {
    const { slug, id } = Route.useParams()
    return <QuizzesPage slug={slug} sectionId={id} />
  },
})
