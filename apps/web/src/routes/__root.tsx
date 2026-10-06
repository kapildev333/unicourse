import { useEffect, useState } from 'react'
import { Outlet, createRootRoute, useNavigate } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ApiError } from '../api/errors'

function Root() {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: (n, e) => !(e instanceof ApiError && e.status < 500) && n < 2,
            refetchOnWindowFocus: false,
          },
        },
      }),
  )
  const navigate = useNavigate()

  useEffect(() => {
    const onUnauthenticated = () => {
      const { pathname, search } = window.location
      if (pathname === '/sign-in') return
      void navigate({ to: '/sign-in', search: { next: pathname + search } })
    }
    window.addEventListener('unicourse:unauthenticated', onUnauthenticated)
    return () => window.removeEventListener('unicourse:unauthenticated', onUnauthenticated)
  }, [navigate])

  return (
    <QueryClientProvider client={queryClient}>
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-surface focus:px-3 focus:py-2 focus:text-sm focus:outline-2 focus:outline-focus"
      >
        Skip to content
      </a>
      <Outlet />
    </QueryClientProvider>
  )
}

export const Route = createRootRoute({ component: Root })
