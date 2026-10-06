import { api, unwrap } from './client'
import type { components } from './types'

export type ImportPreview = components['schemas']['ImportPreview']
export type ImportCounts = components['schemas']['ImportCounts']
export type ImportTarget =
  | { kind: 'courses'; slug: string }
  | { kind: 'sections'; slug: string }
  | { kind: 'enrollments'; slug: string; sectionId: string }

// fetch sets the multipart boundary itself, so no Content-Type here.
const opts = (form: FormData) => ({ body: form as never, bodySerializer: (b: unknown) => b as FormData })

function form(file: File, checksum?: string) {
  const f = new FormData()
  f.append('file', file)
  if (checksum !== undefined) f.append('checksum', checksum)
  return f
}

export function previewImport(t: ImportTarget, file: File): Promise<ImportPreview> {
  const o = opts(form(file))
  switch (t.kind) {
    case 'courses':
      return unwrap(api.POST('/universities/{slug}/courses/import/preview', { params: { path: { slug: t.slug } }, ...o }))
    case 'sections':
      return unwrap(api.POST('/universities/{slug}/sections/import/preview', { params: { path: { slug: t.slug } }, ...o }))
    case 'enrollments':
      return unwrap(
        api.POST('/universities/{slug}/sections/{id}/enrollments/import/preview', {
          params: { path: { slug: t.slug, id: t.sectionId } },
          ...o,
        }),
      )
  }
}

export function commitImport(t: ImportTarget, file: File, checksum: string): Promise<ImportPreview> {
  const o = opts(form(file, checksum))
  switch (t.kind) {
    case 'courses':
      return unwrap(api.POST('/universities/{slug}/courses/import', { params: { path: { slug: t.slug } }, ...o }))
    case 'sections':
      return unwrap(api.POST('/universities/{slug}/sections/import', { params: { path: { slug: t.slug } }, ...o }))
    case 'enrollments':
      return unwrap(
        api.POST('/universities/{slug}/sections/{id}/enrollments/import', {
          params: { path: { slug: t.slug, id: t.sectionId } },
          ...o,
        }),
      )
  }
}
