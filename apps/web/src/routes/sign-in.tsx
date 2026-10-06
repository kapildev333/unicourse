import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/sign-in')({
  validateSearch: (s: Record<string, unknown>): { next?: string } => ({
    next: typeof s.next === 'string' ? s.next : undefined,
  }),
  component: () => <h1>sign-in</h1>,
})
