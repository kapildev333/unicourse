import { useQueryClient } from '@tanstack/react-query'
import { Link, Outlet, createFileRoute } from '@tanstack/react-router'
import { api, unwrap } from '../../api/client'
import { AppShell } from '../../components/app-shell'
import { Button } from '../../components/button'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { useMe } from '../../features/auth/use-me'

function UniversityLayout() {
  const { slug } = Route.useParams()
  const me = useMe()
  const queryClient = useQueryClient()
  if (me.isPending) return <Loading />
  if (me.isError) {
    if (me.error.status === 401) return <Loading />
    return (
      <main id="main" tabIndex={-1} className="mx-auto max-w-6xl px-8 py-6">
        <ErrorBanner onRetry={() => void me.refetch()}>{errorMessage(me.error)}</ErrorBanner>
      </main>
    )
  }
  const membership = me.data.memberships.find((m) => m.university.slug === slug)
  if (!membership) {
    const signOut = async () => {
      try {
        await unwrap(api.POST('/auth/sign-out'))
      } catch {
        // signing out locally anyway
      }
      queryClient.clear()
      window.location.assign('/sign-in')
    }
    return (
      <main id="main" tabIndex={-1} className="mx-auto max-w-md px-4 pt-24">
        <h1 className="text-xl font-semibold tracking-tight">You don't have access to this university</h1>
        <p className="mt-2 text-sm text-ink-muted">
          Your account ({me.data.user.email}) isn't a member of “{slug}”.
        </p>
        <ul className="mt-4 flex flex-col gap-1 text-sm">
          {me.data.memberships.map((m) => (
            <li key={m.university.slug}>
              <Link to="/$slug/me" params={{ slug: m.university.slug }} className="text-accent underline">
                {m.university.name}
              </Link>
            </li>
          ))}
        </ul>
        <Button variant="secondary" size="sm" className="mt-6" onClick={() => void signOut()}>
          Sign out
        </Button>
      </main>
    )
  }
  return (
    <AppShell me={me.data} membership={membership}>
      <Outlet />
    </AppShell>
  )
}

export const Route = createFileRoute('/$slug')({ component: UniversityLayout })
