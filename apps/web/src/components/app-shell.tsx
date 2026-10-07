import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { LogOut, Search } from 'lucide-react'
import { signOut } from '../features/auth/sign-out'
import type { Me, MyMembership } from '../features/auth/use-me'
import { Button } from './button'
import { inputClass } from './form-field'
import { ThemeSwitch } from './theme-switch'

const navLink = {
  className:
    'inline-flex h-10 items-center text-sm decoration-2 underline-offset-8 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus',
  activeProps: {
    className: 'font-bold text-ink underline decoration-accent',
    'aria-current': 'page' as const,
  },
  inactiveProps: {
    className: 'font-medium text-ink-muted hover:text-accent-hover',
  },
}

function HeaderSearch({ slug }: { slug: string }) {
  const navigate = useNavigate()
  return (
    <form
      role="search"
      aria-label="Catalog"
      className="relative order-last basis-full xl:order-none xl:min-w-48 xl:flex-1 xl:basis-auto"
      onSubmit={(e) => {
        e.preventDefault()
        const q = String(new FormData(e.currentTarget).get('q') ?? '').trim()
        void navigate({
          to: '/$slug/catalog',
          params: { slug },
          search: { q: q || undefined },
        })
      }}
    >
      <Search className="pointer-events-none absolute top-1/2 left-4 -translate-y-1/2 text-ink-muted" />
      <input
        name="q"
        type="search"
        maxLength={100}
        aria-label="Find a course"
        placeholder="Search the catalog"
        className="h-10 w-full rounded-full border border-line-strong bg-canvas pr-4 pl-11 text-sm text-ink placeholder:text-ink-muted transition-colors hover:border-ink focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus"
      />
    </form>
  )
}

export function AppShell(props: { me: Me; membership: MyMembership; children: React.ReactNode }) {
  const { me, membership, children } = props
  const slug = membership.university.slug
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const name = membership.university.name
  const isAdmin = membership.role === 'admin' || membership.role === 'dept_admin'

  return (
    <>
      <header className="border-b border-line bg-surface">
        <div className="mx-auto flex min-h-18 max-w-336 flex-wrap items-center gap-x-6 gap-y-3 px-6 py-3 xl:flex-nowrap xl:py-0">
          <div className="flex min-w-0 items-center gap-3">
            <span className="text-2xl leading-none font-bold tracking-tight text-ink">UniCourse</span>
            {me.memberships.length === 1 && (
              <>
                <span aria-hidden="true" className="h-6 w-px bg-line" />
                <span className="min-w-0 max-w-48 truncate text-sm text-ink-muted" title={name}>
                  {name}
                </span>
              </>
            )}
          </div>
          <nav aria-label="Main" className="flex shrink-0 items-center gap-4">
            <Link to="/$slug/me" params={{ slug }} {...navLink}>
              My sections
            </Link>
            <Link to="/$slug/catalog" params={{ slug }} {...navLink}>
              Catalog
            </Link>
            {isAdmin && (
              <Link to="/$slug/courses" params={{ slug }} {...navLink}>
                Courses
              </Link>
            )}
            {isAdmin && (
              <Link to="/$slug/admin" params={{ slug }} {...navLink}>
                Admin
              </Link>
            )}
          </nav>
          <HeaderSearch slug={slug} />
          <div className="ml-auto flex shrink-0 items-center gap-3">
            {me.memberships.length > 1 && (
              <select
                aria-label="University"
                className={inputClass + ' w-40 min-w-0 xl:w-56'}
                value={slug}
                onChange={(e) =>
                  void navigate({
                    to: '/$slug/me',
                    params: { slug: e.target.value },
                  })
                }
              >
                {me.memberships.map((m) => (
                  <option key={m.university.slug} value={m.university.slug}>
                    {m.university.name}
                  </option>
                ))}
              </select>
            )}
            <ThemeSwitch />
            <Button
              variant="secondary"
              title={`Signed in as ${me.user.email}`}
              onClick={() => void signOut(queryClient)}
            >
              <LogOut />
              Sign out
            </Button>
          </div>
        </div>
      </header>
      <main id="main" tabIndex={-1} className="mx-auto max-w-336 px-6 py-8">
        {children}
      </main>
    </>
  )
}
