import { createFileRoute } from '@tanstack/react-router'
import { CoursesLayout } from '../../features/courses/courses-layout'

export const Route = createFileRoute('/$slug/courses')({ component: CoursesLayout })
