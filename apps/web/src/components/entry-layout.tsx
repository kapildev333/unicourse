export function EntryLayout({ children }: { children: React.ReactNode }) {
  return (
    <main id="main" tabIndex={-1} className="flex min-h-screen flex-col items-center bg-canvas px-4 pt-24 pb-12">
      <p className="mb-6 font-serif text-3xl leading-none tracking-tight text-ink">UniCourse</p>
      <div className="w-full max-w-sm rounded-lg border border-line bg-surface p-8">{children}</div>
    </main>
  )
}
