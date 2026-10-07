import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate, useRouterState } from '@tanstack/react-router'
import { LogOut, Search } from 'lucide-react'
import { useEffect, useRef } from 'react'
import { signOut } from '../features/auth/sign-out'
import type { Me, MyMembership } from '../features/auth/use-me'
import { Button } from './button'
import { inputClass } from './form-field'
import { ThemeSwitch } from './theme-switch'

const navBase =
  'inline-flex min-h-11 items-center text-sm decoration-2 underline-offset-8 transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus md:min-h-10'

// Current-item rules (S5-05): Courses also covers section detail, Admin also covers the grade scale.
function NavLink(props: { to: string; slug: string; match: RegExp; children: React.ReactNode }) {
  const path = useRouterState({ select: (s) => s.location.pathname })
  const current = props.match.test(path)
  return (
    <Link
      to={props.to}
      params={{ slug: props.slug }}
      aria-current={current ? 'page' : undefined}
      className={`${navBase} ${current ? 'font-bold text-ink underline decoration-accent' : 'font-medium text-ink-muted hover:text-accent-hover'}`}
    >
      {props.children}
    </Link>
  )
}

function HeaderSearch({ slug }: { slug: string }) {
  const navigate = useNavigate()
  const input = useRef<HTMLInputElement>(null)
  // the Catalog's q shows in the field (S5-01); keyed so back/clear resets it
  const q = useRouterState({
    select: (s) => (s.location.pathname.endsWith('/catalog') ? (s.location.search as { q?: string }).q : undefined),
  })
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target as HTMLElement | null
      if (e.key !== '/' || e.metaKey || e.ctrlKey || e.altKey) return
      if (t && (t.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(t.tagName))) return
      e.preventDefault()
      input.current?.focus()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])
  return (
    <form
      role="search"
      aria-label="Catalog"
      className="relative order-last w-full basis-full max-w-xl min-w-0 xl:order-3 xl:mx-auto xl:min-w-64 xl:flex-1 xl:basis-auto"
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
        key={q ?? ''}
        defaultValue={q ?? ''}
        ref={input}
        aria-keyshortcuts="/"
        title="Press / to search"
        name="q"
        type="search"
        maxLength={100}
        aria-label="Find a course"
        placeholder="Find a course"
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
        <div className="mx-auto flex min-h-18 max-w-336 flex-wrap items-center gap-x-6 gap-y-2 px-6 py-2 md:gap-y-3 md:py-3 xl:flex-nowrap xl:py-0">
          <div className="order-1 flex shrink-0 items-center gap-3">
            <span className="text-2xl leading-none font-bold tracking-tight text-ink">UniCourse</span>
            {me.memberships.length === 1 && (
              <>
                <span aria-hidden="true" className="hidden h-6 w-px bg-line md:block" />
                <span
                  className="hidden max-w-48 min-w-0 truncate text-sm text-ink-muted md:inline"
                  title={name}
                >
                  {name}
                </span>
              </>
            )}
          </div>
          {me.memberships.length > 1 && (
            <select
              aria-label="University"
              className={
                inputClass + ' order-3 max-w-full min-w-0 max-md:w-40 md:w-40 xl:order-4 xl:shrink-0 2xl:w-56'
              }
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
          <nav
            aria-label="Main"
            className="order-3 flex flex-wrap items-center gap-x-4 md:order-2 md:mr-auto md:shrink-0 xl:mr-0"
          >
            <NavLink to="/$slug/me" slug={slug} match={/^\/[^/]+\/me\/?$/}>
              My sections
            </NavLink>
            <NavLink to="/$slug/catalog" slug={slug} match={/^\/[^/]+\/catalog\/?$/}>
              Catalog
            </NavLink>
            {isAdmin && (
              <NavLink to="/$slug/courses" slug={slug} match={/^\/[^/]+\/(courses|sections\/[^/]+(\/(announcements|materials))?\/?$)/}>
                Courses
              </NavLink>
            )}
            {isAdmin && (
              <NavLink to="/$slug/admin" slug={slug} match={/^\/[^/]+\/(admin|grade-scale)(\/|$)/}>
                Admin
              </NavLink>
            )}
          </nav>
          <HeaderSearch slug={slug} />
          <div className="order-2 ml-auto flex shrink-0 items-center gap-3 md:order-5 md:ml-0">
            <ThemeSwitch />
            <Button
              variant="secondary"
              aria-label="Sign out"
              title={`Sign out (${me.user.email})`}
              className="max-md:w-10 max-md:px-0"
              onClick={() => void signOut(queryClient)}
            >
              <LogOut />
              <span className="hidden md:inline">Sign out</span>
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
