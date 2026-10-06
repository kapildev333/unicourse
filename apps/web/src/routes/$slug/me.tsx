import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/$slug/me')({ component: () => <h1>me</h1> })
