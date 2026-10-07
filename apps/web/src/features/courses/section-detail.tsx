import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, Users } from 'lucide-react'
import { api, unwrap } from '../../api/client'
import type { ApiError } from '../../api/errors'
import type { components } from '../../api/types'
import { RoleBadge } from '../../components/badge'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { formatMeeting } from '../../lib/schedule'
import { useMySections } from '../my-sections/my-sections'
import { useMe, useUniversity } from '../auth/use-me'
import { personName, sectionAccess } from './access'
import { Roster } from './roster'
import { StaffEditor } from './staff-editor'

type Section = components['schemas']['Section']

const backClass = 'mb-3 inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline'

export function SectionDetailPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  const { membership } = useUniversity()
  const me = useMe().data
  const q = useQuery<Section, ApiError>({
    queryKey: [slug, 'sections', 'detail', sectionId],
    queryFn: () =>
      unwrap(api.GET('/universities/{slug}/sections/{id}', { params: { path: { slug, id: sectionId } } })),
  })
  const mine = useMySections(slug).data
  const canList = membership.role === 'admin' || membership.role === 'dept_admin'
  const back = canList ? (
    <Link
      to="/$slug/courses/sections"
      params={{ slug }}
      search={{ termId: q.data?.termId }}
      className={backClass}
    >
      <ArrowLeft />
      Sections
    </Link>
  ) : (
    <Link to="/$slug/me" params={{ slug }} className={backClass}>
      <ArrowLeft />
      My sections
    </Link>
  )

  if (q.isPending || !me) return <Loading />
  if (q.error) {
    const title = q.error.status === 404 ? 'Section not found' : q.error.status === 403 ? 'No access' : undefined
    const copy = {
      not_found: "This section doesn't exist or was deleted.",
      forbidden: "You aren't staff on or enrolled in this section.",
    }
    return (
      <>
        {back}
        <ErrorBanner title={title} onRetry={title ? undefined : () => void q.refetch()}>
          {errorMessage(q.error, copy)}
        </ErrorBanner>
      </>
    )
  }

  const section = q.data
  const access = sectionAccess(membership, me.user.id, section)
  const label = `${section.course.code} section ${section.number}`
  const ownRole = access.staffRole ?? mine?.terms.flatMap((g) => g.sections).find((s) => s.id === section.id)?.myRole
  const names = section.staff.filter((s) => s.role === 'instructor').map(personName)
  const h2 = 'mb-4 text-xl font-bold'
  const rule = 'mt-8 border-t border-line pt-8'
  return (
    <>
      <div className="mb-8 rounded border border-line bg-surface px-6 pt-6">
        {back}
        <PageHeader
          title={`${section.course.code} · Section ${section.number}`}
          description={
            <>
              {section.course.title} — {section.term.name}
              {names.length > 0 && (
                <span className="mt-1 flex items-center gap-2">
                  <Users />
                  {names.join(', ')}
                </span>
              )}
            </>
          }
          action={
            ownRole && (
              <span>
                <span className="sr-only">You are: </span>
                <RoleBadge role={ownRole} />
              </span>
            )
          }
        />
      </div>
      <section aria-labelledby="meetings-h">
        <h2 id="meetings-h" className={h2}>
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
      <section aria-labelledby="staff-h" className={rule}>
        <h2 id="staff-h" className={h2}>
          Staff
        </h2>
        <StaffEditor
          slug={slug}
          sectionId={section.id}
          sectionLabel={label}
          staff={section.staff}
          pendingStaff={section.pendingStaff}
          canManageInstructors={access.canManageInstructors}
          canManageTAs={access.canManageTAs}
        />
      </section>
      {access.canViewRoster && (
        <section aria-labelledby="roster-h" className={rule}>
          <h2 id="roster-h" className={h2}>
            Roster
          </h2>
          <Roster slug={slug} sectionId={section.id} canEdit={access.canEditRoster} />
        </section>
      )}
    </>
  )
}
