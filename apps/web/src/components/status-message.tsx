export function StatusMessage({ message }: { message: string }) {
  return (
    <p role="status" className="min-h-0 text-sm text-success">
      {message && <span className="mb-4 inline-flex rounded-md bg-success-soft px-3 py-2">✓ {message}</span>}
    </p>
  )
}
