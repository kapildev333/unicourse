import { File as FileIcon, FileArchive, FileText, Image as ImageIcon, Presentation, type LucideIcon } from 'lucide-react'
import type { UploadPolicy } from './queries'

export const accept = (p: UploadPolicy) => p.allowedExtensions.map((e) => '.' + e).join(',')

/** Text after the last dot, lower case; '' when there is none (as the server reads it). */
export const extension = (name: string) => {
  const i = name.lastIndexOf('.')
  return i < 0 ? '' : name.slice(i + 1).toLowerCase()
}

export function formatSize(n: number): string {
  if (n < 1024) return `${n} B`
  const [v, unit] = n < 1024 ** 2 ? [n / 1024, 'KB'] : [n / 1024 ** 2, 'MB']
  return `${v < 10 ? Number(v.toFixed(1)) : Math.round(v)} ${unit}`
}

export function fileIcon(contentType: string): LucideIcon {
  if (contentType.startsWith('image/')) return ImageIcon
  if (contentType === 'application/zip') return FileArchive
  if (contentType.includes('powerpoint') || contentType.includes('presentationml')) return Presentation
  if (
    contentType === 'application/pdf' ||
    contentType.includes('msword') ||
    contentType.includes('wordprocessingml') ||
    contentType.startsWith('text/plain')
  )
    return FileText
  return FileIcon
}
