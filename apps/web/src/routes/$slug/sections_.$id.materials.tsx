import { createFileRoute } from '@tanstack/react-router'
import { MaterialsPage } from '../../features/section/materials'

export const Route = createFileRoute('/$slug/sections_/$id/materials')({
  component: function Page() {
    const { slug, id } = Route.useParams()
    return <MaterialsPage slug={slug} sectionId={id} />
  },
})
