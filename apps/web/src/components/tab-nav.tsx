export const tabLink = {
  className:
    '-mb-px inline-flex h-12 shrink-0 snap-start whitespace-nowrap items-center border-b-2 text-sm font-bold transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus',
  activeProps: { className: 'border-accent text-ink' },
  inactiveProps: {
    className: 'border-transparent text-ink-muted hover:border-line-strong hover:text-ink',
  },
}

export function TabNav({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <nav aria-label={label} className="mb-8 flex snap-x gap-6 overflow-x-auto border-b border-line">
      {children}
    </nav>
  )
}
