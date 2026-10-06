import { createFileRoute } from '@tanstack/react-router'
import { MySectionsPage } from '../../features/my-sections/my-sections'

export const Route = createFileRoute('/$slug/me')({
  component: function Me() {
    const { slug } = Route.useParams()
    return <MySectionsPage slug={slug} />
  },
})
