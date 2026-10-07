import { useQueryClient } from '@tanstack/react-query'
import { LogOut } from 'lucide-react'
import { Link, Outlet, createFileRoute } from '@tanstack/react-router'
import { EntryLayout } from '../../components/entry-layout'
import { AppShell } from '../../components/app-shell'
import { Button } from '../../components/button'
import { ButtonLink } from '../../components/button-link'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { signOut } from '../../features/auth/sign-out'
import { useMe } from '../../features/auth/use-me'

function UniversityLayout() {
  const { slug } = Route.useParams()
  const me = useMe()
  const queryClient = useQueryClient()
  if (me.isPending)
    return (
      <div className="mx-auto max-w-336 px-6 py-8">
        <Loading />
      </div>
    )
  if (me.isError) {
    if (me.error.status === 401)
      return (
        <div className="mx-auto max-w-336 px-6 py-8">
          <Loading />
        </div>
      )
    return (
      <main id="main" tabIndex={-1} className="mx-auto max-w-336 px-6 py-8">
        <ErrorBanner onRetry={() => void me.refetch()}>{errorMessage(me.error)}</ErrorBanner>
      </main>
    )
  }
  const membership = me.data.memberships.find((m) => m.university.slug === slug)
  if (!membership) {
    return (
      <EntryLayout>
        <h1 className="text-2xl text-ink">You don't have access to this university</h1>
        <p className="mt-2 text-sm text-ink-muted">
          Your account ({me.data.user.email}) isn't a member of this university's address.
        </p>
        {me.data.memberships.length !== 1 && (
          <>
            <p className="mt-4 text-sm text-ink">Go to your university:</p>
            <ul className="mt-2 flex flex-col gap-1 text-sm">
              {me.data.memberships.map((m) => (
                <li key={m.university.slug}>
                  <Link
                    to="/$slug/me"
                    params={{ slug: m.university.slug }}
                    className="text-accent underline hover:text-accent-hover"
                  >
                    {m.university.name}
                  </Link>
                </li>
              ))}
            </ul>
          </>
        )}
        <div className="mt-6 flex flex-wrap gap-3">
          {me.data.memberships.length === 1 && (
            <ButtonLink to="/$slug/me" params={{ slug: me.data.memberships[0].university.slug }}>
              Go to {me.data.memberships[0].university.name}
            </ButtonLink>
          )}
          <Button variant="secondary" onClick={() => void signOut(queryClient)}>
            <LogOut size={16} aria-hidden="true" />
            Sign out
          </Button>
        </div>
      </EntryLayout>
    )
  }
  return (
    <AppShell me={me.data} membership={membership}>
      <Outlet />
    </AppShell>
  )
}

export const Route = createFileRoute('/$slug')({ component: UniversityLayout })
