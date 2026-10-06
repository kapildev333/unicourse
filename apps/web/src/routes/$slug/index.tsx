import { Navigate, createFileRoute } from '@tanstack/react-router'

function SlugIndex() {
  const { slug } = Route.useParams()
  return <Navigate to="/$slug/me" params={{ slug }} replace />
}

export const Route = createFileRoute('/$slug/')({ component: SlugIndex })
