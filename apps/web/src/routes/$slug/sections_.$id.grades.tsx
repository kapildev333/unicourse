import { createFileRoute } from '@tanstack/react-router'
import { GradebookPage } from '../../features/grades/gradebook'

export const Route = createFileRoute('/$slug/sections_/$id/grades')({
  component: function Page() {
    const { slug, id } = Route.useParams()
    return <GradebookPage slug={slug} sectionId={id} />
  },
})
