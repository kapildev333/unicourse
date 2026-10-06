import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { vi } from 'vitest'

type Reply = { status?: number; body?: unknown }
export type Routes = Record<string, (req: Request) => Reply | Promise<Reply>>

/** Stubs global fetch. Key is `${METHOD} ${path without /api/v1}`; query is ignored. Returns the vi.fn for call inspection. */
export function mockApi(routes: Routes) {
  const fn = vi.fn(async (req: Request) => {
    const key = `${req.method} ${new URL(req.url).pathname.replace('/api/v1', '')}`
    const h = routes[key]
    if (!h) {
      console.error(`mockApi: unmatched request ${key}`)
      return Response.json({ error: { code: 'internal', message: `unmatched ${key}`, details: [] } }, { status: 500 })
    }
    const { status = 200, body } = await h(req.clone())
    return body === undefined ? new Response(null, { status }) : Response.json(body, { status })
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

export function renderWithQuery(ui: React.ReactElement) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return { client, ...render(<QueryClientProvider client={client}>{ui}</QueryClientProvider>) }
}
