import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/$slug/courses')({ component: () => <h1>courses</h1> })
