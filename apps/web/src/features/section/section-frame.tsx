import { Link } from '@tanstack/react-router'
import { ArrowLeft, Users } from 'lucide-react'
import { RoleBadge } from '../../components/badge'
import { ErrorBanner } from '../../components/error-banner'
import { errorMessage } from '../../components/form-errors'
import { Loading } from '../../components/loading'
import { PageHeader } from '../../components/page-header'
import { personName } from '../courses/access'
import { useMe, useUniversity } from '../auth/use-me'
import { useSection } from '../grades/queries'
import { useContentAccess, type ContentAccess } from './access'
import type { Section } from './queries'
import { SectionTabs } from './section-tabs'

const backClass = 'mb-3 inline-flex items-center gap-1 text-sm font-medium text-accent hover:underline'

function BackLink({ slug, termId }: { slug: string; termId?: string }) {
  const { membership } = useUniversity()
  return membership.role === 'admin' || membership.role === 'dept_admin' ? (
    <Link to="/$slug/courses/sections" params={{ slug }} search={{ termId }} className={backClass}>
      <ArrowLeft />
      Sections
    </Link>
  ) : (
    <Link to="/$slug/me" params={{ slug }} className={backClass}>
      <ArrowLeft />
      My sections
    </Link>
  )
}

/** The title band of section-detail.tsx without its edit action. */
export function SectionBand({ slug, section, ownRole }: { slug: string; section: Section; ownRole?: string }) {
  const names = section.staff.filter((s) => s.role === 'instructor').map(personName)
  return (
    <div className="mb-8 rounded border border-line bg-surface px-6 pt-6">
      <BackLink slug={slug} termId={section.termId} />
      <PageHeader
        title={`${section.course.code} · Section ${section.number}`}
        description={
          <>
            {section.course.title} · {section.term.name}
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
  )
}

function Loaded(props: {
  slug: string
  section: Section
  children: (section: Section, access: ContentAccess) => React.ReactNode
}) {
  const { slug, section, children } = props
  const access = useContentAccess(slug, section)
  return (
    <>
      <SectionBand slug={slug} section={section} ownRole={access.ownRole} />
      <SectionTabs slug={slug} section={section} />
      {children(section, access)}
    </>
  )
}

/** Band + strip + the tab's content. Owns the section query and its loading and error states. */
export function SectionFrame(props: {
  slug: string
  sectionId: string
  children: (section: Section, access: ContentAccess) => React.ReactNode
}) {
  const { slug, sectionId, children } = props
  const q = useSection(slug, sectionId)
  const me = useMe().data
  if (q.isPending || !me) return <Loading />
  if (q.error) {
    const title = q.error.status === 404 ? 'Section not found' : q.error.status === 403 ? 'No access' : undefined
    const copy = {
      not_found: "This section doesn't exist or was deleted.",
      forbidden: "You aren't staff on or enrolled in this section.",
    }
    return (
      <>
        <BackLink slug={slug} />
        <ErrorBanner title={title} onRetry={title ? undefined : () => void q.refetch()}>
          {errorMessage(q.error, copy)}
        </ErrorBanner>
      </>
    )
  }
  return (
    <Loaded slug={slug} section={q.data}>
      {children}
    </Loaded>
  )
}
