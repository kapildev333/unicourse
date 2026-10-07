import { createFileRoute } from '@tanstack/react-router'
import { ScaleEditorPage } from '../../features/grades/scale-editor'

function Page() {
  const { slug } = Route.useParams()
  return <ScaleEditorPage slug={slug} />
}

export const Route = createFileRoute('/$slug/grade-scale')({ component: Page })
