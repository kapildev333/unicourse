import { createFileRoute } from '@tanstack/react-router'
import { SectionDetailPage } from '../../features/courses/section-detail'

function Page() {
  const { slug, id } = Route.useParams()
  return <SectionDetailPage slug={slug} sectionId={id} />
}

export const Route = createFileRoute('/$slug/sections/$id')({ component: Page })
