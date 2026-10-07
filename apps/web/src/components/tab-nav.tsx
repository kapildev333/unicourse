export const tabLink = {
  className:
    '-mb-px inline-flex h-10 items-center border-b-2 px-3 text-sm transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus',
  activeProps: { className: 'border-accent font-semibold text-accent' },
  inactiveProps: {
    className: 'border-transparent font-medium text-ink-muted hover:border-line hover:text-ink',
  },
}

export function TabNav({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <nav aria-label={label} className="mb-6 flex gap-1 overflow-x-auto border-b border-line">
      {children}
    </nav>
  )
}
