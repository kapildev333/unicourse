import { createFileRoute } from '@tanstack/react-router'
import { NotificationsPage } from '../../features/notifications/notifications-page'

export const Route = createFileRoute('/$slug/notifications')({
  component: function Page() {
    const { slug } = Route.useParams()
    return <NotificationsPage slug={slug} />
  },
})
