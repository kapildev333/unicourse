const tones = {
  accent: 'border-accent/30 bg-accent-soft text-accent',
  neutral: 'border-line-strong bg-surface text-ink-muted',
  warning: 'border-[#b54708]/30 bg-[#fffaeb] text-[#93370d]',
}

export function Badge({ tone, children }: { tone: keyof typeof tones; children: React.ReactNode }) {
  return (
    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${tones[tone]}`}>
      {children}
    </span>
  )
}

// Task 11 adds instructor / TA / student here.
const roleLabels: Record<string, string> = { admin: 'Admin', dept_admin: 'Department admin', member: 'Member' }
export const roleLabel = (role: string) => roleLabels[role] ?? role

export function RoleBadge({ role }: { role: string }) {
  return <Badge tone={role === 'member' ? 'neutral' : 'accent'}>{roleLabel(role)}</Badge>
}
