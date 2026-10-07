import { createFileRoute } from '@tanstack/react-router'
import { MyGradesPage } from '../../features/grades/my-grades'

export const Route = createFileRoute('/$slug/grades')({
  component: function Page() {
    const { slug } = Route.useParams()
    return <MyGradesPage slug={slug} />
  },
})
