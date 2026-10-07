import { Pencil } from 'lucide-react'
import { useState } from 'react'
import { Button } from '../../components/button'
import { StatusMessage } from '../../components/status-message'
import { useContentAccess } from './access'
import { DescriptionDialog } from './description-dialog'
import { PlainText } from './plain-text'
import type { Section } from './queries'

export function SectionDescription({ slug, section }: { slug: string; section: Section }) {
  const { canEdit } = useContentAccess(slug, section)
  const [editing, setEditing] = useState(false)
  const [notice, setNotice] = useState('')
  const text = section.description ?? '' // old fixtures have no field
  if (!text && !canEdit) return null
  return (
    <section aria-labelledby="about-h" className="mb-8 border-b border-line pb-8">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <h2 id="about-h" className="text-xl font-bold">
          About this section
        </h2>
        {canEdit && (
          <Button variant="secondary" size="sm" onClick={() => setEditing(true)}>
            <Pencil aria-hidden="true" />
            {text ? 'Edit description' : 'Add description'}
          </Button>
        )}
      </div>
      {canEdit && <StatusMessage message={notice} />}
      {text ? (
        <PlainText text={text} className="max-w-prose text-base" />
      ) : (
        <p className="text-sm text-ink-muted">
          No description yet. Add one to tell students what this section covers and how it runs.
        </p>
      )}
      {editing && (
        <DescriptionDialog
          slug={slug}
          sectionId={section.id}
          initial={text}
          onClose={() => setEditing(false)}
          onSaved={(m) => {
            setEditing(false)
            setNotice(m)
          }}
        />
      )}
    </section>
  )
}
