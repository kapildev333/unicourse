import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { api, unwrap } from '../../api/client'
import type { ApiError } from '../../api/errors'
import type { components } from '../../api/types'
import { RoleBadge } from '../../components/badge'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { useMe, useUniversity } from '../auth/use-me'
import { sectionAccess } from './access'
import { Roster } from './roster'
import { StaffEditor } from './staff-editor'

type Section = components['schemas']['Section']

export function SectionDetailPage({ slug, sectionId }: { slug: string; sectionId: string }) {
  const { membership } = useUniversity()
  const me = useMe().data
  const q = useQuery<Section, ApiError>({
    queryKey: [slug, 'sections', 'detail', sectionId],
    queryFn: () =>
      unwrap(api.GET('/universities/{slug}/sections/{id}', { params: { path: { slug, id: sectionId } } })),
  })
  const canList = membership.role === 'admin' || membership.role === 'dept_admin'
  const back = canList ? (
    <Link
      to="/$slug/courses/sections"
      params={{ slug }}
      search={{ termId: q.data?.termId }}
      className="mb-2 inline-block text-sm text-accent hover:underline"
    >
      ← Sections
    </Link>
  ) : (
    <Link to="/$slug/me" params={{ slug }} className="mb-2 inline-block text-sm text-accent hover:underline">
      ← My sections
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
  const ownRole = access.staffRole ?? (access.manager ? undefined : 'student')
  return (
    <>
      {back}
      <PageHeader
        title={`${section.course.code} · Section ${section.number}`}
        description={`${section.course.title} — ${section.term.name}`}
        action={
          ownRole && (
            <span>
              <span className="sr-only">You are: </span>
              <RoleBadge role={ownRole} />
            </span>
          )
        }
      />
      <section aria-labelledby="staff-h">
        <h2 id="staff-h" className="mb-3 text-base font-semibold">
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
        <section aria-labelledby="roster-h" className="mt-8">
          <h2 id="roster-h" className="mb-3 text-base font-semibold">
            Roster
          </h2>
          <Roster slug={slug} sectionId={section.id} canEdit={access.canEditRoster} />
        </section>
      )}
    </>
  )
}
