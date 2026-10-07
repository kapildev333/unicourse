import { useMe, useUniversity } from '../auth/use-me'
import { sectionAccess } from '../courses/access'
import { Roster } from '../courses/roster'
import { StaffEditor } from '../courses/staff-editor'
import type { Section } from './queries'
import { SectionFrame } from './section-frame'

const h2 = 'mb-4 text-xl font-bold'

export function ParticipantsPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  return (
    <SectionFrame slug={slug} sectionId={sectionId}>
      {(section) => <Participants slug={slug} section={section} />}
    </SectionFrame>
  )
}

function Participants({ slug, section }: { slug: string; section: Section }) {
  const { membership } = useUniversity()
  const me = useMe().data
  const access = sectionAccess(membership, me?.user.id ?? '', section)
  return (
    <>
      <section aria-labelledby="staff-h">
        <h2 id="staff-h" className={h2}>
          Staff <span className="font-normal text-ink-muted">{section.staff.length + section.pendingStaff.length}</span>
        </h2>
        <StaffEditor
          slug={slug}
          sectionId={section.id}
          sectionLabel={`${section.course.code} section ${section.number}`}
          staff={section.staff}
          pendingStaff={section.pendingStaff}
          canManageInstructors={access.canManageInstructors}
          canManageTAs={access.canManageTAs}
        />
      </section>
      {access.canViewRoster && (
        <section aria-labelledby="roster-h" className="mt-8 border-t border-line pt-8">
          <Roster slug={slug} sectionId={section.id} canEdit={access.canEditRoster} />
        </section>
      )}
    </>
  )
}
