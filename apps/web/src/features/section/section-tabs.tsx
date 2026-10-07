import { Link } from '@tanstack/react-router'
import { TabNav, tabLink } from '../../components/tab-nav'
import { useContentAccess } from './access'
import type { Section } from './queries'

export function SectionTabs({ slug, section }: { slug: string; section: Section }) {
  const a = useContentAccess(slug, section)
  if (!a.canRead) return null // Home would be the only tab
  const params = { slug, id: section.id }
  return (
    <TabNav label="Section pages">
      <Link to="/$slug/sections/$id" params={params} activeOptions={{ exact: true }} {...tabLink}>
        Home
      </Link>
      <Link to="/$slug/sections/$id/announcements" params={params} {...tabLink}>
        Announcements
      </Link>
      <Link to="/$slug/sections/$id/materials" params={params} {...tabLink}>
        Materials
      </Link>
      <Link to="/$slug/sections/$id/participants" params={params} {...tabLink}>
        Participants
      </Link>
      {a.canGrades && (
        <Link to="/$slug/sections/$id/grades" params={params} {...tabLink}>
          Grades
        </Link>
      )}
    </TabNav>
  )
}
