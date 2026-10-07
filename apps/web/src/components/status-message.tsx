import { Check } from 'lucide-react'

export function StatusMessage({ message }: { message: string }) {
  return (
    <p role="status" className="text-sm text-success">
      {message && (
        <span className="mb-4 inline-flex items-center gap-2 rounded-md bg-success-soft px-3 py-2">
          <Check />
          {message}
        </span>
      )}
    </p>
  )
}
