import { createFileRoute } from '@tanstack/react-router'
import { AnnouncementsPage } from '../../features/section/announcements'

export const Route = createFileRoute('/$slug/sections_/$id/announcements')({
  component: function Page() {
    const { slug, id } = Route.useParams()
    return <AnnouncementsPage slug={slug} sectionId={id} />
  },
})
