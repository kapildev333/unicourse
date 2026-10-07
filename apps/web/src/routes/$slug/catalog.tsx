import { createFileRoute } from '@tanstack/react-router'
import { CatalogPage } from '../../features/catalog/catalog'

const str = (v: unknown) => (typeof v === 'string' ? v : undefined)

export const Route = createFileRoute('/$slug/catalog')({
  validateSearch: (s: Record<string, unknown>): { termId?: string; departmentId?: string; q?: string } => ({
    termId: str(s.termId),
    departmentId: str(s.departmentId),
    q: str(s.q) || undefined,
  }),
  component: function Catalog() {
    const { slug } = Route.useParams()
    const { termId, departmentId, q } = Route.useSearch()
    const navigate = Route.useNavigate()
    return (
      <CatalogPage
        slug={slug}
        termId={termId}
        departmentId={departmentId}
        q={q}
        onSearchChange={(p) => void navigate({ search: (s) => ({ ...s, ...p }), replace: true })}
      />
    )
  },
})
