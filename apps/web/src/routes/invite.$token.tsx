import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/invite/$token')({ component: () => <h1>invite</h1> })
