import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/$slug/admin')({ component: () => <h1>admin</h1> })
