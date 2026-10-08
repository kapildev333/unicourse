import { ApiError } from '../../api/errors'
import { errorMessage } from '../../components/form-errors'
import { sentence } from '../grades/grade'
import { extension, formatSize } from './files'
import type { UploadPolicy } from './queries'

export type Failed = { name: string; reason: string }

/** Reason a file is refused before sending, or undefined. Order: type, empty, size. Without a policy the server decides. */
export function precheck(f: File, policy?: UploadPolicy): string | undefined {
  if (policy && !policy.allowedExtensions.includes(extension(f.name))) return "This file type can't be uploaded."
  if (f.size === 0) return 'The file is empty.'
  if (policy && f.size > policy.maxBytes) return `It is larger than ${formatSize(policy.maxBytes)}.`
}

/** Sends the files that pass precheck one at a time, in order; a failure never stops the rest. */
export async function runUploads(
  files: File[],
  policy: UploadPolicy | undefined,
  send: (f: File) => Promise<unknown>,
  onProgress: (line: string) => void,
): Promise<{ done: number; total: number; failed: Failed[] }> {
  const failed: Failed[] = []
  const ok: File[] = []
  for (const f of files) {
    const r = precheck(f, policy)
    if (r) failed.push({ name: f.name, reason: r })
    else ok.push(f)
  }
  let done = 0
  for (const [i, f] of ok.entries()) {
    onProgress(`Uploading ${i + 1} of ${ok.length}: ${f.name}`)
    try {
      await send(f)
      done++
    } catch (e) {
      failed.push({ name: f.name, reason: uploadReason(e) })
    }
  }
  return { done, total: files.length, failed }
}

export function uploadReason(e: unknown): string {
  if (!(e instanceof ApiError)) return 'The upload was interrupted. Check your connection and try again.'
  const d = e.details[0]
  if (e.code === 'validation_failed' && d) {
    if (d.field === 'materials') return 'This week already has 200 files.'
    if (d.message.startsWith('is larger than')) return `It is larger than ${d.message.slice('is larger than '.length)}.`
    if (d.message === 'is empty') return 'The file is empty.'
    if (d.message.startsWith('file type not allowed')) return "This file type can't be uploaded."
    return sentence(d.message)
  }
  return errorMessage(e, {
    forbidden: "Only this section's staff and administrators can change its materials.",
  })
}

export function finalLine(r: { done: number; total: number }, week: string): string {
  if (r.done === r.total) return `Uploaded ${r.done} ${r.done === 1 ? 'file' : 'files'} to ${week}.`
  if (r.done === 0) return `No files were uploaded to ${week}.`
  return `Uploaded ${r.done} of ${r.total} files to ${week}.`
}
