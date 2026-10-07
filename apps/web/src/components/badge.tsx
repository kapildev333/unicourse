const tones = {
  accent: 'border-transparent bg-accent-soft text-ink',
  neutral: 'border-line bg-surface text-ink-muted',
  danger: 'border-transparent bg-danger-soft text-danger',
  warning: 'border-transparent bg-warning-soft text-warning',
}

export function Badge(props: {
  tone: keyof typeof tones
  className?: string
  children: React.ReactNode
}) {
  const { tone, className = '', children } = props
  return (
    <span
      className={`inline-flex items-center gap-1 rounded border px-2 py-0.5 text-xs font-bold ${tones[tone]} ${className}`}
    >
      {children}
    </span>
  )
}

const roleLabels: Record<string, string> = {
  admin: 'Admin',
  dept_admin: 'Department admin',
  member: 'Member',
  instructor: 'Instructor',
  ta: 'TA',
  student: 'Student',
}
export const roleLabel = (role: string) => roleLabels[role] ?? role

export function RoleBadge({ role }: { role: string }) {
  return (
    <Badge tone={role === 'member' || role === 'student' ? 'neutral' : 'accent'}>
      {roleLabel(role)}
    </Badge>
  )
}

const actions = {
  add: { label: 'Add', tone: 'accent' },
  exists: { label: 'Exists', tone: 'neutral' },
  invite: { label: 'Invite', tone: 'warning' },
  error: { label: 'Error', tone: 'danger' },
} as const

export function ActionBadge({ action }: { action: keyof typeof actions }) {
  const a = actions[action]
  return <Badge tone={a.tone}>{a.label}</Badge>
}
