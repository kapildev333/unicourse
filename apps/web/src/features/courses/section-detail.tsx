import { Pencil } from 'lucide-react'
import { useState } from 'react'
import { Button } from '../../components/button'
import { StatusMessage } from '../../components/status-message'
import { formatMeeting } from '../../lib/schedule'
import { SectionDescription } from '../section/home'
import type { Section } from '../section/queries'
import { SectionFrame } from '../section/section-frame'
import { useMe, useUniversity } from '../auth/use-me'
import { sectionAccess } from './access'
import { SectionDialog } from './section-dialog'

function EditAction({ section, onClick }: { section: Section; onClick: () => void }) {
  const { membership } = useUniversity()
  const me = useMe().data
  if (!sectionAccess(membership, me?.user.id ?? '', section).manager) return null
  return (
    <Button variant="secondary" onClick={onClick}>
      <Pencil />
      Edit section
    </Button>
  )
}

export function SectionDetailPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  const [editing, setEditing] = useState(false)
  const [notice, setNotice] = useState('')
  return (
    <SectionFrame
      slug={slug}
      sectionId={sectionId}
      bandAction={(section) => <EditAction section={section} onClick={() => setEditing(true)} />}
    >
      {(section) => (
        <>
          <StatusMessage message={notice} />
          {editing && (
            <SectionDialog
              slug={slug}
              term={section.term}
              section={section}
              onClose={() => setEditing(false)}
              onSaved={() => {
                setNotice('Section updated.')
                setEditing(false)
              }}
            />
          )}
          <SectionDescription slug={slug} section={section} />
          <section aria-labelledby="meetings-h">
            <h2 id="meetings-h" className="mb-4 text-xl font-bold">
              Meeting times
            </h2>
            {section.meetings.length > 0 ? (
              <ul className="flex flex-col gap-1 text-sm">
                {section.meetings.map((m) => (
                  <li key={m.id}>{formatMeeting(m, true)}</li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-ink-muted">No meeting times yet.</p>
            )}
          </section>
        </>
      )}
    </SectionFrame>
  )
}
