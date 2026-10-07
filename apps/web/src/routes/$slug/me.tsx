import { createFileRoute } from '@tanstack/react-router'
import { MySectionsPage } from '../../features/my-sections/my-sections'

export const Route = createFileRoute('/$slug/me')({
  validateSearch: (s: Record<string, unknown>): { termId?: string; view?: 'schedule' } => ({
    termId: typeof s.termId === 'string' ? s.termId : undefined,
    view: s.view === 'schedule' ? 'schedule' : undefined,
  }),
  component: function Me() {
    const { slug } = Route.useParams()
    const { termId, view } = Route.useSearch()
    const navigate = Route.useNavigate()
    return (
      <MySectionsPage
        slug={slug}
        termId={termId}
        view={view}
        onSearchChange={(p) => void navigate({ search: (s) => ({ ...s, ...p }), replace: true })}
      />
    )
  },
})
