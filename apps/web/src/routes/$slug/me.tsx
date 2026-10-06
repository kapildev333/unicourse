import { createFileRoute } from '@tanstack/react-router'
import { EmptyState } from '../../components/empty-state'
import { PageHeader } from '../../components/page-header'

// P2-L3 replaces the body.
export const Route = createFileRoute('/$slug/me')({
  component: () => (
    <>
      <PageHeader title="My sections" />
      <EmptyState title="No sections yet" body="Sections you teach, assist or take will appear here." />
    </>
  ),
})
