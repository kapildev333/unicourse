import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/$slug/sections/$id')({ component: () => <h1>section</h1> })
