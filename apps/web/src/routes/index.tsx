import { Navigate, createFileRoute } from '@tanstack/react-router'
import { ErrorBanner } from '../components/error-banner'
import { errorMessage } from '../components/form-errors'
import { Loading } from '../components/loading'
import { useMe } from '../features/auth/use-me'

function RootRedirect() {
  const me = useMe()
  if (me.isPending) return <Loading />
  if (me.isError) {
    if (me.error.status === 401) return <Loading /> // root listener navigates to sign-in
    return (
      <main id="main" tabIndex={-1} className="mx-auto max-w-6xl px-8 py-6">
        <ErrorBanner onRetry={() => void me.refetch()}>{errorMessage(me.error)}</ErrorBanner>
      </main>
    )
  }
  const first = me.data.memberships[0]
  if (!first) return <Navigate to="/sign-in" search={{ error: 'no_membership' }} replace />
  return <Navigate to="/$slug/me" params={{ slug: first.university.slug }} replace />
}

export const Route = createFileRoute('/')({ component: RootRedirect })
