import { createFileRoute } from '@tanstack/react-router'
import { SectionListPage } from '../../../features/courses/section-list'

function Page() {
  const { slug } = Route.useParams()
  const { termId } = Route.useSearch()
  const navigate = Route.useNavigate()
  return (
    <SectionListPage
      slug={slug}
      termId={termId}
      onTermChange={(id) => void navigate({ search: { termId: id || undefined }, replace: true })}
    />
  )
}

export const Route = createFileRoute('/$slug/courses/sections')({
  validateSearch: (s: Record<string, unknown>): { termId?: string } => ({
    termId: typeof s.termId === 'string' ? s.termId : undefined,
  }),
  component: Page,
})
