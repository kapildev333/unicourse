import { useEffect, useState } from 'react'
import { Outlet, createRootRoute, useNavigate } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

function Root() {
  const [queryClient] = useState(() => new QueryClient())
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
      <a href="#main" className="sr-only focus:not-sr-only focus:absolute focus:p-2 focus:bg-white">
        Skip to content
      </a>
      <main id="main" tabIndex={-1} className="p-4">
        <Outlet />
      </main>
    </QueryClientProvider>
  )
}

export const Route = createRootRoute({ component: Root })
